package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/parpardownloader"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetConfigPath returns the path to the configuration file
func (a *App) GetConfigPath() string {
	return a.configPath
}

// GetConfig returns the current configuration (pending config if available, otherwise applied config)
func (a *App) GetConfig() (*config.ConfigData, error) {
	defer a.recoverPanic("GetConfig")

	if a.configPath == "" {
		return nil, fmt.Errorf("no config file specified")
	}

	// Check if file exists
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found")
	}

	// If we have pending config, return that instead of applied config
	a.pendingConfigMux.RLock()
	pendingConfig := a.pendingConfig
	a.pendingConfigMux.RUnlock()

	if pendingConfig != nil {
		slog.Debug("Returning pending configuration to frontend")
		return pendingConfig, nil
	}

	// If we have a loaded config, get its data
	if a.config != nil {
		return a.config, nil
	}

	// Otherwise load it fresh
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	return cfg, nil
}

// GetAppliedConfig returns the currently applied configuration (ignoring pending)
func (a *App) GetAppliedConfig() (*config.ConfigData, error) {
	defer a.recoverPanic("GetAppliedConfig")

	if a.configPath == "" {
		return nil, fmt.Errorf("no config file specified")
	}

	// Check if file exists
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found")
	}

	// Always return the applied config, ignoring pending
	if a.config != nil {
		return a.config, nil
	}

	// Otherwise load it fresh
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig saves the configuration
func (a *App) SaveConfig(configData *config.ConfigData) error {
	defer a.recoverPanic("SaveConfig")

	slog.Info("Saving config", "path", a.configPath, "configData", configData)

	// Ensure version is set to current version when saving
	configData.Version = config.CurrentConfigVersion

	// Validate server connections before saving
	if err := a.validateServerConnections(configData); err != nil {
		return fmt.Errorf("server validation failed: %w", err)
	}

	// Check if there are active uploads or running jobs
	hasActiveWork := a.IsUploading()
	if a.processor != nil {
		runningJobs := a.processor.GetRunningJobs()
		hasActiveWork = hasActiveWork || len(runningJobs) > 0
	}

	if hasActiveWork {
		// Store config as pending and defer application
		a.pendingConfigMux.Lock()
		a.pendingConfig = configData
		a.pendingConfigMux.Unlock()

		// Save to file but don't apply yet
		if err := config.SaveConfig(configData, a.configPath); err != nil {
			return err
		}

		// Emit pending config event to frontend
		pendingStatus := map[string]interface{}{
			"hasPendingConfig": true,
			"message":          "Configuration changes will be applied when current uploads finish",
		}
		if !a.isWebMode {
			runtime.EventsEmit(a.ctx, "config-pending", pendingStatus)
		} else if a.webEventEmitter != nil {
			a.webEventEmitter("config-pending", pendingStatus)
		}

		slog.Info("Config saved but deferred due to active uploads/jobs")
		return nil
	}

	// No active work, apply config immediately
	return a.applyConfigChanges(configData)
}

// applyConfigChanges applies the configuration changes immediately
func (a *App) applyConfigChanges(configData *config.ConfigData) error {
	if err := config.SaveConfig(configData, a.configPath); err != nil {
		return err
	}

	// Reload configuration
	if err := a.loadConfig(); err != nil {
		return err
	}

	// Update the connection pool manager with new configuration
	if a.poolManager != nil {
		if err := a.poolManager.UpdateConfig(a.config); err != nil {
			slog.Error("Failed to update connection pool manager with new config", "error", err)
			// Don't fail the entire config update for this, but log the error
		} else {
			slog.Info("Connection pool manager updated with new configuration")
		}
	}

	// Clear any pending config since we just applied changes
	a.pendingConfigMux.Lock()
	a.pendingConfig = nil
	a.pendingConfigMux.Unlock()

	// Emit a config update event to the frontend for both desktop and web modes
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "config-updated", configData)
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("config-updated", configData)
	}

	slog.Info("Config applied successfully")
	return nil
}

