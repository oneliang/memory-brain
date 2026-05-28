package profile

import (
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// TestAnalyzer_NewAnalyzer tests analyzer creation
func TestAnalyzer_NewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()
	if analyzer == nil {
		t.Fatal("analyzer should not be nil")
	}
}

// TestAnalyzer_AnalyzeToolPreference_Basic tests tool preference analysis
func TestAnalyzer_AnalyzeToolPreference_Basic(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{
			ID:        "obs_1",
			SessionID: "session_1",
			UserID:    "user_1",
			HookType:  "post_tool_use",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"tool_name":   "bash",
				"tool_result": "success",
			},
		},
		{
			ID:        "obs_2",
			SessionID: "session_1",
			UserID:    "user_1",
			HookType:  "post_tool_use",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"tool_name":   "bash",
				"tool_result": "success",
			},
		},
		{
			ID:        "obs_3",
			SessionID: "session_1",
			UserID:    "user_1",
			HookType:  "post_tool_use",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"tool_name":   "read",
				"tool_result": "success",
			},
		},
	}

	card := analyzer.AnalyzeToolPreference(observations)

	if card == nil {
		t.Fatal("should return a profile card")
	}

	if card.Type != "PreferenceCard" {
		t.Errorf("expected PreferenceCard, got %s", card.Type)
	}

	content := card.Content
	if content["preference_type"] != "tool_preference" {
		t.Errorf("expected preference_type=tool_preference, got %v", content["preference_type"])
	}

	topTool, ok := content["top_tool"].(string)
	if !ok || topTool != "bash" {
		t.Errorf("expected top_tool=bash, got %v", topTool)
	}
}

// TestAnalyzer_AnalyzeToolPreference_Empty tests empty observations
func TestAnalyzer_AnalyzeToolPreference_Empty(t *testing.T) {
	analyzer := NewAnalyzer()

	card := analyzer.AnalyzeToolPreference([]types.Observation{})
	if card != nil {
		t.Error("empty observations should return nil")
	}
}

// TestAnalyzer_AnalyzeToolPreference_NoToolUse tests observations without tool use
func TestAnalyzer_AnalyzeToolPreference_NoToolUse(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "test"},
		},
	}

	card := analyzer.AnalyzeToolPreference(observations)
	if card != nil {
		t.Error("observations without post_tool_use should return nil")
	}
}

// TestAnalyzer_AnalyzeToolPreference_MixedTools tests multiple tools
func TestAnalyzer_AnalyzeToolPreference_MixedTools(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "read"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "edit"}},
	}

	card := analyzer.AnalyzeToolPreference(observations)

	if card == nil {
		t.Fatal("should return a card")
	}

	preferredTools, ok := card.Content["preferred_tools"].([]string)
	if !ok {
		t.Fatal("preferred_tools should be a string array")
	}

	if len(preferredTools) < 1 {
		t.Error("should have at least one preferred tool")
	}

	// bash should be first (most frequent)
	if preferredTools[0] != "bash" {
		t.Errorf("expected bash as most preferred, got %s", preferredTools[0])
	}
}

// TestAnalyzer_AnalyzeWorkPatterns_Basic tests pattern analysis
func TestAnalyzer_AnalyzeWorkPatterns(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{SessionID: "s1", HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "read"}},
		{SessionID: "s1", HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "edit"}},
		{SessionID: "s1", HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
		{SessionID: "s2", HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "read"}},
		{SessionID: "s2", HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "edit"}},
	}

	pattern := analyzer.AnalyzeWorkPatterns(observations)

	if pattern == nil {
		t.Fatal("should return a pattern card")
	}

	if pattern.Type != "tool_sequence" {
		t.Errorf("expected tool_sequence, got %s", pattern.Type)
	}

	if pattern.Pattern == "" {
		t.Error("pattern should have a sequence")
	}

	if pattern.Frequency < 1 {
		t.Error("frequency should be positive")
	}
}

// TestAnalyzer_AnalyzeWorkPatterns_Empty tests empty pattern analysis
func TestAnalyzer_AnalyzeWorkPatterns_Empty(t *testing.T) {
	analyzer := NewAnalyzer()

	pattern := analyzer.AnalyzeWorkPatterns([]types.Observation{})
	if pattern != nil {
		t.Error("empty observations should return nil pattern")
	}
}

// TestAnalyzer_AnalyzeWorkPatterns_SingleTool tests single tool (no sequence)
func TestAnalyzer_AnalyzeWorkPatterns_SingleTool(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{SessionID: "s1", HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
	}

	pattern := analyzer.AnalyzeWorkPatterns(observations)
	if pattern != nil {
		t.Error("single tool should not form a sequence")
	}
}

