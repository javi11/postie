package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/opencontainers/selinux/pkg/pwalkdir"
)

const (
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusComplete = "complete"
	StatusError    = "error"
)

func (w *Watcher) scanDirectory(ctx context.Context) error {
	// First handle any failed items
	if err := w.handleFailedItems(ctx); err != nil {
		return fmt.Errorf("error handling failed items: %w", err)
	}

	return pwalkdir.Walk(w.watchFolder, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if dir.IsDir() {
			return nil
		}

		info, err := dir.Info()
		if err != nil {
			return err
		}

		// Skip if file is too small
		if info.Size() < w.cfg.MinFileSize {
			return nil
		}

		// Check ignore patterns
		for _, pattern := range w.cfg.IgnorePatterns {
			if matched, err := filepath.Match(pattern, info.Name()); err == nil && matched {
				return nil
			}
		}

		// Check if file is open
		if isFileOpen(path) {
			return nil
		}

		// Add to queue if not already present
		if err := w.addToQueue(ctx, path, info.Size()); err != nil {
			return fmt.Errorf("error adding to queue: %w", err)
		}

		return nil
	})
}

func (w *Watcher) addToQueue(ctx context.Context, path string, size int64) error {
	// Check if file is already in queue
	var exists bool
	err := w.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM queue_items WHERE path = ?)", path).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	_, err = w.db.ExecContext(ctx,
		"INSERT INTO queue_items (path, size, status) VALUES (?, ?, ?)",
		path, size, StatusPending,
	)
	return err
}

func (w *Watcher) getNextBatch(ctx context.Context) ([]QueueItem, error) {
	var items []QueueItem
	rows, err := w.db.QueryContext(ctx,
		"SELECT id, path, size, status FROM queue_items WHERE status = ? ORDER BY created_at ASC LIMIT 10",
		StatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.ErrorContext(ctx, "Error closing rows", "error", err)
		}
	}()

	for rows.Next() {
		var item QueueItem
		if err := rows.Scan(&item.ID, &item.Path, &item.Size, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (w *Watcher) updateItemStatus(ctx context.Context, id int64, status string) error {
	_, err := w.db.ExecContext(ctx,
		"UPDATE queue_items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, id,
	)
	return err
}

func (w *Watcher) deleteItem(ctx context.Context, id int64) error {
	_, err := w.db.ExecContext(ctx, "DELETE FROM queue_items WHERE id = ?", id)
	return err
}

func isFileOpen(path string) bool {
	file, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return true
	}

	_ = file.Close()

	return false
}

func (w *Watcher) isWithinSchedule() bool {
	if w.cfg.Schedule.StartTime == "" || w.cfg.Schedule.EndTime == "" {
		return true
	}

	now := time.Now()
	start, err := time.Parse("15:04", w.cfg.Schedule.StartTime)
	if err != nil {
		return true
	}
	end, err := time.Parse("15:04", w.cfg.Schedule.EndTime)
	if err != nil {
		return true
	}

	current := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	startTime := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())
	endTime := time.Date(now.Year(), now.Month(), now.Day(), end.Hour(), end.Minute(), 0, 0, now.Location())

	return current.After(startTime) && current.Before(endTime)
}

func (w *Watcher) handleFailedItems(ctx context.Context) error {
	rows, err := w.db.QueryContext(ctx,
		"SELECT id, path FROM queue_items WHERE status = ?",
		StatusError,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.ErrorContext(ctx, "Error closing rows", "error", err)
		}
	}()

	// Collect items to delete
	type failedItem struct {
		id   int64
		path string
	}
	var itemsToDelete []failedItem

	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return err
		}

		// Check if file still exists
		if _, err := os.Stat(path); err == nil {
			// File exists, collect for deletion
			itemsToDelete = append(itemsToDelete, failedItem{id: id, path: path})
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Now delete the collected items
	for _, item := range itemsToDelete {
		if err := w.deleteItem(ctx, item.id); err != nil {
			return fmt.Errorf("error deleting failed item %d: %w", item.id, err)
		}
	}

	return nil
}
