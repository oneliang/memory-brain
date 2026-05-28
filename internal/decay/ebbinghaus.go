package decay

import (
	"math"
	"time"

	"github.com/oneliang/memory-brain/pkg/types"
)

// DefaultDecayDays is the default decay period (7 days)
const DefaultDecayDays = 7

// MinStrength is the minimum strength threshold (0.1)
const MinStrength = 0.1

// DeleteThreshold is the threshold for deleting memories (0.3)
const DeleteThreshold = 0.3

// DecayFactor is the decay factor per period (0.9, borrowed from agentmemory)
const DecayFactor = 0.9

// ApplyDecay applies Ebbinghaus decay to profile cards
func ApplyDecay(cards []types.ProfileCard, decayDays int) []types.ProfileCard {
	now := time.Now()
	var remaining []types.ProfileCard

	for _, card := range cards {
		daysSince := now.Sub(card.Metadata.Timestamp).Hours() / 24

		if daysSince > float64(decayDays) {
			decayPeriods := int(daysSince / float64(decayDays))
			// Apply decay: strength *= 0.9^decay_periods
			card.Metadata.Strength = math.Max(MinStrength,
				card.Metadata.Strength*math.Pow(DecayFactor, float64(decayPeriods)))

			// Delete if below threshold
			if card.Metadata.Strength < DeleteThreshold {
				continue // Skip this card (effectively delete)
			}
		}

		remaining = append(remaining, card)
	}

	return remaining
}

// ApplyDecayToPatterns applies decay to pattern cards
func ApplyDecayToPatterns(patterns []types.PatternCard, decayDays int) []types.PatternCard {
	now := time.Now()
	var remaining []types.PatternCard

	for _, pattern := range patterns {
		daysSince := now.Sub(pattern.LastSeen).Hours() / 24

		if daysSince > float64(decayDays) {
			decayPeriods := int(daysSince / float64(decayDays))
			// Decay confidence instead of strength for patterns
			pattern.Confidence = math.Max(MinStrength,
				pattern.Confidence*math.Pow(DecayFactor, float64(decayPeriods)))

			if pattern.Confidence < DeleteThreshold {
				continue
			}
		}

		remaining = append(remaining, pattern)
	}

	return remaining
}

// CalculateStrength calculates initial strength based on importance and frequency
func CalculateStrength(importance float64, frequency int) float64 {
	// Base strength from importance
	base := importance

	// Boost from frequency (up to 0.2 extra)
	freqBoost := math.Min(0.2, float64(frequency)*0.02)

	return math.Min(1.0, base+freqBoost)
}

// ShouldDecay checks if a card should be decayed
func ShouldDecay(card *types.ProfileCard, decayDays int) bool {
	daysSince := time.Since(card.Metadata.Timestamp).Hours() / 24
	return daysSince > float64(decayDays)
}

// GetDecayPeriods returns the number of decay periods
func GetDecayPeriods(timestamp time.Time, decayDays int) int {
	daysSince := time.Since(timestamp).Hours() / 24
	return int(daysSince / float64(decayDays))
}