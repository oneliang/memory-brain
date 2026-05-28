package summary

import (
	"fmt"
	"strings"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// Generator generates session summaries from observations
type Generator struct{}

// NewGenerator creates a new summary generator
func NewGenerator() *Generator {
	return &Generator{}
}

// BuildSyntheticSummary generates a summary from observations without LLM
func (g *Generator) BuildSyntheticSummary(observations []types.Observation) *types.SessionSummary {
	if len(observations) == 0 {
		return nil
	}

	// Extract components
	title := g.extractTitle(observations)
	facts := g.extractFacts(observations)
	concepts := g.extractConcepts(observations)
	narrative := g.buildNarrative(observations)
	files := g.extractFiles(observations)
	importance := g.calculateImportance(observations)

	// Get session info from first observation
	var sessionID, userID string
	for _, obs := range observations {
		if obs.SessionID != "" {
			sessionID = obs.SessionID
		}
		if obs.UserID != "" {
			userID = obs.UserID
		}
		if sessionID != "" && userID != "" {
			break
		}
	}

	return &types.SessionSummary{
		ID:         fmt.Sprintf("sum_%s_%d", sessionID, time.Now().Unix()),
		SessionID:  sessionID,
		UserID:     userID,
		Title:      title,
		Facts:      facts,
		Concepts:   concepts,
		Narrative:  narrative,
		Files:      files,
		Importance: importance,
		CreatedAt:  time.Now(),
	}
}

// extractTitle extracts a title from the first user prompt
func (g *Generator) extractTitle(observations []types.Observation) string {
	for _, obs := range observations {
		if obs.HookType == "user_prompt_submit" {
			prompt, ok := obs.Data["prompt"].(string)
			if !ok {
				continue
			}

			// Truncate to first meaningful sentence
			if len(prompt) > 100 {
				// Find first sentence break
				for i, c := range prompt {
					if c == '.' || c == '!' || c == '?' || c == '\n' {
						if i > 20 && i < 100 {
							return prompt[:i+1]
						}
					}
				}
				return prompt[:100] + "..."
			}
			return prompt
		}
	}

	// Fallback: use first tool name
	for _, obs := range observations {
		if obs.HookType == "post_tool_use" {
			tool, ok := obs.Data["tool_name"].(string)
			if ok {
				return fmt.Sprintf("Session using %s", tool)
			}
		}
	}

	return "Untitled Session"
}

// extractFacts extracts key facts from successful tool calls
func (g *Generator) extractFacts(observations []types.Observation) []string {
	var facts []string

	for _, obs := range observations {
		if obs.HookType != "post_tool_use" {
			continue
		}

		toolName, ok := obs.Data["tool_name"].(string)
		if !ok {
			continue
		}

		// Extract facts based on tool type
		switch toolName {
		case "read":
			if file, ok := obs.Data["file_path"].(string); ok {
				facts = append(facts, fmt.Sprintf("Read file: %s", file))
			}
		case "write", "edit":
			if file, ok := obs.Data["file_path"].(string); ok {
				facts = append(facts, fmt.Sprintf("Modified file: %s", file))
			}
		case "bash":
			if cmd, ok := obs.Data["tool_input"].(string); ok {
				// Truncate long commands
				if len(cmd) > 50 {
					cmd = cmd[:50] + "..."
				}
				facts = append(facts, fmt.Sprintf("Executed: %s", cmd))
			}
		}
	}

	// Limit to 10 facts
	if len(facts) > 10 {
		facts = facts[:10]
	}

	return facts
}

// extractConcepts extracts technical concepts mentioned
func (g *Generator) extractConcepts(observations []types.Observation) []string {
	conceptSet := make(map[string]bool)

	// Keywords that indicate concepts
	conceptKeywords := []string{
		"架构", "设计", "API", "REST", "数据库", "缓存",
		"测试", "单元测试", "集成测试",
		"Go", "Python", "JavaScript", "TypeScript",
		"architecture", "design", "database", "cache",
		"test", "testing", "unit test",
	}

	for _, obs := range observations {
		// Check user prompts for concepts
		if obs.HookType == "user_prompt_submit" {
			prompt, ok := obs.Data["prompt"].(string)
			if !ok {
				continue
			}

			for _, kw := range conceptKeywords {
				if strings.Contains(prompt, kw) {
					conceptSet[kw] = true
				}
			}
		}

		// Check tool inputs for file types
		if obs.HookType == "post_tool_use" {
			toolInput, ok := obs.Data["tool_input"].(string)
			if !ok {
				continue
			}

			// Detect file extensions
			exts := extractFileExtensions(toolInput)
			for ext := range exts {
				conceptSet[ext] = true
			}
		}
	}

	var concepts []string
	for concept := range conceptSet {
		concepts = append(concepts, concept)
	}

	// Limit to 10 concepts
	if len(concepts) > 10 {
		concepts = concepts[:10]
	}

	return concepts
}

// buildNarrative builds a narrative from tool sequence
func (g *Generator) buildNarrative(observations []types.Observation) string {
	var steps []string

	for _, obs := range observations {
		if obs.HookType == "user_prompt_submit" {
			prompt, ok := obs.Data["prompt"].(string)
			if ok && len(prompt) > 0 {
				// Truncate long prompts
				if len(prompt) > 80 {
					prompt = prompt[:80] + "..."
				}
				steps = append(steps, fmt.Sprintf("User asked: \"%s\"", prompt))
			}
		}

		if obs.HookType == "post_tool_use" {
			tool, ok := obs.Data["tool_name"].(string)
			if !ok {
				continue
			}

			step := fmt.Sprintf("Used %s", tool)

			// Add file info if available
			if file, ok := obs.Data["file_path"].(string); ok {
				step += fmt.Sprintf(" on %s", file)
			}

			// Add result status
			if result, ok := obs.Data["tool_result"].(string); ok {
				if result == "error" {
					step += " (failed)"
				} else if result != "" {
					step += " (success)"
				}
			}

			steps = append(steps, step)
		}
	}

	if len(steps) == 0 {
		return "No activity recorded."
	}

	// Build narrative
	narrative := strings.Join(steps, ". ")
	if len(narrative) > 500 {
		narrative = narrative[:500] + "..."
	}

	return narrative
}

// extractFiles extracts file paths from observations
func (g *Generator) extractFiles(observations []types.Observation) []string {
	fileSet := make(map[string]bool)

	for _, obs := range observations {
		if obs.HookType != "post_tool_use" {
			continue
		}

		// Direct file_path field
		if file, ok := obs.Data["file_path"].(string); ok {
			fileSet[file] = true
		}

		// Extract from tool_input
		if input, ok := obs.Data["tool_input"].(string); ok {
			files := extractFilePaths(input)
			for file := range files {
				fileSet[file] = true
			}
		}
	}

	var files []string
	for file := range fileSet {
		files = append(files, file)
	}

	// Limit to 20 files
	if len(files) > 20 {
		files = files[:20]
	}

	return files
}

// calculateImportance calculates session importance
func (g *Generator) calculateImportance(observations []types.Observation) float64 {
	if len(observations) == 0 {
		return 0.0
	}

	// Count successful operations
	successCount := 0
	toolCount := 0

	for _, obs := range observations {
		if obs.HookType == "post_tool_use" {
			toolCount++
			if result, ok := obs.Data["tool_result"].(string); ok {
				if result != "" && result != "error" {
					successCount++
				}
			}
		}
	}

	// Base importance from success rate
	successRate := 0.0
	if toolCount > 0 {
		successRate = float64(successCount) / float64(toolCount)
	}

	// Boost from activity count
	activityBoost := float64(len(observations)) / 20.0
	if activityBoost > 0.5 {
		activityBoost = 0.5
	}

	importance := successRate + activityBoost
	if importance > 1.0 {
		importance = 1.0
	}

	return importance
}

// Helper functions

func extractFileExtensions(s string) map[string]bool {
	exts := make(map[string]bool)

	commonExts := []string{".go", ".py", ".js", ".ts", ".json", ".yaml", ".md", ".txt"}
	for _, ext := range commonExts {
		if strings.Contains(s, ext) {
			exts[ext] = true
		}
	}

	return exts
}

func extractFilePaths(s string) map[string]bool {
	files := make(map[string]bool)

	// Simple path detection (starts with / or ./ or contains common extensions)
	words := strings.Fields(s)
	for _, word := range words {
		// Check if it looks like a file path
		if strings.HasPrefix(word, "/") ||
			strings.HasPrefix(word, "./") ||
			strings.HasPrefix(word, "~/") ||
			strings.Contains(word, ".go") ||
			strings.Contains(word, ".py") ||
			strings.Contains(word, ".js") ||
			strings.Contains(word, ".ts") ||
			strings.Contains(word, ".json") ||
			strings.Contains(word, ".yaml") ||
			strings.Contains(word, ".md") {
			// Clean up the path
			word = strings.TrimSuffix(word, ",")
			word = strings.TrimSuffix(word, ";")
			word = strings.TrimPrefix(word, "\"")
			word = strings.TrimSuffix(word, "\"")
			if len(word) > 3 && len(word) < 200 {
				files[word] = true
			}
		}
	}

	return files
}