package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/pool"
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
		return err // Pass through - validateServerConnections already has descriptive messages
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
		pendingStatus := map[string]any{
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

// poolConfigChanged checks if pool-related configuration has changed
func (a *App) poolConfigChanged(newConfig *config.ConfigData) bool {
	if a.config == nil {
		return true // No existing config, so pool needs to be created
	}

	oldConfig := a.config

	// Compare servers
	if len(oldConfig.Servers) != len(newConfig.Servers) {
		return true
	}

	for i, newServer := range newConfig.Servers {
		oldServer := oldConfig.Servers[i]
		if newServer.Host != oldServer.Host ||
			newServer.Port != oldServer.Port ||
			newServer.Username != oldServer.Username ||
			newServer.Password != oldServer.Password ||
			newServer.SSL != oldServer.SSL ||
			newServer.MaxConnections != oldServer.MaxConnections ||
			newServer.MaxConnectionIdleTimeInSeconds != oldServer.MaxConnectionIdleTimeInSeconds ||
			newServer.MaxConnectionTTLInSeconds != oldServer.MaxConnectionTTLInSeconds ||
			newServer.InsecureSSL != oldServer.InsecureSSL ||
			((newServer.Enabled == nil) != (oldServer.Enabled == nil)) ||
			(newServer.Enabled != nil && oldServer.Enabled != nil && *newServer.Enabled != *oldServer.Enabled) {
			return true
		}
	}

	// Compare connection pool settings
	if oldConfig.ConnectionPool.MinConnections != newConfig.ConnectionPool.MinConnections ||
		oldConfig.ConnectionPool.HealthCheckInterval != newConfig.ConnectionPool.HealthCheckInterval {
		return true
	}

	// Compare posting settings that affect the pool
	if oldConfig.Posting.MaxRetries != newConfig.Posting.MaxRetries ||
		oldConfig.Posting.RetryDelay != newConfig.Posting.RetryDelay {
		return true
	}

	return false
}

// applyConfigChanges applies the configuration changes immediately
func (a *App) applyConfigChanges(configData *config.ConfigData) error {
	if err := config.SaveConfig(configData, a.configPath); err != nil {
		return err
	}

	poolCfgChanged := a.poolConfigChanged(configData)

	// Reload configuration
	if err := a.loadConfig(); err != nil {
		return err
	}

	// Initialize queue if not yet created (e.g. after setup wizard on first start)
	if a.queue == nil {
		if err := a.initializeQueue(); err != nil {
			slog.Error("Failed to initialize queue after config change", "error", err)
		}
	}

	if err := a.initializeProcessor(); err != nil {
		slog.Error("Failed to re-initialize processor after config change", "error", err)
	}

	// Always check watcher configuration
	if err := a.initializeWatcher(); err != nil {
		slog.Error("Failed to re-initialize watcher after config change", "error", err)
	}

	// Update or create the connection pool manager
	if a.poolManager != nil && poolCfgChanged {
		if err := a.poolManager.UpdateConfig(a.config); err != nil {
			slog.Error("Failed to update connection pool manager with new config", "error", err)
			// Don't fail the entire config update for this, but log the error
		} else {
			slog.Info("Connection pool manager updated with new configuration")
		}
	} else if a.poolManager != nil {
		slog.Info("Pool configuration unchanged, skipping pool update")
	} else if a.poolManager == nil && len(a.config.Servers) > 0 {
		// Create pool manager if it doesn't exist but we now have servers configured
		slog.Info("Creating connection pool manager for newly configured servers")
		poolManager, err := pool.New(a.config)
		if err != nil {
			slog.Error("Failed to create connection pool manager", "error", err)
			// Don't fail the entire config update, but log the error
		} else {
			a.poolManager = poolManager
			slog.Info("Connection pool manager created successfully")

			// Re-initialize processor with the new pool manager
			if err := a.initializeProcessor(); err != nil {
				slog.Error("Failed to re-initialize processor with new pool manager", "error", err)
			}
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
	var invalidServers []string
	for i, server := range configData.Servers {
		if server.Host != "" {
			validServers++
		} else {
			invalidServers = append(invalidServers, fmt.Sprintf("Server %d: missing host", i+1))
		}
	}

	// If no valid servers, skip connection test
	if validServers == 0 {
		if len(invalidServers) > 0 {
			slog.Warn("No valid servers found for validation", "issues", invalidServers)
		}
		return nil
	}

	slog.Info("Starting server connection validation",
		"validServers", validServers,
		"totalServers", len(configData.Servers))

	// Create a context with timeout for validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Individual server validation for better error reporting
	var failedServers []string
	var lastError error

	for i, server := range configData.Servers {
		if server.Host == "" {
			continue // Skip invalid servers
		}

		slog.Debug("Validating individual server",
			"index", i,
			"host", server.Host,
			"port", server.Port,
			"ssl", server.SSL)

		// Test individual server connection with timeout
		if err := a.validateIndividualServer(ctx, &server, i+1); err != nil {
			serverDesc := fmt.Sprintf("%s:%d", server.Host, server.Port)
			failedServers = append(failedServers, serverDesc)
			lastError = err
			slog.Error("Server validation failed",
				"server", serverDesc,
				"index", i+1,
				"error", err)
		} else {
			slog.Info("Server validated successfully",
				"server", fmt.Sprintf("%s:%d", server.Host, server.Port),
				"index", i+1)
		}
	}

	// If any servers failed, return detailed error
	if len(failedServers) > 0 {
		errorMsg := fmt.Sprintf("Failed to connect to %d server(s): %s",
			len(failedServers),
			strings.Join(failedServers, ", "))

		if lastError != nil {
			errorMsg = fmt.Sprintf("%s. Last error: %v", errorMsg, lastError)
		}

		slog.Error("Server connection validation completed with failures",
			"failedCount", len(failedServers),
			"successCount", validServers-len(failedServers))

		return fmt.Errorf("%s", errorMsg)
	}

	slog.Info("All server connections validated successfully", "serverCount", validServers)
	return nil
}

// validateIndividualServer tests a single server connection
func (a *App) validateIndividualServer(ctx context.Context, server *config.ServerConfig, serverNum int) error {
	// Convert to ServerData format for testing
	serverData := ServerData{
		Host:           server.Host,
		Port:           server.Port,
		Username:       server.Username,
		Password:       server.Password,
		SSL:            server.SSL,
		MaxConnections: server.MaxConnections,
	}

	// Create a channel to handle the validation result
	resultChan := make(chan ValidationResult, 1)
	errorChan := make(chan error, 1)

	// Run validation in a goroutine with timeout
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("validation panic: %v", r)
			}
		}()

		result := a.TestProviderConnectivity(serverData)
		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		if !result.Valid {
			return fmt.Errorf("server %d (%s:%d): %s", serverNum, server.Host, server.Port, result.Error)
		}
		return nil
	case err := <-errorChan:
		return fmt.Errorf("server %d (%s:%d): %w", serverNum, server.Host, server.Port, err)
	case <-ctx.Done():
		return fmt.Errorf("server %d (%s:%d): connection timeout after 30 seconds", serverNum, server.Host, server.Port)
	}
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

	return nil
}

