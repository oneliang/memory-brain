package profile

import (
	"fmt"
	"sort"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// Analyzer analyzes observations to extract user profile cards
type Analyzer struct{}

// NewAnalyzer creates a new profile analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzeToolPreference extracts tool preference from observations
func (a *Analyzer) AnalyzeToolPreference(observations []types.Observation) *types.ProfileCard {
	// Count tool usage frequency
	toolFreq := make(map[string]int)
	toolSuccess := make(map[string]int)

	for _, obs := range observations {
		if obs.HookType != "post_tool_use" {
			continue
		}

		toolName, ok := obs.Data["tool_name"].(string)
		if !ok {
			continue
		}

		toolFreq[toolName]++

		// Check success from tool_result
		if result, ok := obs.Data["tool_result"].(string); ok {
			if result != "" && result != "error" {
				toolSuccess[toolName]++
			}
		}
	}

	if len(toolFreq) == 0 {
		return nil
	}

	// Find most used tool
	var topTool string
	var maxFreq int
	for tool, freq := range toolFreq {
		if freq > maxFreq {
			maxFreq = freq
			topTool = tool
		}
	}

	// Calculate success rate
	successRate := 0.0
	if maxFreq > 0 {
		successRate = float64(toolSuccess[topTool]) / float64(maxFreq)
	}

	// Generate preference content
	content := map[string]interface{}{
		"preference_type":   "tool_preference",
		"preferred_tools":   getTopTools(toolFreq, 5),
		"top_tool":          topTool,
		"top_tool_freq":     maxFreq,
		"top_tool_success":  successRate,
		"confidence":        calculateConfidence(maxFreq, successRate),
	}

	return &types.ProfileCard{
		ID:      fmt.Sprintf("pref_tool_%d", time.Now().Unix()),
		Type:    "PreferenceCard",
		Content: content,
		Metadata: types.CardMetadata{
			Timestamp:  time.Now(),
			Importance: calculateImportance(maxFreq, successRate),
			Strength:   1.0,
			DecayDays:  7,
		},
	}
}

// AnalyzeWorkPatterns extracts behavioral patterns from observations
func (a *Analyzer) AnalyzeWorkPatterns(observations []types.Observation) *types.PatternCard {
	// Extract tool sequences
	sequences := extractToolSequences(observations)

	if len(sequences) == 0 {
		return nil
	}

	// Find most common sequence
	var topSequence string
	var maxCount int
	for seq, count := range sequences {
		if count > maxCount {
			maxCount = count
			topSequence = seq
		}
	}

	if topSequence == "" {
		return nil
	}

	return &types.PatternCard{
		ID:         fmt.Sprintf("pattern_seq_%d", time.Now().Unix()),
		UserID:     "", // Set by caller
		Type:       "tool_sequence",
		Pattern:    topSequence,
		Frequency:  maxCount,
		Confidence: calculatePatternConfidence(maxCount, len(sequences)),
		LastSeen:   time.Now(),
		CreatedAt:  time.Now(),
	}
}

// AnalyzeIntent extracts user intent from UserPromptSubmit observations
func (a *Analyzer) AnalyzeIntent(observations []types.Observation) *types.ProfileCard {
	var intents []string

	for _, obs := range observations {
		if obs.HookType != "user_prompt_submit" {
			continue
		}

		prompt, ok := obs.Data["prompt"].(string)
		if !ok {
			continue
		}

		// Simple intent classification based on keywords
		intent := classifyIntent(prompt)
		if intent != "" {
			intents = append(intents, intent)
		}
	}

	if len(intents) == 0 {
		return nil
	}

	// Count intents
	intentFreq := make(map[string]int)
	for _, intent := range intents {
		intentFreq[intent]++
	}

	content := map[string]interface{}{
		"preference_type": "intent_preference",
		"intent_freq":     intentFreq,
		"primary_intent":  getPrimaryIntent(intentFreq),
	}

	return &types.ProfileCard{
		ID:      fmt.Sprintf("pref_intent_%d", time.Now().Unix()),
		Type:    "PreferenceCard",
		Content: content,
		Metadata: types.CardMetadata{
			Timestamp:  time.Now(),
			Importance: 0.7,
			Strength:   1.0,
			DecayDays:  7,
		},
	}
}

// Helper functions

func getTopTools(toolFreq map[string]int, limit int) []string {
	// Sort by frequency
	var tools []string
	for tool := range toolFreq {
		tools = append(tools, tool)
	}

	// Sort by frequency using sort.Slice
	sort.Slice(tools, func(i, j int) bool {
		return toolFreq[tools[i]] > toolFreq[tools[j]]
	})

	if len(tools) > limit {
		tools = tools[:limit]
	}

	return tools
}

func calculateConfidence(freq int, successRate float64) float64 {
	// Base confidence from frequency
	baseConf := float64(freq) / 10.0 // Normalize by expected usage
	if baseConf > 1.0 {
		baseConf = 1.0
	}

	// Adjust by success rate
	return baseConf * successRate
}

func calculateImportance(freq int, successRate float64) float64 {
	// Higher frequency = higher importance
	freqScore := float64(freq) / 20.0
	if freqScore > 1.0 {
		freqScore = 1.0
	}

	// Combine with success rate
	return (freqScore * 0.6) + (successRate * 0.4)
}

func extractToolSequences(observations []types.Observation) map[string]int {
	sequences := make(map[string]int)

	// Group by session
	sessionTools := make(map[string][]string)
	for _, obs := range observations {
		if obs.HookType != "post_tool_use" {
			continue
		}

		toolName, ok := obs.Data["tool_name"].(string)
		if !ok {
			continue
		}

		sessionTools[obs.SessionID] = append(sessionTools[obs.SessionID], toolName)
	}

	// Extract sequences (pairs and triples)
	for _, tools := range sessionTools {
		// Pairs
		for i := 0; i < len(tools)-1; i++ {
			seq := fmt.Sprintf("%s -> %s", tools[i], tools[i+1])
			sequences[seq]++
		}

		// Triples
		for i := 0; i < len(tools)-2; i++ {
			seq := fmt.Sprintf("%s -> %s -> %s", tools[i], tools[i+1], tools[i+2])
			sequences[seq]++
		}
	}

	return sequences
}

func calculatePatternConfidence(maxCount int, totalPatterns int) float64 {
	if totalPatterns == 0 {
		return 0.0
	}
	conf := float64(maxCount) / float64(totalPatterns)
	if conf > 1.0 {
		conf = 1.0
	}
	return conf
}

func classifyIntent(prompt string) string {
	// Simple keyword-based classification
	promptLower := prompt

	// Development keywords
	if containsAny(promptLower, []string{"实现", "开发", "代码", "编写", "创建", "添加", "修改", "implement", "code", "create", "add", "modify"}) {
		return "development"
	}

	// Debug keywords
	if containsAny(promptLower, []string{"调试", "修复", "错误", "bug", "问题", "debug", "fix", "error", "issue"}) {
		return "debugging"
	}

	// Query keywords
	if containsAny(promptLower, []string{"查询", "搜索", "查找", "是什么", "怎么", "query", "search", "find", "what", "how"}) {
		return "query"
	}

	// Management keywords
	if containsAny(promptLower, []string{"管理", "配置", "部署", "设置", "manage", "config", "deploy", "setup"}) {
		return "management"
	}

	return "general"
}

func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if len(s) >= len(kw) {
			for i := 0; i <= len(s)-len(kw); i++ {
				if s[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}

func getPrimaryIntent(intentFreq map[string]int) string {
	var primary string
	var maxFreq int
	for intent, freq := range intentFreq {
		if freq > maxFreq {
			maxFreq = freq
			primary = intent
		}
	}
	return primary
}