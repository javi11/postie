package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UploadFiles uploads the selected files using the queue system
func (a *App) UploadFiles() error {
	defer a.recoverPanic("UploadFiles")

	// Open file dialog
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select files to upload",
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

		// Skip directories
		if info.IsDir() {
			slog.Info("Skipping directory", "path", filePath)
			continue
		}

		// Add file to queue
		if err := a.queue.AddFile(context.Background(), filePath, info.Size()); err != nil {
			slog.Warn("Could not add selected file to queue, skipping", "file", filePath, "error", err)
			continue
		}

		addedCount++
		slog.Info("Selected file added to queue", "file", filePath, "size", info.Size())
	}

	if addedCount > 0 {
		slog.Info("Added selected files to queue", "added", addedCount, "total", len(files))
		
		// Emit event to refresh queue in frontend
		if !a.isWebMode {
			runtime.EventsEmit(a.ctx, "queue-updated")
		} else if a.webEventEmitter != nil {
			a.webEventEmitter("queue-updated", nil)
		}
	}

	if addedCount == 0 {
		return fmt.Errorf("no valid files could be added to queue")
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
