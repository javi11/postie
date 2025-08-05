package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/postie/internal/watcher"
)

func (a *App) initializeWatcher() error {
	defer a.recoverPanic("initializeWatcher")

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

	// Ensure watch directory exists - only set permissions if creating new directory
	if _, err := os.Stat(watchDir); os.IsNotExist(err) {
		if err := os.MkdirAll(watchDir, 0755); err != nil {
			return fmt.Errorf("failed to create watch directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check watch directory: %w", err)
	}

	// Create separate context for watcher
	a.watchCtx, a.watchCancel = context.WithCancel(context.Background())

	// Initialize watcher
	a.watcher = watcher.New(watcherCfg, a.queue, a.processor, watchDir)

	// Start watcher
	go func() {
		if err := a.watcher.Start(a.watchCtx); err != nil && err != context.Canceled {
			slog.Error("Watcher error", "error", err)
		}
	}()

	slog.Info("Watcher initialized successfully", "watchDir", watchDir)
	return nil
}
