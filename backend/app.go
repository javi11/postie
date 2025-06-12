package backend

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/processor"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/internal/watcher"
	"github.com/javi11/postie/pkg/postie"
	_ "github.com/mattn/go-sqlite3"
)

// App struct for the Wails application
type App struct {
	ctx          context.Context
	config       *config.ConfigData
	configPath   string
	postie       *postie.Postie
	progress     *ProgressTracker
	progressMux  sync.RWMutex
	uploading    bool
	uploadingMux sync.RWMutex
	uploadCtx    context.Context
	uploadCancel context.CancelFunc
	queue        *queue.Queue
	processor    *processor.Processor
	watcher      *watcher.Watcher
	watchCtx     context.Context
	watchCancel  context.CancelFunc
	procCtx      context.Context
	procCancel   context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Get the directory where the executable is located
	exePath, err := os.Executable()
	var configPath string
	if err != nil {
		// Fallback to current directory if we can't get executable path
		slog.Warn("Could not get executable path, using current directory", "error", err)
		configPath = "./config.yaml"
	} else {
		// Use the same directory as the executable
		exeDir := filepath.Dir(exePath)
		configPath = filepath.Join(exeDir, "config.yaml")
	}

	slog.Info("Using config path", "path", configPath)

	return &App{
		configPath: configPath,
		progress: &ProgressTracker{
			Stage: "Ready",
		},
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Ensure the config directory exists
	configDir := filepath.Dir(a.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		slog.Error("Failed to create config directory", "dir", configDir, "error", err)
	}

	// Load initial configuration
	if err := a.loadConfig(); err != nil {
		slog.Info("Config file not found or invalid, creating default config", "path", a.configPath, "error", err)
		// Continue with empty config - create default
		if err := a.createDefaultConfig(); err != nil {
			slog.Error("Failed to create default config", "error", err)
		} else {
			slog.Info("Default config created successfully", "path", a.configPath)
			// Try to load the default config after creating it
			if err := a.loadConfig(); err != nil {
				slog.Error("Failed to load default config", "error", err)
			} else {
				slog.Info("Default config loaded successfully")
			}
		}
	} else {
		slog.Info("Config loaded successfully", "path", a.configPath)
	}

	// Initialize queue (always available)
	if err := a.initializeQueue(); err != nil {
		slog.Error("Failed to initialize queue", "error", err)
	}

	// Initialize processor if configuration is valid
	if err := a.initializeProcessor(); err != nil {
		slog.Error("Failed to initialize processor", "error", err)
	}

	// Initialize watcher if enabled and configuration is valid
	if err := a.initializeWatcher(); err != nil {
		slog.Error("Failed to initialize watcher", "error", err)
	}

	// Ensure par2 executable is available
	go a.ensurePar2Executable(ctx)
}

// GetAppStatus returns the current application status
func (a *App) GetAppStatus() map[string]interface{} {
	status := map[string]interface{}{
		"hasPostie":           a.postie != nil,
		"hasConfig":           a.config != nil,
		"configPath":          a.configPath,
		"uploading":           a.IsUploading(),
		"criticalConfigError": false, // Default to false
	}

	if a.config != nil {
		configData := a.config
		hasServers := len(configData.Servers) > 0
		status["hasServers"] = hasServers
		status["serverCount"] = len(configData.Servers)

		// Check if all servers have valid configuration (at least host filled)
		validServers := 0
		for _, server := range configData.Servers {
			if server.Host != "" {
				validServers++
			}
		}
		status["validServerCount"] = validServers
		status["configValid"] = hasServers && validServers > 0
		status["needsConfiguration"] = !hasServers || validServers == 0

		// Set criticalConfigError if we have servers configured but postie instance creation failed
		if hasServers && validServers > 0 && a.postie == nil {
			status["criticalConfigError"] = true
		}
	} else {
		status["hasServers"] = false
		status["serverCount"] = 0
		status["validServerCount"] = 0
		status["configValid"] = false
		status["needsConfiguration"] = true
	}

	return status
}

// GetProcessorStatus returns processor status information
func (a *App) GetProcessorStatus() map[string]interface{} {
	status := map[string]interface{}{
		"hasProcessor": a.processor != nil,
		"runningJobs":  0,
	}

	if a.processor != nil {
		runningJobs := a.processor.GetRunningJobs()
		status["runningJobs"] = len(runningJobs)
		status["runningJobIDs"] = getKeys(runningJobs)
	}

	return status
}

// GetRunningJobs returns currently running jobs from the processor
func (a *App) GetRunningJobs() ([]processor.RunningJobItem, error) {
	if a.processor == nil {
		return []processor.RunningJobItem{}, nil
	}

	return a.processor.GetRunningJobItems(), nil
}

// GetRunningJobDetails returns detailed information about currently running jobs
func (a *App) GetRunningJobDetails() ([]*processor.RunningJobDetails, error) {
	if a.processor == nil {
		return []*processor.RunningJobDetails{}, nil
	}

	return a.processor.GetRunningJobDetails(), nil
}

// Helper function to get keys from map
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
