package backend

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	return a.GetLogsPaginated(0, 0) // 0, 0 means get last 1MB like before
}

// GetLogsPaginated returns paginated log content
// limit: number of lines to return (0 = use original 1MB limit)
// offset: number of lines to skip from the end (0 = start from most recent)
func (a *App) GetLogsPaginated(limit, offset int) (string, error) {
	file, err := os.Open(a.appPaths.Log)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close log file", "error", err)
		}
	}()

	if limit == 0 {
		// Original behavior - read last 1MB
		stat, err := file.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to get log file stats: %w", err)
		}

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

	// New paginated behavior
	return a.readLogLines(file, limit, offset)
}

// readLogLines reads lines from the log file with pagination
func (a *App) readLogLines(file *os.File, limit, offset int) (string, error) {
	// For very large files (>10MB), we should optimize this
	// For now, we'll read the entire file for simplicity
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	if len(content) == 0 {
		return "", nil
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Remove empty last line if it exists (common with newline-terminated files)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	totalLines := len(lines)
	if totalLines == 0 {
		return "", nil
	}

	// Validate parameters
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		return "", fmt.Errorf("limit must be positive")
	}

	// Calculate start and end indices
	// We want the most recent lines, so we work backwards from the end
	endIndex := totalLines - offset
	if endIndex <= 0 {
		return "", nil // No lines to return
	}

	startIndex := endIndex - limit
	if startIndex < 0 {
		startIndex = 0
	}

	// Ensure we don't go out of bounds
	if startIndex >= totalLines {
		return "", nil
	}
	if endIndex > totalLines {
		endIndex = totalLines
	}

	// Get the requested lines
	requestedLines := lines[startIndex:endIndex]

	// Join with newlines and return
	return strings.Join(requestedLines, "\n"), nil
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

// HandleDroppedFiles processes files that are dropped onto the application window
func (a *App) HandleDroppedFiles(filePaths []string) error {
	if len(filePaths) == 0 {
		return fmt.Errorf("no files dropped")
	}

	slog.Info("Files dropped onto window", "count", len(filePaths), "files", filePaths)

	// Check if configuration is valid before proceeding
	status := a.GetAppStatus()
	if needsConfig, ok := status["needsConfiguration"].(bool); ok && needsConfig {
		return fmt.Errorf("configuration required: Please configure at least one server in the Settings page before uploading files")
	}

	// If we have a queue, add files to it
	if a.queue != nil {
		addedCount := 0
		for _, filePath := range filePaths {
			// Get file info
			info, err := os.Stat(filePath)
			if err != nil {
				slog.Warn("Could not get file info for dropped file, skipping", "file", filePath, "error", err)
				continue
			}

			// Skip directories for now
			if info.IsDir() {
				slog.Info("Skipping directory", "path", filePath)
				continue
			}

			// Add file to queue
			if err := a.queue.AddFile(context.Background(), filePath, info.Size()); err != nil {
				slog.Warn("Could not add dropped file to queue, skipping", "file", filePath, "error", err)
				continue
			}

			addedCount++
			slog.Info("Dropped file added to queue", "file", filepath.Base(filePath), "size", info.Size())
		}

		if addedCount > 0 {
			slog.Info("Added dropped files to queue", "added", addedCount, "total", len(filePaths))
			// Emit event to refresh queue in frontend
			runtime.EventsEmit(a.ctx, "queue-updated")
		}

		if addedCount == 0 {
			return fmt.Errorf("no valid files could be added to queue")
		}

		return nil
	}

	return fmt.Errorf("queue not initialized")
}
