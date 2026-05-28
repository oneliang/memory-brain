package search

import (
	"context"
	"log"
	"sync"

	"github.com/oneliang/memory-brain/pkg/types"
)

// HybridSearchEngine combines BM25 and Vector search with RRF fusion
type HybridSearchEngine struct {
	bm25Index   *BM25Index
	vectorIndex *VectorIndex
	mu          sync.RWMutex
}

// NewHybridSearchEngine creates a new hybrid search engine
func NewHybridSearchEngine(vectorStoragePath string) (*HybridSearchEngine, error) {
	// Create BM25 index
	bm25 := NewBM25Index()

	// Create Vector index (may fail if chromem not available)
	vector, err := NewVectorIndex(vectorStoragePath)
	if err != nil {
		// Log warning but continue without vector search
		vector = nil
	}

	return &HybridSearchEngine{
		bm25Index:   bm25,
		vectorIndex: vector,
	}, nil
}

// AddDocument adds a document to both indexes
func (e *HybridSearchEngine) AddDocument(ctx context.Context, doc *IndexDoc) error {
	// Add to BM25
	e.bm25Index.AddDocument(doc)

	// Add to Vector (if available)
	if e.vectorIndex != nil {
		vectorDoc := &VectorDoc{
			ID:        doc.ID,
			Content:   doc.Title + " " + doc.Content,
			SessionID: doc.SessionID,
			Metadata: map[string]string{
				"session_id": doc.SessionID,
				"title":      doc.Title,
			},
		}
		if err := e.vectorIndex.AddDocument(ctx, vectorDoc); err != nil {
			log.Printf("Failed to add document to vector index: %v", err)
		}
	}

	return nil
}

// Search performs hybrid search with RRF fusion
func (e *HybridSearchEngine) Search(ctx context.Context, query string, limit int) ([]types.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var bm25Results, vectorResults []types.SearchResult

	// BM25 search
	bm25Results = e.bm25Index.Search(query, limit)

	// Vector search (if available)
	if e.vectorIndex != nil {
		vectorResults, _ = e.vectorIndex.Search(ctx, query, limit)
	}

	// Fuse with RRF
	return HybridSearch(bm25Results, vectorResults, limit), nil
}

// SearchWithWeights performs weighted hybrid search
func (e *HybridSearchEngine) SearchWithWeights(ctx context.Context, query string, limit int, bm25Weight, vectorWeight float64) ([]types.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var bm25Results, vectorResults []types.SearchResult

	bm25Results = e.bm25Index.Search(query, limit)

	if e.vectorIndex != nil {
		vectorResults, _ = e.vectorIndex.Search(ctx, query, limit)
	}

	return WeightedSearch(bm25Results, vectorResults, "mixed", limit), nil
}

// GetBM25Index returns the BM25 index for direct access
func (e *HybridSearchEngine) GetBM25Index() *BM25Index {
	return e.bm25Index
}

// GetVectorIndex returns the Vector index for direct access
func (e *HybridSearchEngine) GetVectorIndex() *VectorIndex {
	return e.vectorIndex
}

// HasVectorIndex checks if vector index is available
func (e *HybridSearchEngine) HasVectorIndex() bool {
	return e.vectorIndex != nil
}

// Stats returns search engine statistics
func (e *HybridSearchEngine) Stats() map[string]interface{} {
	stats := e.bm25Index.Stats()
	stats["vector_available"] = e.HasVectorIndex()
	if e.vectorIndex != nil {
		stats["vector_docs"] = e.vectorIndex.GetDocumentCount()
	}
	return stats
}

// Clear clears all indexes
func (e *HybridSearchEngine) Clear() error {
	e.bm25Index.Clear()
	if e.vectorIndex != nil {
		e.vectorIndex.Clear()
	}
	return nil
}

// BuildFromSummaries builds indexes from session summaries
func (e *HybridSearchEngine) BuildFromSummaries(ctx context.Context, summaries []types.SessionSummary) error {
	// Build BM25 index
	e.bm25Index.BuildIndexFromSummaries(summaries)

	// Build Vector index (if available)
	if e.vectorIndex != nil {
		return e.vectorIndex.BuildIndexFromSummaries(ctx, summaries)
	}

	return nil
}