package search

import (
	"testing"

	"github.com/oneliang/memory-brain/pkg/types"
)

func TestRRFFusion_Basic(t *testing.T) {
	bm25Results := []types.SearchResult{
		{ObsID: "a", Score: 0.8, Source: "bm25"},
		{ObsID: "b", Score: 0.6, Source: "bm25"},
		{ObsID: "c", Score: 0.4, Source: "bm25"},
	}

	vectorResults := []types.SearchResult{
		{ObsID: "b", Score: 0.9, Source: "vector"},
		{ObsID: "d", Score: 0.7, Source: "vector"},
		{ObsID: "a", Score: 0.5, Source: "vector"},
	}

	fused := RRFFusion(bm25Results, vectorResults)

	if len(fused) != 4 {
		t.Errorf("Expected 4 unique results, got %d", len(fused))
	}

	// "b" appears in both lists, should have highest combined score
	if fused[0].ObsID != "b" {
		t.Errorf("Expected 'b' as top result (appears in both lists), got '%s'", fused[0].ObsID)
	}
}

func TestRRFFusion_EmptyResults(t *testing.T) {
	fused := RRFFusion(nil, nil)

	if len(fused) != 0 {
		t.Errorf("Expected 0 results for empty inputs, got %d", len(fused))
	}
}

func TestRRFFusion_SingleList(t *testing.T) {
	bm25Results := []types.SearchResult{
		{ObsID: "a", Score: 0.8, Source: "bm25"},
		{ObsID: "b", Score: 0.6, Source: "bm25"},
	}

	fused := RRFFusion(bm25Results, nil)

	if len(fused) != 2 {
		t.Errorf("Expected 2 results, got %d", len(fused))
	}

	// Order should be preserved from single list
	if fused[0].ObsID != "a" {
		t.Errorf("Expected 'a' as first result, got '%s'", fused[0].ObsID)
	}
}

func TestRRFFusion_WithWeights(t *testing.T) {
	bm25Results := []types.SearchResult{
		{ObsID: "a", Score: 0.8, Source: "bm25"},
		{ObsID: "b", Score: 0.6, Source: "bm25"},
	}

	vectorResults := []types.SearchResult{
		{ObsID: "b", Score: 0.9, Source: "vector"},
		{ObsID: "a", Score: 0.7, Source: "vector"},
	}

	// Use custom weights (bm25=0.5, vector=0.5)
	fused := RRFFusion(bm25Results, vectorResults, 0.5, 0.5)

	if len(fused) != 2 {
		t.Errorf("Expected 2 results, got %d", len(fused))
	}
}

func TestRRFFusion_DifferentOrders(t *testing.T) {
	// BM25: a first, b second
	bm25Results := []types.SearchResult{
		{ObsID: "a", Score: 0.8, Source: "bm25"},
		{ObsID: "b", Score: 0.6, Source: "bm25"},
	}

	// Vector: b first, a second
	vectorResults := []types.SearchResult{
		{ObsID: "b", Score: 0.9, Source: "vector"},
		{ObsID: "a", Score: 0.7, Source: "vector"},
	}

	fused := RRFFusion(bm25Results, vectorResults)

	// Both appear in both lists, but b is ranked higher in vector (higher weight)
	if fused[0].ObsID != "b" {
		t.Errorf("Expected 'b' as top result (higher vector rank), got '%s'", fused[0].ObsID)
	}
}

func TestRRFConstants(t *testing.T) {
	// Verify RRF_K constant
	if RRF_K != 60 {
		t.Errorf("Expected RRF_K=60 (standard value), got %d", RRF_K)
	}

	// Verify default weights
	if BM25Weight != 0.4 {
		t.Errorf("Expected BM25Weight=0.4, got %f", BM25Weight)
	}
	if VectorWeight != 0.6 {
		t.Errorf("Expected VectorWeight=0.6, got %f", VectorWeight)
	}
}

func TestDiversifyBySession(t *testing.T) {
	results := []types.SearchResult{
		{ObsID: "a", SessionID: "s1", Score: 0.9},
		{ObsID: "b", SessionID: "s1", Score: 0.8},
		{ObsID: "c", SessionID: "s1", Score: 0.7},
		{ObsID: "d", SessionID: "s2", Score: 0.6},
		{ObsID: "e", SessionID: "s2", Score: 0.5},
		{ObsID: "f", SessionID: "s3", Score: 0.4},
	}

	// Diversify with max 2 per session and limit = 5 (won't trigger fill loop)
	diversified := DiversifyBySession(results, 5, 2)

	// With limit=5 and maxPerSession=2:
	// First loop: s1:2, s2:2, s3:1 = 5 results (limit reached)
	// Second loop won't run
	// s1 should have exactly 2
	s1Count := 0
	for _, r := range diversified {
		if r.SessionID == "s1" {
			s1Count++
		}
	}

	if s1Count > 2 {
		t.Errorf("Expected at most 2 from s1 (limit=5 triggers no fill), got %d", s1Count)
	}

	// Total should be exactly 5 (limit reached)
	if len(diversified) != 5 {
		t.Errorf("Expected 5 total results, got %d", len(diversified))
	}
}

func TestDiversifyBySession_SingleSession(t *testing.T) {
	results := []types.SearchResult{
		{ObsID: "a", SessionID: "s1", Score: 0.9},
		{ObsID: "b", SessionID: "s1", Score: 0.8},
		{ObsID: "c", SessionID: "s1", Score: 0.7},
	}

	// With limit=2 and maxPerSession=2
	diversified := DiversifyBySession(results, 2, 2)

	// Should only have 2 results (limit reached after 2 from s1)
	if len(diversified) != 2 {
		t.Errorf("Expected 2 results after diversification (limit=2), got %d", len(diversified))
	}
}

func TestDiversifyBySession_Empty(t *testing.T) {
	diversified := DiversifyBySession(nil, 10, 2)

	if len(diversified) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(diversified))
	}
}