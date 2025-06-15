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
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"createdAt"`
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
	`

	_, err := db.Exec(schema)
	return err
}

// AddFile adds a file to the queue for processing
func (q *Queue) AddFile(ctx context.Context, path string, size int64) error {
	job := FileJob{
		Path:      path,
		Size:      size,
		Priority:  0,
		CreatedAt: time.Now(),
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
	job := FileJob{
		Path:      path,
		Size:      size,
		Priority:  priority,
		CreatedAt: time.Now(),
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
		q.queue.Delete(ctx, msg.ID)
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
	defer rows.Close()

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
			RetryCount:  received,
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
	defer completedRows.Close()

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

	return items, completedRows.Err()
}

// RemoveFromQueue removes an item from the queue by ID (handles both active and completed)
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

	// If not found in completed items, try to remove from active queue
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

// ClearQueue removes all completed items from the queue
func (q *Queue) ClearQueue() error {
	// Get all NZB paths before deleting
	rows, err := q.db.Query("SELECT nzb_path FROM completed_items")
	if err != nil {
		return err
	}
	defer rows.Close()

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
	defer rows.Close()

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

// GetQueueStats returns statistics about the queue including completed items
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

	// For compatibility with existing UI
	stats["error"] = 0

	// Add total including completed
	stats["total_including_completed"] = pending + complete

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
