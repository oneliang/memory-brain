package search

import (
	"context"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// TestHybridSearchEngine_Creation tests engine creation
func TestHybridSearchEngine_Creation(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create hybrid engine: %v", err)
	}

	if engine == nil {
		t.Fatal("engine should not be nil")
	}

	if engine.GetBM25Index() == nil {
		t.Error("BM25 index should be initialized")
	}

	// Vector index may be nil if chromem fails
	// This is acceptable behavior
}

// TestHybridSearchEngine_AddDocument tests adding documents
func TestHybridSearchEngine_AddDocument(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	doc := &IndexDoc{
		ID:        "doc_1",
		Title:     "Test Document",
		Content:   "This is test content for hybrid search",
		SessionID: "session_1",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()
	err = engine.AddDocument(ctx, doc)
	if err != nil {
		t.Errorf("AddDocument should not error: %v", err)
	}

	// BM25 should have the document
	bm25Results := engine.GetBM25Index().Search("test", 5)
	if len(bm25Results) == 0 {
		t.Error("BM25 should return results after adding document")
	}
}

// TestHybridSearchEngine_Search tests hybrid search
func TestHybridSearchEngine_Search(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Add documents first
	docs := []*IndexDoc{
		{ID: "doc_1", Title: "Go Programming", Content: "Learning Go language basics", SessionID: "s1"},
		{ID: "doc_2", Title: "Python Tutorial", Content: "Python programming guide", SessionID: "s2"},
	}

	ctx := context.Background()
	for _, doc := range docs {
		engine.AddDocument(ctx, doc)
	}

	// Perform search
	results, err := engine.Search(ctx, "programming", 5)
	if err != nil {
		t.Errorf("Search should not error: %v", err)
	}

	// Should return results from BM25 (vector may not be available)
	if len(results) == 0 {
		t.Error("Search should return results")
	}
}

// TestHybridSearchEngine_SearchWithWeights tests weighted search
func TestHybridSearchEngine_SearchWithWeights(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	doc := &IndexDoc{
		ID:        "doc_1",
		Title:     "Test Document",
		Content:   "Test content",
		SessionID: "s1",
	}

	ctx := context.Background()
	engine.AddDocument(ctx, doc)

	results, err := engine.SearchWithWeights(ctx, "test", 5, 0.7, 0.3)
	if err != nil {
		t.Errorf("SearchWithWeights should not error: %v", err)
	}

	// Should return results
	if len(results) == 0 {
		t.Error("SearchWithWeights should return results")
	}
}

// TestHybridSearchEngine_Stats tests statistics
func TestHybridSearchEngine_Stats(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	stats := engine.Stats()

	if stats == nil {
		t.Fatal("stats should not be nil")
	}

	// Should have BM25 stats
	if _, ok := stats["total_docs"]; !ok {
		t.Error("stats should contain total_docs")
	}

	// Should indicate vector availability
	if _, ok := stats["vector_available"]; !ok {
		t.Error("stats should contain vector_available")
	}
}

// TestHybridSearchEngine_Clear tests clearing engine
func TestHybridSearchEngine_Clear(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Add document
	doc := &IndexDoc{ID: "doc_1", Title: "Test", Content: "Content"}
	engine.AddDocument(context.Background(), doc)

	// Clear
	err = engine.Clear()
	if err != nil {
		t.Errorf("Clear should not error: %v", err)
	}

	// BM25 should be empty
	stats := engine.GetBM25Index().Stats()
	if stats["total_docs"] != 0 {
		t.Error("BM25 should be empty after clear")
	}
}

// TestHybridSearchEngine_BuildFromSummaries tests building from summaries
func TestHybridSearchEngine_BuildFromSummaries(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	summaries := []types.SessionSummary{
		{
			ID:        "s1",
			SessionID: "session_1",
			UserID:    "user_1",
			Title:     "Test Session",
			Narrative: "This session discusses programming",
			CreatedAt: time.Now(),
		},
	}

	ctx := context.Background()
	err = engine.BuildFromSummaries(ctx, summaries)
	if err != nil {
		// May fail if vector build fails (Ollama not available)
		t.Logf("BuildFromSummaries: %v", err)
	}

	// BM25 should have documents
	stats := engine.GetBM25Index().Stats()
	if stats["total_docs"] == 0 {
		t.Error("BM25 should have documents after build")
	}
}

// TestHybridSearchEngine_HasVectorIndex tests vector availability check
func TestHybridSearchEngine_HasVectorIndex(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// May or may not have vector index depending on chromem availability
	// Just test that the method works
	hasVector := engine.HasVectorIndex()
	_ = hasVector // Value depends on runtime conditions
}

// TestHybridSearchEngine_GetVectorIndex tests getting vector index
func TestHybridSearchEngine_GetVectorIndex(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	vectorIndex := engine.GetVectorIndex()
	// May be nil if chromem not available - acceptable
	_ = vectorIndex
}

// TestHybridSearchEngine_EmptySearch tests search on empty engine
func TestHybridSearchEngine_EmptySearch(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()
	results, err := engine.Search(ctx, "nonexistent", 5)
	if err != nil {
		t.Errorf("Search on empty engine should not error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Empty engine should return empty results, got %d", len(results))
	}
}

// TestHybridSearchEngine_MultipleDocuments tests multiple document handling
func TestHybridSearchEngine_MultipleDocuments(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	for i := 0; i < 10; i++ {
		doc := &IndexDoc{
			ID:        string(rune('a' + i)),
			Title:     "Document " + string(rune('a'+i)),
			Content:   "Content for document",
			SessionID: "session_1",
		}
		engine.AddDocument(context.Background(), doc)
	}

	// All should be indexed in BM25
	stats := engine.GetBM25Index().Stats()
	if stats["total_docs"] != 10 {
		t.Errorf("expected 10 documents, got %v", stats["total_docs"])
	}
}

// TestHybridSearchEngine_RRFIntegration tests that RRF fusion is used
func TestHybridSearchEngine_RRFIntegration(t *testing.T) {
	engine, err := NewHybridSearchEngine("")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Add documents with different keywords
	docs := []*IndexDoc{
		{ID: "bm25_doc", Title: "API Design", Content: "REST API architecture patterns", SessionID: "s1"},
		{ID: "vector_doc", Title: "Architecture Guide", Content: "System design and architecture", SessionID: "s2"},
	}

	for _, doc := range docs {
		engine.AddDocument(context.Background(), doc)
	}

	// Search for "architecture" - should return both if RRF works
	results, _ := engine.Search(context.Background(), "architecture", 5)

	// Should return results
	if len(results) < 1 {
		t.Error("Search should return at least one result for 'architecture'")
	}
}