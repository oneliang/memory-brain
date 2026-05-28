package search

import (
	"testing"
)

func TestBM25Index_AddDocument(t *testing.T) {
	idx := NewBM25Index()

	doc := &IndexDoc{
		ID:      "doc_1",
		Title:   "架构设计方案",
		Content: "采用 REST API + Hook 机制实现独立服务",
	}

	idx.AddDocument(doc)

	stats := idx.Stats()
	totalDocs := stats["total_docs"].(int)
	if totalDocs != 1 {
		t.Errorf("Expected 1 document, got %d", totalDocs)
	}
}

func TestBM25Index_Search(t *testing.T) {
	idx := NewBM25Index()

	// Add multiple documents
	docs := []*IndexDoc{
		{ID: "doc_1", Title: "Architecture Design", Content: "REST API design document"},
		{ID: "doc_2", Title: "User Profile System", Content: "Four layer memory integration"},
		{ID: "doc_3", Title: "Search Engine", Content: "BM25 and Vector hybrid search"},
	}

	for _, doc := range docs {
		idx.AddDocument(doc)
	}

	// Search for "architecture"
	results := idx.Search("architecture", 5)

	if len(results) == 0 {
		t.Error("Expected results for 'architecture', got none")
	}

	// doc_1 should be in results (contains "Architecture" in title)
	found := false
	for _, r := range results {
		if r.ObsID == "doc_1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected doc_1 in results for 'architecture'")
	}
}

func TestBM25Index_SearchMultipleKeywords(t *testing.T) {
	idx := NewBM25Index()

	docs := []*IndexDoc{
		{ID: "doc_1", Title: "Go 语言编程", Content: "使用 Go 实现 API 服务"},
		{ID: "doc_2", Title: "Python 开发", Content: "Python 数据处理"},
		{ID: "doc_3", Title: "Go 和 Python 对比", Content: "Go Python 性能对比"},
	}

	for _, doc := range docs {
		idx.AddDocument(doc)
	}

	// Search for "Go"
	results := idx.Search("Go", 5)

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'Go', got %d", len(results))
	}
}

func TestBM25Index_EmptySearch(t *testing.T) {
	idx := NewBM25Index()

	// Search empty index
	results := idx.Search("test", 5)

	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty index, got %d", len(results))
	}
}

func TestBM25Index_Clear(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument(&IndexDoc{ID: "doc_1", Title: "Test", Content: "Content"})
	idx.Clear()

	stats := idx.Stats()
	totalDocs := stats["total_docs"].(int)
	if totalDocs != 0 {
		t.Errorf("Expected 0 documents after clear, got %d", totalDocs)
	}
}

func TestBM25Index_DocumentCount(t *testing.T) {
	idx := NewBM25Index()

	for i := 0; i < 10; i++ {
		idx.AddDocument(&IndexDoc{
			ID:      "doc_" + string(rune('0'+i)),
			Title:   "Test",
			Content: "Content",
		})
	}

	stats := idx.Stats()
	totalDocs := stats["total_docs"].(int)
	if totalDocs != 10 {
		t.Errorf("Expected 10 documents, got %d", totalDocs)
	}
}