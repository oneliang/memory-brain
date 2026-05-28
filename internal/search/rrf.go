package search

import (
	"sort"

	"github.com/oneliang/memory-brain/pkg/types"
)

// RRF constants (borrowed from agentmemory)
const (
	// RRF_K is the Reciprocal Rank Fusion smoothing parameter
	RRF_K = 60

	// Default weights for each stream
	BM25Weight   = 0.4
	VectorWeight = 0.6
	ContextWeight = 0.3
)

// RRFResult represents a fused result
type RRFResult struct {
	types.SearchResult
	BM25Rank   int
	VectorRank int
	ContextRank int
	CombinedScore float64
}

// RRFFusion performs Reciprocal Rank Fusion on multiple result streams
func RRFFusion(bm25Results, vectorResults []types.SearchResult, weights ...float64) []types.SearchResult {
	// Use default weights if not provided
	var bm25Weight, vectorWeight, contextWeight float64
	switch len(weights) {
	case 0:
		bm25Weight = BM25Weight
		vectorWeight = VectorWeight
		contextWeight = ContextWeight
	case 1:
		bm25Weight = weights[0]
		vectorWeight = VectorWeight
		contextWeight = ContextWeight
	case 2:
		bm25Weight = weights[0]
		vectorWeight = weights[1]
		contextWeight = ContextWeight
	default:
		bm25Weight = weights[0]
		vectorWeight = weights[1]
		contextWeight = weights[2]
	}

	// Normalize weights
	totalWeight := bm25Weight + vectorWeight + contextWeight
	bm25Weight /= totalWeight
	vectorWeight /= totalWeight
	contextWeight /= totalWeight

	// Track results by ID
	combined := make(map[string]*RRFResult)

	// Process BM25 results
	for i, result := range bm25Results {
		rank := float64(i + 1)
		score := bm25Weight * (1.0 / (float64(RRF_K) + rank))
		combined[result.ObsID] = &RRFResult{
			SearchResult:  result,
			BM25Rank:      i + 1,
			VectorRank:    -1, // Not found in vector
			ContextRank:   -1,
			CombinedScore: score,
		}
	}

	// Process Vector results
	for i, result := range vectorResults {
		rank := float64(i + 1)
		score := vectorWeight * (1.0 / (float64(RRF_K) + rank))

		if existing, ok := combined[result.ObsID]; ok {
			// Document exists in BM25, add to combined score
			existing.VectorRank = i + 1
			existing.CombinedScore += score
		} else {
			// New document from vector search
			combined[result.ObsID] = &RRFResult{
				SearchResult:  result,
				BM25Rank:      -1,
				VectorRank:    i + 1,
				ContextRank:   -1,
				CombinedScore: score,
			}
		}
	}

	// Convert to slice and sort
	var results []types.SearchResult
	for _, r := range combined {
		r.SearchResult.Score = r.CombinedScore
		r.SearchResult.Source = "rrf"
		results = append(results, r.SearchResult)
	}

	// Sort by combined score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// DiversifyBySession diversifies results to avoid same session dominance
func DiversifyBySession(results []types.SearchResult, limit, maxPerSession int) []types.SearchResult {
	var diversified []types.SearchResult
	sessionCounts := make(map[string]int)

	for _, result := range results {
		count := sessionCounts[result.SessionID]
		if count >= maxPerSession {
			continue // Skip, already have max from this session
		}

		diversified = append(diversified, result)
		sessionCounts[result.SessionID] = count + 1

		if len(diversified) >= limit {
			break
		}
	}

	// If still not enough, fill with remaining
	if len(diversified) < limit {
		for _, result := range results {
			// Check if already included
			var included bool
			for _, d := range diversified {
				if d.ObsID == result.ObsID {
					included = true
					break
				}
			}
			if !included {
				diversified = append(diversified, result)
				if len(diversified) >= limit {
					break
				}
			}
		}
	}

	return diversified
}

// HybridSearch performs combined BM25 + Vector search with RRF fusion
func HybridSearch(bm25Results, vectorResults []types.SearchResult, limit int) []types.SearchResult {
	// Fuse results with RRF
	fused := RRFFusion(bm25Results, vectorResults)

	// Diversify by session (max 3 per session, borrowed from agentmemory)
	return DiversifyBySession(fused, limit, 3)
}

// WeightedSearch performs weighted search based on query type
func WeightedSearch(bm25Results, vectorResults []types.SearchResult, queryType string, limit int) []types.SearchResult {
	var weights []float64

	switch queryType {
	case "keyword":
		// Prefer BM25 for keyword queries
		weights = []float64{0.8, 0.2, 0.0}
	case "semantic":
		// Prefer Vector for semantic queries
		weights = []float64{0.2, 0.8, 0.0}
	case "mixed":
		// Balanced weights
		weights = []float64{0.4, 0.6, 0.0}
	default:
		// Default weights
		weights = []float64{BM25Weight, VectorWeight, ContextWeight}
	}

	fused := RRFFusion(bm25Results, vectorResults, weights...)
	return DiversifyBySession(fused, limit, 3)
}