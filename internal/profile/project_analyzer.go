package profile

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/oneliang/memory-brain/pkg/hash"
	"github.com/oneliang/memory-brain/pkg/types"
)

// ProjectAnalyzer analyzes directories and generates profiles
type ProjectAnalyzer struct{}

// NewProjectAnalyzer creates a new project analyzer
func NewProjectAnalyzer() *ProjectAnalyzer {
	return &ProjectAnalyzer{}
}

// AnalyzeProject analyzes a directory and generates a profile
func (a *ProjectAnalyzer) AnalyzeProject(directory string) (*types.ProjectProfile, error) {
	// Verify directory exists
	info, err := os.Stat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	// Scan directory
	ctx, err := NewAnalysisContext(directory)
	if err != nil {
		return nil, err
	}

	// Detect category
	category, categoryScores := DetectCategory(ctx)

	// Generate summary
	summary := a.generateSummary(category, ctx)

	// Generate highlights
	highlights := a.generateHighlights(ctx)

	// Compute project hash
	projectHash := hash.ProjectHash(directory)

	return &types.ProjectProfile{
		ID:          fmt.Sprintf("dir_%s", projectHash),
		Directory:   directory,
		ProjectHash: projectHash,
		Category:    category,
		Summary:     summary,
		Stats: types.DirStats{
			TotalFiles:     ctx.GetTotalFiles(),
			TotalDirs:      ctx.GetTotalDirs(),
			ExtensionCount: ctx.ExtensionCount,
			CategoryScores: categoryScores,
		},
		Highlights:   highlights,
		LastAccessed: time.Now(),
	}, nil
}

// generateSummary generates a category-specific summary
func (a *ProjectAnalyzer) generateSummary(category string, ctx *AnalysisContext) string {
	totalFiles := ctx.GetTotalFiles()
	totalDirs := ctx.GetTotalDirs()

	switch category {
	case CategoryDevelopment:
		techs := a.detectTechStack(ctx)
		if len(techs) > 0 {
			return fmt.Sprintf("开发项目，技术栈：%s。包含 %d 个文件，%d 个目录。",
				strings.Join(techs, "、"), totalFiles, totalDirs)
		}
		return fmt.Sprintf("开发项目，包含 %d 个文件，%d 个目录。", totalFiles, totalDirs)

	case CategoryDocumentation:
		docCount := a.countDocFiles(ctx)
		return fmt.Sprintf("文档库，包含 %d 个文档文件，%d 个目录。", docCount, totalDirs)

	case CategoryOperations:
		dataCount := a.countDataFiles(ctx)
		return fmt.Sprintf("数据/运营目录，包含 %d 个数据文件，%d 个目录。", dataCount, totalDirs)

	case CategoryDesign:
		designCount := a.countDesignFiles(ctx)
		return fmt.Sprintf("设计资源目录，包含 %d 个设计文件，%d 个目录。", designCount, totalDirs)

	case CategoryMedia:
		mediaCount := a.countMediaFiles(ctx)
		return fmt.Sprintf("媒体文件目录，包含 %d 个媒体文件，%d 个目录。", mediaCount, totalDirs)

	case CategoryMixed:
		return fmt.Sprintf("混合用途目录，包含 %d 个文件，%d 个目录。", totalFiles, totalDirs)

	case CategoryUnknown:
		return fmt.Sprintf("未分类目录，包含 %d 个文件，%d 个目录。", totalFiles, totalDirs)

	default:
		return fmt.Sprintf("目录包含 %d 个文件，%d 个目录。", totalFiles, totalDirs)
	}
}

// generateHighlights generates key file/pattern highlights
func (a *ProjectAnalyzer) generateHighlights(ctx *AnalysisContext) []string {
	var highlights []string

	// Add top extensions
	type extCount struct {
		ext   string
		count int
	}
	var exts []extCount
	for ext, count := range ctx.ExtensionCount {
		if count >= 3 { // Only include extensions with 3+ files
			exts = append(exts, extCount{ext, count})
		}
	}

	// Sort by count descending
	sort.Slice(exts, func(i, j int) bool {
		return exts[i].count > exts[j].count
	})

	// Add top 3 extensions
	for i := 0; i < len(exts) && i < 3; i++ {
		highlights = append(highlights, fmt.Sprintf("%s 文件：%d 个", exts[i].ext, exts[i].count))
	}

	return highlights
}

// detectTechStack detects development technologies
func (a *ProjectAnalyzer) detectTechStack(ctx *AnalysisContext) []string {
	var techs []string

	techIndicators := map[string]string{
		"go.mod":          "Go",
		"package.json":    "Node.js",
		"requirements.txt": "Python",
		"cargo.toml":      "Rust",
		"pom.xml":         "Java/Maven",
		"makefile":        "Make",
	}

	// Sort keys for deterministic output
	var files []string
	for file := range techIndicators {
		files = append(files, file)
	}
	sort.Strings(files)

	for _, file := range files {
		tech := techIndicators[file]
		if ctx.HasFile(file) {
			techs = append(techs, tech)
		}
	}

	// Check by extension
	if ctx.HasExtension(".go") {
		techs = appendIfMissing(techs, "Go")
	}
	if ctx.HasExtension(".js") || ctx.HasExtension(".ts") {
		techs = appendIfMissing(techs, "JavaScript/TypeScript")
	}
	if ctx.HasExtension(".py") {
		techs = appendIfMissing(techs, "Python")
	}

	return techs
}

// countDocFiles counts documentation files
func (a *ProjectAnalyzer) countDocFiles(ctx *AnalysisContext) int {
	count := 0
	docExts := []string{".pdf", ".epub", ".docx", ".doc", ".md", ".txt", ".tex", ".pptx", ".ppt", ".rtf"}
	for _, ext := range docExts {
		count += ctx.GetExtensionCount(ext)
	}
	return count
}

// countDataFiles counts data/operations files
func (a *ProjectAnalyzer) countDataFiles(ctx *AnalysisContext) int {
	count := 0
	dataExts := []string{".csv", ".xlsx", ".xls", ".sql", ".db", ".sqlite", ".parquet", ".tsv"}
	for _, ext := range dataExts {
		count += ctx.GetExtensionCount(ext)
	}
	return count
}

// countDesignFiles counts design files
func (a *ProjectAnalyzer) countDesignFiles(ctx *AnalysisContext) int {
	count := 0
	designExts := []string{".psd", ".sketch", ".fig", ".ai", ".xd", ".svg", ".eps", ".png", ".jpg", ".jpeg", ".gif"}
	for _, ext := range designExts {
		count += ctx.GetExtensionCount(ext)
	}
	return count
}

// countMediaFiles counts media files
func (a *ProjectAnalyzer) countMediaFiles(ctx *AnalysisContext) int {
	count := 0
	mediaExts := []string{".mp4", ".mkv", ".avi", ".mov", ".mp3", ".wav", ".flac", ".aac", ".ogg"}
	for _, ext := range mediaExts {
		count += ctx.GetExtensionCount(ext)
	}
	return count
}

// Helper functions

func appendIfMissing(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
