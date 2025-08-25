package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/javi11/postie/internal/database"
	"github.com/javi11/postie/internal/queue"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// QueueItem represents a queue item for the frontend - matches queue.QueueItem
type QueueItem struct {
	ID           string     `json:"id"`
	Path         string     `json:"path"`
	FileName     string     `json:"fileName"`
	Size         int64      `json:"size"`
	Status       string     `json:"status"`
	RetryCount   int        `json:"retryCount"`
	Priority     int        `json:"priority"`
	ErrorMessage *string    `json:"errorMessage"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CompletedAt  *time.Time `json:"completedAt"`
	NzbPath      *string    `json:"nzbPath"`
}

// QueueStats represents queue statistics
type QueueStats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Running  int `json:"running"`
	Complete int `json:"complete"`
	Error    int `json:"error"`
}

// PaginationParams defines parameters for paginated queries
type PaginationParams struct {
	Page   int    `json:"page"`   // 1-based page number
	Limit  int    `json:"limit"`  // Items per page
	SortBy string `json:"sortBy"` // Sort field: "created", "priority", "status", "filename"
	Order  string `json:"order"`  // Sort order: "asc", "desc"
}

// PaginatedQueueResult contains paginated queue items and metadata
type PaginatedQueueResult struct {
	Items        []QueueItem `json:"items"`
	TotalItems   int         `json:"totalItems"`
	TotalPages   int         `json:"totalPages"`
	CurrentPage  int         `json:"currentPage"`
	ItemsPerPage int         `json:"itemsPerPage"`
	HasNext      bool        `json:"hasNext"`
	HasPrev      bool        `json:"hasPrev"`
}

func (a *App) initializeQueue() error {
	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}

	// Stop previous queue if running
	if a.queue != nil {
		_ = a.queue.Close()
		a.queue = nil
	}

	// Close previous database if exists
	if a.database != nil {
		_ = a.database.Close()
		a.database = nil
	}

	// Stop processor context to clean up any running operations
	if a.procCancel != nil {
		a.procCancel()
		a.procCancel = nil
		a.procCtx = nil
	}

	databaseCfg := a.config.GetDatabaseConfig()

	// Create context for queue and processor
	a.procCtx, a.procCancel = context.WithCancel(context.Background())

	// Initialize database
	db, err := database.New(a.procCtx, databaseCfg)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Store database reference for cleanup
	a.database = db

	// Run database migrations
	if err := db.EnsureMigrationCompatibility(); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Initialize queue with database (always available for manual file additions)
	a.queue, err = queue.New(a.procCtx, db)
	if err != nil {
		return fmt.Errorf("failed to create queue: %w", err)
	}

	slog.Info("Queue initialized successfully")
	return nil
}

// AddFilesToQueue adds multiple files to the queue for processing
func (a *App) AddFilesToQueue() error {
	defer a.recoverPanic("AddFilesToQueue")

	if a.queue == nil {
		slog.Error("Queue not initialized - this should not happen")
		return fmt.Errorf("queue not initialized - please restart the application")
	}

	// Open file dialog for multiple files
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select files to add to queue",
	})
	if err != nil {
		return fmt.Errorf("error opening file dialog: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files selected")
	}

	// Add files directly to queue
	addedCount := 0
	for _, filePath := range files {
		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			slog.Warn("Could not get file info, skipping", "file", filePath, "error", err)
			continue
		}

		// Add file directly to queue
		if err := a.queue.AddFile(context.Background(), filePath, info.Size()); err != nil {
			slog.Warn("Could not add file to queue, skipping", "file", filePath, "error", err)
			continue
		}

		addedCount++
		slog.Info("File added directly to queue", "file", filepath.Base(filePath), "size", info.Size())
	}

	slog.Info("Added files directly to queue", "added", addedCount, "total", len(files))

	// Emit event to refresh queue in frontend for both desktop and web modes
	if !a.isWebMode {
		runtime.EventsEmit(a.ctx, "queue-updated")
	} else if a.webEventEmitter != nil {
		a.webEventEmitter("queue-updated", nil)
	}

	return nil
}

// GetQueueItems returns the current queue items from the queue
func (a *App) GetQueueItems(params PaginationParams) (*PaginatedQueueResult, error) {
	defer a.recoverPanic("GetQueueItems")

	if a.queue == nil {
		return &PaginatedQueueResult{
			Items:        []QueueItem{},
			TotalItems:   0,
			TotalPages:   0,
			CurrentPage:  params.Page,
			ItemsPerPage: params.Limit,
			HasNext:      false,
			HasPrev:      false,
		}, nil
	}

	// Convert backend params to queue params
	queueParams := queue.PaginationParams{
		Page:   params.Page,
		Limit:  params.Limit,
		SortBy: params.SortBy,
		Order:  params.Order,
	}

	result, err := a.queue.GetQueueItems(queueParams)
	if err != nil {
		return nil, err
	}

	// Convert queue items to backend items
	var items []QueueItem
	for _, queueItem := range result.Items {
		item := QueueItem{
			ID:           queueItem.ID,
			Path:         queueItem.Path,
			FileName:     queueItem.FileName,
			Size:         queueItem.Size,
			Status:       queueItem.Status,
			RetryCount:   queueItem.RetryCount,
			Priority:     queueItem.Priority,
			ErrorMessage: queueItem.ErrorMessage,
			CreatedAt:    queueItem.CreatedAt,
			UpdatedAt:    queueItem.UpdatedAt,
			CompletedAt:  queueItem.CompletedAt,
			NzbPath:      queueItem.NzbPath,
		}
		items = append(items, item)
	}

	return &PaginatedQueueResult{
		Items:        items,
		TotalItems:   result.TotalItems,
		TotalPages:   result.TotalPages,
		CurrentPage:  result.CurrentPage,
		ItemsPerPage: result.ItemsPerPage,
		HasNext:      result.HasNext,
		HasPrev:      result.HasPrev,
	}, nil
}

// RemoveFromQueue removes an item from the queue via queue
func (a *App) RemoveFromQueue(id string) error {
	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}

	err := a.queue.RemoveFromQueue(id)
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

// DebugQueueItem returns debug information about a queue item
func (a *App) DebugQueueItem(id string) (map[string]interface{}, error) {
	if a.queue == nil {
		return nil, fmt.Errorf("queue not initialized")
	}

	return a.queue.DebugQueueItem(id)
}

// ClearQueue removes all completed and failed items from the queue via queue
func (a *App) ClearQueue() error {
	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}

	err := a.queue.ClearQueue()
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

// GetQueueStats returns statistics about the queue via queue
func (a *App) GetQueueStats() (QueueStats, error) {
	if a.queue == nil {
		return QueueStats{
			Total:    0,
			Pending:  0,
			Running:  0,
			Complete: 0,
			Error:    0,
		}, nil
	}

	stats, err := a.queue.GetQueueStats()
	if err != nil {
		return QueueStats{}, err
	}

	// Convert map[string]interface{} to QueueStats struct
	queueStats := QueueStats{}

	if total, ok := stats["total"].(int); ok {
		queueStats.Total = total
	}
	if pending, ok := stats["pending"].(int); ok {
		queueStats.Pending = pending
	}
	if running, ok := stats["running"].(int); ok {
		queueStats.Running = running
	}
	if complete, ok := stats["complete"].(int); ok {
		queueStats.Complete = complete
	}
	if errorCount, ok := stats["error"].(int); ok {
		queueStats.Error = errorCount
	}

	return queueStats, nil
}

// DownloadNZB downloads the NZB file for a completed item
func (a *App) DownloadNZB(id string) error {
	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}

	// Get the NZB path for the completed item
	nzbPath, err := a.queue.GetCompletedItemNzbPath(id)
	if err != nil {
		return fmt.Errorf("failed to get NZB path: %w", err)
	}

	// Check if the NZB file exists
	if _, err := os.Stat(nzbPath); os.IsNotExist(err) {
		return fmt.Errorf("NZB file not found: %s", nzbPath)
	}

	// Get the filename from the path
	fileName := filepath.Base(nzbPath)

	// Use Wails runtime to save the file
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save NZB File",
		DefaultFilename: fileName,
	})

	if err != nil {
		return fmt.Errorf("failed to show save dialog: %w", err)
	}

	// If user cancelled the dialog, savePath will be empty
	if savePath == "" {
		return nil // User cancelled, not an error
	}

	// Read the NZB file content
	nzbContent, err := os.ReadFile(nzbPath)
	if err != nil {
		return fmt.Errorf("failed to read NZB file: %w", err)
	}

	// Write the file to the selected location
	if err := os.WriteFile(savePath, nzbContent, 0644); err != nil {
		return fmt.Errorf("failed to save NZB file: %w", err)
	}

	slog.Info("NZB file downloaded successfully", "id", id, "from", nzbPath, "to", savePath)
	return nil
}

// GetNZBContent returns the content of an NZB file for a completed item
func (a *App) GetNZB(id string) (content string, fileName string, err error) {
	if a.queue == nil {
		return "", "", fmt.Errorf("queue not initialized")
	}

	// Get the NZB path for the completed item
	nzbPath, err := a.queue.GetCompletedItemNzbPath(id)
	if err != nil {
		return "", "", fmt.Errorf("failed to get NZB path: %w", err)
	}

	// Check if the NZB file exists
	if _, err := os.Stat(nzbPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("NZB file not found: %s", nzbPath)
	}

	// Read the NZB file content
	nzbContent, err := os.ReadFile(nzbPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read NZB file: %w", err)
	}

	f := filepath.Base(nzbPath)

	return string(nzbContent), f, nil
}

// SetQueueItemPriority updates the priority of a pending queue item by id and reorders the queue
func (a *App) SetQueueItemPriority(id string, priority int) error {
	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}
	if err := a.queue.SetQueueItemPriorityWithReorder(context.Background(), id, priority); err != nil {
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

// Database migration methods exposed through the App

// GetMigrationStatus returns the current migration status
func (a *App) GetMigrationStatus() (map[string]interface{}, error) {
	if a.database == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	status, err := migrationRunner.GetStatus()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"currentVersion": status.CurrentVersion,
	}, nil
}

// RunMigrations runs all pending migrations
func (a *App) RunMigrations() error {
	if a.database == nil {
		return fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	return migrationRunner.MigrateUp()
}

// RollbackMigration rolls back the last applied migration
func (a *App) RollbackMigration() error {
	if a.database == nil {
		return fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	return migrationRunner.MigrateDown()
}

// MigrateTo migrates to a specific version
func (a *App) MigrateTo(version int64) error {
	if a.database == nil {
		return fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	return migrationRunner.MigrateTo(version)
}

// ResetDatabase drops all tables and re-runs all migrations
func (a *App) ResetDatabase() error {
	if a.database == nil {
		return fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	return migrationRunner.Reset()
}

// IsLegacyDatabase checks if the database is a legacy (non-goose) database
func (a *App) IsLegacyDatabase() (bool, error) {
	if a.database == nil {
		return false, fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	return migrationRunner.IsLegacyDatabase()
}

// RecreateDatabase drops all tables and recreates them using migrations
func (a *App) RecreateDatabase() error {
	if a.database == nil {
		return fmt.Errorf("database not initialized")
	}

	migrationRunner := a.database.GetMigrationRunner()
	return migrationRunner.RecreateDatabase()
}