// TestAnalyzer_AnalyzeIntent_Basic tests intent analysis
func TestAnalyzer_AnalyzeIntent(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "implement a new feature"},
		},
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "debug this error"},
		},
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "implement another feature"},
		},
	}

	card := analyzer.AnalyzeIntent(observations)

	if card == nil {
		t.Fatal("should return a profile card")
	}

	if card.Type != "PreferenceCard" {
		t.Errorf("expected PreferenceCard, got %s", card.Type)
	}

	primaryIntent, ok := card.Content["primary_intent"].(string)
	if !ok || primaryIntent == "" {
		t.Error("should have a primary intent")
	}
}

// TestAnalyzer_AnalyzeIntent_Empty tests empty intent analysis
func TestAnalyzer_AnalyzeIntent_Empty(t *testing.T) {
	analyzer := NewAnalyzer()

	card := analyzer.AnalyzeIntent([]types.Observation{})
	if card != nil {
		t.Error("empty observations should return nil")
	}
}

// TestAnalyzer_AnalyzeIntent_NoPrompts tests without prompts
func TestAnalyzer_AnalyzeIntent_NoPrompts(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
	}

	card := analyzer.AnalyzeIntent(observations)
	if card != nil {
		t.Error("observations without user_prompt_submit should return nil")
	}
}

// TestAnalyzer_AnalyzeIntent_ChineseKeywords tests Chinese intent classification
func TestAnalyzer_AnalyzeIntent_ChineseKeywords(t *testing.T) {
	analyzer := NewAnalyzer()

	observations := []types.Observation{
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "实现新功能"},
		},
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "修复这个bug"},
		},
	}

	card := analyzer.AnalyzeIntent(observations)

	if card == nil {
		t.Fatal("should return a card")
	}

	intentFreq, ok := card.Content["intent_freq"].(map[string]int)
	if !ok {
		t.Fatal("intent_freq should be a map")
	}

	// Should classify Chinese keywords
	if len(intentFreq) < 2 {
		t.Errorf("should have classified at least 2 intents, got %d", len(intentFreq))
	}
}

// TestGetTopTools tests top tools extraction
func TestGetTopTools(t *testing.T) {
	toolFreq := map[string]int{
		"bash":  10,
		"read":  5,
		"edit":  3,
		"write": 2,
		"grep":  1,
	}

	topTools := getTopTools(toolFreq, 3)

	if len(topTools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(topTools))
	}

	// Should be sorted by frequency
	if topTools[0] != "bash" {
		t.Errorf("expected bash first, got %s", topTools[0])
	}
}

// TestCalculateConfidence tests confidence calculation
func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		freq       int
		successRate float64
	}{
		{5, 1.0},
		{10, 0.8},
		{20, 0.5},
		{1, 0.9},
	}

	for _, tt := range tests {
		conf := calculateConfidence(tt.freq, tt.successRate)
		if conf < 0 || conf > 1 {
			t.Errorf("confidence should be between 0 and 1, got %f", conf)
		}
	}
}

// TestCalculateImportance tests importance calculation
func TestCalculateImportance(t *testing.T) {
	tests := []struct {
		freq       int
		successRate float64
	}{
		{10, 1.0},
		{20, 0.9},
		{5, 0.5},
	}

	for _, tt := range tests {
		imp := calculateImportance(tt.freq, tt.successRate)
		if imp < 0 || imp > 1 {
			t.Errorf("importance should be between 0 and 1, got %f", imp)
		}
	}
}

// TestClassifyIntent tests intent classification helper
func TestClassifyIntent(t *testing.T) {
	tests := []struct {
		prompt    string
		expected  string
	}{
		{"implement a new feature", "development"},
		{"fix this bug", "debugging"},
		{"what is this function", "query"},
		{"configure the settings", "management"},
		{"hello world", "general"},
		{"实现新功能", "development"},
		{"修复错误", "debugging"},
	}

	for _, tt := range tests {
		result := classifyIntent(tt.prompt)
		if result != tt.expected {
			t.Errorf("classifyIntent(%s) = %s, expected %s", tt.prompt, result, tt.expected)
		}
	}
}

// TestContainsAny tests contains any helper
func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		keywords []string
		expected bool
	}{
		{"implement a feature", []string{"implement"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"fix the bug", []string{"fix", "bug"}, true},
		{"", []string{"test"}, false},
	}

	for _, tt := range tests {
		result := containsAny(tt.s, tt.keywords)
		if result != tt.expected {
			t.Errorf("containsAny(%s, %v) = %v, expected %v", tt.s, tt.keywords, result, tt.expected)
		}
	}
}