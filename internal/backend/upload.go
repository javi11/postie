package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UploadFiles uploads the selected files using the queue system
func (a *App) UploadFiles() error {
	defer a.recoverPanic("UploadFiles")

	// Open file dialog - allow both files and directories
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select files or folders to upload",
	})
	if err != nil {
		return fmt.Errorf("error opening file dialog: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	// Check if configuration is valid before proceeding
	status := a.GetAppStatus()
	if status.NeedsConfiguration {
		return fmt.Errorf("configuration required: Please configure at least one server in the Settings page before uploading files")
	}

	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}

	// Add files to queue
	addedCount := 0
	for _, filePath := range files {
		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			slog.Warn("Could not get file info for selected file, skipping", "file", filePath, "error", err)
			continue
		}

		// Handle directories by processing them as single NZB units
		if info.IsDir() {
			slog.Info("Processing selected directory", "path", filePath)

			// Process directory recursively to collect files
			filesByFolder, sizeByFolder, err := processDirectoryRecursively(filePath)
			if err != nil {
				slog.Error("Error processing directory", "path", filePath, "error", err)
				continue
			}

			// Add each folder as a single queue entry (following watcher pattern)
			for folderPath, files := range filesByFolder {
				if len(files) == 0 {
					continue
				}

				folderSize := sizeByFolder[folderPath]
				folderName := filepath.Base(folderPath)

				slog.Info("Adding folder to queue", "folder", folderName, "files", len(files), "size", folderSize)

				// Add the folder to the queue with FOLDER: prefix to indicate it's a folder
				folderQueuePath := "FOLDER:" + folderPath
				if err := a.queue.AddFile(context.Background(), folderQueuePath, folderSize); err != nil {
					slog.Warn("Could not add folder to queue, skipping", "folder", folderPath, "error", err)
					continue
				}

				addedCount++
				slog.Info("Selected folder added to queue", "folder", folderName, "files", len(files), "size", folderSize)

				// Log the files in the folder for debugging
				for _, file := range files {
					slog.Debug("File in selected folder", "folder", folderName, "file", filepath.Base(file))
				}
			}

			continue
		}

		// Handle individual files (existing logic)
		if err := a.queue.AddFile(context.Background(), filePath, info.Size()); err != nil {
			slog.Warn("Could not add selected file to queue, skipping", "file", filePath, "error", err)
			continue
		}

		addedCount++
		slog.Info("Selected file added to queue", "file", filePath, "size", info.Size())
	}

	if addedCount > 0 {
		slog.Info("Added selected items to queue", "added", addedCount, "total", len(files))

		// Emit event to refresh queue in frontend
		if !a.isWebMode {
			runtime.EventsEmit(a.ctx, "queue-updated")
		} else if a.webEventEmitter != nil {
			a.webEventEmitter("queue-updated", nil)
		}
	}

	if addedCount == 0 {
		return fmt.Errorf("no valid files or folders could be added to queue")
	}

	return nil
}

// IsUploading returns whether any uploads are in progress (via processor)
func (a *App) IsUploading() bool {
	if a.processor == nil {
		return false
	}

	runningJobs := a.processor.GetRunningJobs()
	return len(runningJobs) > 0
}

// CancelUpload is no longer used - individual jobs are cancelled via CancelJob
// This function is kept for backward compatibility but does nothing
func (a *App) CancelUpload() error {
	return fmt.Errorf("use CancelJob() to cancel individual jobs instead")
}

// SelectFolder opens a native folder picker dialog and returns the selected folder path
// Returns empty string if no folder was selected
func (a *App) SelectFolder() (string, error) {
	defer a.recoverPanic("SelectFolder")

	// Open directory dialog
	folderPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select folder to upload",
	})
	if err != nil {
		return "", fmt.Errorf("error opening folder dialog: %w", err)
	}

	return folderPath, nil
}

// UploadFolder uploads all files from a folder as a single NZB
// The folder structure will be preserved in the article subjects
func (a *App) UploadFolder(folderPath string) error {
	defer a.recoverPanic("UploadFolder")

	if folderPath == "" {
		return fmt.Errorf("folder path is empty")
	}

	// Check if folder exists
	info, err := os.Stat(folderPath)
	if err != nil {
		return fmt.Errorf("error accessing folder: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a folder: %s", folderPath)
	}

	// Check if configuration is valid before proceeding
	status := a.GetAppStatus()
	if status.NeedsConfiguration {
		return fmt.Errorf("configuration required: Please configure at least one server in the Settings page before uploading files")
	}

	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}

	// Process directory recursively to collect files
	filesByFolder, sizeByFolder, err := processDirectoryRecursively(folderPath)
	if err != nil {
		return fmt.Errorf("error processing folder: %w", err)
	}

	// Check if folder has any files
	totalFiles := 0
	for _, files := range filesByFolder {
		totalFiles += len(files)
	}
	if totalFiles == 0 {
		return fmt.Errorf("folder is empty or contains no files: %s", folderPath)
	}

	// Add each folder as a single queue entry (following watcher pattern)
	addedCount := 0
	for folderSubPath, files := range filesByFolder {
		if len(files) == 0 {
			continue
		}

		folderSize := sizeByFolder[folderSubPath]
		folderName := filepath.Base(folderSubPath)

		slog.Info("Adding folder to queue", "folder", folderName, "files", len(files), "size", folderSize)

		// Add the folder to the queue with FOLDER: prefix to indicate it's a folder
		folderQueuePath := "FOLDER:" + folderSubPath
		if err := a.queue.AddFile(context.Background(), folderQueuePath, folderSize); err != nil {
			slog.Warn("Could not add folder to queue, skipping", "folder", folderSubPath, "error", err)
			continue
		}

		addedCount++
		slog.Info("Folder added to queue", "folder", folderName, "files", len(files), "size", folderSize)

		// Log the files in the folder for debugging
		for _, file := range files {
			slog.Debug("File in folder", "folder", folderName, "file", filepath.Base(file))
		}
	}

	if addedCount > 0 {
		slog.Info("Added folder to queue", "folder", filepath.Base(folderPath), "subfolders", addedCount, "totalFiles", totalFiles)

		// Emit event to refresh queue in frontend
		if !a.isWebMode {
			runtime.EventsEmit(a.ctx, "queue-updated")
		} else if a.webEventEmitter != nil {
			a.webEventEmitter("queue-updated", nil)
		}
	}

	if addedCount == 0 {
		return fmt.Errorf("no valid files or folders could be added to queue")
	}

	return nil
}
