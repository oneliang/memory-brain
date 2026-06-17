package extractors

import (
	"sync"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/pkg/types"
)

// Registry manages multiple extractors
type Registry struct {
	extractors []Extractor
	mu         sync.RWMutex
}

// NewRegistry creates a new extractor registry
func NewRegistry() *Registry {
	return &Registry{
		extractors: make([]Extractor, 0),
	}
}

// Register adds an extractor to the registry
func (r *Registry) Register(extractor Extractor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.extractors = append(r.extractors, extractor)
}

// GetAll returns all registered extractors
func (r *Registry) GetAll() []Extractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Extractor, len(r.extractors))
	copy(result, r.extractors)
	return result
}

// GetSync returns all synchronous extractors
func (r *Registry) GetSync() []Extractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Extractor
	for _, ext := range r.extractors {
		if !ext.IsAsync() {
			result = append(result, ext)
		}
	}
	return result
}

// GetAsync returns all asynchronous extractors
func (r *Registry) GetAsync() []Extractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Extractor
	for _, ext := range r.extractors {
		if ext.IsAsync() {
			result = append(result, ext)
		}
	}
	return result
}

// ExtractAll runs all extractors on an observation and merges results
func (r *Registry) ExtractAll(obs *types.Observation) (*graph.ExtractionResult, error) {
	r.mu.RLock()
	extractors := make([]Extractor, len(r.extractors))
	copy(extractors, r.extractors)
	r.mu.RUnlock()

	merged := &graph.ExtractionResult{
		Entities:  make([]graph.Entity, 0),
		Relations: make([]graph.Relation, 0),
	}

	for _, ext := range extractors {
		if ext.IsAsync() {
			continue // Skip async extractors in synchronous extraction
		}

		result, err := ext.Extract(obs)
		if err != nil {
			continue // Continue with other extractors on error
		}

		if result != nil {
			merged.Entities = append(merged.Entities, result.Entities...)
			merged.Relations = append(merged.Relations, result.Relations...)
		}
	}

	return merged, nil
}

// Count returns the number of registered extractors
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.extractors)
}
