package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
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

	// Ensure output directory exists - only set permissions if creating new directory
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w, %s", err, outputDir)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check output directory: %w, %s", err, outputDir)
	}

	// Get watcher config for delete original file setting
	watcherCfg := a.config.GetWatcherConfig()

	// Initialize processor (always needed)
	a.processor = processor.New(processor.ProcessorOptions{
		Queue:                     a.queue,
		Config:                    a.config,
		QueueConfig:               queueCfg,
		PoolManager:               a.poolManager,
		OutputFolder:              outputDir,
		DeleteOriginalFile:        watcherCfg.DeleteOriginalFile,
		MaintainOriginalExtension: a.config.GetMaintainOriginalExtension(),
		WatchFolder:               watcherCfg.WatchDirectory,
		CanProcessNextItem:        a.canProcessNextItem,
	})

	// Start processor
	go func() {
		if err := a.processor.Start(a.procCtx); err != nil && !errors.Is(err, context.Canceled) {
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
