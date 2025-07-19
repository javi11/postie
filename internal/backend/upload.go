package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UploadFiles uploads the selected files
func (a *App) UploadFiles() error {
	defer a.recoverPanic("UploadFiles")
	
	a.uploadingMux.Lock()
	if a.uploading {
		a.uploadingMux.Unlock()
		return fmt.Errorf("upload already in progress")
	}
	a.uploading = true
	a.uploadingMux.Unlock()

	// Open file dialog
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select files to upload",
	})
	if err != nil {
		a.uploadingMux.Lock()
		a.uploading = false
		a.uploadingMux.Unlock()
		return fmt.Errorf("error opening file dialog: %w", err)
	}

	if len(files) == 0 {
		a.uploadingMux.Lock()
		a.uploading = false
		a.uploadingMux.Unlock()
		return nil
	}

	// Check if configuration is valid before proceeding with upload
	status := a.GetAppStatus()
	if status.NeedsConfiguration {
		a.uploadingMux.Lock()
		a.uploading = false
		a.uploadingMux.Unlock()
		return fmt.Errorf("configuration required: Please configure at least one server in the Settings page before uploading files")
	}

	// Convert to fileinfo and calculate total bytes
	var fileInfos []fileinfo.FileInfo
	var totalBytes int64
	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			a.uploadingMux.Lock()
			a.uploading = false
			a.uploadingMux.Unlock()
			return fmt.Errorf("error getting file info: %w", err)
		}
		fileInfos = append(fileInfos, fileinfo.FileInfo{
			Path: filePath,
			Size: uint64(info.Size()),
		})
		totalBytes += info.Size()
	}

	// Create cancellable context for this upload
	a.uploadCtx, a.uploadCancel = context.WithCancel(context.Background())

	// Start upload in background
	go func() {
		defer func() {
			a.uploadingMux.Lock()
			a.uploading = false
			a.uploadCancel = nil
			a.uploadCtx = nil
			a.uploadingMux.Unlock()
		}()

		// Set up progress callback - this will handle all progress updates
		progressCallback := func(stage string, current, total int64, details string, speed float64, secondsLeft float64, elapsedTime float64) {
			var fileProgress float64
			if total > 0 {
				fileProgress = float64(current) / float64(total) * 100
			}

			a.progressMux.Lock()
			a.progress.Stage = stage
			a.progress.Details = details
			a.progress.CurrentFileProgress = fileProgress
			a.progress.CurrentFileBytes = current
			a.progress.Speed = speed
			a.progress.SecondsLeft = secondsLeft
			a.progress.ElapsedTime = elapsedTime
			a.progressMux.Unlock()

			// Emit progress event to frontend for both desktop and web modes
			if !a.isWebMode {
				runtime.EventsEmit(a.ctx, "progress", a.getProgress())
			} else if a.webEventEmitter != nil {
				a.webEventEmitter("progress", a.getProgress())
			}
		}

		// Set progress callback on postie instance
		if a.postie != nil {
			a.postie.SetProgressCallback(progressCallback)
		}

		// Initialize progress state
		a.updateProgress(ProgressUpdate{
			Stage:               "Initializing",
			IsRunning:           true,
			CompletedFiles:      0,
			Percentage:          0.0,
			CurrentFile:         stringPtr("Adding files to queue..."),
			TotalFiles:          intPtr(len(fileInfos)),
			TotalBytes:          int64Ptr(totalBytes),
			TransferredBytes:    int64Ptr(0),
			CurrentFileBytes:    int64Ptr(0),
			CurrentFileProgress: float64Ptr(0.0),
			ElapsedTime:         float64Ptr(0.0),
		})

		// Track transferred bytes
		var transferredBytes int64

		for i, f := range fileInfos {
			// Check if upload was cancelled
			select {
			case <-a.uploadCtx.Done():
				a.updateProgress(ProgressUpdate{
					Stage:          "Cancelled",
					IsRunning:      false,
					CompletedFiles: i,
					Percentage:     0.0,
					CurrentFile:    stringPtr("Upload cancelled"),
				})
				return
			default:
			}

			fileName := filepath.Base(f.Path)
			currentFileSize := int64(f.Size)

			// Update current file info - the callback will handle the detailed progress
			var percentage float64
			if len(fileInfos) > 0 {
				percentage = float64(i) / float64(len(fileInfos)) * 100
			}
			a.updateProgress(ProgressUpdate{
				CurrentFile:         stringPtr(fileName),
				CompletedFiles:      i,
				Stage:               "Adding to queue",
				IsRunning:           true,
				TransferredBytes:    int64Ptr(transferredBytes),
				CurrentFileBytes:    int64Ptr(0),
				CurrentFileProgress: float64Ptr(0.0),
				Percentage:          percentage,
				ElapsedTime:         float64Ptr(0.0),
			})

			// Add file to queue instead of processing directly
			if err := a.queue.AddFile(a.uploadCtx, f.Path, int64(f.Size)); err != nil {
				if a.uploadCtx.Err() != nil {
					a.updateProgress(ProgressUpdate{
						CurrentFile:    stringPtr("Upload cancelled"),
						CompletedFiles: i,
						Stage:          "Cancelled",
						IsRunning:      false,
					})
					return
				}
				slog.Error("Error adding file to queue", "error", err, "file", f.Path)
				a.updateProgress(ProgressUpdate{
					CurrentFile:    stringPtr(fileName),
					CompletedFiles: i,
					Stage:          fmt.Sprintf("Error: %v", err),
					IsRunning:      false,
				})
				return
			}

			// File added to queue, add its size to transferred bytes
			transferredBytes += currentFileSize

			// Update progress for completed file
			if len(fileInfos) > 0 {
				percentage = float64(i+1) / float64(len(fileInfos)) * 100
			}
			a.updateProgress(ProgressUpdate{
				CompletedFiles:      i + 1,
				TransferredBytes:    int64Ptr(transferredBytes),
				CurrentFileProgress: float64Ptr(100.0),
				CurrentFileBytes:    int64Ptr(currentFileSize),
				Percentage:          percentage,
			})
		}

		if len(fileInfos) > 0 {
			a.updateProgress(ProgressUpdate{
				CurrentFile:      stringPtr("All files added to queue"),
				CompletedFiles:   len(fileInfos),
				Stage:            "Queued",
				IsRunning:        false,
				Percentage:       100.0,
				TransferredBytes: int64Ptr(transferredBytes),
			})
		}
	}()

	return nil
}

// IsUploading returns whether an upload is in progress
func (a *App) IsUploading() bool {
	a.uploadingMux.RLock()
	defer a.uploadingMux.RUnlock()
	return a.uploading
}

// CancelUpload cancels the currently running direct upload
func (a *App) CancelUpload() error {
	defer a.recoverPanic("CancelUpload")
	
	a.uploadingMux.Lock()
	defer a.uploadingMux.Unlock()

	if !a.uploading {
		return fmt.Errorf("no upload in progress")
	}

	if a.uploadCancel != nil {
		a.uploadCancel()
		return nil
	}

	return fmt.Errorf("upload cannot be cancelled")
}
