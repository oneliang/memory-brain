package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/oneliang/memory-brain/pkg/hash"
	"github.com/oneliang/memory-brain/pkg/types"
)

const (
	// DefaultStorageDir is the default storage directory
	DefaultStorageDir = ".memory-brain"

	// ProfileFile is the profile storage file
	ProfileFile = "profile.jsonl"

	// PatternsFile is the patterns storage file
	PatternsFile = "patterns.jsonl"

	// KnowledgeFile is the vector knowledge file
	KnowledgeFile = "knowledge.db"

	// SessionsArchiveDir is the sessions archive directory
	SessionsArchiveDir = "sessions_archive"

	// SessionsDir is the session profile directory
	SessionsDir = "sessions"

	// ProjectsDir is the project profile directory
	ProjectsDir = "projects"

	// ProjectMetaFile is the project metadata file
	ProjectMetaFile = "meta.json"
)

// Storage handles file-based storage for Memory Brain
type Storage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewStorage creates a new storage instance
func NewStorage(baseDir string) *Storage {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			baseDir = DefaultStorageDir
		} else {
			baseDir = filepath.Join(homeDir, DefaultStorageDir)
		}
	}
	return &Storage{baseDir: baseDir}
}

// GetUserDir returns the user-specific directory
func (s *Storage) GetUserDir(userID string) string {
	return filepath.Join(s.baseDir, "users", userID)
}

// EnsureUserDir creates user directory if not exists
func (s *Storage) EnsureUserDir(userID string) error {
	userDir := s.GetUserDir(userID)
	return os.MkdirAll(userDir, 0755)
}

// AppendProfile appends a profile card to profile.jsonl
func (s *Storage) AppendProfile(userID string, card *types.ProfileCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureUserDir(userID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetUserDir(userID), ProfileFile)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(card)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}

