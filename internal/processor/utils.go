package processor

import (
	"os"
	"path/filepath"
	"strings"
)

// isSymlink checks if the given path is a symbolic link using Lstat.
// Returns true if the path is a symlink, false otherwise.
func isSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

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