// validateServerConnections validates that all configured servers can be connected to
func (a *App) validateServerConnections(configData *config.ConfigData) error {
	// Skip validation if no servers are configured
	if len(configData.Servers) == 0 {
		return nil
	}

	// Check if all servers have required fields
	validServers := 0
	for _, server := range configData.Servers {
		if server.Host != "" {
			validServers++
		}
	}

	// If no valid servers, skip connection test
	if validServers == 0 {
		return nil
	}

	slog.Info("Validating server connections", "server_count", validServers)

	// Try to create NNTP pool - this will test connections to all servers
	pool, err := configData.GetNNTPPool()
	if err != nil {
		slog.Error("Server connection validation failed", "error", err)
		return fmt.Errorf("failed to connect to one or more servers: %w", err)
	}

	// Close the pool immediately as we were just testing connections
	if pool != nil {
		pool.Quit()
	}

	slog.Info("Server connections validated successfully")
	return nil
}

// SelectConfigFile allows user to select a config file
func (a *App) SelectConfigFile() (string, error) {
	defer a.recoverPanic("SelectConfigFile")

	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select config file",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "YAML files (*.yaml, *.yml)",
				Pattern:     "*.yaml;*.yml",
			},
		},
	})
	if err != nil {
		return "", err
	}

	if file != "" {
		a.configPath = file
		if err := a.loadConfig(); err != nil {
			return "", err
		}
	}

	return file, nil
}

func (a *App) loadConfig() error {
	if a.configPath == "" {
		return fmt.Errorf("no config file specified")
	}

	cfg, err := config.Load(a.configPath)
	if err != nil {
		return err
	}
	a.config = cfg

	// Check if we have any valid servers configured
	validServers := 0
	for _, server := range cfg.Servers {
		if server.Host != "" {
			validServers++
		}
	}

	// Only create postie instance if we have valid servers
	if validServers == 0 {
		slog.Info("No valid servers configured, skipping postie instance creation")
		return nil
	}

	slog.Info("Successfully created postie instance")

	// Re-initialize queue and watcher with new config
	if err := a.initializeQueue(); err != nil {
		slog.Error("Failed to re-initialize queue after config change", "error", err)
	}
	if err := a.initializeProcessor(); err != nil {
		slog.Error("Failed to re-initialize processor after config change", "error", err)
	}
	if err := a.initializeWatcher(); err != nil {
		slog.Error("Failed to re-initialize watcher after config change", "error", err)
	}

	return nil
}

func (a *App) createDefaultConfig() error {
	// Directory creation is handled in GetAppPaths()
	defaultConfig := config.GetDefaultConfig()

	// Set the par2 path to the OS-specific location
	defaultConfig.Par2.Par2Path = a.appPaths.Par2

	// Set the database path to the OS-specific location
	defaultConfig.Queue.DatabasePath = a.appPaths.Database

	if err := config.SaveConfig(&defaultConfig, a.configPath); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	slog.Info("Default config file created", "path", a.configPath)
	return nil
}

// ensurePar2Executable downloads par2 executable if it doesn't exist
func (a *App) ensurePar2Executable(ctx context.Context) {
	defer a.recoverPanic("ensurePar2Executable")

	// Use the OS-specific par2 path
	par2Path := a.appPaths.Par2

	slog.Info("Checking for par2 executable", "path", par2Path)

	// Check if par2 executable already exists
	if _, err := os.Stat(par2Path); err == nil {
		slog.Info("Par2 executable already exists", "path", par2Path)
		return
	}

	slog.Info("Par2 executable not found, downloading...")

	// Emit progress event to frontend for both desktop and web modes
	status := config.Par2DownloadStatus{
		Status:  "downloading",
		Message: "Downloading par2 executable...",
	}
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "par2-download-status", status)
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("par2-download-status", status)
	}

	// Download par2 executable
	execPath, err := parpardownloader.DownloadParParCmd(par2Path)
	if err != nil {
		slog.Error("Failed to download par2 executable", "error", err)
		errorStatus := config.Par2DownloadStatus{
			Status:  "error",
			Message: fmt.Sprintf("Failed to download par2 executable: %v", err),
		}
		if !a.isWebMode {
			runtime.EventsEmit(a.ctx, "par2-download-status", errorStatus)
		} else if a.webEventEmitter != nil {
			a.webEventEmitter("par2-download-status", errorStatus)
		}
		return
	}

	slog.Info("Par2 executable downloaded successfully", "path", execPath)
	completedStatus := config.Par2DownloadStatus{
		Status:  "completed",
		Message: "Par2 executable downloaded successfully",
	}
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "par2-download-status", completedStatus)
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("par2-download-status", completedStatus)
	}
}

