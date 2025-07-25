package backend

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/javi11/postie/internal/processor"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) initializeProcessor() error {
	defer a.recoverPanic("initializeProcessor")

	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}

	// Stop previous processor if running
	if a.processor != nil {
		a.processor = nil
	}

	// Only initialize processor if we have valid servers configured
	validServers := 0
	for _, server := range a.config.Servers {
		if server.Host != "" {
			validServers++
		}
	}

	if validServers == 0 {
		slog.Info("No valid servers configured, skipping processor initialization")
		return nil
	}

	if a.postie == nil {
		slog.Info("No postie instance available, skipping processor initialization")
		return nil
	}

	if a.queue == nil {
		slog.Warn("Queue not available, skipping processor initialization")
		return nil
	}

	queueCfg := a.config.GetQueueConfig()

	// Get output directory from configuration
	outputDir := a.config.GetOutputDir()

	// If output directory is relative, make it relative to OS-specific data directory
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(a.appPaths.Data, outputDir)
	}

	// Event emitter for progress updates
	eventEmitter := func(eventName string, optionalData ...interface{}) {
		if a.ctx != nil {
			// If this is a progress event, update our internal progress tracker
			if eventName == "progress" && len(optionalData) > 0 {
				if progressData, ok := optionalData[0].(map[string]interface{}); ok {
					a.progressMux.Lock()

					// Update progress tracker with data from processor
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

			// Emit events for both desktop and web modes
			if !a.isWebMode {
				runtime.EventsEmit(a.ctx, eventName, optionalData...)
			} else if a.webEventEmitter != nil {
				// For web mode, send the data appropriately
				var data interface{}
				if len(optionalData) > 0 {
					data = optionalData[0]
				}
				a.webEventEmitter(eventName, data)
			}
		}
	}

	// Get watcher config for delete original file setting
	watcherCfg := a.config.GetWatcherConfig()

	// Initialize processor (always needed)
	a.processor = processor.New(processor.ProcessorOptions{
		Queue:                     a.queue,
		Postie:                    a.postie,
		Config:                    queueCfg,
		OutputFolder:              outputDir,
		EventEmitter:              eventEmitter,
		DeleteOriginalFile:        watcherCfg.DeleteOriginalFile,
		MaintainOriginalExtension: a.config.GetMaintainOriginalExtension(),
		WatchFolder:               watcherCfg.WatchDirectory, // Pass watch folder for folder structure maintenance
	})

	// Start processor
	go func() {
		if err := a.processor.Start(a.procCtx); err != nil && err != context.Canceled {
			slog.Error("Processor error", "error", err)
		}
	}()

	slog.Info("Processor initialized successfully", "outputDir", outputDir)
	return nil
}

// CancelJob cancels a running job via processor
func (a *App) CancelJob(id string) error {
	defer a.recoverPanic("CancelJob")

	if a.processor == nil {
		return fmt.Errorf("processor not initialized")
	}

	err := a.processor.CancelJob(id)
	if err != nil {
		return err
	}

	// Emit event to refresh queue in frontend for both desktop and web modes
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "queue-updated")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("queue-updated", nil)
	}
	return nil
}
