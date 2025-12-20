package backend

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/javi11/nntppool/v2"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/database"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/processor"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/internal/watcher"
	_ "github.com/mattn/go-sqlite3"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ServerData represents the server configuration data from the frontend
type ServerData struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	SSL            bool   `json:"ssl"`
	MaxConnections int    `json:"maxConnections"`
}

// SetupWizardData represents the complete setup wizard data from the frontend
type SetupWizardData struct {
	Servers         []ServerData `json:"servers"`
	OutputDirectory string       `json:"outputDirectory"`
	WatchDirectory  string       `json:"watchDirectory"`
}

// ValidationResult represents the result of server validation
type ValidationResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error"`
}

// AppStatus represents the current application status
type AppStatus struct {
	HasConfig           bool   `json:"hasConfig"`
	ConfigPath          string `json:"configPath"`
	Uploading           bool   `json:"uploading"`
	CriticalConfigError bool   `json:"criticalConfigError"`
	Error               string `json:"error"`
	IsFirstStart        bool   `json:"isFirstStart"`
	HasServers          bool   `json:"hasServers"`
	ServerCount         int    `json:"serverCount"`
	ValidServerCount    int    `json:"validServerCount"`
	ConfigValid         bool   `json:"configValid"`
	NeedsConfiguration  bool   `json:"needsConfiguration"`
}

// ProcessorStatus represents the current processor status
type ProcessorStatus struct {
	HasProcessor  bool     `json:"hasProcessor"`
	RunningJobs   int      `json:"runningJobs"`
	RunningJobIDs []string `json:"runningJobIDs"`
}

// NntpPoolMetrics represents NNTP connection pool metrics
type NntpPoolMetrics struct {
	Timestamp               string                `json:"timestamp"`
	ActiveConnections       int                   `json:"activeConnections"`
	TotalBytesUploaded      int64                 `json:"totalBytesUploaded"`
	TotalBytesDownloaded    int64                 `json:"totalBytesDownloaded"`
	TotalArticlesPosted     int64                 `json:"totalArticlesPosted"`
	TotalArticlesDownloaded int64                 `json:"totalArticlesDownloaded"`
	TotalErrors             int64                 `json:"totalErrors"`
	ProviderErrors          map[string]int64      `json:"providerErrors"`
	Providers               []NntpProviderMetrics `json:"providers"`
}

// NntpProviderMetrics represents metrics for individual NNTP providers
type NntpProviderMetrics struct {
	Host                    string `json:"host"`
	State                   string `json:"state"`
	ActiveConnections       int    `json:"activeConnections"`
	MaxConnections          int    `json:"maxConnections"`
	TotalBytesUploaded      int64  `json:"totalBytesUploaded"`
	TotalBytesDownloaded    int64  `json:"totalBytesDownloaded"`
	TotalArticlesPosted     int64  `json:"totalArticlesPosted"`
	TotalArticlesDownloaded int64  `json:"totalArticlesDownloaded"`
	TotalErrors             int64  `json:"totalErrors"`
}

// App struct for the Wails application
type App struct {
	ctx                  context.Context
	config               *config.ConfigData
	configPath           string
	appPaths             *AppPaths
	database             *database.Database
	poolManager          *pool.Manager
	queue                *queue.Queue
	processor            *processor.Processor
	watcher              *watcher.Watcher
	watchCtx             context.Context
	watchCancel          context.CancelFunc
	procCtx              context.Context
	procCancel           context.CancelFunc
	criticalErrorMessage string
	loggingError         string // Error message if file logging setup failed
	actualLogPath        string // The actual log path being used (may differ from appPaths.Log if fallback was used)
	isWebMode            bool
	webEventEmitter      func(eventType string, data interface{})
	firstStart           bool
	pendingConfig        *config.ConfigData
	pendingConfigMux     sync.RWMutex
}

