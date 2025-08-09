package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/javi11/postie/internal/database"
	_ "github.com/mattn/go-sqlite3"
	"maragu.dev/goqite"
)

var _ QueueInterface = (*Queue)(nil)

// QueueInterface defines the interface for queue operations
type QueueInterface interface {
	AddFile(ctx context.Context, path string, size int64) error
	GetQueueItems() ([]QueueItem, error)
	RemoveFromQueue(id string) error
	ClearQueue() error
	GetQueueStats() (map[string]interface{}, error)
	SetQueueItemPriorityWithReorder(ctx context.Context, id string, newPriority int) error
	IsPathInQueue(path string) (bool, error)
}

type Queue struct {
	queue     *goqite.Queue
	db        *sql.DB
	runCtx    context.Context
	runCancel context.CancelFunc
}

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

type FileJob struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Priority   int       `json:"priority"`
	CreatedAt  time.Time `json:"createdAt"`
	RetryCount int       `json:"retryCount"`
}

type CompletedItem struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Priority    int       `json:"priority"`
	NzbPath     string    `json:"nzbPath"`
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt"`
	JobData     []byte    `json:"jobData"`
}

const (
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusComplete = "complete"
	StatusError    = "error"
)

func New(ctx context.Context, database *database.Database) (*Queue, error) {
	if database == nil {
		return nil, fmt.Errorf("database instance is required")
	}

	// Create goqite queue
	queue := goqite.New(goqite.NewOpts{
		DB:   database.DB,
		Name: "file_jobs",
	})

	runCtx, runCancel := context.WithCancel(ctx)

	return &Queue{
		queue:     queue,
		db:        database.DB,
		runCtx:    runCtx,
		runCancel: runCancel,
	}, nil
}


