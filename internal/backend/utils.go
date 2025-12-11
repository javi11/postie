package backend

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Helper function to get keys from map
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// readLogLines reads lines from the log file with pagination
func readLogLines(file *os.File, limit, offset int) (string, error) {
	// For very large files (>10MB), we should optimize this
	// For now, we'll read the entire file for simplicity
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	if len(content) == 0 {
		return "", nil
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Remove empty last line if it exists (common with newline-terminated files)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	totalLines := len(lines)
	if totalLines == 0 {
		return "", nil
	}

	// Validate parameters
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		return "", fmt.Errorf("limit must be positive")
	}

	// Calculate start and end indices
	// We want the most recent lines, so we work backwards from the end
	endIndex := totalLines - offset
	if endIndex <= 0 {
		return "", nil // No lines to return
	}

	startIndex := endIndex - limit
	if startIndex < 0 {
		startIndex = 0
	}

	// Ensure we don't go out of bounds
	if startIndex >= totalLines {
		return "", nil
	}
	if endIndex > totalLines {
		endIndex = totalLines
	}

	// Get the requested lines
	requestedLines := lines[startIndex:endIndex]

	// Join with newlines and return
	return strings.Join(requestedLines, "\n"), nil
}

// wrapSaveConfigError wraps configuration save errors with user-friendly messages
func (a *App) wrapSaveConfigError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Permission errors
	if os.IsPermission(err) || strings.Contains(errStr, "permission denied") {
		return fmt.Errorf("Permission denied: cannot write configuration to '%s'. Please check directory permissions", a.configPath)
	}

	// Directory doesn't exist
	if os.IsNotExist(err) {
		return fmt.Errorf("Configuration directory does not exist: '%s'", filepath.Dir(a.configPath))
	}

	// Disk full
	if strings.Contains(errStr, "no space left") {
		return fmt.Errorf("Not enough disk space to save configuration")
	}

	// Server validation errors - pass through as-is (already descriptive)
	if strings.Contains(errStr, "server validation failed") {
		return err
	}

	// Generic fallback with path context
	return fmt.Errorf("Failed to save configuration to '%s': %v", a.configPath, err)
}

// processDirectoryRecursively processes a directory and returns collected files grouped by folder
func processDirectoryRecursively(dirPath string) (map[string][]string, map[string]int64, error) {
	filesByFolder := make(map[string][]string)
	sizeByFolder := make(map[string]int64)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Error walking directory", "path", path, "error", err)
			return nil // Continue processing other files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the parent directory
		folderPath := filepath.Dir(path)

		// Add file to the folder's file list
		filesByFolder[folderPath] = append(filesByFolder[folderPath], path)
		sizeByFolder[folderPath] += info.Size()

		slog.Debug("Found file in directory", "file", path, "folder", folderPath, "size", info.Size())

		return nil
	})

	return filesByFolder, sizeByFolder, err
}
