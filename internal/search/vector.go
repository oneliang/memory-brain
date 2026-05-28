package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/philippgille/chromem-go"
	"github.com/oneliang/memory-brain/pkg/types"
)

// VectorIndex represents a Chromem vector index
type VectorIndex struct {
	db           *chromem.DB
	collection   *chromem.Collection
	storagePath  string
	docCount     int
	mu           sync.RWMutex
	persist      bool
}

// VectorDoc represents a document for vector indexing
type VectorDoc struct {
	ID        string
	Content   string
	SessionID string
	Metadata  map[string]string
}

// NewVectorIndex creates a new vector index with optional persistence
func NewVectorIndex(storagePath string) (*VectorIndex, error) {
	var db *chromem.DB
	var persist bool
	var err error

	if storagePath != "" {
		// Ensure storage directory exists
		if err = os.MkdirAll(storagePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}

		// Use persistent DB (path, forceNew=false to load existing)
		persistentDB, err := chromem.NewPersistentDB(storagePath, false)
		if err != nil {
			// Fallback to in-memory DB if persistence fails
			db = chromem.NewDB()
			persist = false
		} else {
			db = persistentDB
			persist = true
		}
	} else {
		// In-memory DB
		db = chromem.NewDB()
		persist = false
	}

	// Create collection with Ollama embedding (local, no API key required)
	// Use OpenAI-compatible API endpoint (Ollama supports this at /v1/embeddings)
	// This avoids the request body format mismatch issue with native Ollama API
	// See: https://github.com/Chromem-go/chromem-go/issues/102
	embeddingFunc := chromem.NewEmbeddingFuncOpenAICompat(
		"http://localhost:11434/v1", // Ollama OpenAI-compatible endpoint
		"",                          // No API key needed for Ollama
		"nomic-embed-text",          // Model name
		nil,                         // No normalized param needed
	)

	// Get existing collection or create new one
	var collection *chromem.Collection
	if persist {
		// Try to get existing collection from persistent DB
		collection = db.GetCollection("memory_brain", embeddingFunc)
	}
	if collection == nil {
		// Collection doesn't exist, create new one
		metadata := map[string]string{"storage_path": storagePath}
		collection, err = db.CreateCollection("memory_brain", metadata, embeddingFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return &VectorIndex{
		db:          db,
		collection:  collection,
		storagePath: storagePath,
		persist:     persist,
	}, nil
}

// AddDocument adds a document to the vector index
func (idx *VectorIndex) AddDocument(ctx context.Context, doc *VectorDoc) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Create Chromem document
	chromemDoc := chromem.Document{
		ID:      doc.ID,
		Content: doc.Content,
		Metadata: doc.Metadata,
	}

	// Add to collection
	err := idx.collection.AddDocuments(ctx, []chromem.Document{chromemDoc}, 1)
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	idx.docCount++
	return nil
}

// RemoveDocument removes a document from the vector index
// NOTE: Chromem-go does not provide a direct deletion API. This method
// is a stub and does not actually remove documents. Alternatives:
// 1. Rebuild the entire index without the document (expensive)
// 2. Use soft-delete: add "deleted" metadata flag and filter in Search
// 3. Let documents expire naturally through the decay mechanism
func (idx *VectorIndex) RemoveDocument(docID string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Chromem doesn't have direct deletion API
	// Documents remain in the index until it's rebuilt
	return nil
}

// Search performs vector similarity search
func (idx *VectorIndex) Search(ctx context.Context, query string, limit int) ([]types.SearchResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.docCount == 0 {
		return []types.SearchResult{}, nil
	}

	// Query the collection
	results, err := idx.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	// Convert to SearchResult
	var searchResults []types.SearchResult
	for _, result := range results {
		sessionID := result.Metadata["session_id"]
		title := result.Metadata["title"]
		if title == "" {
			title = truncateSummary(result.Content, 50)
		}

		searchResults = append(searchResults, types.SearchResult{
			ObsID:     result.ID,
			SessionID: sessionID,
			Title:     title,
			Score:     float64(result.Similarity),
			Source:    "vector",
			Summary:   truncateSummary(result.Content, 100),
		})
	}

	return searchResults, nil
}

// GetDocumentCount returns the number of documents
func (idx *VectorIndex) GetDocumentCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docCount
}

// Clear clears the vector index
func (idx *VectorIndex) Clear() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.docCount = 0
	return nil
}

// BuildIndexFromSummaries builds vector index from session summaries
func (idx *VectorIndex) BuildIndexFromSummaries(ctx context.Context, summaries []types.SessionSummary) error {
	for _, summary := range summaries {
		doc := &VectorDoc{
			ID:        summary.ID,
			Content:   summary.Narrative + " " + summary.Title,
			SessionID: summary.SessionID,
			Metadata: map[string]string{
				"session_id": summary.SessionID,
				"title":      summary.Title,
				"user_id":    summary.UserID,
			},
		}
		if err := idx.AddDocument(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// Persist returns whether the index is persistent
func (idx *VectorIndex) Persist() bool {
	return idx.persist
}

// StoragePath returns the storage path
func (idx *VectorIndex) StoragePath() string {
	return idx.storagePath
}

// DefaultVectorPath is the default storage path for vector index
func DefaultVectorPath(userID string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".memory-brain", "users", userID, "knowledge.db")
	}
	return filepath.Join(homeDir, ".memory-brain", "users", userID, "knowledge.db")
}