// getCrashLogPath returns the path for crash logs
// It tries to use the app's data directory, falling back to temp directory or current directory
func getCrashLogPath(appPaths *AppPaths) string {
	// Try to use the app's data directory first
	if appPaths != nil && appPaths.Data != "" {
		crashPath := filepath.Join(appPaths.Data, "postie_crash.log")
		// Verify the directory is writable
		if _, err := verifyLogDirectory(crashPath); err == nil {
			return crashPath
		}
	}

	// Try temp directory as fallback
	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, "postie", "postie_crash.log")
	if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err == nil {
		return tempPath
	}

	// Last resort: current directory
	return "postie_crash.log"
}

// recoverPanic handles panic recovery with logging
func (a *App) recoverPanic(methodName string) {
	if r := recover(); r != nil {
		stack := debug.Stack()
		slog.Error("Panic recovered in app method",
			"method", methodName,
			"panic", r,
			"stack", string(stack))

		// Set critical error message if we don't have one already
		if a.criticalErrorMessage == "" {
			a.criticalErrorMessage = fmt.Sprintf("Critical error in %s: %v", methodName, r)
		}

		// Write to crash log file for debugging, especially useful on Windows
		crashLogPath := getCrashLogPath(a.appPaths)
		if crashFile, err := os.OpenFile(crashLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = fmt.Fprintf(crashFile, "=== POSTIE BACKEND PANIC ===\n")
			_, _ = fmt.Fprintf(crashFile, "Time: %s\n", time.Now().Format(time.RFC3339))
			_, _ = fmt.Fprintf(crashFile, "Method: %s\n", methodName)
			_, _ = fmt.Fprintf(crashFile, "OS: %s\n", runtime.GOOS)
			_, _ = fmt.Fprintf(crashFile, "Arch: %s\n", runtime.GOARCH)
			_, _ = fmt.Fprintf(crashFile, "Go Version: %s\n", runtime.Version())
			_, _ = fmt.Fprintf(crashFile, "Panic: %v\n\n", r)
			_, _ = fmt.Fprintf(crashFile, "Stack trace:\n%s\n", string(stack))
			_, _ = fmt.Fprintf(crashFile, "=== END PANIC REPORT ===\n\n")
			_ = crashFile.Close()
		}
	}
}

// verifyLogDirectory checks if the log directory is writable
// Returns the verified log path, or an alternative path if the original fails
func verifyLogDirectory(logPath string) (string, error) {
	logDir := filepath.Dir(logPath)

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Test write permissions by creating a temporary file
	testFile := filepath.Join(logDir, ".postie_write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return "", fmt.Errorf("log directory %s is not writable: %w", logDir, err)
	}

	// Write a small amount of data to verify actual write capability
	if _, err := f.WriteString("test"); err != nil {
		_ = f.Close()
		_ = os.Remove(testFile)
		return "", fmt.Errorf("failed to write to log directory %s: %w", logDir, err)
	}

	_ = f.Close()
	_ = os.Remove(testFile)

	return logPath, nil
}

// getFallbackLogPath returns a fallback log path using the system temp directory
func getFallbackLogPath() string {
	tempDir := os.TempDir()
	return filepath.Join(tempDir, "postie", "postie.log")
}

// setupLogging configures logging with Windows-specific optimizations
// Returns the actual log path being used (may differ from input if fallback was needed) and any error
func setupLogging(logPath string) (string, error) {
	// Verify the log directory is writable
	verifiedPath, err := verifyLogDirectory(logPath)
	if err != nil {
		// Try fallback to temp directory
		slog.Warn("Primary log directory not writable, trying fallback",
			"originalPath", logPath,
			"error", err)

		fallbackPath := getFallbackLogPath()
		verifiedPath, err = verifyLogDirectory(fallbackPath)
		if err != nil {
			return "", fmt.Errorf("neither primary nor fallback log directory is writable: primary=%s, fallback=%s: %w",
				logPath, fallbackPath, err)
		}

		slog.Info("Using fallback log path", "path", verifiedPath)
	}

	// Configure lumberjack with Windows-optimized settings
	// Disable compression on Windows to avoid file locking issues during rotation
	compress := runtime.GOOS != "windows"

	logFile := &lumberjack.Logger{
		Filename:   verifiedPath,
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   compress,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Logging initialized successfully",
		"logPath", verifiedPath,
		"os", runtime.GOOS,
		"compression", compress)

	return verifiedPath, nil
}