// AddFile adds a file to the queue for processing
func (q *Queue) AddFile(ctx context.Context, path string, size int64) error {
	// Check if the path already exists in pending queue, completed items, or errored items
	exists, err := q.IsPathInQueue(path)
	if err != nil {
		return fmt.Errorf("failed to check if path exists: %w", err)
	}
	if exists {
		slog.DebugContext(ctx, "File already exists in queue, ignoring", "path", path)
		return nil
	}

	job := FileJob{
		Path:       path,
		Size:       size,
		Priority:   0,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	slog.InfoContext(ctx, "Adding file to queue", "path", path, "size", size)

	return q.queue.Send(ctx, goqite.Message{
		Body:     jobData,
		Priority: job.Priority,
	})
}

// AddFileWithoutDuplicateCheck adds a file to the queue without checking for duplicates
// This is useful when original files are deleted after processing, making duplicate checks unnecessary
func (q *Queue) AddFileWithoutDuplicateCheck(ctx context.Context, path string, size int64) error {
	job := FileJob{
		Path:       path,
		Size:       size,
		Priority:   0,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	slog.InfoContext(ctx, "Adding file to queue", "path", path, "size", size)

	return q.queue.Send(ctx, goqite.Message{
		Body:     jobData,
		Priority: job.Priority,
	})
}

// AddFileWithPriority adds a file to the queue with a specific priority
func (q *Queue) AddFileWithPriority(ctx context.Context, path string, size int64, priority int) error {
	// Check if the path already exists in pending queue, completed items, or errored items
	exists, err := q.IsPathInQueue(path)
	if err != nil {
		return fmt.Errorf("failed to check if path exists: %w", err)
	}
	if exists {
		slog.DebugContext(ctx, "File already exists in queue, ignoring", "path", path)
		return nil
	}

	job := FileJob{
		Path:       path,
		Size:       size,
		Priority:   priority,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	slog.InfoContext(ctx, "Adding file to queue", "path", path, "size", size)

	return q.queue.Send(ctx, goqite.Message{
		Body:     jobData,
		Priority: job.Priority,
	})
}

// AddFileWithPriorityWithoutDuplicateCheck adds a file to the queue with a specific priority without checking for duplicates
// This is useful when original files are deleted after processing, making duplicate checks unnecessary
func (q *Queue) AddFileWithPriorityWithoutDuplicateCheck(ctx context.Context, path string, size int64, priority int) error {
	job := FileJob{
		Path:       path,
		Size:       size,
		Priority:   priority,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	slog.InfoContext(ctx, "Adding file to queue", "path", path, "size", size)

	return q.queue.Send(ctx, goqite.Message{
		Body:     jobData,
		Priority: job.Priority,
	})
}

// ReceiveFile gets the next file job from the queue and removes it immediately
func (q *Queue) ReceiveFile(ctx context.Context) (*goqite.Message, *FileJob, error) {
	msg, err := q.queue.Receive(ctx)
	if err != nil {
		return nil, nil, err
	}

	// If no message available, return nil (goqite returns nil, nil when no messages)
	if msg == nil {
		return nil, nil, nil
	}

	var job FileJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		// Delete the invalid message
		_ = q.queue.Delete(ctx, msg.ID)
		return nil, nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Delete the message from the queue immediately when starting to process
	// If processing fails, we'll re-add it to the queue
	if err := q.queue.Delete(ctx, msg.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to remove job from queue: %w", err)
	}

	return msg, &job, nil
}

// CompleteFile marks a file job as completed and adds it to completed_items table
func (q *Queue) CompleteFile(ctx context.Context, msgID goqite.ID, nzbPath string, job *FileJob) error {
	// Insert into completed_items table
	created := job.CreatedAt.Format("2006-01-02T15:04:05.000Z")

	// Marshal job data for storage
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job data: %w", err)
	}

	_, err = q.db.Exec(`
		INSERT INTO completed_items (id, path, size, priority, nzb_path, created_at, job_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, string(msgID), job.Path, job.Size, job.Priority, nzbPath, created, jobData)

	if err != nil {
		return fmt.Errorf("failed to insert completed item: %w", err)
	}

	return nil
}

// ExtendTimeout extends the processing timeout for a file job
func (q *Queue) ExtendTimeout(ctx context.Context, msgID goqite.ID, duration time.Duration) error {
	return q.queue.Extend(ctx, msgID, duration)
}

// GetQueueItems returns queue items for display including completed items
func (q *Queue) GetQueueItems() ([]QueueItem, error) {
	var items []QueueItem

	// Get active queue items (only pending items, not running ones)
	// goqite uses higher numbers for higher priority, so ORDER BY priority DESC
	rows, err := q.db.Query(`
		SELECT id, created, updated, body, timeout, received
		FROM goqite 
		WHERE queue = 'file_jobs'
		ORDER BY priority DESC, created ASC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	for rows.Next() {
		var id, created, updated, timeout string
		var body []byte
		var received int

		err := rows.Scan(&id, &created, &updated, &body, &timeout, &received)
		if err != nil {
			return nil, err
		}

		// Parse the job data
		var job FileJob
		if err := json.Unmarshal(body, &job); err != nil {
			continue // Skip invalid jobs
		}

		// All items in the queue are pending (processor handles running state separately)
		status := StatusPending
		var completedAt *time.Time

		// Parse timestamps
		createdTime, _ := time.Parse("2006-01-02T15:04:05.000Z", created)
		updatedTime, _ := time.Parse("2006-01-02T15:04:05.000Z", updated)

		item := QueueItem{
			ID:          id,
			Path:        job.Path,
			FileName:    getFileName(job.Path),
			Size:        job.Size,
			Status:      status,
			RetryCount:  job.RetryCount,
			Priority:    job.Priority,
			CreatedAt:   createdTime,
			UpdatedAt:   updatedTime,
			CompletedAt: completedAt,
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get completed items
	completedRows, err := q.db.Query(`
		SELECT id, path, size, priority, nzb_path, created_at, completed_at
		FROM completed_items
		ORDER BY completed_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := completedRows.Close(); err != nil {
			slog.Error("Failed to close completed rows", "error", err)
		}
	}()

	for completedRows.Next() {
		var id, path, nzbPath, createdAt, completedAt string
		var size int64
		var priority int

		err := completedRows.Scan(&id, &path, &size, &priority, &nzbPath, &createdAt, &completedAt)
		if err != nil {
			return nil, err
		}

		// Parse timestamps
		createdTime, _ := time.Parse("2006-01-02T15:04:05.000Z", createdAt)
		completedTime, _ := time.Parse("2006-01-02T15:04:05.000Z", completedAt)

		item := QueueItem{
			ID:          id,
			Path:        path,
			FileName:    getFileName(path),
			Size:        size,
			Status:      StatusComplete,
			RetryCount:  0,
			Priority:    priority,
			CreatedAt:   createdTime,
			UpdatedAt:   completedTime,
			CompletedAt: &completedTime,
			NzbPath:     &nzbPath,
		}

		items = append(items, item)
	}

	if err := completedRows.Err(); err != nil {
		return nil, err
	}

	// Get errored items
	erroredRows, err := q.db.Query(`
		SELECT id, path, size, priority, error_message, created_at, errored_at, job_data
		FROM errored_items
		ORDER BY errored_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := erroredRows.Close(); err != nil {
			slog.Error("Failed to close errored rows", "error", err)
		}
	}()

	for erroredRows.Next() {
		var id, path, errMsg, createdAt, erroredAt string
		var size int64
		var priority int
		var jobData []byte

		err := erroredRows.Scan(&id, &path, &size, &priority, &errMsg, &createdAt, &erroredAt, &jobData)
		if err != nil {
			return nil, err
		}

		var job FileJob
		if err := json.Unmarshal(jobData, &job); err != nil {
			continue // Skip invalid jobs
		}

		// Parse timestamps
		createdTime, _ := time.Parse("2006-01-02T15:04:05.000Z", createdAt)
		erroredTime, _ := time.Parse("2006-01-02T15:04:05.000Z", erroredAt)

		item := QueueItem{
			ID:           id,
			Path:         path,
			FileName:     getFileName(path),
			Size:         size,
			Status:       StatusError,
			RetryCount:   job.RetryCount,
			Priority:     priority,
			ErrorMessage: &errMsg,
			CreatedAt:    createdTime,
			UpdatedAt:    erroredTime,
		}

		items = append(items, item)
	}

	return items, erroredRows.Err()
}

// RemoveFromQueue removes an item from the queue by ID (handles active, completed, and errored items)
func (q *Queue) RemoveFromQueue(id string) error {
	// First check if the item exists in completed items
	var exists bool
	err := q.db.QueryRow("SELECT EXISTS(SELECT 1 FROM completed_items WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check completed items: %w", err)
	}

	if exists {
		// Remove from completed items
		return q.RemoveCompletedItem(id)
	}

	// Check if the item exists in errored items
	err = q.db.QueryRow("SELECT EXISTS(SELECT 1 FROM errored_items WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check errored items: %w", err)
	}

	if exists {
		// Remove from errored items
		return q.RemoveErroredItem(id)
	}

	// If not found in completed or errored items, try to remove from active queue
	return q.queue.Delete(q.runCtx, goqite.ID(id))
}

// RemoveCompletedItem removes a completed item and its associated NZB file
func (q *Queue) RemoveCompletedItem(id string) error {
	// Get the NZB path before deleting
	var nzbPath string
	err := q.db.QueryRow("SELECT nzb_path FROM completed_items WHERE id = ?", id).Scan(&nzbPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("completed item not found: %s", id)
		}
		return fmt.Errorf("failed to get NZB path: %w", err)
	}

	// Delete the database record
	_, err = q.db.Exec("DELETE FROM completed_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete completed item: %w", err)
	}

	// Delete the NZB file
	if err := os.Remove(nzbPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to delete NZB file", "path", nzbPath, "error", err)
		// Don't return error here as the database record is already deleted
	}

	return nil
}

// RemoveErroredItem removes an errored item from the database
func (q *Queue) RemoveErroredItem(id string) error {
	// Delete the database record
	_, err := q.db.Exec("DELETE FROM errored_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete errored item: %w", err)
	}

	return nil
}

// ClearQueue removes all completed, errored, and active items from the queue
func (q *Queue) ClearQueue() error {
	// Get all NZB paths before deleting
	rows, err := q.db.Query("SELECT nzb_path FROM completed_items")
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	var nzbPaths []string
	for rows.Next() {
		var nzbPath string
		if err := rows.Scan(&nzbPath); err != nil {
			continue
		}
		nzbPaths = append(nzbPaths, nzbPath)
	}

	// Clear completed items from database
	_, err = q.db.Exec("DELETE FROM completed_items")
	if err != nil {
		return err
	}

	// Clear errored items from database
	_, err = q.db.Exec("DELETE FROM errored_items")
	if err != nil {
		return err
	}

	// Delete all NZB files
	for _, nzbPath := range nzbPaths {
		if err := os.Remove(nzbPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to delete NZB file during clear", "path", nzbPath, "error", err)
		}
	}

	// Also clear active queue items
	_, err = q.db.Exec("DELETE FROM goqite WHERE queue = 'file_jobs'")
	return err
}

// ClearCompletedItems removes only completed items and their NZB files
func (q *Queue) ClearCompletedItems() error {
	// Get all NZB paths before deleting
	rows, err := q.db.Query("SELECT nzb_path FROM completed_items")
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	var nzbPaths []string
	for rows.Next() {
		var nzbPath string
		if err := rows.Scan(&nzbPath); err != nil {
			continue
		}
		nzbPaths = append(nzbPaths, nzbPath)
	}

	// Clear completed items from database
	_, err = q.db.Exec("DELETE FROM completed_items")
	if err != nil {
		return err
	}

	// Delete all NZB files
	for _, nzbPath := range nzbPaths {
		if err := os.Remove(nzbPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to delete NZB file", "path", nzbPath, "error", err)
		}
	}

	return nil
}

// GetQueueStats returns statistics about the queue including completed and errored items
func (q *Queue) GetQueueStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total pending count (all items in queue are pending)
	var pending int
	err := q.db.QueryRow("SELECT COUNT(*) FROM goqite WHERE queue = 'file_jobs'").Scan(&pending)
	if err != nil {
		return nil, err
	}
	stats["pending"] = pending
	stats["total"] = pending

	// Running count is handled by processor, not queue
	stats["running"] = 0

	// Count completed items
	var complete int
	err = q.db.QueryRow("SELECT COUNT(*) FROM completed_items").Scan(&complete)
	if err == nil {
		stats["complete"] = complete
	} else {
		stats["complete"] = 0
	}

	// Count errored items
	var errorCount int
	err = q.db.QueryRow("SELECT COUNT(*) FROM errored_items").Scan(&errorCount)
	if err == nil {
		stats["error"] = errorCount
	} else {
		stats["error"] = 0
	}

	// Add total including completed and errored
	stats["total_including_completed"] = pending + complete + errorCount

	return stats, nil
}

