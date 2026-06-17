package graph

import "time"

// Node represents an entity node in the knowledge graph
type Node struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`       // Open: any string (e.g., "person", "org", "file", "tool")
	Name      string    `json:"name"`       // Entity name
	Properties string   `json:"properties"` // JSON string for flexible properties
	CreatedAt time.Time `json:"created_at"`
}

// Edge represents a relationship between two nodes
type Edge struct {
	ID        int64     `json:"id"`
	SourceID  int64     `json:"source_id"`
	TargetID  int64     `json:"target_id"`
	Relation  string    `json:"relation"` // e.g., "uses", "modifies", "works_at"
	Weight    float64   `json:"weight"`
	Timestamp time.Time `json:"timestamp"`
}

// Entity represents an extracted entity (for Extractor output)
type Entity struct {
	Type       string                 `json:"type"` // "person", "org", "loc", "tech", "file", "tool", etc.
	Name       string                 `json:"name"` // Entity name
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Relation represents an extracted relationship (for Extractor output)
type Relation struct {
	Source string  `json:"source"` // Source entity name
	Target string  `json:"target"` // Target entity name
	Type   string  `json:"type"`   // "uses", "modifies", "works_at", "works_on", "mentions", etc.
	Weight float64 `json:"weight,omitempty"`
}

// ExtractionResult represents the output of an Extractor
type ExtractionResult struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// QueryResult represents a result from graph query
type QueryResult struct {
	Node   Node    `json:"node"`
	Score  float64 `json:"score"`
	Hops   int     `json:"hops"`
	Path   []int64 `json:"path,omitempty"` // Node IDs in the path
}

// GraphConfig represents configuration for the graph module
type GraphConfig struct {
	Enabled bool         `json:"enabled" yaml:"enabled"`
	Ollama  OllamaConfig `json:"ollama" yaml:"ollama"`
	Query   QueryConfig  `json:"query" yaml:"query"`
}

// OllamaConfig represents Ollama configuration for NER+RE
type OllamaConfig struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Model    string `json:"model" yaml:"model"`
	Timeout  int    `json:"timeout" yaml:"timeout"` // seconds
}

// QueryConfig represents query configuration
type QueryConfig struct {
	MaxHops      int `json:"max_hops" yaml:"max_hops"`
	DefaultLimit int `json:"default_limit" yaml:"default_limit"`
}