// ReadProfiles reads all profile cards for a user
func (s *Storage) ReadProfiles(userID string) ([]types.ProfileCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.GetUserDir(userID), ProfileFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.ProfileCard{}, nil
		}
		return nil, err
	}

	var cards []types.ProfileCard
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var card types.ProfileCard
		if err := json.Unmarshal([]byte(line), &card); err != nil {
			continue // Skip malformed lines
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// AppendPattern appends a pattern card to patterns.jsonl
func (s *Storage) AppendPattern(userID string, pattern *types.PatternCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureUserDir(userID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetUserDir(userID), PatternsFile)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(pattern)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}

// ReadPatterns reads all pattern cards for a user
func (s *Storage) ReadPatterns(userID string) ([]types.PatternCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.GetUserDir(userID), PatternsFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.PatternCard{}, nil
		}
		return nil, err
	}

	var patterns []types.PatternCard
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var pattern types.PatternCard
		if err := json.Unmarshal([]byte(line), &pattern); err != nil {
			continue
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

// WriteProfiles overwrites profile.jsonl with provided cards (for decay persistence)
func (s *Storage) WriteProfiles(userID string, cards []types.ProfileCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureUserDir(userID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetUserDir(userID), ProfileFile)
	file, err := os.Create(filePath) // Create/truncate file
	if err != nil {
		return err
	}
	defer file.Close()

	for _, card := range cards {
		data, err := json.Marshal(card)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// WritePatterns overwrites patterns.jsonl with provided patterns (for decay persistence)
func (s *Storage) WritePatterns(userID string, patterns []types.PatternCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureUserDir(userID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetUserDir(userID), PatternsFile)
	file, err := os.Create(filePath) // Create/truncate file
	if err != nil {
		return err
	}
	defer file.Close()

	for _, pattern := range patterns {
		data, err := json.Marshal(pattern)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// SaveSessionSummary saves a session summary to archive
func (s *Storage) SaveSessionSummary(userID, sessionID string, summary *types.SessionSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	archiveDir := filepath.Join(s.GetUserDir(userID), SessionsArchiveDir)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(archiveDir, fmt.Sprintf("%s_summary.json", sessionID))
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// ReadSessionSummary reads a session summary from archive
func (s *Storage) ReadSessionSummary(userID, sessionID string) (*types.SessionSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.GetUserDir(userID), SessionsArchiveDir, fmt.Sprintf("%s_summary.json", sessionID))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var summary types.SessionSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// splitLines splits string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// ========== Session-level storage methods ==========

// GetSessionDir returns the session-specific directory
func (s *Storage) GetSessionDir(userID, sessionID string) string {
	return filepath.Join(s.GetUserDir(userID), SessionsDir, sessionID)
}

// EnsureSessionDir creates session directory if not exists
func (s *Storage) EnsureSessionDir(userID, sessionID string) error {
	sessionDir := s.GetSessionDir(userID, sessionID)
	return os.MkdirAll(sessionDir, 0755)
}

// AppendSessionProfile appends a profile card to session's profile.jsonl
func (s *Storage) AppendSessionProfile(userID, sessionID string, card *types.ProfileCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureSessionDir(userID, sessionID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetSessionDir(userID, sessionID), ProfileFile)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(card)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}

// ReadSessionProfiles reads all profile cards for a session
func (s *Storage) ReadSessionProfiles(userID, sessionID string) ([]types.ProfileCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.GetSessionDir(userID, sessionID), ProfileFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.ProfileCard{}, nil
		}
		return nil, err
	}

	var cards []types.ProfileCard
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var card types.ProfileCard
		if err := json.Unmarshal([]byte(line), &card); err != nil {
			continue
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// WriteSessionProfiles overwrites session's profile.jsonl with provided cards
func (s *Storage) WriteSessionProfiles(userID, sessionID string, cards []types.ProfileCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureSessionDir(userID, sessionID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetSessionDir(userID, sessionID), ProfileFile)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, card := range cards {
		data, err := json.Marshal(card)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// AppendSessionPattern appends a pattern card to session's patterns.jsonl
func (s *Storage) AppendSessionPattern(userID, sessionID string, pattern *types.PatternCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureSessionDir(userID, sessionID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetSessionDir(userID, sessionID), PatternsFile)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(pattern)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}

// ReadSessionPatterns reads all pattern cards for a session
func (s *Storage) ReadSessionPatterns(userID, sessionID string) ([]types.PatternCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.GetSessionDir(userID, sessionID), PatternsFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.PatternCard{}, nil
		}
		return nil, err
	}

	var patterns []types.PatternCard
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var pattern types.PatternCard
		if err := json.Unmarshal([]byte(line), &pattern); err != nil {
			continue
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

// WriteSessionPatterns overwrites session's patterns.jsonl with provided patterns
func (s *Storage) WriteSessionPatterns(userID, sessionID string, patterns []types.PatternCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureSessionDir(userID, sessionID); err != nil {
		return err
	}

	filePath := filepath.Join(s.GetSessionDir(userID, sessionID), PatternsFile)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, pattern := range patterns {
		data, err := json.Marshal(pattern)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// ========== Project-level storage methods ==========

// GetProjectDir returns the project-specific directory (using SHA256 hash of path)
func (s *Storage) GetProjectDir(directory string) string {
	hash := computeProjectHash(directory)
	return filepath.Join(s.baseDir, ProjectsDir, hash)
}

// computeProjectHash computes SHA256 hash of directory path
func computeProjectHash(directory string) string {
	return hash.ProjectHash(directory)
}

// EnsureProjectDir creates project directory if not exists
func (s *Storage) EnsureProjectDir(directory string) error {
	projectDir := s.GetProjectDir(directory)
	return os.MkdirAll(projectDir, 0755)
}

// AppendProjectProfile appends a profile card to project's profile.jsonl
func (s *Storage) AppendProjectProfile(directory string, card *types.ProfileCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureProjectDir(directory); err != nil {
		return err
	}

	projectDir := s.GetProjectDir(directory)
	filePath := filepath.Join(projectDir, ProfileFile)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(card)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}

// ReadProjectProfiles reads all profile cards for a project
func (s *Storage) ReadProjectProfiles(directory string) ([]types.ProfileCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectDir := s.GetProjectDir(directory)
	filePath := filepath.Join(projectDir, ProfileFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.ProfileCard{}, nil
		}
		return nil, err
	}

	var cards []types.ProfileCard
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var card types.ProfileCard
		if err := json.Unmarshal([]byte(line), &card); err != nil {
			continue
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// WriteProjectProfiles overwrites project's profile.jsonl with provided cards
func (s *Storage) WriteProjectProfiles(directory string, cards []types.ProfileCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureProjectDir(directory); err != nil {
		return err
	}

	projectDir := s.GetProjectDir(directory)
	filePath := filepath.Join(projectDir, ProfileFile)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, card := range cards {
		data, err := json.Marshal(card)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// SaveProjectMeta saves project metadata
func (s *Storage) SaveProjectMeta(directory string, meta *types.ProjectProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureProjectDir(directory); err != nil {
		return err
	}

	projectDir := s.GetProjectDir(directory)
	filePath := filepath.Join(projectDir, ProjectMetaFile)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// ReadProjectMeta reads project metadata
func (s *Storage) ReadProjectMeta(directory string) (*types.ProjectProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectDir := s.GetProjectDir(directory)
	filePath := filepath.Join(projectDir, ProjectMetaFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var meta types.ProjectProfile
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}