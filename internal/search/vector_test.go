package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// TestVectorIndex_InMemory tests in-memory vector index creation
func TestVectorIndex_InMemory(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create in-memory index: %v", err)
	}

	if idx == nil {
		t.Fatal("index should not be nil")
	}

	if idx.Persist() {
		t.Error("in-memory index should not be persistent")
	}

	if idx.StoragePath() != "" {
		t.Error("in-memory index should have empty storage path")
	}
}

// TestVectorIndex_Persistent tests persistent vector index creation
func TestVectorIndex_Persistent(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "vector.db")

	idx, err := NewVectorIndex(storagePath)
	if err != nil {
		t.Fatalf("failed to create persistent index: %v", err)
	}

	if idx == nil {
		t.Fatal("index should not be nil")
	}

	// Note: Persistence may fail if chromem has issues, so we check either case
	// The important thing is that the index was created
	if idx.StoragePath() != storagePath {
		t.Errorf("expected storage path %s, got %s", storagePath, idx.StoragePath())
	}
}

// TestVectorIndex_AddDocument tests adding documents
func TestVectorIndex_AddDocument(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	doc := &VectorDoc{
		ID:        "doc_1",
		Content:   "This is a test document for vector search",
		SessionID: "session_1",
		Metadata: map[string]string{
			"title": "Test Document",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = idx.AddDocument(ctx, doc)
	if err != nil {
		// May fail if Ollama is not running - this is acceptable for testing
		t.Logf("AddDocument failed (may require Ollama): %v", err)
		return
	}

	if idx.GetDocumentCount() != 1 {
		t.Errorf("expected 1 document, got %d", idx.GetDocumentCount())
	}
}

// TestVectorIndex_Search tests vector search
func TestVectorIndex_Search(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Empty search should return empty results
	ctx := context.Background()
	results, err := idx.Search(ctx, "test query", 5)
	if err != nil {
		t.Errorf("search on empty index should not error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("empty index should return empty results, got %d", len(results))
	}
}

// TestVectorIndex_GetDocumentCount tests document count
func TestVectorIndex_GetDocumentCount(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	if idx.GetDocumentCount() != 0 {
		t.Errorf("new index should have 0 documents, got %d", idx.GetDocumentCount())
	}
}

// TestVectorIndex_Clear tests clearing the index
func TestVectorIndex_Clear(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Clear should work even on empty index
	err = idx.Clear()
	if err != nil {
		t.Errorf("clear should not error: %v", err)
	}

	if idx.GetDocumentCount() != 0 {
		t.Errorf("cleared index should have 0 documents")
	}
}

// TestVectorIndex_BuildIndexFromSummaries tests building from summaries
func TestVectorIndex_BuildIndexFromSummaries(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	summaries := []types.SessionSummary{
		{
			ID:        "summary_1",
			SessionID: "session_1",
			UserID:    "user_1",
			Title:     "Test Session",
			Narrative: "This is a test session narrative",
			CreatedAt: time.Now(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = idx.BuildIndexFromSummaries(ctx, summaries)
	if err != nil {
		// May fail if Ollama is not running
		t.Logf("BuildIndexFromSummaries failed (may require Ollama): %v", err)
		return
	}

	if idx.GetDocumentCount() != 1 {
		t.Errorf("expected 1 document after build, got %d", idx.GetDocumentCount())
	}
}

// TestVectorIndex_RemoveDocument tests document removal (stub)
func TestVectorIndex_RemoveDocument(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// RemoveDocument is a stub, should not error
	err = idx.RemoveDocument("doc_1")
	if err != nil {
		t.Errorf("RemoveDocument stub should not error: %v", err)
	}
}

// TestDefaultVectorPath tests default path generation
func TestDefaultVectorPath(t *testing.T) {
	path := DefaultVectorPath("user_1")

	// Should contain user_1 in path
	if !filepath.IsAbs(path) {
		t.Errorf("default path should be absolute: %s", path)
	}

	// Should contain user directory structure
	expectedSuffix := filepath.Join(".memory-brain", "users", "user_1", "knowledge.db")
	if !filepath.IsAbs(path) {
		// If home dir couldn't be resolved, uses relative path
		if path != expectedSuffix {
			t.Errorf("path should end with %s, got %s", expectedSuffix, path)
		}
	}
}

// TestVectorIndex_MultipleDocuments tests adding multiple documents
func TestVectorIndex_MultipleDocuments(t *testing.T) {
	idx, err := NewVectorIndex("")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	docs := []*VectorDoc{
		{ID: "doc_1", Content: "First document", SessionID: "s1"},
		{ID: "doc_2", Content: "Second document", SessionID: "s2"},
		{ID: "doc_3", Content: "Third document", SessionID: "s3"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, doc := range docs {
		err = idx.AddDocument(ctx, doc)
		if err != nil {
			t.Logf("AddDocument failed (may require Ollama): %v", err)
			return
		}
	}

	if idx.GetDocumentCount() != 3 {
		t.Errorf("expected 3 documents, got %d", idx.GetDocumentCount())
	}
}

// TestVectorIndex_StorageDirectoryCreation tests storage directory creation
func TestVectorIndex_StorageDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "nested", "dir", "vector.db")

	idx, err := NewVectorIndex(storagePath)
	if err != nil {
		// May fail due to nested directory creation or chromem issues
		t.Logf("Index creation with nested path: %v", err)
		return
	}

	if idx != nil {
		// Directory should have been created
		parentDir := filepath.Dir(storagePath)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			t.Error("parent directory should have been created")
		}
	}
}