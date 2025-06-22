package backend

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/processor"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/internal/watcher"
	"github.com/javi11/postie/pkg/postie"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/natefinch/lumberjack.v2"
)

// App struct for the Wails application
type App struct {
	ctx                  context.Context
	config               *config.ConfigData
	configPath           string
	appPaths             *AppPaths
	postie               *postie.Postie
	progress             *ProgressTracker
	progressMux          sync.RWMutex
	uploading            bool
	uploadingMux         sync.RWMutex
	uploadCtx            context.Context
	uploadCancel         context.CancelFunc
	queue                *queue.Queue
	processor            *processor.Processor
	watcher              *watcher.Watcher
	watchCtx             context.Context
	watchCancel          context.CancelFunc
	procCtx              context.Context
	procCancel           context.CancelFunc
	criticalErrorMessage string
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Get OS-specific paths
	appPaths, err := GetAppPaths()
	if err != nil {
		// Fallback to current directory if we can't get OS-specific paths
		slog.Warn("Could not get OS-specific paths, using current directory", "error", err)
		appPaths = &AppPaths{
			Config:   "./config.yaml",
			Database: "./postie_queue.db",
			Par2:     "./parpar",
			Data:     ".",
			Log:      "./postie.log",
		}
	}

	// Setup logging
	logFile := &lumberjack.Logger{
		Filename:   appPaths.Log,
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     28, //days
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Using OS-specific paths",
		"config", appPaths.Config,
		"database", appPaths.Database,
		"par2", appPaths.Par2,
		"data", appPaths.Data,
		"log", appPaths.Log)

	return &App{
		configPath: appPaths.Config,
		appPaths:   appPaths,
		progress: &ProgressTracker{
			Stage: "Ready",
		},
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Note: Directory creation is now handled in GetAppPaths()
	slog.Info("Application starting with OS-specific paths",
		"config", a.appPaths.Config,
		"database", a.appPaths.Database,
		"par2", a.appPaths.Par2)

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
		a.criticalErrorMessage = err.Error()
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
		"error":               "",
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
			status["error"] = a.criticalErrorMessage
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

// RetryJob retries a failed job
func (a *App) RetryJob(id string) error {
	if a.queue == nil {
		return nil
	}
	err := a.queue.RetryErroredJob(a.ctx, id)
	if err != nil {
		return err
	}

	runtime.EventsEmit(a.ctx, "queue:updated")
	return nil
}

// Helper function to get keys from map
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetLogs returns the content of the log file.
func (a *App) GetLogs() (string, error) {
	file, err := os.Open(a.appPaths.Log)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get log file stats: %w", err)
	}

	// Read last 1MB of the file
	const maxLogSize = 1024 * 1024
	start := stat.Size() - maxLogSize
	if start < 0 {
		start = 0
	}

	buffer := make([]byte, stat.Size()-start)
	_, err = file.ReadAt(buffer, start)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	return string(buffer), nil
}

// NavigateToSettings emits an event to navigate to the settings page
func (a *App) NavigateToSettings() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "navigate-to-settings")
	}
}

// NavigateToDashboard emits an event to navigate to the dashboard page
func (a *App) NavigateToDashboard() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "navigate-to-dashboard")
	}
}