// Close closes the queue context (database is managed externally)
func (q *Queue) Close() error {
	if q.runCancel != nil {
		q.runCancel()
	}
	return nil
}

// IsPathInQueue checks if a file path already exists in pending queue, completed items, or errored items
func (q *Queue) IsPathInQueue(path string) (bool, error) {
	// Check pending queue items
	var count int
	err := q.db.QueryRow(`
		SELECT COUNT(*) FROM goqite 
		WHERE queue = 'file_jobs' AND json_extract(body, '$.path') = ?
	`, path).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending queue: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	// Check completed items
	err = q.db.QueryRow("SELECT COUNT(*) FROM completed_items WHERE path = ?", path).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check completed items: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	// Check errored items
	err = q.db.QueryRow("SELECT COUNT(*) FROM errored_items WHERE path = ?", path).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check errored items: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	return false, nil
}

func getFileName(path string) string {
	// Simple filename extraction
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// GetCompletedItemNzbPath returns the NZB path for a completed item
func (q *Queue) GetCompletedItemNzbPath(id string) (string, error) {
	var nzbPath string
	err := q.db.QueryRow("SELECT nzb_path FROM completed_items WHERE id = ?", id).Scan(&nzbPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("completed item not found: %s", id)
		}
		return "", fmt.Errorf("failed to get NZB path: %w", err)
	}
	return nzbPath, nil
}

