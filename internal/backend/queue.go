package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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

func (a *App) initializeQueue() error {
	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}

	// Stop previous queue if running
	if a.queue != nil {
		_ = a.queue.Close()
		a.queue = nil
	}

	// Stop processor context to clean up any running operations
	if a.procCancel != nil {
		a.procCancel()
		a.procCancel = nil
		a.procCtx = nil
	}

	queueCfg := a.config.GetQueueConfig()

	// Get output directory from configuration
	outputDir := a.config.GetOutputDir()

	// If output directory is relative, make it relative to OS-specific data directory
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(a.appPaths.Data, outputDir)
	}

	// Ensure output directory exists (always needed for queue operations)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create context for queue and processor
	a.procCtx, a.procCancel = context.WithCancel(context.Background())

	// Initialize queue (always available for manual file additions)
	var err2 error
	a.queue, err2 = queue.New(a.procCtx, queueCfg)
	if err2 != nil {
		return fmt.Errorf("failed to create queue: %w", err2)
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
func (a *App) GetQueueItems() ([]QueueItem, error) {
	defer a.recoverPanic("GetQueueItems")
	
	if a.queue == nil {
		return []QueueItem{}, nil
	}

	queueItems, err := a.queue.GetQueueItems()
	if err != nil {
		return nil, err
	}

	// Convert queue.QueueItem to app.QueueItem (they should be compatible now)
	var items []QueueItem
	for _, queueItem := range queueItems {
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

	return items, nil
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
func (a *App) GetNZBContent(id string) (string, error) {
	if a.queue == nil {
		return "", fmt.Errorf("queue not initialized")
	}

	// Get the NZB path for the completed item
	nzbPath, err := a.queue.GetCompletedItemNzbPath(id)
	if err != nil {
		return "", fmt.Errorf("failed to get NZB path: %w", err)
	}

	// Check if the NZB file exists
	if _, err := os.Stat(nzbPath); os.IsNotExist(err) {
		return "", fmt.Errorf("NZB file not found: %s", nzbPath)
	}

	// Read the NZB file content
	nzbContent, err := os.ReadFile(nzbPath)
	if err != nil {
		return "", fmt.Errorf("failed to read NZB file: %w", err)
	}

	return string(nzbContent), nil
}

// SetQueueItemPriority updates the priority of a pending queue item by id
func (a *App) SetQueueItemPriority(id string, priority int) error {
	if a.queue == nil {
		return fmt.Errorf("queue not initialized")
	}
	if err := a.queue.SetQueueItemPriority(id, priority); err != nil {
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
