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