// DebugQueueItem returns debug information about a specific queue item
func (q *Queue) DebugQueueItem(id string) (map[string]interface{}, error) {
	var received int
	var timeout, created, updated string
	var body []byte

	err := q.db.QueryRow(`
		SELECT received, timeout, created, updated, body
		FROM goqite 
		WHERE id = ? AND queue = 'file_jobs'
	`, id).Scan(&received, &timeout, &created, &updated, &body)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get job debug info: %w", err)
	}

	// Parse timeout
	timeoutTime, timeoutErr := time.Parse("2006-01-02T15:04:05.000Z", timeout)
	now := time.Now()

	// Check processor status
	debug := map[string]interface{}{
		"id":                id,
		"received":          received,
		"timeout":           timeout,
		"created":           created,
		"updated":           updated,
		"currentTime":       now.Format("2006-01-02T15:04:05.000Z"),
		"timeoutParseError": timeoutErr,
		"isBeforeTimeout":   timeoutErr == nil && now.Before(timeoutTime),
		"shouldBeRunning":   timeoutErr == nil && now.Before(timeoutTime) && received > 0,
	}

	if timeoutErr == nil {
		debug["timeoutDiff"] = timeoutTime.Sub(now).String()
	}

	return debug, nil
}

// ReaddJob re-adds a job to the queue (used when processing fails)
func (q *Queue) ReaddJob(ctx context.Context, job *FileJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return q.queue.Send(ctx, goqite.Message{
		Body:     jobData,
		Priority: job.Priority,
	})
}