// GetWatchDirectory returns the current watch directory
func (a *App) GetWatchDirectory() string {
	if a.config != nil {
		watcherCfg := a.config.GetWatcherConfig()
		if watcherCfg.WatchDirectory != "" {
			return watcherCfg.WatchDirectory
		}
	}
	// Use the OS-specific data directory for watch folder as default
	watchDir := filepath.Join(a.appPaths.Data, "watch")
	return watchDir
}

// GetOutputDirectory returns the current output directory
func (a *App) GetOutputDirectory() string {
	if a.config != nil {
		outputDir := a.config.GetOutputDir()
		if outputDir != "" {
			// If output directory is relative, make it relative to OS-specific data directory
			if !filepath.IsAbs(outputDir) {
				outputDir = filepath.Join(a.appPaths.Data, outputDir)
			}
			return outputDir
		}
	}
	// Default fallback to OS-specific data directory
	return filepath.Join(a.appPaths.Data, "output")
}

// SelectWatchDirectory allows user to select a watch directory
func (a *App) SelectWatchDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select watch directory",
	})
	if err != nil {
		return "", err
	}

	return dir, nil
}

// SelectOutputDirectory allows user to select an output directory
func (a *App) SelectOutputDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select output directory",
	})
	if err != nil {
		return "", err
	}

	return dir, nil
}

// SelectTempDirectory allows user to select a temporary directory for PAR2 files
func (a *App) SelectTempDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select temporary directory for PAR2 files",
	})
	if err != nil {
		return "", err
	}

	return dir, nil
}

// HasPendingConfigChanges returns whether there are pending config changes
func (a *App) HasPendingConfigChanges() bool {
	defer a.recoverPanic("HasPendingConfigChanges")

	a.pendingConfigMux.RLock()
	defer a.pendingConfigMux.RUnlock()
	return a.pendingConfig != nil
}

// GetPendingConfigStatus returns the status of pending config changes
func (a *App) GetPendingConfigStatus() map[string]interface{} {
	defer a.recoverPanic("GetPendingConfigStatus")

	a.pendingConfigMux.RLock()
	defer a.pendingConfigMux.RUnlock()

	status := map[string]interface{}{
		"hasPendingConfig": a.pendingConfig != nil,
		"canApplyNow":      !a.IsUploading(),
	}

	if a.processor != nil {
		runningJobs := a.processor.GetRunningJobs()
		status["canApplyNow"] = status["canApplyNow"].(bool) && len(runningJobs) == 0
	}

	if a.pendingConfig != nil {
		status["message"] = "Configuration changes are pending. They will be applied when uploads finish."
	}

	return status
}

// ApplyPendingConfig manually applies pending configuration changes
func (a *App) ApplyPendingConfig() error {
	defer a.recoverPanic("ApplyPendingConfig")

	a.pendingConfigMux.Lock()
	pendingConfig := a.pendingConfig
	a.pendingConfigMux.Unlock()

	if pendingConfig == nil {
		return fmt.Errorf("no pending configuration changes")
	}

	// Check if we can apply now (no active uploads/jobs)
	hasActiveWork := a.IsUploading()
	if a.processor != nil {
		runningJobs := a.processor.GetRunningJobs()
		hasActiveWork = hasActiveWork || len(runningJobs) > 0
	}

	if hasActiveWork {
		return fmt.Errorf("cannot apply configuration while uploads or jobs are running")
	}

	// Apply the pending configuration
	return a.applyConfigChanges(pendingConfig)
}

// DiscardPendingConfig discards pending configuration changes
func (a *App) DiscardPendingConfig() error {
	defer a.recoverPanic("DiscardPendingConfig")

	a.pendingConfigMux.Lock()
	defer a.pendingConfigMux.Unlock()

	if a.pendingConfig == nil {
		return fmt.Errorf("no pending configuration changes to discard")
	}

	a.pendingConfig = nil

	// Emit event to frontend
	status := map[string]interface{}{
		"hasPendingConfig": false,
		"message":          "Pending configuration changes have been discarded",
	}
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "config-pending", status)
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("config-pending", status)
	}

	slog.Info("Pending configuration changes discarded")
	return nil
}
