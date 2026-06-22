package profile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxScanDepth  = 3
	maxScanFiles  = 1000
)

var errScanComplete = errors.New("scan complete")

// AnalysisContext holds scanned directory information for category detection
type AnalysisContext struct {
	RootDir        string
	Files          []string            // All file paths (relative to root)
	Dirs           []string            // All directory paths (relative to root)
	ExtensionCount map[string]int      // Extension -> count
	FileNames      map[string]bool     // File names (lowercase)
	DirNames       map[string]bool     // Directory names (lowercase)
}

// NewAnalysisContext scans a directory and builds analysis context
func NewAnalysisContext(rootDir string) (*AnalysisContext, error) {
	ctx := &AnalysisContext{
		RootDir:        rootDir,
		Files:          []string{},
		Dirs:           []string{},
		ExtensionCount: make(map[string]int),
		FileNames:      make(map[string]bool),
		DirNames:       make(map[string]bool),
	}

	err := ctx.scan()
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

// scan walks the directory tree with depth and file count limits
func (ctx *AnalysisContext) scan() error {
	fileCount := 0

	err := filepath.Walk(ctx.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log but continue on errors
			return nil
		}

		// Calculate depth
		relPath, err := filepath.Rel(ctx.RootDir, path)
		if err != nil {
			return nil
		}

		depth := 0
		if relPath != "." {
			depth = strings.Count(relPath, string(os.PathSeparator)) + 1
		}

		// Skip if too deep
		if depth > maxScanDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories and files
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			if relPath != "." {
				ctx.Dirs = append(ctx.Dirs, relPath)
				ctx.DirNames[strings.ToLower(info.Name())] = true
			}
		} else {
			// Check file count limit BEFORE adding
			if fileCount >= maxScanFiles {
				return errScanComplete // Short-circuit the walk
			}

			ctx.Files = append(ctx.Files, relPath)
			ctx.FileNames[strings.ToLower(info.Name())] = true

			// Count extensions
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext != "" {
				ctx.ExtensionCount[ext]++
			}

			fileCount++
		}

		return nil
	})

	// Treat errScanComplete as success
	if err == errScanComplete {
		return nil
	}
	return err
}

// HasExtension checks if any file has the given extension
func (ctx *AnalysisContext) HasExtension(ext string) bool {
	count, exists := ctx.ExtensionCount[strings.ToLower(ext)]
	return exists && count > 0
}

// HasDir checks if a directory with the given name exists
func (ctx *AnalysisContext) HasDir(name string) bool {
	return ctx.DirNames[strings.ToLower(name)]
}

// HasFile checks if a file with the given name exists
func (ctx *AnalysisContext) HasFile(name string) bool {
	return ctx.FileNames[strings.ToLower(name)]
}

// GetExtensionCount returns the count of files with the given extension
func (ctx *AnalysisContext) GetExtensionCount(ext string) int {
	return ctx.ExtensionCount[strings.ToLower(ext)]
}

// GetTotalFiles returns total number of files scanned
func (ctx *AnalysisContext) GetTotalFiles() int {
	return len(ctx.Files)
}

// GetTotalDirs returns total number of directories scanned
func (ctx *AnalysisContext) GetTotalDirs() int {
	return len(ctx.Dirs)
}
