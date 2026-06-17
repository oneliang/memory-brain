package extractors

import (
	"regexp"
	"strings"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/pkg/types"
)

// RuleExtractor extracts entities and relations using rules (no ML)
type RuleExtractor struct {
	patterns []*regexp.Regexp
	dict     map[string]string // keyword -> entity type
}

// NewRuleExtractor creates a new rule-based extractor
func NewRuleExtractor() *RuleExtractor {
	return &RuleExtractor{
		patterns: []*regexp.Regexp{
			// File paths
			regexp.MustCompile(`(?:/[a-zA-Z0-9_.-]+)+\.[a-zA-Z]{1,10}`),
			// Windows paths
			regexp.MustCompile(`[A-Z]:\\(?:[a-zA-Z0-9_.-]+\\)*[a-zA-Z0-9_.-]+\.[a-zA-Z]{1,10}`),
			// Email addresses
			regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			// URLs
			regexp.MustCompile(`https?://[^\s"'<>]+`),
			// Version numbers
			regexp.MustCompile(`v?\d+\.\d+(?:\.\d+)?`),
		},
		dict: map[string]string{
			// Technologies
			"PostgreSQL": "tech", "Postgres": "tech", "MySQL": "tech", "MariaDB": "tech",
			"Redis": "tech", "MongoDB": "tech", "SQLite": "tech",
			"Kubernetes": "tech", "K8s": "tech", "Docker": "tech",
			"REST": "tech", "GraphQL": "tech", "gRPC": "tech",
			"React": "tech", "Vue": "tech", "Angular": "tech",
			"Nginx": "tech", "Apache": "tech",
			// Programming languages
			"Go": "lang", "Golang": "lang", "Python": "lang", "JavaScript": "lang",
			"TypeScript": "lang", "Rust": "lang", "Java": "lang", "C++": "lang",
			"C#": "lang", "Ruby": "lang", "PHP": "lang", "Swift": "lang",
			// Tools
			"git": "tool", "npm": "tool", "yarn": "tool", "pnpm": "tool",
			"make": "tool", "cmake": "tool", "bazel": "tool",
		},
	}
}

// Name returns the name of the extractor
func (e *RuleExtractor) Name() string {
	return "rule"
}

// IsAsync returns false (rule extraction is fast)
func (e *RuleExtractor) IsAsync() bool {
	return false
}

// Extract extracts entities and relations from an observation
func (e *RuleExtractor) Extract(obs *types.Observation) (*graph.ExtractionResult, error) {
	result := &graph.ExtractionResult{
		Entities:  make([]graph.Entity, 0),
		Relations: make([]graph.Relation, 0),
	}

	seen := make(map[string]bool) // Deduplicate entities

	// 1. Extract tool entity
	if toolName, ok := obs.Data["tool_name"].(string); ok && toolName != "" {
		entity := graph.Entity{
			Type: "tool",
			Name: toolName,
		}
		if !seen[entity.Type+":"+entity.Name] {
			result.Entities = append(result.Entities, entity)
			seen[entity.Type+":"+entity.Name] = true

			// Relation: user -> uses -> tool
			result.Relations = append(result.Relations, graph.Relation{
				Source: obs.UserID,
				Target: toolName,
				Type:   "uses",
				Weight: 1.0,
			})
		}
	}

	// 2. Extract file entity
	if filePath, ok := obs.Data["file_path"].(string); ok && filePath != "" {
		entity := graph.Entity{
			Type: "file",
			Name: filePath,
		}
		if !seen[entity.Type+":"+entity.Name] {
			result.Entities = append(result.Entities, entity)
			seen[entity.Type+":"+entity.Name] = true

			// Relation: user -> modifies -> file
			result.Relations = append(result.Relations, graph.Relation{
				Source: obs.UserID,
				Target: filePath,
				Type:   "modifies",
				Weight: 1.0,
			})
		}
	}

	// 3. Extract command entity (from bash tool)
	if toolName, ok := obs.Data["tool_name"].(string); ok && toolName == "bash" {
		if toolInput, ok := obs.Data["tool_input"].(map[string]interface{}); ok {
			if cmd, ok := toolInput["command"].(string); ok && cmd != "" {
				// Truncate long commands
				cmdName := cmd
				if len(cmdName) > 100 {
					cmdName = cmdName[:100] + "..."
				}

				entity := graph.Entity{
					Type: "command",
					Name: cmdName,
				}
				if !seen[entity.Type+":"+entity.Name] {
					result.Entities = append(result.Entities, entity)
					seen[entity.Type+":"+entity.Name] = true

					// Relation: user -> executes -> command
					result.Relations = append(result.Relations, graph.Relation{
						Source: obs.UserID,
						Target: cmdName,
						Type:   "executes",
						Weight: 1.0,
					})

					// Extract additional entities from command
					e.extractFromText(cmd, result, obs.UserID, seen)
				}
			}
		} else if cmd, ok := obs.Data["tool_input"].(string); ok && cmd != "" {
			// Alternative format: tool_input is a string
			cmdName := cmd
			if len(cmdName) > 100 {
				cmdName = cmdName[:100] + "..."
			}

			entity := graph.Entity{
				Type: "command",
				Name: cmdName,
			}
			if !seen[entity.Type+":"+entity.Name] {
				result.Entities = append(result.Entities, entity)
				seen[entity.Type+":"+entity.Name] = true

				result.Relations = append(result.Relations, graph.Relation{
					Source: obs.UserID,
					Target: cmdName,
					Type:   "executes",
					Weight: 1.0,
				})

				e.extractFromText(cmd, result, obs.UserID, seen)
			}
		}
	}

	// 4. Extract from prompt (user_prompt_submit)
	if prompt, ok := obs.Data["prompt"].(string); ok && prompt != "" {
		e.extractFromText(prompt, result, obs.UserID, seen)

		// Extract intent
		intent := classifyIntent(prompt)
		if intent != "" && intent != "general" {
			entity := graph.Entity{
				Type: "intent",
				Name: intent,
			}
			if !seen[entity.Type+":"+entity.Name] {
				result.Entities = append(result.Entities, entity)
				seen[entity.Type+":"+entity.Name] = true

				result.Relations = append(result.Relations, graph.Relation{
					Source: obs.UserID,
					Target: intent,
					Type:   "works_on",
					Weight: 1.0,
				})
			}
		}
	}

	return result, nil
}

