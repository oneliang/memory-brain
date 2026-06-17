package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/internal/graph/extractors"
	"github.com/oneliang/memory-brain/pkg/types"
)

// setupTestStore creates a temporary store for testing
func setupTestStore(t *testing.T) (*graph.Store, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "graph_e2e_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := graph.NewStoreWithPath(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

// TestE2EGraphPipeline tests the complete graph extraction and query pipeline
func TestE2EGraphPipeline(t *testing.T) {
	// 1. Setup: Create store and extractors
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create registry with rule extractor
	registry := extractors.NewRegistry()
	registry.Register(extractors.NewRuleExtractor())

	// 2. Simulate observations from a coding session
	observations := []*types.Observation{
		{
			ID:       "obs1",
			UserID:   "user1",
			HookType: "user_prompt_submit",
			Data: map[string]interface{}{
				"prompt": "帮我配置 PostgreSQL 和 Redis",
			},
			Timestamp: time.Now(),
		},
		{
			ID:       "obs2",
			UserID:   "user1",
			HookType: "post_tool_use",
			Data: map[string]interface{}{
				"tool_name": "bash",
				"tool_input": map[string]interface{}{
					"command": "kubectl get pods",
				},
			},
			Timestamp: time.Now(),
		},
		{
			ID:       "obs3",
			UserID:   "user1",
			HookType: "post_tool_use",
			Data: map[string]interface{}{
				"tool_name": "write",
				"file_path": "/internal/api/server.go",
			},
			Timestamp: time.Now(),
		},
		{
			ID:       "obs4",
			UserID:   "user1",
			HookType: "post_tool_use",
			Data: map[string]interface{}{
				"tool_name": "bash",
				"tool_input": map[string]interface{}{
					"command": "git commit -m 'add api server'",
				},
			},
			Timestamp: time.Now(),
		},
	}

	// 3. Extract and save all observations
	t.Log("=== Step 1: Extract and Save Observations ===")
	for _, obs := range observations {
		// Extract using registry
		result, err := registry.ExtractAll(obs)
		if err != nil {
			t.Fatalf("ExtractAll failed for obs %s: %v", obs.ID, err)
		}

		// Save result
		if err := store.SaveExtractionResult(obs.UserID, result); err != nil {
			t.Fatalf("SaveExtractionResult failed: %v", err)
		}

		t.Logf("  Processed observation %s: %d entities, %d relations",
			obs.ID, len(result.Entities), len(result.Relations))
	}

	// 4. Verify graph statistics
	t.Log("\n=== Step 2: Verify Graph Statistics ===")
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	nodeCount := stats["node_count"].(int)
	edgeCount := stats["edge_count"].(int)
	nodeTypes := stats["node_types"].(map[string]int)

	t.Logf("  Total nodes: %d", nodeCount)
	t.Logf("  Total edges: %d", edgeCount)
	t.Logf("  Node types: %v", nodeTypes)

	// Verify expected node types exist
	expectedTypes := []string{"user", "tech", "tool", "file", "command", "intent"}
	for _, typ := range expectedTypes {
		if count, ok := nodeTypes[typ]; !ok || count == 0 {
			t.Errorf("Expected node type '%s' to exist", typ)
		}
	}

	// 5. Test multi-hop query
	t.Log("\n=== Step 3: Multi-Hop Query ===")
	engine := graph.NewQueryEngine(store)

	opts := &graph.QueryOptions{
		MaxHops:   2,
		Limit:     20,
		MinScore:  0.1,
		DecayRate: 0.7,
	}

	results, err := engine.MultiHopQuery("user1", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	t.Logf("  Found %d connected entities", len(results))
	for i, result := range results {
		if i < 10 { // Show first 10
			t.Logf("    [%d] %s:%s (hops=%d, score=%.2f)",
				i+1, result.Node.Type, result.Node.Name, result.Hops, result.Score)
		}
	}

	// Verify key entities are found
	foundEntities := make(map[string]bool)
	for _, result := range results {
		foundEntities[result.Node.Type+":"+result.Node.Name] = true
	}

	expectedEntities := []string{
		"tech:PostgreSQL",
		"tech:Redis",
		"tool:bash",
		"file:/internal/api/server.go",
	}

	for _, expected := range expectedEntities {
		if !foundEntities[expected] {
			t.Errorf("Expected to find entity '%s' in query results", expected)
		} else {
			t.Logf("  ✓ Found: %s", expected)
		}
	}

	// 6. Test GetContext
	t.Log("\n=== Step 4: Get User Context ===")
	context, err := engine.GetContext("user1", 10)
	if err != nil {
		t.Fatalf("GetContext failed: %v", err)
	}

	t.Logf("  Context keys: %v", getMapKeys(context))

	// Verify context structure
	if context["user_id"] != "user1" {
		t.Errorf("Expected user_id='user1', got '%v'", context["user_id"])
	}

	// tools/techs/files are []map[string]interface{} with name/score/hops
	if tools, ok := context["tools"].([]map[string]interface{}); !ok || len(tools) == 0 {
		t.Error("Expected tools in context")
	} else {
		toolNames := make([]string, len(tools))
		for i, tool := range tools {
			toolNames[i] = tool["name"].(string)
		}
		t.Logf("  Tools: %v", toolNames)
	}

	if techs, ok := context["techs"].([]map[string]interface{}); !ok || len(techs) == 0 {
		t.Error("Expected techs in context")
	} else {
		techNames := make([]string, len(techs))
		for i, tech := range techs {
			techNames[i] = tech["name"].(string)
		}
		t.Logf("  Techs: %v", techNames)
	}

	if files, ok := context["files"].([]map[string]interface{}); !ok || len(files) == 0 {
		t.Error("Expected files in context")
	} else {
		fileNames := make([]string, len(files))
		for i, file := range files {
			fileNames[i] = file["name"].(string)
		}
		t.Logf("  Files: %v", fileNames)
	}

	// 7. Test FindPath
	t.Log("\n=== Step 5: Find Path Between Entities ===")

	userNode, _ := store.GetNodeByTypeAndName("user", "user1")
	postgresNode, _ := store.GetNodeByTypeAndName("tech", "PostgreSQL")

	if userNode != nil && postgresNode != nil {
		path, err := engine.FindPath(userNode.ID, postgresNode.ID, 5)
		if err != nil {
			t.Fatalf("FindPath failed: %v", err)
		}

		if path != nil {
			t.Logf("  Found path from user1 to PostgreSQL (%d hops):", len(path)-1)
			for i, nodeID := range path {
				node, _ := store.GetNode(nodeID)
				if node != nil {
					t.Logf("    [%d] %s:%s", i, node.Type, node.Name)
				}
			}
		} else {
			t.Log("  No path found (may be expected)")
		}
	}

	// 8. Test filtered query
	t.Log("\n=== Step 6: Filtered Query (tech only) ===")
	techOpts := &graph.QueryOptions{
		MaxHops:   2,
		Limit:     10,
		MinScore:  0.1,
		DecayRate: 0.7,
		NodeTypes: []string{"tech"},
	}

	techResults, err := engine.MultiHopQuery("user1", techOpts)
	if err != nil {
		t.Fatalf("Filtered query failed: %v", err)
	}

	t.Logf("  Found %d tech entities", len(techResults))
	for _, result := range techResults {
		if result.Node.Type != "tech" {
			t.Errorf("Expected only tech nodes, got %s", result.Node.Type)
		}
		t.Logf("    - %s (score=%.2f)", result.Node.Name, result.Score)
	}

	t.Log("\n=== E2E Test Complete ===")
}

// TestE2EWithMockOllama tests the full pipeline with mock Ollama
func TestE2EWithMockOllama(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			// Return mock NER+RE result
			// Note: Include relation to user1 so query can find entities
			response := map[string]interface{}{
				"entities": []map[string]interface{}{
					{"type": "person", "name": "张三", "properties": map[string]interface{}{"role": "developer"}},
					{"type": "org", "name": "Google", "properties": nil},
				},
				"relations": []map[string]interface{}{
					{"source": "user1", "relation": "knows", "target": "张三"},
					{"source": "张三", "relation": "works_at", "target": "Google"},
				},
			}
			responseJSON, _ := json.Marshal(response)

			resp := map[string]interface{}{
				"response": string(responseJSON),
				"done":     true,
			}
			json.NewEncoder(w).Encode(resp)
		} else if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"models": [{"name": "qwen2.5:3b"}]}`))
		}
	}))
	defer server.Close()

	// Setup
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create observation
	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "张三在 Google 用 Go 开发",
		},
		Timestamp: time.Now(),
	}

	t.Log("=== E2E with Mock Ollama ===")

	// Note: ExtractAll only runs sync extractors. For async (Ollama), call directly.
	// 1. Run rule extractor (sync)
	ruleExtractor := extractors.NewRuleExtractor()
	ruleResult, err := ruleExtractor.Extract(obs)
	if err != nil {
		t.Fatalf("Rule extractor failed: %v", err)
	}
	t.Logf("  Rule extractor: %d entities, %d relations", len(ruleResult.Entities), len(ruleResult.Relations))

	// 2. Run Ollama extractor (async)
	ollamaExtractor := extractors.NewOllamaExtractor(server.URL, "test-model", 5*time.Second)
	ollamaResult, err := ollamaExtractor.Extract(obs)
	if err != nil {
		t.Fatalf("Ollama extractor failed: %v", err)
	}
	t.Logf("  Ollama extractor: %d entities, %d relations", len(ollamaResult.Entities), len(ollamaResult.Relations))

	// 3. Save both results
	if err := store.SaveExtractionResult(obs.UserID, ruleResult); err != nil {
		t.Fatalf("SaveExtractionResult (rule) failed: %v", err)
	}
	if err := store.SaveExtractionResult(obs.UserID, ollamaResult); err != nil {
		t.Fatalf("SaveExtractionResult (ollama) failed: %v", err)
	}

	// Verify
	stats, _ := store.GetStats()
	t.Logf("  Nodes in graph: %v", stats["node_count"])
	t.Logf("  Edges in graph: %v", stats["edge_count"])

	// Query
	engine := graph.NewQueryEngine(store)
	queryResults, err := engine.MultiHopQuery("user1", graph.DefaultQueryOptions())
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	t.Logf("  Query results: %d entities", len(queryResults))
	for _, result := range queryResults {
		t.Logf("    - %s:%s", result.Node.Type, result.Node.Name)
	}

	// Verify expected entities from Ollama extraction
	foundPerson := false
	foundOrg := false
	for _, result := range queryResults {
		if result.Node.Type == "person" && result.Node.Name == "张三" {
			foundPerson = true
		}
		if result.Node.Type == "org" && result.Node.Name == "Google" {
			foundOrg = true
		}
	}

	if !foundPerson {
		t.Error("Should find person '张三' from Ollama extraction")
	} else {
		t.Log("  ✓ Found person from Ollama: 张三")
	}

	if !foundOrg {
		t.Error("Should find org 'Google' from Ollama extraction")
	} else {
		t.Log("  ✓ Found org from Ollama: Google")
	}
}

// TestE2EAPISimulation simulates the full API flow
func TestE2EAPISimulation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Simulate API handlers
	extractHandler := func(userID string, text string) error {
		obs := &types.Observation{
			ID:       "obs-api",
			UserID:   userID,
			HookType: "user_prompt_submit",
			Data: map[string]interface{}{
				"prompt": text,
			},
			Timestamp: time.Now(),
		}

		registry := extractors.NewRegistry()
		registry.Register(extractors.NewRuleExtractor())

		result, err := registry.ExtractAll(obs)
		if err != nil {
			return err
		}

		return store.SaveExtractionResult(userID, result)
	}

	queryHandler := func(userID string, maxHops, limit int) ([]graph.QueryResult, error) {
		engine := graph.NewQueryEngine(store)
		opts := &graph.QueryOptions{
			MaxHops:   maxHops,
			Limit:     limit,
			MinScore:  0.1,
			DecayRate: 0.7,
		}
		return engine.MultiHopQuery(userID, opts)
	}

	contextHandler := func(userID string) (map[string]interface{}, error) {
		engine := graph.NewQueryEngine(store)
		return engine.GetContext(userID, 10)
	}

	statsHandler := func() (map[string]interface{}, error) {
		return store.GetStats()
	}

	// Test flow
	t.Log("=== API Simulation ===")

	// 1. Extract via API
	t.Log("POST /memory/graph/extract")
	err := extractHandler("user1", "帮我配置 Docker 和 Kubernetes")
	if err != nil {
		t.Fatalf("Extract API failed: %v", err)
	}
	t.Log("  ✓ Extraction completed")

	// 2. Query via API
	t.Log("GET /memory/graph/query?user=user1&hops=2&limit=10")
	queryResults, err := queryHandler("user1", 2, 10)
	if err != nil {
		t.Fatalf("Query API failed: %v", err)
	}
	t.Logf("  ✓ Found %d entities", len(queryResults))

	// 3. Context via API
	t.Log("GET /memory/graph/context?user=user1&limit=10")
	context, err := contextHandler("user1")
	if err != nil {
		t.Fatalf("Context API failed: %v", err)
	}
	t.Logf("  ✓ Context: %v", getMapKeys(context))

	// 4. Stats via API
	t.Log("GET /memory/graph/stats")
	stats, err := statsHandler()
	if err != nil {
		t.Fatalf("Stats API failed: %v", err)
	}
	t.Logf("  ✓ Stats: nodes=%v, edges=%v", stats["node_count"], stats["edge_count"])

	// Verify results
	if len(queryResults) == 0 {
		t.Error("Query should return results")
	}

	foundDocker := false
	foundK8s := false
	for _, result := range queryResults {
		if result.Node.Name == "Docker" {
			foundDocker = true
		}
		if result.Node.Name == "Kubernetes" {
			foundK8s = true
		}
	}

	if !foundDocker {
		t.Error("Should find Docker")
	} else {
		t.Log("  ✓ Found: Docker")
	}

	if !foundK8s {
		t.Error("Should find Kubernetes")
	} else {
		t.Log("  ✓ Found: Kubernetes")
	}
}

// Helper function
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