// NewApp creates a new App application struct
func NewApp() *App {
	var loggingError string
	var actualLogPath string

	// Get OS-specific paths
	appPaths, err := GetAppPaths()
	if err != nil {
		// Fallback to current directory if we can't get OS-specific paths
		slog.Warn("Could not get OS-specific paths, using current directory", "error", err)
		appPaths = &AppPaths{
			Config:   "./config.yaml",
			Database: "./postie.db",
			Par2:     "./parpar",
			Data:     ".",
			Log:      "./postie.log",
		}
	}

	// Setup logging with Windows-specific optimizations
	actualLogPath, err = setupLogging(appPaths.Log)
	if err != nil {
		// Fallback to basic stdout logging if file logging fails
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
		loggingError = fmt.Sprintf("File logging unavailable: %v", err)
		slog.Error("Failed to setup file logging, using stdout only", "error", err)
	}

	slog.Info("Using OS-specific paths",
		"config", appPaths.Config,
		"database", appPaths.Database,
		"par2", appPaths.Par2,
		"data", appPaths.Data,
		"log", appPaths.Log,
		"actualLogPath", actualLogPath)

	return &App{
		configPath:    appPaths.Config,
		appPaths:      appPaths,
		loggingError:  loggingError,
		actualLogPath: actualLogPath,
		isWebMode:     false,
	}
}

// SetWebMode sets the application to web mode
func (a *App) SetWebMode(isWeb bool) {
	slog.Info("Setting web mode", "isWeb", isWeb)
	a.isWebMode = isWeb
}

// SetWebEventEmitter sets the event emitter function for web mode
func (a *App) SetWebEventEmitter(emitter func(eventType string, data interface{})) {
	a.webEventEmitter = emitter
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	defer a.recoverPanic("Startup")

	a.ctx = ctx

	// Note: Directory creation is now handled in GetAppPaths()
	slog.Info("Application starting with OS-specific paths",
		"config", a.appPaths.Config,
		"database", a.appPaths.Database,
		"par2", a.appPaths.Par2)

	// Check if it's the first start BEFORE creating any config
	a.firstStart = a.determineFirstStart()
	slog.Info("First start determination", "isFirstStart", a.firstStart)

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

	// Initialize connection pool manager if we have a valid configuration
	if a.config != nil {
		poolManager, err := pool.New(a.config)
		if err != nil {
			slog.Error("Failed to create connection pool manager", "error", err)
			a.criticalErrorMessage = fmt.Sprintf("Failed to create connection pool: %v", err)
		} else {
			a.poolManager = poolManager
			slog.Info("Connection pool manager created successfully")
		}
	}

	// Initialize queue (always available)
	if err := a.initializeQueue(); err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize queue: %v", err))
	}

	// Initialize processor if configuration is valid
	if err := a.initializeProcessor(); err != nil {
		a.criticalErrorMessage = err.Error()
		slog.Error(fmt.Sprintf("Failed to initialize processor: %v", err))
	}

	// Initialize watcher if enabled and configuration is valid
	if err := a.initializeWatcher(); err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize watcher: %v", err))
	}

	// Ensure par2 executable is available
	go a.ensurePar2Executable(ctx)
}

// Shutdown gracefully shuts down the application and closes all resources
func (a *App) Shutdown() {
	defer a.recoverPanic("Shutdown")

	slog.Info("Application shutdown initiated")

	// Stop watcher if running
	if a.watcher != nil {
		slog.Info("Stopping watcher")
		_ = a.watcher.Close()
	}

	// Stop processor if running
	if a.processor != nil {
		slog.Info("Stopping processor")
		_ = a.processor.Close()
	}

	// Close the connection pool manager if it exists
	if a.poolManager != nil {
		slog.Info("Closing connection pool manager")
		if err := a.poolManager.Close(); err != nil {
			slog.Error("Failed to close connection pool manager", "error", err)
		}
	}

	slog.Info("Application shutdown completed")
}

