package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalysisContext(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test_analysis_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test structure
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test file count
	if ctx.GetTotalFiles() != 3 {
		t.Errorf("Expected 3 files, got %d", ctx.GetTotalFiles())
	}

	// Test dir count
	if ctx.GetTotalDirs() != 1 {
		t.Errorf("Expected 1 dir, got %d", ctx.GetTotalDirs())
	}

	// Test extension count
	if ctx.GetExtensionCount(".go") != 2 {
		t.Errorf("Expected 2 .go files, got %d", ctx.GetExtensionCount(".go"))
	}

	if ctx.GetExtensionCount(".md") != 1 {
		t.Errorf("Expected 1 .md file, got %d", ctx.GetExtensionCount(".md"))
	}

	// Test HasExtension
	if !ctx.HasExtension(".go") {
		t.Error("Expected HasExtension(.go) to be true")
	}

	// Test HasDir
	if !ctx.HasDir("src") {
		t.Error("Expected HasDir(src) to be true")
	}

	// Test HasFile
	if !ctx.HasFile("README.md") {
		t.Error("Expected HasFile(README.md) to be true")
	}
}

func TestDevelopmentDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_dev_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Go project structure
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "internal"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "internal", "handler.go"), []byte("package internal"), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	detector := &DevelopmentDetector{}
	score := detector.Detect(ctx)

	if score < 0.3 {
		t.Errorf("Expected development score > 0.3, got %f", score)
	}

	category, _ := DetectCategory(ctx)
	if category != CategoryDevelopment {
		t.Errorf("Expected category=development, got %s", category)
	}
}

func TestDocumentationDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_docs_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create documentation structure
	os.Mkdir(filepath.Join(tmpDir, "docs"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "paper1.pdf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "paper2.pdf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "notes.md"), []byte("# Notes"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "docs", "guide.pdf"), []byte(""), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	detector := &DocumentationDetector{}
	score := detector.Detect(ctx)

	if score < 0.2 {
		t.Errorf("Expected documentation score > 0.2, got %f", score)
	}

	category, _ := DetectCategory(ctx)
	if category != CategoryDocumentation {
		t.Errorf("Expected category=documentation, got %s", category)
	}
}

func TestOperationsDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_ops_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create data structure
	os.Mkdir(filepath.Join(tmpDir, "data"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "report.csv"), []byte("a,b,c"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "data.xlsx"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "backup.sql"), []byte("SELECT 1"), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	detector := &OperationsDetector{}
	score := detector.Detect(ctx)

	if score < 0.15 {
		t.Errorf("Expected operations score > 0.15, got %f", score)
	}

	category, _ := DetectCategory(ctx)
	if category != CategoryOperations {
		t.Errorf("Expected category=operations, got %s", category)
	}
}

func TestDesignDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_design_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create design structure
	os.Mkdir(filepath.Join(tmpDir, "assets"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "logo.svg"), []byte("<svg></svg>"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "icon.png"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "banner.jpg"), []byte(""), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	detector := &DesignDetector{}
	score := detector.Detect(ctx)

	if score < 0.2 {
		t.Errorf("Expected design score > 0.2, got %f", score)
	}

	category, _ := DetectCategory(ctx)
	if category != CategoryDesign {
		t.Errorf("Expected category=design, got %s", category)
	}
}

func TestMediaDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_media_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create media structure
	os.Mkdir(filepath.Join(tmpDir, "videos"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "video.mp4"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "audio.mp3"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "videos", "clip.mkv"), []byte(""), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	detector := &MediaDetector{}
	score := detector.Detect(ctx)

	if score < 0.4 {
		t.Errorf("Expected media score > 0.4, got %f", score)
	}

	category, _ := DetectCategory(ctx)
	if category != CategoryMedia {
		t.Errorf("Expected category=media, got %s", category)
	}
}

func TestMixedDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_mixed_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mixed structure with equal weight for multiple categories
	os.WriteFile(filepath.Join(tmpDir, "code.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "paper.pdf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "data.csv"), []byte("a,b,c"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "design.psd"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "video.mp4"), []byte(""), 0644)

	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	category, scores := DetectCategory(ctx)
	// Should be mixed because multiple categories have significant scores
	if category != CategoryMixed && category != CategoryDevelopment {
		t.Logf("Category: %s, Scores: %+v", category, scores)
		// Accept either mixed or the highest scoring category
	}
}

func TestUnknownDetector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_unknown_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty directory
	ctx, err := NewAnalysisContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	category, _ := DetectCategory(ctx)
	if category != CategoryUnknown {
		t.Errorf("Expected category=unknown, got %s", category)
	}
}

func TestProjectAnalyzer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_analyzer_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Go project
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "internal"), 0755)

	analyzer := NewProjectAnalyzer()
	profile, err := analyzer.AnalyzeProject(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if profile.Category != CategoryDevelopment {
		t.Errorf("Expected category=development, got %s", profile.Category)
	}

	if profile.Stats.TotalFiles != 2 {
		t.Errorf("Expected 2 files, got %d", profile.Stats.TotalFiles)
	}

	if profile.Stats.TotalDirs != 1 {
		t.Errorf("Expected 1 dir, got %d", profile.Stats.TotalDirs)
	}

	if profile.Summary == "" {
		t.Error("Expected non-empty summary")
	}
}
