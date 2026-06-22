package profile

// DocumentationDetector detects documentation directories
type DocumentationDetector struct{}

func (d *DocumentationDetector) Category() string {
	return CategoryDocumentation
}

func (d *DocumentationDetector) Detect(ctx *AnalysisContext) float64 {
	score := 0.0

	// File extensions (weight: 0.5)
	docFiles := map[string]float64{
		".pdf":   0.5,
		".epub":  0.5,
		".docx":  0.4,
		".doc":   0.4,
		".md":    0.3,
		".txt":   0.2,
		".tex":   0.5,
		".pptx":  0.4,
		".ppt":   0.4,
		".rtf":   0.3,
	}

	extensionScore := 0.0
	for ext, weight := range docFiles {
		count := ctx.GetExtensionCount(ext)
		if count > 0 {
			extensionScore += weight * min(float64(count)/5.0, 1.0) // Normalize by 5 files
		}
	}
	score += 0.5 * min(extensionScore, 1.0)

	// Directory structure (weight: 0.3)
	docDirs := []string{"docs", "doc", "documentation", "notes", "wiki", "papers", "books", "references"}
	dirScore := 0.0
	for _, dir := range docDirs {
		if ctx.HasDir(dir) {
			dirScore += 0.25
		}
	}
	score += 0.3 * min(dirScore, 1.0)

	// File names (weight: 0.2)
	docFileNames := []string{"readme.md", "readme.txt", "changelog.md", "license.md", "contributing.md"}
	fileScore := 0.0
	for _, file := range docFileNames {
		if ctx.HasFile(file) {
			fileScore += 0.2
		}
	}
	score += 0.2 * min(fileScore, 1.0)

	return score
}
