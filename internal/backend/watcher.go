package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/postie/internal/watcher"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) initializeWatcher() error {
	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}

	// Stop previous watcher if running
	if a.watchCancel != nil {
		a.watchCancel()
		a.watchCancel = nil
	}
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}

	watcherCfg := a.config.GetWatcherConfig()

	// Only initialize watcher if it's enabled
	if !watcherCfg.Enabled {
		slog.Info("Watcher disabled in configuration")
		return nil
	}

	// Check dependencies
	if a.queue == nil {
		slog.Warn("Queue not available, skipping watcher initialization")
		return nil
	}

	// Get watch directory from config, or use default if not set
	watchDir := watcherCfg.WatchDirectory
	if watchDir == "" {
		// Use the OS-specific data directory for watch folder as default
		watchDir = filepath.Join(a.appPaths.Data, "watch")
	}

	// Ensure watch directory exists
	if err := os.MkdirAll(watchDir, 0755); err != nil {
		return fmt.Errorf("failed to create watch directory: %w", err)
	}

	// Event emitter for progress updates
	eventEmitter := func(eventName string, optionalData ...interface{}) {
		if a.ctx != nil {
			// If this is a progress event, update our internal progress tracker
			if eventName == "progress" && len(optionalData) > 0 {
				if progressData, ok := optionalData[0].(map[string]interface{}); ok {
					a.progressMux.Lock()

					// Update progress tracker with data from watcher
					if currentFile, ok := progressData["currentFile"].(string); ok {
						a.progress.CurrentFile = currentFile
					}
					if totalFiles, ok := progressData["totalFiles"].(int); ok {
						a.progress.TotalFiles = totalFiles
					}
					if completedFiles, ok := progressData["completedFiles"].(int); ok {
						a.progress.CompletedFiles = completedFiles
					}
					if stage, ok := progressData["stage"].(string); ok {
						a.progress.Stage = stage
					}
					if details, ok := progressData["details"].(string); ok {
						a.progress.Details = details
					}
					if speed, ok := progressData["speed"].(float64); ok {
						a.progress.Speed = speed
					}
					if secondsLeft, ok := progressData["secondsLeft"].(float64); ok {
						a.progress.SecondsLeft = secondsLeft
					}
					if isRunning, ok := progressData["isRunning"].(bool); ok {
						a.progress.IsRunning = isRunning
					}
					if lastUpdate, ok := progressData["lastUpdate"].(int64); ok {
						a.progress.LastUpdate = lastUpdate
					}
					if percentage, ok := progressData["percentage"].(float64); ok {
						a.progress.Percentage = percentage
					}
					if currentFileProgress, ok := progressData["currentFileProgress"].(float64); ok {
						a.progress.CurrentFileProgress = currentFileProgress
					}
					if jobID, ok := progressData["jobID"].(string); ok {
						a.progress.JobID = jobID
					}
					if elapsedTime, ok := progressData["elapsedTime"].(float64); ok {
						a.progress.ElapsedTime = elapsedTime
					}

					// Clear JobID when job is no longer running
					if stage, ok := progressData["stage"].(string); ok {
						if stage == "Completed" || stage == "Cancelled" || stage == "Error" {
							a.progress.JobID = ""
						}
					}

					a.progressMux.Unlock()
				}
			}

			runtime.EventsEmit(a.ctx, eventName, optionalData...)
		}
	}

	// Create separate context for watcher
	a.watchCtx, a.watchCancel = context.WithCancel(context.Background())

	// Initialize watcher
	a.watcher = watcher.New(watcherCfg, a.queue, a.processor, watchDir, eventEmitter)

	// Start watcher
	go func() {
		if err := a.watcher.Start(a.watchCtx); err != nil && err != context.Canceled {
			slog.Error("Watcher error", "error", err)
		}
	}()

	slog.Info("Watcher initialized successfully", "watchDir", watchDir)
	return nil
}
