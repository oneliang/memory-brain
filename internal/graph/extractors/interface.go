package extractors

import (
	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/pkg/types"
)

// Extractor defines the interface for entity and relation extraction
type Extractor interface {
	// Name returns the name of the extractor
	Name() string

	// Extract extracts entities and relations from an observation
	Extract(obs *types.Observation) (*graph.ExtractionResult, error)

	// IsAsync returns true if this extractor should run asynchronously
	IsAsync() bool
}
