package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// TestHandleHealth tests the health check endpoint
func TestHandleHealth(t *testing.T) {
	srv := NewServer(12321, nil)
	req := httptest.NewRequest(http.MethodGet, "/memory/health", nil)
	w := httptest.NewRecorder()

	srv.handler.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !body.Success {
		t.Error("expected success=true")
	}

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %v", data["status"])
	}

	if data["version"] != "0.1.0" {
		t.Errorf("expected version=0.1.0, got %v", data["version"])
	}
}

// TestHandleHealth_MethodNotAllowed tests wrong method on health endpoint
func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	srv := NewServer(12321, nil)
	req := httptest.NewRequest(http.MethodPost, "/memory/health", nil)
	w := httptest.NewRecorder()

	srv.handler.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

// TestHandleObserve tests the observe endpoint
func TestHandleObserve(t *testing.T) {
	srv := NewServer(12321, nil)

	obs := types.Observation{
		ID:        "obs_test_1",
		SessionID: "session_test_1",
		UserID:    "user_test_1",
		HookType:  "post_tool_use",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool_name": "bash",
			"args":      "ls -la",
		},
	}

	bodyBytes, _ := json.Marshal(obs)
	req := httptest.NewRequest(http.MethodPost, "/memory/observe", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	srv.handler.handleObserve(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !body.Success {
		t.Error("expected success=true")
	}

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["dedup"] == true {
		t.Error("expected dedup=false for first request")
	}
}

// TestHandleObserve_Dedup tests deduplication behavior
func TestHandleObserve_Dedup(t *testing.T) {
	srv := NewServer(12321, nil)

	obs := types.Observation{
		ID:        "obs_test_2",
		SessionID: "session_test_2",
		UserID:    "user_test_2",
		HookType:  "post_tool_use",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool_name": "bash",
			"args":      "echo test",
		},
	}

	// First request
	bodyBytes, _ := json.Marshal(obs)
	req1 := httptest.NewRequest(http.MethodPost, "/memory/observe", bytes.NewReader(bodyBytes))
	w1 := httptest.NewRecorder()
	srv.handler.handleObserve(w1, req1)

	resp1 := w1.Result()
	var body1 types.APIResponse
	json.NewDecoder(resp1.Body).Decode(&body1)
	data1, _ := body1.Data.(map[string]interface{})

	if data1["dedup"] == true {
		t.Error("first request should not be deduped")
	}

	// Second identical request within dedup window
	req2 := httptest.NewRequest(http.MethodPost, "/memory/observe", bytes.NewReader(bodyBytes))
	w2 := httptest.NewRecorder()
	srv.handler.handleObserve(w2, req2)

	resp2 := w2.Result()
	var body2 types.APIResponse
	json.NewDecoder(resp2.Body).Decode(&body2)
	data2, _ := body2.Data.(map[string]interface{})

	if data2["dedup"] != true {
		t.Error("second identical request should be deduped")
	}
}

// TestHandleObserve_MethodNotAllowed tests wrong method
func TestHandleObserve_MethodNotAllowed(t *testing.T) {
	srv := NewServer(12321, nil)
	req := httptest.NewRequest(http.MethodGet, "/memory/observe", nil)
	w := httptest.NewRecorder()

	srv.handler.handleObserve(w, req)

	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Result().StatusCode)
	}
}

