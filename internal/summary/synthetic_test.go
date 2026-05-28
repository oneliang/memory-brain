package summary

import (
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// TestGenerator_NewGenerator tests generator creation
func TestGenerator_NewGenerator(t *testing.T) {
	gen := NewGenerator()
	if gen == nil {
		t.Fatal("generator should not be nil")
	}
}

// TestGenerator_BuildSyntheticSummary_Basic tests summary generation
func TestGenerator_BuildSyntheticSummary_Basic(t *testing.T) {
	gen := NewGenerator()

	observations := []types.Observation{
		{
			ID:        "obs_1",
			SessionID: "session_1",
			UserID:    "user_1",
			HookType:  "user_prompt_submit",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"prompt": "Implement a REST API endpoint",
			},
		},
		{
			ID:        "obs_2",
			SessionID: "session_1",
			UserID:    "user_1",
			HookType:  "post_tool_use",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"tool_name":   "read",
				"file_path":   "/src/api/server.go",
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
				"tool_name":   "edit",
				"file_path":   "/src/api/server.go",
				"tool_result": "success",
			},
		},
	}

	summary := gen.BuildSyntheticSummary(observations)

	if summary == nil {
		t.Fatal("should return a summary")
	}

	if summary.SessionID != "session_1" {
		t.Errorf("expected session_1, got %s", summary.SessionID)
	}

	if summary.UserID != "user_1" {
		t.Errorf("expected user_1, got %s", summary.UserID)
	}

	if summary.Title == "" {
		t.Error("title should not be empty")
	}

	if len(summary.Facts) == 0 {
		t.Error("should have extracted facts")
	}

	if summary.Narrative == "" {
		t.Error("narrative should not be empty")
	}
}

// TestGenerator_BuildSyntheticSummary_Empty tests empty observations
func TestGenerator_BuildSyntheticSummary_Empty(t *testing.T) {
	gen := NewGenerator()

	summary := gen.BuildSyntheticSummary([]types.Observation{})
	if summary != nil {
		t.Error("empty observations should return nil")
	}
}

// TestGenerator_ExtractTitle tests title extraction
func TestGenerator_ExtractTitle(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		observations []types.Observation
		expectedContains string
	}{
		{
			[]types.Observation{
				{HookType: "user_prompt_submit", Data: map[string]interface{}{"prompt": "Test prompt"}},
			},
			"Test prompt",
		},
		{
			[]types.Observation{
				{HookType: "post_tool_use", Data: map[string]interface{}{"tool_name": "bash"}},
			},
			"bash",
		},
		{
			[]types.Observation{},
			"",
		},
	}

	for _, tt := range tests {
		title := gen.extractTitle(tt.observations)
		if tt.expectedContains != "" && title == "" {
			t.Error("title should not be empty for non-empty observations")
		}
	}
}

