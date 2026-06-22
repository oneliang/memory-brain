package profile

// DevelopmentDetector detects development projects
type DevelopmentDetector struct{}

func (d *DevelopmentDetector) Category() string {
	return CategoryDevelopment
}

func (d *DevelopmentDetector) Detect(ctx *AnalysisContext) float64 {
	score := 0.0

	// File extensions (weight: 0.5)
	devFiles := map[string]float64{
		".go":    0.5,
		".js":    0.4,
		".ts":    0.4,
		".py":    0.4,
		".java":  0.4,
		".rs":    0.5,
		".c":     0.3,
		".cpp":   0.3,
		".h":     0.2,
		".rb":    0.4,
		".php":   0.4,
		".swift": 0.4,
		".kt":    0.4,
	}

	extensionScore := 0.0
	for ext, weight := range devFiles {
		if ctx.HasExtension(ext) {
			extensionScore += weight
		}
	}
	score += 0.5 * min(extensionScore, 1.0)

	// Directory structure (weight: 0.3)
	devDirs := []string{"src", "tests", "test", "__tests__", "internal", "pkg", "cmd", "lib", "api"}
	dirScore := 0.0
	for _, dir := range devDirs {
		if ctx.HasDir(dir) {
			dirScore += 0.2
		}
	}
	score += 0.3 * min(dirScore, 1.0)

	// Config files (weight: 0.2)
	configFiles := []string{"go.mod", "package.json", "requirements.txt", "cargo.toml", "pom.xml", "makefile", "cmakelists.txt"}
	configScore := 0.0
	for _, file := range configFiles {
		if ctx.HasFile(file) {
			configScore += 0.3
		}
	}
	score += 0.2 * min(configScore, 1.0)

	return score
}
