package storage

import (
	"os"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

func TestStorage_NewStorage(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	if s == nil {
		t.Error("Expected storage instance, got nil")
	}

	if s.baseDir != tmpDir {
		t.Errorf("Expected baseDir=%s, got %s", tmpDir, s.baseDir)
	}
}

func TestStorage_AppendProfile(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	card := &types.ProfileCard{
		ID:   "card_1",
		Type: "PreferenceCard",
		Content: map[string]interface{}{
			"preference": "bash",
		},
		Metadata: types.CardMetadata{
			UserID:    "user_1",
			Timestamp: time.Now(),
			Strength:  1.0,
		},
	}

	err := s.AppendProfile("user_1", card)
	if err != nil {
		t.Errorf("Failed to append profile: %v", err)
	}

	// Verify file exists
	filePath := s.GetUserDir("user_1") + "/" + ProfileFile
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Profile file not created")
	}
}

func TestStorage_ReadProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	// Append multiple profiles
	for i := 1; i <= 3; i++ {
		card := &types.ProfileCard{
			ID:   "card_" + string(rune(i)),
			Type: "PreferenceCard",
			Content: map[string]interface{}{
				"index": i,
			},
		}
		s.AppendProfile("user_1", card)
	}

	// Read profiles
	profiles, err := s.ReadProfiles("user_1")
	if err != nil {
		t.Errorf("Failed to read profiles: %v", err)
	}

	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(profiles))
	}
}

func TestStorage_ReadProfiles_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	// User has no profiles
	profiles, err := s.ReadProfiles("user_nonexistent")
	if err != nil {
		t.Errorf("Expected no error for nonexistent user, got: %v", err)
	}

	if len(profiles) != 0 {
		t.Errorf("Expected 0 profiles for nonexistent user, got %d", len(profiles))
	}
}

func TestStorage_AppendPattern(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	pattern := &types.PatternCard{
		ID:        "pattern_1",
		UserID:    "user_1",
		Type:      "tool_sequence",
		Pattern:   "bash -> read",
		Frequency: 5,
		Confidence: 0.8,
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := s.AppendPattern("user_1", pattern)
	if err != nil {
		t.Errorf("Failed to append pattern: %v", err)
	}

	// Verify file exists
	filePath := s.GetUserDir("user_1") + "/" + PatternsFile
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Patterns file not created")
	}
}

func TestStorage_ReadPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	// Append patterns
	for i := 1; i <= 2; i++ {
		pattern := &types.PatternCard{
			ID:        "pattern_" + string(rune(i)),
			UserID:    "user_1",
			Type:      "tool_sequence",
			Pattern:   "test",
			Frequency: i,
		}
		s.AppendPattern("user_1", pattern)
	}

	patterns, err := s.ReadPatterns("user_1")
	if err != nil {
		t.Errorf("Failed to read patterns: %v", err)
	}

	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}
}

func TestStorage_SaveSessionSummary(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	summary := &types.SessionSummary{
		ID:        "sum_1",
		SessionID: "session_1",
		UserID:    "user_1",
		Title:     "Test Session",
		Narrative: "This is a test session summary",
		CreatedAt: time.Now(),
	}

	err := s.SaveSessionSummary("user_1", "session_1", summary)
	if err != nil {
		t.Errorf("Failed to save session summary: %v", err)
	}

	// Verify file exists
	filePath := s.GetUserDir("user_1") + "/" + SessionsArchiveDir + "/session_1_summary.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Session summary file not created")
	}
}

func TestStorage_ReadSessionSummary(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	// Save summary
	summary := &types.SessionSummary{
		ID:        "sum_1",
		SessionID: "session_1",
		UserID:    "user_1",
		Title:     "Test Session",
		Facts:     []string{"fact1", "fact2"},
		Narrative: "Test narrative",
		CreatedAt: time.Now(),
	}
	s.SaveSessionSummary("user_1", "session_1", summary)

	// Read summary
	readSummary, err := s.ReadSessionSummary("user_1", "session_1")
	if err != nil {
		t.Errorf("Failed to read session summary: %v", err)
	}

	if readSummary.Title != "Test Session" {
		t.Errorf("Expected title 'Test Session', got '%s'", readSummary.Title)
	}

	if len(readSummary.Facts) != 2 {
		t.Errorf("Expected 2 facts, got %d", len(readSummary.Facts))
	}
}

func TestStorage_GetUserDir(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	userDir := s.GetUserDir("user_123")

	expected := tmpDir + "/users/user_123"
	if userDir != expected {
		t.Errorf("Expected userDir=%s, got %s", expected, userDir)
	}
}

func TestStorage_EnsureUserDir(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	err := s.EnsureUserDir("user_new")
	if err != nil {
		t.Errorf("Failed to ensure user dir: %v", err)
	}

	// Verify directory exists
	userDir := s.GetUserDir("user_new")
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		t.Error("User directory not created")
	}
}

func TestStorage_ConcurrentWrite(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStorage(tmpDir)

	// Write profiles concurrently
	done := make(chan bool, 10) // Buffered channel for 10 goroutines
	for i := 0; i < 10; i++ {
		go func(idx int) {
			card := &types.ProfileCard{
				ID:   "card_" + string(rune(idx)),
				Type: "PreferenceCard",
			}
			s.AppendProfile("user_concurrent", card)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all profiles saved
	profiles, _ := s.ReadProfiles("user_concurrent")
	if len(profiles) != 10 {
		t.Errorf("Expected 10 profiles after concurrent write, got %d", len(profiles))
	}
}