func (a *App) createDefaultConfig() error {
	// Directory creation is handled in GetAppPaths()
	defaultConfig := config.GetDefaultConfig()

	// Set the database path to the OS-specific location
	defaultConfig.Database.DatabasePath = a.appPaths.Database

	if err := config.SaveConfig(&defaultConfig, a.configPath); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	slog.Info("Default config file created", "path", a.configPath)
	return nil
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
func (a *App) GetPendingConfigStatus() map[string]any {
	defer a.recoverPanic("GetPendingConfigStatus")

	a.pendingConfigMux.RLock()
	defer a.pendingConfigMux.RUnlock()

	status := map[string]any{
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
	status := map[string]any{
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

// canProcessNextItem returns false if there are pending config changes that prevent new items from being processed
func (a *App) canProcessNextItem() bool {
	a.pendingConfigMux.RLock()
	defer a.pendingConfigMux.RUnlock()

	// If there's no pending config, processing can continue
	if a.pendingConfig == nil {
		return true
	}

	// If there are pending configs, check if we can apply them now
	hasActiveWork := a.IsUploading()
	if a.processor != nil {
		runningJobs := a.processor.GetRunningJobs()
		hasActiveWork = hasActiveWork || len(runningJobs) > 0
	}

	// If no active work, try to apply pending config automatically
	if !hasActiveWork {
		slog.Info("No active work detected, attempting to apply pending configuration")

		// Apply pending config in background to avoid blocking the processor
		go func() {
			if err := a.ApplyPendingConfig(); err != nil {
				slog.Error("Failed to auto-apply pending configuration", "error", err)
			}
		}()
	}

	// Don't process new items while pending config exists
	return false
}
