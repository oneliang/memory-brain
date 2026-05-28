package search

import (
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/oneliang/memory-brain/pkg/types"
)

// BM25 constants (borrowed from agentmemory)
const (
	// K1 is the BM25 k1 parameter (term frequency saturation)
	K1 float64 = 1.2

	// B is the BM25 b parameter (length normalization)
	B float64 = 0.75

	// AvgDocLength is the average document length (estimated)
	AvgDocLength float64 = 100.0
)

// BM25Index represents a BM25 inverted index
type BM25Index struct {
	// Inverted index: term -> list of (docID, frequency)
	index map[string][]docFreq

	// Document lengths
	docLengths map[string]int

	// Total number of documents
	totalDocs int

	// Document store: docID -> content
	documents map[string]*IndexDoc

	mu sync.RWMutex
}

// docFreq represents a document-frequency pair
type docFreq struct {
	docID string
	freq  int
}

// IndexDoc represents a document in the index
type IndexDoc struct {
	ID        string
	Title     string
	Content   string
	SessionID string
	Timestamp int64
}

// NewBM25Index creates a new BM25 index
func NewBM25Index() *BM25Index {
	return &BM25Index{
		index:     make(map[string][]docFreq),
		docLengths: make(map[string]int),
		documents: make(map[string]*IndexDoc),
	}
}

// AddDocument adds a document to the index
func (idx *BM25Index) AddDocument(doc *IndexDoc) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Tokenize content
	tokens := DefaultTokenizer.Tokenize(doc.Content + " " + doc.Title)

	// Count term frequencies
	freqMap := make(map[string]int)
	for _, token := range tokens {
		freqMap[token]++
	}

	// Update inverted index
	for term, freq := range freqMap {
		idx.index[term] = append(idx.index[term], docFreq{
			docID: doc.ID,
			freq:  freq,
		})
	}

	// Store document length
	idx.docLengths[doc.ID] = len(tokens)

	// Store document
	idx.documents[doc.ID] = doc

	// Update total docs
	idx.totalDocs++
}

// RemoveDocument removes a document from the index
func (idx *BM25Index) RemoveDocument(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove from inverted index
	for term, dfs := range idx.index {
		var newDfs []docFreq
		for _, df := range dfs {
			if df.docID != docID {
				newDfs = append(newDfs, df)
			}
		}
		if len(newDfs) == 0 {
			delete(idx.index, term)
		} else {
			idx.index[term] = newDfs
		}
	}

	// Remove document length
	delete(idx.docLengths, docID)

	// Remove document
	delete(idx.documents, docID)

	// Update total docs
	idx.totalDocs--
}

// Search performs BM25 search
func (idx *BM25Index) Search(query string, limit int) []types.SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.totalDocs == 0 {
		return []types.SearchResult{}
	}

	// Tokenize query
	queryTokens := DefaultTokenizer.Tokenize(query)

	// Calculate BM25 scores for each document
	scores := make(map[string]float64)

	for _, term := range queryTokens {
		// Get documents containing this term
		dfs, exists := idx.index[term]
		if !exists {
			continue
		}

		// Calculate IDF
		df := float64(len(dfs)) // Document frequency
		totalDocs := float64(idx.totalDocs)
		idf := math.Log((totalDocs - df + 0.5) / (df + 0.5) + 1)

		// Calculate BM25 score for each document
		for _, dfEntry := range dfs {
			docID := dfEntry.docID
			tf := float64(dfEntry.freq) // Term frequency

			// Document length
			docLen := float64(idx.docLengths[docID])

			// BM25 formula
			score := idf * (tf * (K1 + 1.0)) / (tf + K1 * (1.0 - B + B * (docLen / AvgDocLength)))

			scores[docID] += score
		}
	}

	// Sort by score
	var results []types.SearchResult
	for docID, score := range scores {
		doc := idx.documents[docID]
		if doc == nil {
			continue
		}
		results = append(results, types.SearchResult{
			ObsID:     doc.ID,
			SessionID: doc.SessionID,
			Title:     doc.Title,
			Score:     score,
			Source:    "bm25",
			Summary:   truncateSummary(doc.Content, 100),
		})
	}

	// Sort descending by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// truncateSummary truncates content to maxLen
func truncateSummary(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// GetDocument retrieves a document by ID
func (idx *BM25Index) GetDocument(docID string) *IndexDoc {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.documents[docID]
}

// Stats returns index statistics
func (idx *BM25Index) Stats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return map[string]interface{}{
		"total_docs":   idx.totalDocs,
		"total_terms":  len(idx.index),
		"avg_doc_len":  AvgDocLength,
	}
}

// Clear clears the index
func (idx *BM25Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.index = make(map[string][]docFreq)
	idx.docLengths = make(map[string]int)
	idx.documents = make(map[string]*IndexDoc)
	idx.totalDocs = 0
}

// BuildIndexFromSummaries builds index from session summaries
func (idx *BM25Index) BuildIndexFromSummaries(summaries []types.SessionSummary) {
	for _, summary := range summaries {
		doc := &IndexDoc{
			ID:        summary.ID,
			Title:     summary.Title,
			Content:   summary.Narrative + " " + strings.Join(summary.Facts, " ") + " " + strings.Join(summary.Concepts, " "),
			SessionID: summary.SessionID,
			Timestamp: summary.CreatedAt.Unix(),
		}
		idx.AddDocument(doc)
	}
}