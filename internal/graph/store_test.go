package graph

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestStore creates a temporary store for testing
func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "graph_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewStoreWithPath(dbPath)
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

func TestNewStore(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Verify store is created
	if store == nil {
		t.Fatal("Store should not be nil")
	}

	// Verify database is accessible
	nodeCount, err := store.GetNodeCount()
	if err != nil {
		t.Fatalf("Failed to get node count: %v", err)
	}
	if nodeCount != 0 {
		t.Errorf("Expected 0 nodes, got %d", nodeCount)
	}
}

func TestGetOrCreateNode(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Test creating a new node
	node1, err := store.GetOrCreateNode("person", "张三", map[string]interface{}{
		"role": "developer",
	})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	if node1.ID == 0 {
		t.Error("Node ID should not be 0")
	}
	if node1.Type != "person" {
		t.Errorf("Expected type 'person', got '%s'", node1.Type)
	}
	if node1.Name != "张三" {
		t.Errorf("Expected name '张三', got '%s'", node1.Name)
	}

	// Test getting existing node (should return same ID)
	node2, err := store.GetOrCreateNode("person", "张三", nil)
	if err != nil {
		t.Fatalf("Failed to get existing node: %v", err)
	}
	if node2.ID != node1.ID {
		t.Errorf("Expected same ID %d, got %d", node1.ID, node2.ID)
	}

	// Test creating different node
	node3, err := store.GetOrCreateNode("tool", "bash", nil)
	if err != nil {
		t.Fatalf("Failed to create different node: %v", err)
	}
	if node3.ID == node1.ID {
		t.Error("Different node should have different ID")
	}
}

func TestCreateEdge(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create two nodes
	userNode, _ := store.GetOrCreateNode("user", "user1", nil)
	toolNode, _ := store.GetOrCreateNode("tool", "bash", nil)

	// Create edge
	edge, err := store.CreateEdge(userNode.ID, toolNode.ID, "uses", 1.5)
	if err != nil {
		t.Fatalf("Failed to create edge: %v", err)
	}
	if edge.ID == 0 {
		t.Error("Edge ID should not be 0")
	}
	if edge.SourceID != userNode.ID {
		t.Errorf("Expected source ID %d, got %d", userNode.ID, edge.SourceID)
	}
	if edge.TargetID != toolNode.ID {
		t.Errorf("Expected target ID %d, got %d", toolNode.ID, edge.TargetID)
	}
	if edge.Relation != "uses" {
		t.Errorf("Expected relation 'uses', got '%s'", edge.Relation)
	}
	if edge.Weight != 1.5 {
		t.Errorf("Expected weight 1.5, got %f", edge.Weight)
	}
}

func TestGetOutgoingEdges(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create nodes
	userNode, _ := store.GetOrCreateNode("user", "user1", nil)
	tool1Node, _ := store.GetOrCreateNode("tool", "bash", nil)
	tool2Node, _ := store.GetOrCreateNode("tool", "git", nil)

	// Create edges
	store.CreateEdge(userNode.ID, tool1Node.ID, "uses", 1.0)
	store.CreateEdge(userNode.ID, tool2Node.ID, "uses", 2.0)

	// Get outgoing edges
	edges, err := store.GetOutgoingEdges(userNode.ID)
	if err != nil {
		t.Fatalf("Failed to get outgoing edges: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(edges))
	}

	// Verify edges are for correct targets
	targetIDs := make(map[int64]bool)
	for _, edge := range edges {
		targetIDs[edge.TargetID] = true
	}
	if !targetIDs[tool1Node.ID] || !targetIDs[tool2Node.ID] {
		t.Error("Edges should point to tool1 and tool2")
	}
}

func TestSaveExtractionResult(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	result := &ExtractionResult{
		Entities: []Entity{
			{Type: "person", Name: "张三", Properties: map[string]interface{}{"role": "dev"}},
			{Type: "tech", Name: "PostgreSQL", Properties: nil},
		},
		Relations: []Relation{
			{Source: "user1", Target: "PostgreSQL", Type: "works_on", Weight: 1.0},
		},
	}

	err := store.SaveExtractionResult("user1", result)
	if err != nil {
		t.Fatalf("Failed to save extraction result: %v", err)
	}

	// Verify entities were created
	nodeCount, _ := store.GetNodeCount()
	if nodeCount < 3 { // user1 + 张三 + PostgreSQL
		t.Errorf("Expected at least 3 nodes, got %d", nodeCount)
	}

	// Verify edges were created
	edgeCount, _ := store.GetEdgeCount()
	if edgeCount < 1 {
		t.Errorf("Expected at least 1 edge, got %d", edgeCount)
	}
}

func TestGetStats(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create some test data
	store.GetOrCreateNode("user", "user1", nil)
	store.GetOrCreateNode("tool", "bash", nil)
	store.GetOrCreateNode("tool", "git", nil)

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	nodeCount, ok := stats["node_count"].(int)
	if !ok || nodeCount != 3 {
		t.Errorf("Expected 3 nodes in stats, got %v", stats["node_count"])
	}

	nodeTypes, ok := stats["node_types"].(map[string]int)
	if !ok {
		t.Fatal("node_types should be a map")
	}
	if nodeTypes["tool"] != 2 {
		t.Errorf("Expected 2 tool nodes, got %d", nodeTypes["tool"])
	}
}

func TestTruncate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create some data
	store.GetOrCreateNode("user", "user1", nil)
	store.GetOrCreateNode("tool", "bash", nil)

	// Verify data exists
	nodeCount, _ := store.GetNodeCount()
	if nodeCount == 0 {
		t.Fatal("Should have nodes before truncate")
	}

	// Truncate
	err := store.Truncate()
	if err != nil {
		t.Fatalf("Failed to truncate: %v", err)
	}

	// Verify data is gone
	nodeCount, _ = store.GetNodeCount()
	if nodeCount != 0 {
		t.Errorf("Expected 0 nodes after truncate, got %d", nodeCount)
	}
}

func TestSearchNodes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create test nodes
	store.GetOrCreateNode("tool", "bash", nil)
	store.GetOrCreateNode("tool", "git", nil)
	store.GetOrCreateNode("file", "server.go", nil)

	// Search for "b"
	nodes, err := store.SearchNodes("b", 10)
	if err != nil {
		t.Fatalf("Failed to search nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node matching 'b', got %d", len(nodes))
	}
	if len(nodes) > 0 && nodes[0].Name != "bash" {
		t.Errorf("Expected 'bash', got '%s'", nodes[0].Name)
	}
}
