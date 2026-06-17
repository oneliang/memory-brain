package extractors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/pkg/types"
)

func TestOllamaExtractor_Name(t *testing.T) {
	ext := NewOllamaExtractor("http://localhost:11434", "qwen2.5:3b", 30*time.Second)
	if ext.Name() != "ollama" {
		t.Errorf("Expected name 'ollama', got '%s'", ext.Name())
	}
}

func TestOllamaExtractor_IsAsync(t *testing.T) {
	ext := NewOllamaExtractor("http://localhost:11434", "qwen2.5:3b", 30*time.Second)
	if !ext.IsAsync() {
		t.Error("Ollama extractor should be async")
	}
}

func TestOllamaExtractor_Defaults(t *testing.T) {
	ext := NewOllamaExtractor("", "", 0)
	if ext.endpoint != "http://localhost:11434" {
		t.Errorf("Expected default endpoint, got '%s'", ext.endpoint)
	}
	if ext.model != "qwen2.5:3b" {
		t.Errorf("Expected default model, got '%s'", ext.model)
	}
	if ext.timeout != 30*time.Second {
		t.Errorf("Expected default timeout, got %v", ext.timeout)
	}
}

func TestOllamaExtractor_Extract_Mock(t *testing.T) {
	// Create mock Ollama server
	mockResponse := graph.ExtractionResult{
		Entities: []graph.Entity{
			{Type: "person", Name: "张三", Properties: map[string]interface{}{"role": "developer"}},
			{Type: "org", Name: "Google", Properties: nil},
		},
		Relations: []graph.Relation{
			{Source: "张三", Type: "works_at", Target: "Google", Weight: 1.0},
		},
	}

	responseJSON, _ := json.Marshal(mockResponse)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected path /api/generate, got %s", r.URL.Path)
		}

		// Verify request format
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got '%s'", req.Model)
		}
		if req.Stream != false {
			t.Error("Stream should be false")
		}
		if req.Format != "json" {
			t.Errorf("Expected format 'json', got '%s'", req.Format)
		}
		if req.Think == nil || *req.Think != false {
			t.Error("Think should be disabled (false)")
		}

		// Return mock response
		ollamaResp := OllamaResponse{
			Response: string(responseJSON),
			Done:     true,
		}
		json.NewEncoder(w).Encode(ollamaResp)
	}))
	defer server.Close()

	ext := NewOllamaExtractor(server.URL, "test-model", 5*time.Second)

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "张三在 Google 工作",
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify entities
	if len(result.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(result.Entities))
	}

	var foundPerson, foundOrg bool
	for _, entity := range result.Entities {
		if entity.Type == "person" && entity.Name == "张三" {
			foundPerson = true
		}
		if entity.Type == "org" && entity.Name == "Google" {
			foundOrg = true
		}
	}

	if !foundPerson {
		t.Error("Should find person entity '张三'")
	}
	if !foundOrg {
		t.Error("Should find org entity 'Google'")
	}

	// Verify relations
	if len(result.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(result.Relations))
	}

	if len(result.Relations) > 0 {
		rel := result.Relations[0]
		if rel.Source != "张三" || rel.Type != "works_at" || rel.Target != "Google" {
			t.Errorf("Unexpected relation: %+v", rel)
		}
	}
}

func TestOllamaExtractor_Extract_EmptyText(t *testing.T) {
	ext := NewOllamaExtractor("http://localhost:11434", "test-model", 5*time.Second)

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "post_tool_use",
		Data:     map[string]interface{}{}, // No text data
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract should not fail for empty text: %v", err)
	}

	if len(result.Entities) != 0 || len(result.Relations) != 0 {
		t.Error("Expected empty result for empty text")
	}
}

func TestOllamaExtractor_Extract_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	ext := NewOllamaExtractor(server.URL, "test-model", 5*time.Second)

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "test",
		},
	}

	_, err := ext.Extract(obs)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestOllamaExtractor_Extract_OllamaError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OllamaResponse{
			Error: "model not found",
			Done:  true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ext := NewOllamaExtractor(server.URL, "nonexistent-model", 5*time.Second)

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "test",
		},
	}

	_, err := ext.Extract(obs)
	if err == nil {
		t.Error("Expected error for Ollama error response")
	}
}

