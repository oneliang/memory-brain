package profile

// OperationsDetector detects operations/data directories
type OperationsDetector struct{}

func (d *OperationsDetector) Category() string {
	return CategoryOperations
}

func (d *OperationsDetector) Detect(ctx *AnalysisContext) float64 {
	score := 0.0

	// File extensions (weight: 0.5)
	dataFiles := map[string]float64{
		".csv":      0.5,
		".xlsx":     0.5,
		".xls":      0.5,
		".sql":      0.4,
		".db":       0.4,
		".sqlite":   0.4,
		".parquet":  0.5,
		".json":     0.2, // Could be data or config
		".xml":      0.2,
		".tsv":      0.5,
	}

	extensionScore := 0.0
	for ext, weight := range dataFiles {
		count := ctx.GetExtensionCount(ext)
		if count > 0 {
			extensionScore += weight * min(float64(count)/5.0, 1.0)
		}
	}
	score += 0.5 * min(extensionScore, 1.0)

	// Directory structure (weight: 0.3)
	dataDirs := []string{"data", "datasets", "reports", "exports", "backups", "archives", "logs", "analytics"}
	dirScore := 0.0
	for _, dir := range dataDirs {
		if ctx.HasDir(dir) {
			dirScore += 0.2
		}
	}
	score += 0.3 * min(dirScore, 1.0)

	// File names (weight: 0.2)
	dataFileNames := []string{"data.csv", "report.xlsx", "export.sql", "backup.db", "analytics.json"}
	fileScore := 0.0
	for _, file := range dataFileNames {
		if ctx.HasFile(file) {
			fileScore += 0.25
		}
	}
	score += 0.2 * min(fileScore, 1.0)

	return score
}