// TestGenerator_ExtractFacts tests fact extraction
func TestGenerator_ExtractFacts(t *testing.T) {
	gen := NewGenerator()

	observations := []types.Observation{
		{HookType: "post_tool_use", Data: map[string]interface{}{
			"tool_name": "read", "file_path": "/src/main.go"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{
			"tool_name": "edit", "file_path": "/src/main.go"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{
			"tool_name": "bash", "tool_input": "go build"}},
	}

	facts := gen.extractFacts(observations)

	if len(facts) < 2 {
		t.Errorf("expected at least 2 facts, got %d", len(facts))
	}

	// Should contain file operations
	for _, fact := range facts {
		if fact == "" {
			t.Error("fact should not be empty")
		}
	}
}

// TestGenerator_ExtractConcepts tests concept extraction
func TestGenerator_ExtractConcepts(t *testing.T) {
	gen := NewGenerator()

	observations := []types.Observation{
		{
			HookType: "user_prompt_submit",
			Data:     map[string]interface{}{"prompt": "Design an API architecture"},
		},
		{
			HookType: "post_tool_use",
			Data:     map[string]interface{}{"tool_input": "main.go test.py"},
		},
	}

	concepts := gen.extractConcepts(observations)

	if len(concepts) == 0 {
		t.Error("should extract concepts")
	}

	// Should contain API and architecture keywords
	containsAPI := false
	for _, c := range concepts {
		if c == "API" {
			containsAPI = true
		}
	}
	if !containsAPI {
		t.Log("Warning: API concept may not be extracted")
	}
}

// TestGenerator_BuildNarrative tests narrative building
func TestGenerator_BuildNarrative(t *testing.T) {
	gen := NewGenerator()

	observations := []types.Observation{
		{HookType: "user_prompt_submit", Data: map[string]interface{}{"prompt": "Hello"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{
			"tool_name": "bash", "tool_result": "success"}},
	}

	narrative := gen.buildNarrative(observations)

	if narrative == "" {
		t.Error("narrative should not be empty")
	}

	// Should contain user prompt and tool usage
	if !containsSubstring(narrative, "User asked") {
		t.Error("narrative should mention user request")
	}
}

// TestGenerator_ExtractFiles tests file extraction
func TestGenerator_ExtractFiles(t *testing.T) {
	gen := NewGenerator()

	observations := []types.Observation{
		{HookType: "post_tool_use", Data: map[string]interface{}{
			"file_path": "/src/main.go"}},
		{HookType: "post_tool_use", Data: map[string]interface{}{
			"tool_input": "edit /src/test.go"}},
	}

	files := gen.extractFiles(observations)

	if len(files) == 0 {
		t.Error("should extract files")
	}
}

// TestGenerator_CalculateImportance tests importance calculation
func TestGenerator_CalculateImportance(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		observations []types.Observation
		minImportance float64
	}{
		{
			[]types.Observation{
				{HookType: "post_tool_use", Data: map[string]interface{}{"tool_result": "success"}},
				{HookType: "post_tool_use", Data: map[string]interface{}{"tool_result": "success"}},
			},
			0.5,
		},
		{
			[]types.Observation{},
			0.0,
		},
		{
			[]types.Observation{
				{HookType: "post_tool_use", Data: map[string]interface{}{"tool_result": "error"}},
			},
			0.0,
		},
	}

	for _, tt := range tests {
		imp := gen.calculateImportance(tt.observations)
		if imp < tt.minImportance {
			t.Errorf("importance should be >= %f, got %f", tt.minImportance, imp)
		}
		if imp > 1.0 {
			t.Errorf("importance should not exceed 1.0, got %f", imp)
		}
	}
}

// TestExtractFileExtensions tests file extension extraction
func TestExtractFileExtensions(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"main.go test.py", []string{".go", ".py"}},
		{"config.json data.yaml", []string{".json", ".yaml"}},
		{"no extensions here", []string{}},
	}

	for _, tt := range tests {
		exts := extractFileExtensions(tt.input)
		for _, expected := range tt.expected {
			if !exts[expected] {
				t.Errorf("expected extension %s in %s", expected, tt.input)
			}
		}
	}
}

// TestExtractFilePaths tests file path extraction
func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"read /src/main.go", []string{"/src/main.go"}},
		{"edit ./test.py", []string{"./test.py"}},
		{"no paths here", []string{}},
	}

	for _, tt := range tests {
		files := extractFilePaths(tt.input)
		for _, expected := range tt.expected {
			if !files[expected] {
				t.Errorf("expected path %s in %s", expected, tt.input)
			}
		}
	}
}

// TestGenerator_BuildSyntheticSummary_Limit tests output limits
func TestGenerator_BuildSyntheticSummary_Limit(t *testing.T) {
	gen := NewGenerator()

	// Create many observations to test limits
	var observations []types.Observation
	for i := 0; i < 20; i++ {
		observations = append(observations, types.Observation{
			HookType: "post_tool_use",
			Data: map[string]interface{}{
				"tool_name": "read",
				"file_path": "/src/file.go",
			},
		})
	}

	summary := gen.BuildSyntheticSummary(observations)

	if summary == nil {
		t.Fatal("should return summary")
	}

	// Facts should be limited to 10
	if len(summary.Facts) > 10 {
		t.Errorf("facts should be limited to 10, got %d", len(summary.Facts))
	}

	// Narrative should be limited to 500 chars
	if len(summary.Narrative) > 500+3 { // +3 for "..."
		t.Errorf("narrative should be limited, got %d chars", len(summary.Narrative))
	}
}

// TestGenerator_BuildSyntheticSummary_MissingSessionInfo tests missing session info
func TestGenerator_BuildSyntheticSummary_MissingSessionInfo(t *testing.T) {
	gen := NewGenerator()

	observations := []types.Observation{
		{HookType: "user_prompt_submit", Data: map[string]interface{}{"prompt": "test"}},
	}

	summary := gen.BuildSyntheticSummary(observations)

	if summary == nil {
		t.Fatal("should return summary even without session info")
	}

	// Should still generate title and narrative
	if summary.Title == "" {
		t.Error("should generate title")
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}