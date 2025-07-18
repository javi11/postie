package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/javi11/postie/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"maragu.dev/goqite"
)

type Queue struct {
	queue     *goqite.Queue
	db        *sql.DB
	dbPath    string
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

func New(ctx context.Context, cfg config.QueueConfig) (*Queue, error) {
	dbPath := cfg.DatabasePath
	if dbPath == "" {
		dbPath = "postie_queue.db"
	}

	slog.InfoContext(ctx, fmt.Sprintf("Using %s database at %s", cfg.DatabaseType, dbPath))

	// For now, only SQLite is fully implemented
	if cfg.DatabaseType != "sqlite" {
		return nil, fmt.Errorf("database type %s is not yet implemented, please use 'sqlite'", cfg.DatabaseType)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_timeout=5000&_fk=true")
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Initialize goqite schema and completed items table
	if err := initGoqiteSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize goqite schema: %w", err)
	}

	// Create goqite queue
	queue := goqite.New(goqite.NewOpts{
		DB:   db,
		Name: "file_jobs",
	})

	runCtx, runCancel := context.WithCancel(ctx)

	return &Queue{
		queue:     queue,
		db:        db,
		dbPath:    dbPath,
		runCtx:    runCtx,
		runCancel: runCancel,
	}, nil
}

func initGoqiteSchema(db *sql.DB) error {
	// Use the goqite SQLite schema
	schema := `
		create table if not exists goqite (
		  id text primary key default ('m_' || lower(hex(randomblob(16)))),
		  created text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
		  updated text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
		  queue text not null,
		  body blob not null,
		  timeout text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
		  received integer not null default 0
		) strict;

		create trigger if not exists goqite_updated_timestamp after update on goqite begin
		  update goqite set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
		end;

		create index if not exists goqite_queue_created_idx on goqite (queue, created);

		-- Table for completed items
		create table if not exists completed_items (
		  id text primary key,
		  path text not null,
		  size integer not null,
		  priority integer not null default 0,
		  nzb_path text not null,
		  created_at text not null,
		  completed_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
		  job_data blob not null
		);

		create index if not exists completed_items_completed_at_idx on completed_items (completed_at);
		create index if not exists completed_items_path_idx on completed_items (path);

		-- Table for errored items
		create table if not exists errored_items (
			id text primary key,
			path text not null,
			size integer not null,
			priority integer not null default 0,
			error_message text not null,
			created_at text not null,
			errored_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
			job_data blob not null
		);

		create index if not exists errored_items_errored_at_idx on errored_items (errored_at);
		create index if not exists errored_items_path_idx on errored_items (path);
	`

	_, err := db.Exec(schema)
	return err
}

// AddFile adds a file to the queue for processing
func (q *Queue) AddFile(ctx context.Context, path string, size int64) error {
	// Check if the path already exists in pending queue, completed items, or errored items
	exists, err := q.pathExists(path)
	if err != nil {
		return fmt.Errorf("failed to check if path exists: %w", err)
	}
	if exists {
		slog.InfoContext(ctx, "File already exists in queue, ignoring", "path", path)
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
		Body: jobData,
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
		Body: jobData,
	})
}

// AddFileWithPriority adds a file to the queue with a specific priority
func (q *Queue) AddFileWithPriority(ctx context.Context, path string, size int64, priority int) error {
	// Check if the path already exists in pending queue, completed items, or errored items
	exists, err := q.pathExists(path)
	if err != nil {
		return fmt.Errorf("failed to check if path exists: %w", err)
	}
	if exists {
		slog.InfoContext(ctx, "File already exists in queue, ignoring", "path", path)
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
		Body: jobData,
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
		Body: jobData,
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
	rows, err := q.db.Query(`
		SELECT id, created, updated, body, timeout, received
		FROM goqite 
		WHERE queue = 'file_jobs'
		ORDER BY json_extract(body, '$.Priority') DESC, created ASC
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

// Close closes the database connection
func (q *Queue) Close() error {
	if q.runCancel != nil {
		q.runCancel()
	}
	if q.db != nil {
		return q.db.Close()
	}
	return nil
}

// pathExists checks if a file path already exists in pending queue, completed items, or errored items
func (q *Queue) pathExists(path string) (bool, error) {
	// Check pending queue items
	var count int
	err := q.db.QueryRow(`
		SELECT COUNT(*) FROM goqite 
		WHERE queue = 'file_jobs' AND json_extract(body, '$.Path') = ?
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
		Body: jobData,
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