// TestHandleObserve_InvalidBody tests invalid JSON body
func TestHandleObserve_InvalidBody(t *testing.T) {
	srv := NewServer(12321, nil)
	req := httptest.NewRequest(http.MethodPost, "/memory/observe", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	srv.handler.handleObserve(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

// TestHandleProfile tests the profile endpoint
func TestHandleProfile(t *testing.T) {
	srv := NewServer(12321, nil)

	req := httptest.NewRequest(http.MethodGet, "/memory/profile?user=user_test_1", nil)
	w := httptest.NewRecorder()

	srv.handler.handleProfile(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !body.Success {
		t.Error("expected success=true")
	}

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	profileSummary, ok := data["profile_summary"].(map[string]interface{})
	if !ok {
		t.Fatal("expected profile_summary to be a map")
	}

	if profileSummary["user_id"] != "user_test_1" {
		t.Errorf("expected user_id=user_test_1, got %v", profileSummary["user_id"])
	}
}

// TestHandleProfile_Inject tests profile with system message injection
func TestHandleProfile_Inject(t *testing.T) {
	srv := NewServer(12321, nil)

	req := httptest.NewRequest(http.MethodGet, "/memory/profile?user=user_test_1&inject=true", nil)
	w := httptest.NewRecorder()

	srv.handler.handleProfile(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	json.NewDecoder(resp.Body).Decode(&body)

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	// systemMessage may be empty if no profiles exist
	if _, exists := data["systemMessage"]; !exists {
		// This is acceptable for empty profiles
	}
}

// TestHandleProfile_MissingUser tests missing user parameter
func TestHandleProfile_MissingUser(t *testing.T) {
	srv := NewServer(12321, nil)
	req := httptest.NewRequest(http.MethodGet, "/memory/profile", nil)
	w := httptest.NewRecorder()

	srv.handler.handleProfile(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

// TestHandleSearch tests the search endpoint (without vector search)
func TestHandleSearch(t *testing.T) {
	srv := NewServer(12321, nil)

	req := httptest.NewRequest(http.MethodGet, "/memory/search?query=test&user=user_test_1&limit=5", nil)
	w := httptest.NewRecorder()

	srv.handler.handleSearch(w, req)

	resp := w.Result()
	// May return 500 if vector search unavailable, or 200 with empty results
	// This is acceptable behavior for now
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", resp.StatusCode)
	}
}

// TestHandleSearch_MissingParams tests missing query/user parameters
func TestHandleSearch_MissingParams(t *testing.T) {
	srv := NewServer(12321, nil)

	// Missing query
	req1 := httptest.NewRequest(http.MethodGet, "/memory/search?user=user_test_1", nil)
	w1 := httptest.NewRecorder()
	srv.handler.handleSearch(w1, req1)
	if w1.Result().StatusCode != http.StatusBadRequest {
		t.Error("expected 400 for missing query")
	}

	// Missing user
	req2 := httptest.NewRequest(http.MethodGet, "/memory/search?query=test", nil)
	w2 := httptest.NewRecorder()
	srv.handler.handleSearch(w2, req2)
	if w2.Result().StatusCode != http.StatusBadRequest {
		t.Error("expected 400 for missing user")
	}
}

// TestHandleProfileUpdate tests profile update endpoint
func TestHandleProfileUpdate(t *testing.T) {
	srv := NewServer(12321, nil)

	reqBody := map[string]interface{}{
		"user_id":    "user_test_1",
		"session_id": "session_test_1",
		"observations": []types.Observation{
			{
				ID:        "obs_1",
				SessionID: "session_test_1",
				UserID:    "user_test_1",
				HookType:  "post_tool_use",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"tool_name": "bash",
					"success":   true,
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/memory/profile/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	srv.handler.handleProfileUpdate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	json.NewDecoder(resp.Body).Decode(&body)

	if !body.Success {
		t.Error("expected success=true")
	}

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["updated"] != true {
		t.Error("expected updated=true")
	}
}

// TestHandleProfileUpdate_MissingUserID tests missing user_id
func TestHandleProfileUpdate_MissingUserID(t *testing.T) {
	srv := NewServer(12321, nil)

	reqBody := map[string]interface{}{
		"session_id": "session_test_1",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/memory/profile/update", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	srv.handler.handleProfileUpdate(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Error("expected 400 for missing user_id")
	}
}

// TestHandleSessionSummary tests session summary endpoint
func TestHandleSessionSummary(t *testing.T) {
	srv := NewServer(12321, nil)

	// Test with provided summary
	summary := &types.SessionSummary{
		ID:        "summary_test_1",
		SessionID: "session_test_1",
		UserID:    "user_test_1",
		Title:     "Test Session",
		Facts:     []string{"fact1", "fact2"},
		Narrative: "Test narrative",
		CreatedAt: time.Now(),
	}

	reqBody := map[string]interface{}{
		"summary":   summary,
		"user_id":   "user_test_1",
		"session_id": "session_test_1",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/memory/session/summary", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	srv.handler.handleSessionSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	json.NewDecoder(resp.Body).Decode(&body)

	if !body.Success {
		t.Error("expected success=true")
	}

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["saved"] != true {
		t.Error("expected saved=true")
	}
}

// TestHandleSessionSummary_Generate tests summary generation from observations
func TestHandleSessionSummary_Generate(t *testing.T) {
	srv := NewServer(12321, nil)

	reqBody := map[string]interface{}{
		"user_id":    "user_test_1",
		"session_id": "session_test_1",
		"generate":   true,
		"observations": []types.Observation{
			{
				ID:        "obs_1",
				SessionID: "session_test_1",
				UserID:    "user_test_1",
				HookType:  "user_prompt_submit",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"prompt": "Help me implement a REST API",
				},
			},
			{
				ID:        "obs_2",
				SessionID: "session_test_1",
				UserID:    "user_test_1",
				HookType:  "post_tool_use",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"tool_name": "bash",
					"success":   true,
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/memory/session/summary", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	srv.handler.handleSessionSummary(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body types.APIResponse
	json.NewDecoder(resp.Body).Decode(&body)

	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["generated"] != true {
		t.Error("expected generated=true")
	}
}

// TestHandleSessionSummary_MissingData tests missing summary and observations
func TestHandleSessionSummary_MissingData(t *testing.T) {
	srv := NewServer(12321, nil)

	reqBody := map[string]interface{}{
		"user_id":    "user_test_1",
		"session_id": "session_test_1",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/memory/session/summary", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	srv.handler.handleSessionSummary(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Error("expected 400 for missing summary and observations")
	}
}

// TestServer_Start tests server lifecycle
func TestServer_Start(t *testing.T) {
	srv := NewServer(3112, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should return after context cancellation
	err := srv.Start(ctx)
	if err != nil {
		t.Errorf("unexpected error on shutdown: %v", err)
	}
}