// GetAppStatus returns the current application status
func (a *App) GetAppStatus() AppStatus {
	defer a.recoverPanic("GetAppStatus")

	status := AppStatus{
		HasConfig:           a.config != nil,
		ConfigPath:          a.configPath,
		Uploading:           a.IsUploading(),
		CriticalConfigError: false, // Default to false
		Error:               "",
		IsFirstStart:        a.isFirstStart(),
	}

	if a.config != nil {
		configData := a.config
		hasServers := len(configData.Servers) > 0
		status.HasServers = hasServers
		status.ServerCount = len(configData.Servers)

		// Check if all servers have valid configuration (at least host filled)
		validServers := 0
		for _, server := range configData.Servers {
			if server.Host != "" {
				validServers++
			}
		}
		status.ValidServerCount = validServers
		status.ConfigValid = hasServers && validServers > 0
		status.NeedsConfiguration = !hasServers || validServers == 0
	} else {
		status.HasServers = false
		status.ServerCount = 0
		status.ValidServerCount = 0
		status.ConfigValid = false
		status.NeedsConfiguration = true
	}

	slog.Debug("Current application status", "status", status)

	return status
}

// GetLoggingStatus returns information about logging configuration
func (a *App) GetLoggingStatus() map[string]interface{} {
	// Determine the actual log path being used
	logPath := a.actualLogPath
	if logPath == "" {
		logPath = a.appPaths.Log
	}

	status := map[string]interface{}{
		"configuredLogPath": a.appPaths.Log,            // The originally configured path
		"actualLogPath":     logPath,                   // The actual path being used (may be fallback)
		"usingFallback":     logPath != a.appPaths.Log, // True if using a fallback path
		"os":                runtime.GOOS,
		"canWrite":          false,
		"fileExists":        false,
		"fileLoggingActive": a.loggingError == "", // True if file logging is working
		"error":             a.loggingError,       // Any error from logging setup
	}

	// Check if log file exists
	if _, err := os.Stat(logPath); err == nil {
		status["fileExists"] = true

		// Get file info for additional details
		if info, err := os.Stat(logPath); err == nil {
			status["fileSize"] = info.Size()
			status["lastModified"] = info.ModTime().Format(time.RFC3339)
		}
	}

	// Test current write permissions
	logDir := filepath.Dir(logPath)
	testFile := filepath.Join(logDir, ".write_test")
	if f, err := os.Create(testFile); err != nil {
		if status["error"] == "" {
			status["error"] = fmt.Sprintf("Cannot write to log directory: %v", err)
		}
	} else {
		_ = f.Close()
		_ = os.Remove(testFile)
		status["canWrite"] = true
	}

	return status
}

