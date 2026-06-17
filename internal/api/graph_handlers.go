package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/internal/graph/extractors"
	"github.com/oneliang/memory-brain/pkg/types"
)

// GraphHandler handles graph-related API requests
type GraphHandler struct {
	stores    map[string]*graph.Store // userID -> store
	registry  *extractors.Registry
	config    *graph.GraphConfig
}

// NewGraphHandler creates a new graph handler
func NewGraphHandler(config *graph.GraphConfig) *GraphHandler {
	return &GraphHandler{
		stores:   make(map[string]*graph.Store),
		registry: extractors.NewRegistry(),
		config:   config,
	}
}

// GetStore returns or creates a store for a user
func (h *GraphHandler) GetStore(userID string) (*graph.Store, error) {
	if store, ok := h.stores[userID]; ok {
		return store, nil
	}

	store, err := graph.NewStore(userID)
	if err != nil {
		return nil, err
	}

	h.stores[userID] = store
	return store, nil
}

// RegisterExtractors registers default extractors
func (h *GraphHandler) RegisterExtractors() {
	// Register rule extractor (synchronous)
	h.registry.Register(extractors.NewRuleExtractor())

	// Register Ollama extractor (asynchronous) if enabled
	if h.config != nil && h.config.Enabled && h.config.Ollama.Endpoint != "" {
		ollamaExtractor := extractors.NewOllamaExtractor(
			h.config.Ollama.Endpoint,
			h.config.Ollama.Model,
			0, // Use default timeout
		)
		h.registry.Register(ollamaExtractor)
	}
}

// HandleExtract handles POST /memory/graph/extract
func (h *GraphHandler) HandleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var obs types.Observation
	if err := json.NewDecoder(r.Body).Decode(&obs); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if obs.UserID == "" {
		writeError(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	// Get or create store
	store, err := h.GetStore(obs.UserID)
	if err != nil {
		writeError(w, "Failed to get store: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract using all synchronous extractors
	result, err := h.registry.ExtractAll(&obs)
	if err != nil {
		writeError(w, "Extraction failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save to graph
	if err := store.SaveExtractionResult(obs.UserID, result); err != nil {
		writeError(w, "Failed to save: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Run async extractors in background
	go func() {
		for _, ext := range h.registry.GetAsync() {
			asyncResult, err := ext.Extract(&obs)
			if err != nil {
				log.Printf("Async extractor %s failed: %v", ext.Name(), err)
				continue
			}
			if asyncResult != nil {
				if err := store.SaveExtractionResult(obs.UserID, asyncResult); err != nil {
					log.Printf("Failed to save async result: %v", err)
				}
			}
		}
	}()

	writeSuccess(w, map[string]interface{}{
		"success":   true,
		"user_id":   obs.UserID,
		"entities":  len(result.Entities),
		"relations": len(result.Relations),
	})
}

// HandleQuery handles GET /memory/graph/query
func (h *GraphHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	// Parse options
	opts := graph.DefaultQueryOptions()

	if hops := r.URL.Query().Get("hops"); hops != "" {
		if h, err := strconv.Atoi(hops); err == nil && h > 0 {
			opts.MaxHops = h
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			opts.Limit = l
		}
	}

	// Get store
	store, err := h.GetStore(userID)
	if err != nil {
		writeError(w, "Failed to get store: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute query
	engine := graph.NewQueryEngine(store)
	results, err := engine.MultiHopQuery(userID, opts)
	if err != nil {
		writeError(w, "Query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"user_id": userID,
		"hops":    opts.MaxHops,
		"results": results,
		"count":   len(results),
	})
}

// HandleContext handles GET /memory/graph/context
func (h *GraphHandler) HandleContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	// Parse limit
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Get store
	store, err := h.GetStore(userID)
	if err != nil {
		writeError(w, "Failed to get store: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get context
	engine := graph.NewQueryEngine(store)
	context, err := engine.GetContext(userID, limit)
	if err != nil {
		writeError(w, "Failed to get context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeSuccess(w, context)
}

// HandleStats handles GET /memory/graph/stats
func (h *GraphHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// Return stats for all users
		stats := make(map[string]interface{})
		for uid, store := range h.stores {
			storeStats, err := store.GetStats()
			if err != nil {
				continue
			}
			stats[uid] = storeStats
		}
		writeSuccess(w, stats)
		return
	}

	// Get store for specific user
	store, err := h.GetStore(userID)
	if err != nil {
		writeError(w, "Failed to get store: "+err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := store.GetStats()
	if err != nil {
		writeError(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeSuccess(w, stats)
}

// Close closes all stores
func (h *GraphHandler) Close() {
	for _, store := range h.stores {
		store.Close()
	}
}
