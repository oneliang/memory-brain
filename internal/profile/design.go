package profile

// DesignDetector detects design resource directories
type DesignDetector struct{}

func (d *DesignDetector) Category() string {
	return CategoryDesign
}

func (d *DesignDetector) Detect(ctx *AnalysisContext) float64 {
	score := 0.0

	// File extensions (weight: 0.5)
	designFiles := map[string]float64{
		".psd":     0.6,
		".sketch":  0.6,
		".fig":     0.6,
		".ai":      0.6,
		".xd":      0.6,
		".svg":     0.4,
		".eps":     0.5,
		".png":     0.2,
		".jpg":     0.2,
		".jpeg":    0.2,
		".gif":     0.2,
		".webp":    0.2,
		".ico":     0.3,
	}

	extensionScore := 0.0
	for ext, weight := range designFiles {
		count := ctx.GetExtensionCount(ext)
		if count > 0 {
			extensionScore += weight * min(float64(count)/5.0, 1.0)
		}
	}
	score += 0.5 * min(extensionScore, 1.0)

	// Directory structure (weight: 0.3)
	designDirs := []string{"assets", "designs", "design", "mockups", "wireframes", "images", "icons", "graphics", "ui", "ux"}
	dirScore := 0.0
	for _, dir := range designDirs {
		if ctx.HasDir(dir) {
			dirScore += 0.15
		}
	}
	score += 0.3 * min(dirScore, 1.0)

	// File names (weight: 0.2)
	designFileNames := []string{"logo.svg", "icon.png", "banner.jpg", "mockup.psd", "wireframe.fig"}
	fileScore := 0.0
	for _, file := range designFileNames {
		if ctx.HasFile(file) {
			fileScore += 0.25
		}
	}
	score += 0.2 * min(fileScore, 1.0)

	return score
}