// extractFromText extracts entities from text using patterns and dictionary
func (e *RuleExtractor) extractFromText(text string, result *graph.ExtractionResult, userID string, seen map[string]bool) {
	// Pattern matching
	for _, pattern := range e.patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			entityType := e.classifyPattern(match)
			entity := graph.Entity{
				Type: entityType,
				Name: match,
			}
			if !seen[entity.Type+":"+entity.Name] {
				result.Entities = append(result.Entities, entity)
				seen[entity.Type+":"+entity.Name] = true

				result.Relations = append(result.Relations, graph.Relation{
					Source: userID,
					Target: match,
					Type:   "mentions",
					Weight: 0.5, // Lower weight for pattern matches
				})
			}
		}
	}

	// Dictionary matching
	for keyword, entityType := range e.dict {
		if strings.Contains(text, keyword) {
			entity := graph.Entity{
				Type: entityType,
				Name: keyword,
			}
			if !seen[entity.Type+":"+entity.Name] {
				result.Entities = append(result.Entities, entity)
				seen[entity.Type+":"+entity.Name] = true

				result.Relations = append(result.Relations, graph.Relation{
					Source: userID,
					Target: keyword,
					Type:   "mentions",
					Weight: 0.8,
				})
			}
		}
	}
}

// classifyPattern determines entity type from pattern match
func (e *RuleExtractor) classifyPattern(match string) string {
	if strings.Contains(match, "@") {
		return "email"
	}
	if strings.HasPrefix(match, "http://") || strings.HasPrefix(match, "https://") {
		return "url"
	}
	if strings.Contains(match, "/") || strings.Contains(match, "\\") {
		return "file"
	}
	if regexp.MustCompile(`^v?\d+\.\d+`).MatchString(match) {
		return "version"
	}
	return "unknown"
}

// classifyIntent classifies user intent from prompt (copied from server.go)
func classifyIntent(prompt string) string {
	promptLower := strings.ToLower(prompt)

	// Development keywords
	devKeywords := []string{"实现", "开发", "代码", "编写", "创建", "添加", "修改", "implement", "code", "create", "add", "modify", "build"}
	for _, kw := range devKeywords {
		if strings.Contains(promptLower, kw) {
			return "development"
		}
	}

	// Debug keywords
	debugKeywords := []string{"调试", "修复", "错误", "bug", "问题", "debug", "fix", "error", "issue", "解决"}
	for _, kw := range debugKeywords {
		if strings.Contains(promptLower, kw) {
			return "debugging"
		}
	}

	// Query keywords
	queryKeywords := []string{"查询", "搜索", "查找", "是什么", "怎么", "如何", "query", "search", "find", "what", "how", "explain", "解释"}
	for _, kw := range queryKeywords {
		if strings.Contains(promptLower, kw) {
			return "query"
		}
	}

	// Management keywords
	mgmtKeywords := []string{"管理", "配置", "部署", "设置", "manage", "config", "deploy", "setup", "install"}
	for _, kw := range mgmtKeywords {
		if strings.Contains(promptLower, kw) {
			return "management"
		}
	}

	// Review keywords
	reviewKeywords := []string{"检查", "审查", "优化", "重构", "review", "optimize", "refactor", "测试", "test"}
	for _, kw := range reviewKeywords {
		if strings.Contains(promptLower, kw) {
			return "review"
		}
	}

	return "general"
}