// SetQueueItemPriority updates the priority of a pending queue item by id
func (q *Queue) SetQueueItemPriority(id string, priority int) error {
	// Get the job body for the given id
	var body []byte
	err := q.db.QueryRow("SELECT body FROM goqite WHERE id = ? AND queue = 'file_jobs'", id).Scan(&body)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("pending queue item not found: %s", id)
		}
		return fmt.Errorf("failed to get queue item: %w", err)
	}

	// Unmarshal, update priority, marshal
	var job FileJob
	if err := json.Unmarshal(body, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}
	job.Priority = priority
	newBody, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal updated job: %w", err)
	}

	// Update the body in the database
	_, err = q.db.Exec("UPDATE goqite SET body = ?, updated = strftime('%Y-%m-%dT%H:%M:%fZ') WHERE id = ? AND queue = 'file_jobs'", newBody, id)
	if err != nil {
		return fmt.Errorf("failed to update queue item priority: %w", err)
	}
	return nil
}

// SetQueueItemPriorityWithReorder updates the priority of a pending queue item using goqite's native priority
// Higher priority numbers (0, 1, 2, ...) are processed first
func (q *Queue) SetQueueItemPriorityWithReorder(ctx context.Context, id string, newPriority int) error {
	// Get the current job data to update it
	var body []byte
	err := q.db.QueryRow("SELECT body FROM goqite WHERE id = ? AND queue = 'file_jobs'", id).Scan(&body)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("pending queue item not found: %s", id)
		}
		return fmt.Errorf("failed to get queue item: %w", err)
	}

	// Unmarshal and update priority in job data
	var job FileJob
	if err := json.Unmarshal(body, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	job.Priority = newPriority
	newBody, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal updated job: %w", err)
	}

	// Update both the body and the goqite priority field
	_, err = q.db.Exec(`
		UPDATE goqite 
		SET body = ?, priority = ?, updated = strftime('%Y-%m-%dT%H:%M:%fZ') 
		WHERE id = ? AND queue = 'file_jobs'
	`, newBody, newPriority, id)

	if err != nil {
		return fmt.Errorf("failed to update queue item priority: %w", err)
	}

	return nil
}

// MarkAsError marks a file job as errored and adds it to the errored_items table
func (q *Queue) MarkAsError(ctx context.Context, msgID goqite.ID, job *FileJob, errMsg string) error {
	created := job.CreatedAt.Format("2006-01-02T15:04:05.000Z")

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job data: %w", err)
	}

	_, err = q.db.Exec(`
		INSERT INTO errored_items (id, path, size, priority, error_message, created_at, job_data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, string(msgID), job.Path, job.Size, job.Priority, errMsg, created, jobData)

	if err != nil {
		return fmt.Errorf("failed to insert errored item: %w", err)
	}

	return nil
}

// RetryErroredJob retries an errored job by moving it back to the main queue
func (q *Queue) RetryErroredJob(ctx context.Context, id string) error {
	var jobData []byte
	err := q.db.QueryRow("SELECT job_data FROM errored_items WHERE id = ?", id).Scan(&jobData)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("errored item not found: %s", id)
		}
		return fmt.Errorf("failed to get errored item: %w", err)
	}

	var job FileJob
	if err := json.Unmarshal(jobData, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job data: %w", err)
	}

	// Reset retry count and re-add to queue
	job.RetryCount = 0
	if err := q.ReaddJob(ctx, &job); err != nil {
		return fmt.Errorf("failed to re-add job to queue: %w", err)
	}

	// Delete from errored items
	_, err = q.db.Exec("DELETE FROM errored_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete errored item: %w", err)
	}

	return nil
}
