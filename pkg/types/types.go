package types

import "time"

// Observation represents a captured event from Hook
type Observation struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id"`
	HookType  string                 `json:"hook_type"`
	Timestamp time.Time               `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Hash      string                 `json:"hash"` // SHA-256 dedup
	Directory string                 `json:"directory,omitempty"`   // Working directory for project-level profile
	ProjectID string                 `json:"project_id,omitempty"`  // Project identifier (optional)
}

// ProfileCard represents a user profile memory card
type ProfileCard struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"` // PreferenceCard, PatternCard, etc.
	Content      map[string]interface{} `json:"content"`
	Metadata     CardMetadata           `json:"metadata"`
}

// CardMetadata contains metadata for a profile card
type CardMetadata struct {
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	Timestamp    time.Time `json:"timestamp"`
	SourceEvents []string  `json:"source_events"`
	Importance   float64   `json:"importance"`
	Tags         []string  `json:"tags"`
	LinkedCards  []string  `json:"linked_cards"`
	Strength     float64   `json:"strength"` // Decay strength
	DecayDays    int       `json:"decay_days"`
	Scope        string    `json:"scope"`      // "user" | "session" | "project"
	Directory    string    `json:"directory,omitempty"` // For project scope
}

// SearchResult represents a search result
type SearchResult struct {
	ObsID     string  `json:"obs_id"`
	SessionID string  `json:"session_id"`
	Title     string  `json:"title"`
	Score     float64 `json:"score"`
	Source    string  `json:"source"` // bm25, vector, rrf
	Summary   string  `json:"summary"`
}

// SessionSummary represents an episodic memory summary
type SessionSummary struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Facts       []string  `json:"facts"`
	Concepts    []string  `json:"concepts"`
	Narrative   string    `json:"narrative"`
	Files       []string  `json:"files"`
	Importance  float64   `json:"importance"`
	CreatedAt   time.Time `json:"created_at"`
}

// PatternCard represents a behavioral pattern
type PatternCard struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"` // tool_sequence, time_pattern, etc.
	Pattern     string    `json:"pattern"`
	Frequency   int       `json:"frequency"`
	Confidence  float64   `json:"confidence"`
	LastSeen    time.Time `json:"last_seen"`
	CreatedAt   time.Time `json:"created_at"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ProfileResponse represents the response for GET /memory/profile
type ProfileResponse struct {
	SystemMessage     string                 `json:"systemMessage,omitempty"`
	AdditionalContext string                 `json:"additionalContext,omitempty"`
	ProfileSummary    map[string]interface{} `json:"profile_summary,omitempty"`
}

// SearchResponse represents the response for GET /memory/search
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Lessons []string       `json:"lessons,omitempty"`
}

// ProjectProfile represents a directory-level profile
type ProjectProfile struct {
	ID          string    `json:"id"`
	Directory   string    `json:"directory"`      // Original path
	ProjectHash string    `json:"project_hash"`   // SHA256 of directory path
	Category    string    `json:"category"`       // Primary category
	Summary     string    `json:"summary"`        // Directory summary
	Stats       DirStats  `json:"stats"`          // Directory statistics
	Highlights  []string  `json:"highlights"`     // Key files/patterns
	LastAccessed time.Time `json:"last_accessed"`
}

// DirStats represents directory statistics
type DirStats struct {
	TotalFiles     int            `json:"total_files"`
	TotalDirs      int            `json:"total_dirs"`
	ExtensionCount map[string]int `json:"extension_count"` // Extension -> count
	CategoryScores []CategoryScore `json:"category_scores"` // All category scores
}

// CategoryScore represents a category detection score
type CategoryScore struct {
	Category string  `json:"category"`
	Score    float64 `json:"score"`
}

// SessionProfile represents session-level profile data
type SessionProfile struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id"`
	ProfileSummary map[string]interface{} `json:"profile_summary,omitempty"`
	SystemMessage  string               `json:"system_message,omitempty"`
}