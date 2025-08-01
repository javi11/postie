package processor

import (
	"path/filepath"
	"strings"
)

// isWithinPath checks if a file path is within a given directory path
func isWithinPath(filePath, dirPath string) bool {
	if dirPath == "" {
		return false
	}

	// Clean both paths to handle .. and . components
	cleanFilePath := filepath.Clean(filePath)
	cleanDirPath := filepath.Clean(dirPath)

	// Make both paths absolute for proper comparison
	absFilePath, err := filepath.Abs(cleanFilePath)
	if err != nil {
		return false
	}

	absDirPath, err := filepath.Abs(cleanDirPath)
	if err != nil {
		return false
	}

	// Check if the file path starts with the directory path
	rel, err := filepath.Rel(absDirPath, absFilePath)
	if err != nil {
		return false
	}

	// If the relative path starts with "..", the file is outside the directory
	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}
