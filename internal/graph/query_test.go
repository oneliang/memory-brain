package graph

import (
	"testing"
)

// setupTestGraph creates a test graph with known structure
func setupTestGraph(t *testing.T) (*Store, *QueryEngine, func()) {
	t.Helper()

	store, cleanup := setupTestStore(t)

	// Create test graph structure:
	// user1 -> uses -> bash
	// user1 -> modifies -> server.go
	// bash -> executes -> kubectl
	// server.go -> uses -> postgres

	userNode, _ := store.GetOrCreateNode("user", "user1", nil)
	bashNode, _ := store.GetOrCreateNode("tool", "bash", nil)
	fileNode, _ := store.GetOrCreateNode("file", "server.go", nil)
	kubectlNode, _ := store.GetOrCreateNode("command", "kubectl", nil)
	postgresNode, _ := store.GetOrCreateNode("tech", "postgres", nil)

	// Create edges
	store.CreateEdge(userNode.ID, bashNode.ID, "uses", 2.0)
	store.CreateEdge(userNode.ID, fileNode.ID, "modifies", 1.5)
	store.CreateEdge(bashNode.ID, kubectlNode.ID, "executes", 1.0)
	store.CreateEdge(fileNode.ID, postgresNode.ID, "uses", 1.0)

	engine := NewQueryEngine(store)
	return store, engine, cleanup
}

func TestMultiHopQuery_1Hop(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	opts := &QueryOptions{
		MaxHops:   1,
		Limit:     10,
		MinScore:  0.1,
		DecayRate: 0.7,
	}

	results, err := engine.MultiHopQuery("user1", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	// Should find bash and server.go (direct neighbors)
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 1-hop, got %d", len(results))
	}

	// Verify results are sorted by score
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Error("Results should be sorted by score descending")
		}
	}

	// Verify all results have hops = 1
	for _, result := range results {
		if result.Hops != 1 {
			t.Errorf("Expected hops=1, got %d for %s", result.Hops, result.Node.Name)
		}
	}
}

func TestMultiHopQuery_2Hops(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	opts := &QueryOptions{
		MaxHops:   2,
		Limit:     10,
		MinScore:  0.1,
		DecayRate: 0.7,
	}

	results, err := engine.MultiHopQuery("user1", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	// Should find bash, server.go, kubectl, postgres
	if len(results) < 4 {
		t.Errorf("Expected at least 4 results for 2-hop, got %d", len(results))
	}

	// Find kubectl (2-hop)
	var foundKubectl bool
	for _, result := range results {
		if result.Node.Name == "kubectl" {
			foundKubectl = true
			if result.Hops != 2 {
				t.Errorf("Expected kubectl at hops=2, got %d", result.Hops)
			}
			break
		}
	}
	if !foundKubectl {
		t.Error("Should find kubectl in 2-hop query")
	}
}

func TestMultiHopQuery_ScoreDecay(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	opts := &QueryOptions{
		MaxHops:   2,
		Limit:     10,
		MinScore:  0.0,
		DecayRate: 0.5, // Aggressive decay
	}

	results, err := engine.MultiHopQuery("user1", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	// 1-hop nodes should have higher scores than 2-hop nodes
	var max1HopScore, max2HopScore float64
	for _, result := range results {
		if result.Hops == 1 && result.Score > max1HopScore {
			max1HopScore = result.Score
		}
		if result.Hops == 2 && result.Score > max2HopScore {
			max2HopScore = result.Score
		}
	}

	if max2HopScore > max1HopScore {
		t.Errorf("2-hop scores should be lower than 1-hop scores due to decay: 1-hop=%.2f, 2-hop=%.2f",
			max1HopScore, max2HopScore)
	}
}

func TestMultiHopQuery_WithFilters(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	opts := &QueryOptions{
		MaxHops:   2,
		Limit:     10,
		MinScore:  0.1,
		DecayRate: 0.7,
		NodeTypes: []string{"tool"}, // Only return tool nodes
	}

	results, err := engine.MultiHopQuery("user1", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	// All results should be tool type
	for _, result := range results {
		if result.Node.Type != "tool" {
			t.Errorf("Expected only tool nodes, got %s", result.Node.Type)
		}
	}
}

func TestMultiHopQuery_Limit(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	opts := &QueryOptions{
		MaxHops:   2,
		Limit:     2, // Limit to 2 results
		MinScore:  0.0,
		DecayRate: 0.7,
	}

	results, err := engine.MultiHopQuery("user1", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	if len(results) > 2 {
		t.Errorf("Expected at most 2 results, got %d", len(results))
	}
}

func TestMultiHopQuery_UnknownUser(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	opts := DefaultQueryOptions()
	results, err := engine.MultiHopQuery("unknown_user", opts)
	if err != nil {
		t.Fatalf("MultiHopQuery should not error for unknown user: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for unknown user, got %d", len(results))
	}
}

func TestGetContext(t *testing.T) {
	_, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	context, err := engine.GetContext("user1", 10)
	if err != nil {
		t.Fatalf("GetContext failed: %v", err)
	}

	// Verify context structure
	if context["user_id"] != "user1" {
		t.Errorf("Expected user_id='user1', got '%v'", context["user_id"])
	}

	// Should have tools, techs, files
	if _, ok := context["tools"]; !ok {
		t.Error("Context should have 'tools' field")
	}
	if _, ok := context["files"]; !ok {
		t.Error("Context should have 'files' field")
	}
}

func TestFindPath(t *testing.T) {
	store, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	// Get node IDs
	userNode, _ := store.GetNodeByTypeAndName("user", "user1")
	postgresNode, _ := store.GetNodeByTypeAndName("tech", "postgres")

	// Find path: user1 -> file:server.go -> tech:postgres
	path, err := engine.FindPath(userNode.ID, postgresNode.ID, 5)
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}

	if path == nil {
		t.Fatal("Expected to find a path")
	}

	if len(path) < 3 {
		t.Errorf("Expected path length >= 3, got %d", len(path))
	}

	if path[0] != userNode.ID {
		t.Errorf("Path should start with user node")
	}
	if path[len(path)-1] != postgresNode.ID {
		t.Errorf("Path should end with postgres node")
	}
}

func TestFindPath_NoPath(t *testing.T) {
	store, engine, cleanup := setupTestGraph(t)
	defer cleanup()

	// Create isolated node
	isolatedNode, _ := store.GetOrCreateNode("user", "isolated", nil)
	userNode, _ := store.GetNodeByTypeAndName("user", "user1")

	path, err := engine.FindPath(userNode.ID, isolatedNode.ID, 5)
	if err != nil {
		t.Fatalf("FindPath should not error: %v", err)
	}

	if path != nil {
		t.Error("Expected no path to isolated node")
	}
}

func TestDefaultQueryOptions(t *testing.T) {
	opts := DefaultQueryOptions()

	if opts.MaxHops != 2 {
		t.Errorf("Default MaxHops should be 2, got %d", opts.MaxHops)
	}
	if opts.Limit != 10 {
		t.Errorf("Default Limit should be 10, got %d", opts.Limit)
	}
	if opts.MinScore != 0.1 {
		t.Errorf("Default MinScore should be 0.1, got %f", opts.MinScore)
	}
	if opts.DecayRate != 0.7 {
		t.Errorf("Default DecayRate should be 0.7, got %f", opts.DecayRate)
	}
}
