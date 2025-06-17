package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/parpardownloader"
	"github.com/javi11/postie/pkg/postie"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetConfigPath returns the path to the configuration file
func (a *App) GetConfigPath() string {
	return a.configPath
}

// GetConfig returns the current configuration
func (a *App) GetConfig() (*config.ConfigData, error) {
	if a.configPath == "" {
		return nil, fmt.Errorf("no config file specified")
	}

	// Check if file exists
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found")
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

// SaveConfig saves the configuration
func (a *App) SaveConfig(configData *config.ConfigData) error {
	slog.Info("Saving config", "path", a.configPath, "configData", configData)

	// Validate server connections before saving
	if err := a.validateServerConnections(configData); err != nil {
		return fmt.Errorf("server validation failed: %w", err)
	}

	if err := config.SaveConfig(configData, a.configPath); err != nil {
		return err
	}

	// Reload configuration
	if err := a.loadConfig(); err != nil {
		return err
	}

	// Emit a config update event to the frontend
	runtime.EventsEmit(a.ctx, "config-updated", configData)

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
		a.postie = nil
		return nil
	}

	if a.postie != nil {
		a.postie.Close()
		a.postie = nil
	}

	// Create postie instance
	a.postie, err = postie.New(context.Background(), cfg)
	if err != nil {
		slog.Error("Failed to create postie instance", "error", err)
		// Don't return error here - allow app to continue without postie
		// The user can configure servers later and restart
		a.postie = nil
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
	// Use the OS-specific par2 path
	par2Path := a.appPaths.Par2

	slog.Info("Checking for par2 executable", "path", par2Path)

	// Check if par2 executable already exists
	if _, err := os.Stat(par2Path); err == nil {
		slog.Info("Par2 executable already exists", "path", par2Path)
		return
	}

	slog.Info("Par2 executable not found, downloading...")

	// Emit progress event to frontend
	runtime.EventsEmit(a.ctx, "par2-download-status", map[string]interface{}{
		"status":  "downloading",
		"message": "Downloading par2 executable...",
	})

	// Download par2 executable
	execPath, err := parpardownloader.DownloadParParCmd(par2Path)
	if err != nil {
		slog.Error("Failed to download par2 executable", "error", err)
		runtime.EventsEmit(a.ctx, "par2-download-status", map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Failed to download par2 executable: %v", err),
		})
		return
	}

	slog.Info("Par2 executable downloaded successfully", "path", execPath)
	runtime.EventsEmit(a.ctx, "par2-download-status", map[string]interface{}{
		"status":  "completed",
		"message": "Par2 executable downloaded successfully",
	})
}

// GetWatchDirectory returns the current watch directory
func (a *App) GetWatchDirectory() string {
	// Use the OS-specific data directory for watch folder
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
