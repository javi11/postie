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
	"strings"
	"sync"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/processor"
	"github.com/javi11/postie/internal/queue"
	"github.com/javi11/postie/internal/watcher"
	"github.com/javi11/postie/pkg/postie"
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
	HasPostie           bool   `json:"hasPostie"`
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

// App struct for the Wails application
type App struct {
	ctx                  context.Context
	config               *config.ConfigData
	configPath           string
	appPaths             *AppPaths
	postie               *postie.Postie
	queue                *queue.Queue
	processor            *processor.Processor
	watcher              *watcher.Watcher
	watchCtx             context.Context
	watchCancel          context.CancelFunc
	procCtx              context.Context
	procCancel           context.CancelFunc
	criticalErrorMessage string
	isWebMode            bool
	webEventEmitter      func(eventType string, data interface{})
	firstStart           bool
	pendingConfig        *config.ConfigData
	pendingConfigMux     sync.RWMutex
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
		if crashFile, err := os.OpenFile("postie_backend_crash.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = fmt.Fprintf(crashFile, "=== POSTIE BACKEND PANIC ===\n")
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

// setupLogging configures logging with Windows-specific optimizations
func setupLogging(logPath string) error {
	// Ensure log directory exists with proper permissions
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Configure lumberjack with Windows-optimized settings
	logFile := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Logging initialized successfully",
		"logPath", logPath,
		"os", runtime.GOOS)

	return nil
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

	// Setup logging with Windows-specific optimizations
	if err := setupLogging(appPaths.Log); err != nil {
		// Fallback to basic stdout logging if file logging fails
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
		slog.Error("Failed to setup file logging, using stdout only", "error", err)
	}

	slog.Info("Using OS-specific paths",
		"config", appPaths.Config,
		"database", appPaths.Database,
		"par2", appPaths.Par2,
		"data", appPaths.Data,
		"log", appPaths.Log)

	return &App{
		configPath: appPaths.Config,
		appPaths:   appPaths,
		isWebMode:  false,
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
func (a *App) GetAppStatus() AppStatus {
	defer a.recoverPanic("GetAppStatus")

	status := AppStatus{
		HasPostie:           a.postie != nil,
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

		// Set criticalConfigError if we have servers configured but postie instance creation failed
		if hasServers && validServers > 0 && a.postie == nil {
			status.CriticalConfigError = true
			status.Error = a.criticalErrorMessage
		}
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
	status := map[string]interface{}{
		"logPath":    a.appPaths.Log,
		"os":         runtime.GOOS,
		"canWrite":   false,
		"fileExists": false,
		"error":      "",
	}

	// Check if log file exists
	if _, err := os.Stat(a.appPaths.Log); err == nil {
		status["fileExists"] = true
	}

	// Test write permissions
	testFile := filepath.Join(filepath.Dir(a.appPaths.Log), ".write_test")
	if f, err := os.Create(testFile); err != nil {
		status["error"] = fmt.Sprintf("Cannot write to log directory: %v", err)
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
	defer a.recoverPanic("GetLogs")

	return a.GetLogsPaginated(0, 0) // 0, 0 means get last 1MB like before
}

// GetLogsPaginated returns paginated log content
// limit: number of lines to return (0 = use original 1MB limit)
// offset: number of lines to skip from the end (0 = start from most recent)
func (a *App) GetLogsPaginated(limit, offset int) (string, error) {
	defer a.recoverPanic("GetLogsPaginated")

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
			// Emit event to refresh queue in frontend for both desktop and web modes
			if !a.isWebMode {
				wailsruntime.EventsEmit(a.ctx, "queue-updated")
			} else if a.webEventEmitter != nil {
				a.webEventEmitter("queue-updated", nil)
			}
		}

		if addedCount == 0 {
			return fmt.Errorf("no valid files could be added to queue")
		}

		return nil
	}

	return fmt.Errorf("queue not initialized")
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
	cfg, err := config.Load(a.configPath)
	if err != nil {
		slog.Info("Config file exists but cannot be loaded, treating as first start", "error", err)
		return true
	}

	// Check if config has any configured servers with actual host values
	hasValidServers := false
	for _, server := range cfg.Servers {
		if server.Host != "" {
			hasValidServers = true
			break
		}
	}

	if !hasValidServers {
		slog.Info("Config file exists but has no valid servers, treating as first start")
		return true
	}

	slog.Info("Config file exists with valid servers, not first start")
	return false
}

// isFirstStart returns whether this is the first time the application is being run
func (a *App) isFirstStart() bool {
	return a.firstStart
}

// SetupWizardComplete saves the configuration from the setup wizard
func (a *App) SetupWizardComplete(wizardData SetupWizardData) error {
	defer a.recoverPanic("SetupWizardComplete")

	slog.Info("Completing setup wizard", "data", wizardData)

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
	}

	// Set output directory
	if wizardData.OutputDirectory != "" {
		cfg.OutputDir = wizardData.OutputDirectory
	}

	// Set the par2 path to the OS-specific location
	cfg.Par2.Par2Path = a.appPaths.Par2

	// Set the database path to the OS-specific location
	cfg.Queue.DatabasePath = a.appPaths.Database

	// Save configuration
	if err := a.SaveConfig(&cfg); err != nil {
		return fmt.Errorf("failed to save wizard configuration: %w", err)
	}

	// Mark as no longer first start since setup is complete
	a.firstStart = false

	slog.Info("Setup wizard completed successfully")
	return nil
}

// ValidateNNTPServer validates an NNTP server configuration by attempting to connect
func (a *App) ValidateNNTPServer(serverData ServerData) ValidationResult {
	defer a.recoverPanic("ValidateNNTPServer")

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

	// Create temporary server config for validation
	enabled := true
	serverConfig := config.ServerConfig{
		Host:                           serverData.Host,
		Port:                           serverData.Port,
		Username:                       serverData.Username,
		Password:                       serverData.Password,
		SSL:                            serverData.SSL,
		MaxConnections:                 1, // Use single connection for validation
		Enabled:                        &enabled,
		MaxConnectionIdleTimeInSeconds: 300,
		MaxConnectionTTLInSeconds:      3600,
	}

	// Create minimal config with just this server
	cfg := config.ConfigData{
		Servers: []config.ServerConfig{serverConfig},
	}

	// Attempt to create NNTP pool - this validates the connection parameters
	pool, err := cfg.GetNNTPPool()
	if err != nil {
		return ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to connect to server: %v", err),
		}
	}
	defer pool.Quit()

	// If we got here, the connection was successful
	slog.Info("NNTP server validation successful", "host", serverData.Host, "port", serverData.Port)
	return ValidationResult{
		Valid: true,
		Error: "",
	}
}
