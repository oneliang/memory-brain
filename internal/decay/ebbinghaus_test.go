package decay

import (
	"math"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// TestApplyDecay_Basic tests basic decay calculation
func TestApplyDecay_Basic(t *testing.T) {
	now := time.Now()
	cards := []types.ProfileCard{
		{
			ID:   "card_1",
			Type: "PreferenceCard",
			Metadata: types.CardMetadata{
				UserID:    "user_1",
				Timestamp: now.AddDate(0, 0, -14), // 14 days ago
				Strength:  1.0,
			},
		},
	}

	remaining := ApplyDecay(cards, DefaultDecayDays)

	if len(remaining) != 1 {
		t.Fatalf("expected 1 card remaining, got %d", len(remaining))
	}

	// 14 days = 2 decay periods with default 7 days
	// strength = 1.0 * 0.9^2 = 0.81
	expected := 1.0 * math.Pow(DecayFactor, 2)
	actual := remaining[0].Metadata.Strength

	if math.Abs(actual-expected) > 0.01 {
		t.Errorf("expected strength ~%.2f, got %.2f", expected, actual)
	}
}

// TestApplyDecay_DeleteThreshold tests deletion below threshold
func TestApplyDecay_DeleteThreshold(t *testing.T) {
	now := time.Now()
	cards := []types.ProfileCard{
		{
			ID:   "card_below",
			Type: "PreferenceCard",
			Metadata: types.CardMetadata{
				Timestamp: now.AddDate(0, 0, -100), // 100 days ago
				Strength:  0.5,                      // Will decay below threshold
			},
		},
		{
			ID:   "card_above",
			Type: "PreferenceCard",
			Metadata: types.CardMetadata{
				Timestamp: now,
				Strength:  0.5, // Fresh, no decay
			},
		},
	}

	remaining := ApplyDecay(cards, DefaultDecayDays)

	// card_below should be deleted, card_above should remain
	for _, card := range remaining {
		if card.ID == "card_below" {
			t.Error("card_below should have been deleted")
		}
	}

	if len(remaining) != 1 {
		t.Errorf("expected 1 card remaining, got %d", len(remaining))
	}
}

// TestApplyDecay_MinStrength tests minimum strength floor
func TestApplyDecay_MinStrength(t *testing.T) {
	now := time.Now()
	cards := []types.ProfileCard{
		{
			ID:   "card_min",
			Type: "PreferenceCard",
			Metadata: types.CardMetadata{
				Timestamp: now.AddDate(0, 0, -30), // 30 days ago (will decay but stay above DeleteThreshold)
				Strength:  0.5,
			},
		},
	}

	remaining := ApplyDecay(cards, DefaultDecayDays)

	if len(remaining) != 1 {
		t.Fatal("card should remain with 0.5 initial strength")
	}

	// After ~4 decay periods: 0.5 * 0.9^4 ≈ 0.33, which is above DeleteThreshold
	// Strength should be at least MinStrength (0.1)
	if remaining[0].Metadata.Strength < MinStrength {
		t.Errorf("strength should not be below MinStrength (%.2f), got %.4f", MinStrength, remaining[0].Metadata.Strength)
	}
}

// TestApplyDecay_FreshCards tests cards that shouldn't decay yet
func TestApplyDecay_FreshCards(t *testing.T) {
	now := time.Now()
	cards := []types.ProfileCard{
		{
			ID:   "card_fresh",
			Type: "PreferenceCard",
			Metadata: types.CardMetadata{
				Timestamp: now.AddDate(0, 0, -3), // 3 days ago, less than decay window
				Strength:  1.0,
			},
		},
	}

	remaining := ApplyDecay(cards, DefaultDecayDays)

	if len(remaining) != 1 {
		t.Fatal("fresh card should remain")
	}

	// Strength should not change for fresh cards
	if remaining[0].Metadata.Strength != 1.0 {
		t.Errorf("fresh card strength should remain 1.0, got %.2f", remaining[0].Metadata.Strength)
	}
}

// TestApplyDecay_EmptyInput tests empty input
func TestApplyDecay_EmptyInput(t *testing.T) {
	remaining := ApplyDecay([]types.ProfileCard{}, DefaultDecayDays)

	if len(remaining) != 0 {
		t.Errorf("expected empty result, got %d cards", len(remaining))
	}
}

// TestApplyDecayToPatterns_Basic tests pattern decay
func TestApplyDecayToPatterns_Basic(t *testing.T) {
	now := time.Now()
	patterns := []types.PatternCard{
		{
			ID:         "pattern_1",
			UserID:     "user_1",
			Type:       "tool_usage",
			Pattern:    "bash",
			Frequency:  10,
			Confidence: 1.0,
			LastSeen:   now.AddDate(0, 0, -14), // 14 days ago
		},
	}

	remaining := ApplyDecayToPatterns(patterns, DefaultDecayDays)

	if len(remaining) != 1 {
		t.Fatalf("expected 1 pattern remaining, got %d", len(remaining))
	}

	// Confidence should decay: 1.0 * 0.9^2 = 0.81
	expected := 1.0 * math.Pow(DecayFactor, 2)
	actual := remaining[0].Confidence

	if math.Abs(actual-expected) > 0.01 {
		t.Errorf("expected confidence ~%.2f, got %.2f", expected, actual)
	}
}

// TestApplyDecayToPatterns_DeleteThreshold tests pattern deletion
func TestApplyDecayToPatterns_DeleteThreshold(t *testing.T) {
	now := time.Now()
	patterns := []types.PatternCard{
		{
			ID:         "pattern_delete",
			Confidence: 0.25, // Below DeleteThreshold
			LastSeen:   now.AddDate(0, 0, -100),
		},
		{
			ID:         "pattern_keep",
			Confidence: 0.5,
			LastSeen:   now, // Fresh
		},
	}

	remaining := ApplyDecayToPatterns(patterns, DefaultDecayDays)

	for _, p := range remaining {
		if p.ID == "pattern_delete" {
			t.Error("pattern_delete should have been removed")
		}
	}
}

// TestCalculateStrength tests strength calculation
func TestCalculateStrength(t *testing.T) {
	tests := []struct {
		importance float64
		frequency  int
		expected   float64
	}{
		{0.5, 0, 0.5},      // No frequency boost
		{0.5, 5, 0.6},      // 5 * 0.02 = 0.1 boost
		{0.5, 10, 0.7},     // 10 * 0.02 = 0.2 boost (max)
		{0.8, 20, 1.0},     // Capped at 1.0
		{1.0, 0, 1.0},      // Already max
	}

	for _, tt := range tests {
		actual := CalculateStrength(tt.importance, tt.frequency)
		if math.Abs(actual-tt.expected) > 0.01 {
			t.Errorf("CalculateStrength(%f, %d) = %.2f, expected %.2f",
				tt.importance, tt.frequency, actual, tt.expected)
		}
	}
}

// TestShouldDecay tests decay timing check
func TestShouldDecay(t *testing.T) {
	now := time.Now()

	freshCard := &types.ProfileCard{
		Metadata: types.CardMetadata{
			Timestamp: now.AddDate(0, 0, -3),
		},
	}

	oldCard := &types.ProfileCard{
		Metadata: types.CardMetadata{
			Timestamp: now.AddDate(0, 0, -10),
		},
	}

	if ShouldDecay(freshCard, DefaultDecayDays) {
		t.Error("fresh card should not need decay")
	}

	if !ShouldDecay(oldCard, DefaultDecayDays) {
		t.Error("old card should need decay")
	}
}

// TestGetDecayPeriods tests decay period calculation
func TestGetDecayPeriods(t *testing.T) {
	tests := []struct {
		daysAgo int
		decayDays int
		expected int
	}{
		{3, 7, 0},   // Less than one period
		{7, 7, 1},   // Exactly one period
		{14, 7, 2},  // Two periods
		{21, 7, 3},  // Three periods
		{10, 5, 2},  // Two periods with 5-day window
	}

	for _, tt := range tests {
		timestamp := time.Now().AddDate(0, 0, -tt.daysAgo)
		actual := GetDecayPeriods(timestamp, tt.decayDays)
		if actual != tt.expected {
			t.Errorf("GetDecayPeriods(%d days ago, %d) = %d, expected %d",
				tt.daysAgo, tt.decayDays, actual, tt.expected)
		}
	}
}

// TestConstants tests that constants are sensible values
func TestConstants(t *testing.T) {
	if DefaultDecayDays <= 0 {
		t.Error("DefaultDecayDays should be positive")
	}

	if MinStrength <= 0 || MinStrength >= 1 {
		t.Error("MinStrength should be between 0 and 1")
	}

	if DeleteThreshold <= MinStrength || DeleteThreshold >= 1 {
		t.Error("DeleteThreshold should be between MinStrength and 1")
	}

	if DecayFactor <= 0 || DecayFactor >= 1 {
		t.Error("DecayFactor should be between 0 and 1 (decay per period)")
	}
}