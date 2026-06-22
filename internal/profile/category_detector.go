package profile

import (
	"sort"

	"github.com/oneliang/memory-brain/pkg/types"
)

// Category constants
const (
	CategoryDevelopment   = "development"
	CategoryDocumentation = "documentation"
	CategoryOperations    = "operations"
	CategoryDesign        = "design"
	CategoryMedia         = "media"
	CategoryMixed         = "mixed"
	CategoryUnknown       = "unknown"
)

// CategoryDetector interface for category detection
type CategoryDetector interface {
	Category() string
	Detect(ctx *AnalysisContext) float64
}

// DetectCategory runs all detectors and returns the primary category
func DetectCategory(ctx *AnalysisContext) (string, []types.CategoryScore) {
	detectors := []CategoryDetector{
		&DevelopmentDetector{},
		&DocumentationDetector{},
		&OperationsDetector{},
		&DesignDetector{},
		&MediaDetector{},
	}

	var scores []types.CategoryScore
	for _, detector := range detectors {
		score := detector.Detect(ctx)
		if score > 0 {
			scores = append(scores, types.CategoryScore{
				Category: detector.Category(),
				Score:    score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Determine primary category
	if len(scores) == 0 {
		return CategoryUnknown, scores
	}

	// Check if mixed (second score > 70% of first)
	if len(scores) >= 2 && scores[1].Score > scores[0].Score*0.7 {
		return CategoryMixed, scores
	}

	return scores[0].Category, scores
}