// GetProcessorStatus returns processor status information
func (a *App) GetProcessorStatus() ProcessorStatus {
	status := ProcessorStatus{
		HasProcessor:  a.processor != nil,
		RunningJobs:   0,
		RunningJobIDs: []string{},
	}

	if a.processor != nil {
		runningJobs := a.processor.GetRunningJobs()
		status.RunningJobs = len(runningJobs)
		status.RunningJobIDs = getKeys(runningJobs)
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

// GetRunningJobsDetails returns detailed information about currently running jobs
func (a *App) GetRunningJobsDetails() ([]processor.RunningJobDetails, error) {
	if a.processor == nil {
		return []processor.RunningJobDetails{}, nil
	}

	res := make([]processor.RunningJobDetails, 0, len(a.processor.GetRunningJobDetails()))
	p := a.processor.GetRunningJobDetails()

	for _, job := range p {
		res = append(res, job)
	}

	return res, nil
}

// RetryJob retries a failed job
func (a *App) RetryJob(id string) error {
	defer a.recoverPanic("RetryJob")

	if a.queue == nil {
		return nil
	}
	err := a.queue.RetryErroredJob(a.ctx, id)
	if err != nil {
		return err
	}

	// Emit events for both desktop and web modes
	if !a.isWebMode {
		wailsruntime.EventsEmit(a.ctx, "queue:updated")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("queue:updated", nil)
	}
	return nil
}

// RetryScript manually retries a failed post-upload script execution
func (a *App) RetryScript(id string) error {
	defer a.recoverPanic("RetryScript")

	if a.queue == nil {
		return fmt.Errorf("queue is not available")
	}

	// Reset the script status to pending_retry with immediate retry
	// Pass nil for firstFailureAt to preserve existing value (if any) or set it on first failure
	nextRetry := time.Now()
	if err := a.queue.UpdateScriptStatus(a.ctx, id, "pending_retry", 0, "", &nextRetry, nil); err != nil {
		return fmt.Errorf("failed to reset script status: %w", err)
	}

	// Emit events for both desktop and web modes
	if !a.isWebMode {
		wailsruntime.EventsEmit(a.ctx, "queue:updated")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("queue:updated", nil)
	}

	return nil
}

// GetLogs returns the content of the log file.
func (a *App) GetLogs() (string, error) {
	defer a.recoverPanic("GetLogs")

	return a.GetLogsPaginated(0, 0) // 0, 0 means get last 1MB like before
}

// GetLogsPaginated returns paginated log content
// limit: number of lines to return (0 = use original 1MB limit)
// offset: number of lines to skip from the end (0 = start from most recent)
func (a *App) GetLogsPaginated(limit, offset int) (string, error) {
	defer a.recoverPanic("GetLogsPaginated")

	// Use actual log path if available (may differ from configured path if using fallback)
	logPath := a.actualLogPath
	if logPath == "" {
		logPath = a.appPaths.Log
	}

	file, err := os.Open(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to open log file at %s: %w", logPath, err)
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
	return readLogLines(file, limit, offset)
}

// NavigateToSettings emits an event to navigate to the settings page
func (a *App) NavigateToSettings() {
	if a.ctx != nil && !a.isWebMode {
		wailsruntime.EventsEmit(a.ctx, "navigate-to-settings")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("navigate-to-settings", nil)
	}
}

// NavigateToDashboard emits an event to navigate to the dashboard page
func (a *App) NavigateToDashboard() {
	if a.ctx != nil && !a.isWebMode {
		wailsruntime.EventsEmit(a.ctx, "navigate-to-dashboard")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("navigate-to-dashboard", nil)
	}
}

// HandleDroppedFiles processes files that are dropped onto the application window
func (a *App) HandleDroppedFiles(filePaths []string) error {
	defer a.recoverPanic("HandleDroppedFiles")

	if len(filePaths) == 0 {
		return fmt.Errorf("no files dropped")
	}

	slog.Info("Files dropped onto window", "count", len(filePaths), "files", filePaths)

	// Check if configuration is valid before proceeding
	status := a.GetAppStatus()
	if status.NeedsConfiguration {
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

			// Handle directories by processing them as single NZB units
			if info.IsDir() {
				slog.Info("Processing dropped directory", "path", filePath)

				// Process directory recursively to collect files
				filesByFolder, sizeByFolder, err := processDirectoryRecursively(filePath)
				if err != nil {
					slog.Error("Error processing directory", "path", filePath, "error", err)
					continue
				}

				// Add each folder as a single queue entry (following watcher pattern)
				for folderPath, files := range filesByFolder {
					if len(files) == 0 {
						continue
					}

					folderSize := sizeByFolder[folderPath]
					folderName := filepath.Base(folderPath)

					slog.Info("Adding folder to queue", "folder", folderName, "files", len(files), "size", folderSize)

					// Add the folder to the queue with FOLDER: prefix to indicate it's a folder
					folderQueuePath := "FOLDER:" + folderPath
					if err := a.queue.AddFile(context.Background(), folderQueuePath, folderSize); err != nil {
						slog.Warn("Could not add folder to queue, skipping", "folder", folderPath, "error", err)
						continue
					}

					addedCount++
					slog.Info("Dropped folder added to queue", "folder", folderName, "files", len(files), "size", folderSize)

					// Log the files in the folder for debugging
					for _, file := range files {
						slog.Debug("File in dropped folder", "folder", folderName, "file", filepath.Base(file))
					}
				}

				continue
			}

			// Handle individual files (existing logic)
			if err := a.queue.AddFile(context.Background(), filePath, info.Size()); err != nil {
				slog.Warn("Could not add dropped file to queue, skipping", "file", filePath, "error", err)
				continue
			}

			addedCount++
			slog.Info("Dropped file added to queue", "file", filepath.Base(filePath), "size", info.Size())
		}

		if addedCount > 0 {
			slog.Info("Added dropped items to queue", "added", addedCount, "total", len(filePaths))
			// Emit event to refresh queue in frontend for both desktop and web modes
			if !a.isWebMode {
				wailsruntime.EventsEmit(a.ctx, "queue-updated")
			} else if a.webEventEmitter != nil {
				a.webEventEmitter("queue-updated", nil)
			}
		}

		if addedCount == 0 {
			return fmt.Errorf("no valid files or folders could be added to queue")
		}

		return nil
	}

	return fmt.Errorf("queue not initialized")
}

// SetupWizardComplete saves the configuration from the setup wizard
func (a *App) SetupWizardComplete(wizardData SetupWizardData) error {
	defer a.recoverPanic("SetupWizardComplete")

	slog.Info("Starting setup wizard completion",
		"serverCount", len(wizardData.Servers),
		"hasOutputDir", wizardData.OutputDirectory != "")

	// Validate input data
	if len(wizardData.Servers) == 0 {
		slog.Error("Setup wizard failed: no servers provided")
		return fmt.Errorf("at least one server must be configured")
	}

	if wizardData.OutputDirectory == "" {
		slog.Error("Setup wizard failed: no output directory provided")
		return fmt.Errorf("output directory must be specified")
	}

	// Validate all servers have required fields
	for i, serverData := range wizardData.Servers {
		if serverData.Host == "" {
			slog.Error("Setup wizard failed: server missing host", "serverIndex", i)
			return fmt.Errorf("server %d: host is required", i+1)
		}
		if serverData.Port <= 0 || serverData.Port > 65535 {
			slog.Error("Setup wizard failed: invalid server port", "serverIndex", i, "port", serverData.Port)
			return fmt.Errorf("server %d: port must be between 1 and 65535", i+1)
		}
		if serverData.MaxConnections <= 0 {
			slog.Warn("Server has invalid max connections, setting to default", "serverIndex", i, "maxConnections", serverData.MaxConnections)
			serverData.MaxConnections = 5 // Set reasonable default
		}
	}

	// Create new config based on wizard data
	cfg := config.GetDefaultConfig()

	// Ensure version is set
	cfg.Version = config.CurrentConfigVersion

	// Set servers from wizard
	cfg.Servers = make([]config.ServerConfig, len(wizardData.Servers))
	for i, serverData := range wizardData.Servers {
		enabled := true
		server := config.ServerConfig{
			Host:           serverData.Host,
			Port:           serverData.Port,
			Username:       serverData.Username,
			Password:       serverData.Password,
			SSL:            serverData.SSL,
			MaxConnections: serverData.MaxConnections,
			Enabled:        &enabled,
		}
		cfg.Servers[i] = server
		slog.Debug("Configured server", "index", i, "host", serverData.Host, "port", serverData.Port, "ssl", serverData.SSL)
	}

	// Set output directory
	cfg.OutputDir = wizardData.OutputDirectory
	slog.Debug("Set output directory", "path", wizardData.OutputDirectory)

	// Set the par2 path to the OS-specific location
	cfg.Par2.Par2Path = a.appPaths.Par2

	// Set the database path to the OS-specific location
	cfg.Database.DatabasePath = a.appPaths.Database

	// Save configuration with enhanced error context
	slog.Info("Saving setup wizard configuration", "configPath", a.configPath)
	if err := a.SaveConfig(&cfg); err != nil {
		slog.Error("Failed to save setup wizard configuration", "error", err, "configPath", a.configPath)
		return a.wrapSaveConfigError(err)
	}

	// Mark as no longer first start since setup is complete
	a.firstStart = false

	slog.Info("Setup wizard completed successfully", "configPath", a.configPath)
	return nil
}

// ValidateNNTPServer validates an NNTP server configuration using TestProviderConnectivity
func (a *App) ValidateNNTPServer(serverData ServerData) ValidationResult {
	defer a.recoverPanic("ValidateNNTPServer")

	// Use the new TestProviderConnectivity method for more efficient validation
	return a.TestProviderConnectivity(serverData)
}

// TestProviderConnectivity tests an individual provider's connectivity using the new nntppool method
func (a *App) TestProviderConnectivity(serverData ServerData) ValidationResult {
	defer a.recoverPanic("TestProviderConnectivity")

	// Basic validation
	if serverData.Host == "" {
		return ValidationResult{
			Valid: false,
			Error: "Host is required",
		}
	}
	if serverData.Port <= 0 || serverData.Port > 65535 {
		return ValidationResult{
			Valid: false,
			Error: "Port must be between 1 and 65535",
		}
	}

	// Convert to nntppool provider config
	providerConfig := nntppool.UsenetProviderConfig{
		Host:                           serverData.Host,
		Port:                           serverData.Port,
		Username:                       serverData.Username,
		Password:                       serverData.Password,
		TLS:                            serverData.SSL,
		MaxConnections:                 1, // Use single connection for testing
		MaxConnectionIdleTimeInSeconds: 300,
		MaxConnectionTTLInSeconds:      3600,
		InsecureSSL:                    false,
	}

	// Use the new TestProviderConnectivity method from nntppool
	// Note: The exact parameters may need adjustment based on the actual nntppool API
	ctx := context.Background()
	err := nntppool.TestProviderConnectivity(ctx, providerConfig, nil, nil)
	if err != nil {
		slog.Warn("Provider connectivity test failed", "host", serverData.Host, "port", serverData.Port, "error", err)
		return ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Connection test failed: %v", err),
		}
	}

	slog.Info("Provider connectivity test successful", "host", serverData.Host, "port", serverData.Port)
	return ValidationResult{
		Valid: true,
		Error: "",
	}
}

// PauseProcessing pauses the processor to prevent new jobs from starting
func (a *App) PauseProcessing() error {
	defer a.recoverPanic("PauseProcessing")

	if a.processor == nil {
		return fmt.Errorf("processor not initialized")
	}

	a.processor.PauseProcessing()
	slog.Info("Processing paused via API")

	// Emit events for both desktop and web modes
	if !a.isWebMode {
		wailsruntime.EventsEmit(a.ctx, "processing:paused")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("processing:paused", nil)
	}

	return nil
}

// ResumeProcessing resumes the processor to allow new jobs to start
func (a *App) ResumeProcessing() error {
	defer a.recoverPanic("ResumeProcessing")

	if a.processor == nil {
		return fmt.Errorf("processor not initialized")
	}

	a.processor.ResumeProcessing()
	slog.Info("Processing resumed via API")

	// Emit events for both desktop and web modes
	if !a.isWebMode {
		wailsruntime.EventsEmit(a.ctx, "processing:resumed")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("processing:resumed", nil)
	}

	return nil
}

// IsProcessingPaused returns whether the processor is currently paused
func (a *App) IsProcessingPaused() bool {
	defer a.recoverPanic("IsProcessingPaused")

	if a.processor == nil {
		return false
	}

	return a.processor.IsPaused()
}

// IsProcessingAutoPaused returns whether the processor was automatically paused due to provider unavailability
func (a *App) IsProcessingAutoPaused() bool {
	defer a.recoverPanic("IsProcessingAutoPaused")

	if a.processor == nil {
		return false
	}

	return a.processor.IsAutoPaused()
}

// GetAutoPauseReason returns the reason for automatic pause, if any
func (a *App) GetAutoPauseReason() string {
	defer a.recoverPanic("GetAutoPauseReason")

	if a.processor == nil {
		return ""
	}

	return a.processor.GetAutoPauseReason()
}

// GetNntpPoolMetrics returns NNTP connection pool metrics from the singleton pool manager
func (a *App) GetNntpPoolMetrics() (NntpPoolMetrics, error) {
	defer a.recoverPanic("GetNntpPoolMetrics")

	// Default empty metrics if no pool available
	emptyMetrics := NntpPoolMetrics{
		Timestamp:               time.Now().Format(time.RFC3339),
		ActiveConnections:       0,
		TotalBytesUploaded:      0,
		TotalBytesDownloaded:    0,
		TotalArticlesPosted:     0,
		TotalArticlesDownloaded: 0,
		TotalErrors:             0,
		ProviderErrors:          make(map[string]int64),
		Providers:               []NntpProviderMetrics{},
	}

	// Get metrics from the connection pool manager
	if a.poolManager == nil {
		slog.Warn("Connection pool manager not available for metrics")
		return emptyMetrics, nil
	}

	// Get metrics from the pool manager
	snapshot, err := a.poolManager.GetMetrics()
	if err != nil {
		slog.Error("Failed to get metrics from pool manager", "error", err)
		return emptyMetrics, fmt.Errorf("failed to get pool metrics: %w", err)
	}

	// Sum active connections from all providers
	activeConnections := 0
	for _, provider := range snapshot.ProviderMetrics {
		activeConnections += provider.ActiveConnections
	}

	// Convert pool metrics to our metrics structure
	metrics := NntpPoolMetrics{
		Timestamp:               snapshot.Timestamp.Format(time.RFC3339),
		ActiveConnections:       activeConnections,
		TotalBytesUploaded:      snapshot.BytesUploaded,
		TotalBytesDownloaded:    snapshot.BytesDownloaded,
		TotalArticlesPosted:     snapshot.ArticlesPosted,
		TotalArticlesDownloaded: snapshot.ArticlesDownloaded,
		TotalErrors:             snapshot.TotalErrors,
		ProviderErrors:          snapshot.ProviderErrors,
	}

	// Convert provider metrics from map to array
	providers := make([]NntpProviderMetrics, 0, len(snapshot.ProviderMetrics))
	for _, provider := range snapshot.ProviderMetrics {
		providers = append(providers, NntpProviderMetrics{
			Host:                    provider.Host,
			State:                   provider.State,
			ActiveConnections:       provider.ActiveConnections,
			MaxConnections:          provider.MaxConnections,
			TotalBytesUploaded:      provider.BytesUploaded,
			TotalBytesDownloaded:    provider.BytesDownloaded,
			TotalArticlesPosted:     provider.ArticlesPosted,
			TotalArticlesDownloaded: provider.ArticlesDownloaded,
			TotalErrors:             provider.TotalErrors,
		})
	}
	metrics.Providers = providers

	return metrics, nil
}

// determineFirstStart checks if this is the first time the application is being run
// This must be called BEFORE any config creation
func (a *App) determineFirstStart() bool {
	// Check if config file exists
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		slog.Info("Config file does not exist, treating as first start", "configPath", a.configPath)
		return true
	}

	// If config file exists, try to load it to check if it has meaningful content
	_, err := config.Load(a.configPath)
	if err != nil {
		slog.Info("Config file exists but cannot be loaded, treating as first start", "error", err)
		return true
	}

	return false
}

// isFirstStart returns whether this is the first time the application is being run
func (a *App) isFirstStart() bool {
	return a.firstStart
}