func TestOllamaExtractor_CombineText(t *testing.T) {
	ext := NewOllamaExtractor("http://localhost:11434", "test-model", 5*time.Second)

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
	}{
		{
			name: "from prompt",
			data: map[string]interface{}{
				"prompt": "Hello world",
			},
			expected: "Hello world",
		},
		{
			name: "from tool_input command",
			data: map[string]interface{}{
				"tool_input": map[string]interface{}{
					"command": "ls -la",
				},
			},
			expected: "ls -la",
		},
		{
			name: "from tool_input input",
			data: map[string]interface{}{
				"tool_input": map[string]interface{}{
					"input": "some input",
				},
			},
			expected: "some input",
		},
		{
			name: "from tool_result",
			data: map[string]interface{}{
				"tool_result": "output text",
			},
			expected: "output text",
		},
		{
			name: "combined",
			data: map[string]interface{}{
				"prompt": "prompt text",
				"tool_input": map[string]interface{}{
					"command": "git status",
				},
				"tool_result": "result text",
			},
			expected: "prompt text\ngit status\nresult text",
		},
		{
			name:     "empty",
			data:     map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &types.Observation{
				Data: tt.data,
			}
			result := ext.combineText(obs)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestOllamaExtractor_CombineText_TruncateLongResult(t *testing.T) {
	ext := NewOllamaExtractor("http://localhost:11434", "test-model", 5*time.Second)

	// Create a long tool result (>500 chars)
	longResult := ""
	for i := 0; i < 600; i++ {
		longResult += "x"
	}

	obs := &types.Observation{
		Data: map[string]interface{}{
			"tool_result": longResult,
		},
	}

	result := ext.combineText(obs)

	// Should be truncated to 500 + "..."
	expectedLen := 503 // 500 + 3 for "..."
	if len(result) != expectedLen {
		t.Errorf("Expected length %d, got %d", expectedLen, len(result))
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean json",
			input:    `{"entities": []}`,
			expected: `{"entities": []}`,
		},
		{
			name:     "json with prefix",
			input:    `Here is the result: {"entities": []}`,
			expected: `{"entities": []}`,
		},
		{
			name:     "json with suffix",
			input:    `{"entities": []} Done!`,
			expected: `{"entities": []}`,
		},
		{
			name:     "json with prefix and suffix",
			input:    `Result: {"entities": [{"name": "test"}]} End.`,
			expected: `{"entities": [{"name": "test"}]}`,
		},
		{
			name:     "no json",
			input:    "No JSON here",
			expected: "",
		},
		{
			name:     "invalid braces",
			input:    "}wrong{",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestOllamaExtractor_CheckHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected path /api/tags, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models": []}`))
	}))
	defer server.Close()

	ext := NewOllamaExtractor(server.URL, "test-model", 5*time.Second)

	err := ext.CheckHealth()
	if err != nil {
		t.Errorf("CheckHealth should succeed: %v", err)
	}
}

func TestOllamaExtractor_CheckHealth_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ext := NewOllamaExtractor(server.URL, "test-model", 5*time.Second)

	err := ext.CheckHealth()
	if err == nil {
		t.Error("CheckHealth should fail for error status")
	}
}

func TestOllamaExtractor_GetModelInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected path /api/tags, got %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"models": []map[string]interface{}{
				{"name": "qwen2.5:3b", "size": 1000000},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ext := NewOllamaExtractor(server.URL, "test-model", 5*time.Second)

	info, err := ext.GetModelInfo()
	if err != nil {
		t.Fatalf("GetModelInfo failed: %v", err)
	}

	models, ok := info["models"].([]interface{})
	if !ok {
		t.Fatal("Expected models array in response")
	}

	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}
}
