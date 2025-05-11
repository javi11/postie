package watcher

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/pkg/fileinfo"
	"github.com/javi11/postie/pkg/postie"
	_ "github.com/mattn/go-sqlite3"
)

type Watcher struct {
	cfg          config.WatcherConfig
	postie       *postie.Postie
	db           *sql.DB
	isRunning    bool
	runningMux   sync.Mutex
	watchFolder  string
	outputFolder string
}

type QueueItem struct {
	ID        int64
	Path      string
	Size      int64
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func New(
	ctx context.Context,
	cfg config.WatcherConfig,
	configPath string,
	postie *postie.Postie,
	watchFolder string,
	outputFolder string,
) (*Watcher, error) {
	dbPath := filepath.Join(filepath.Dir(configPath), "postie_queue.db")

	slog.InfoContext(ctx, fmt.Sprintf("Using database at %s", dbPath))

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := initDB(db); err != nil {
		return nil, err
	}

	return &Watcher{
		cfg:          cfg,
		postie:       postie,
		db:           db,
		watchFolder:  watchFolder,
		outputFolder: outputFolder,
	}, nil
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS queue_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			size INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (w *Watcher) resetRunningItems(ctx context.Context) error {
	_, err := w.db.ExecContext(ctx, `
		UPDATE queue_items 
		SET status = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE status = ?`, StatusPending, StatusRunning)
	return err
}

func (w *Watcher) Start(ctx context.Context) error {
	// Reset any running items to pending
	if err := w.resetRunningItems(ctx); err != nil {
		return fmt.Errorf("error resetting running items: %w", err)
	}

	ticker := time.NewTicker(w.cfg.CheckInterval)
	defer ticker.Stop()

	slog.InfoContext(ctx, fmt.Sprintf("Starting watching %s with interval %s", w.watchFolder, w.cfg.CheckInterval))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.processQueue(ctx); err != nil {
				return err
			}
		}
	}
}

func (w *Watcher) processQueue(ctx context.Context) error {
	if !w.isWithinSchedule() || w.isRunning {
		slog.InfoContext(ctx, "Not within schedule or already running")

		return nil
	}

	w.runningMux.Lock()
	defer w.runningMux.Unlock()

	w.isRunning = true
	defer func() {
		w.isRunning = false
	}()

	// Scan directory for new files
	if err := w.scanDirectory(ctx); err != nil {
		return fmt.Errorf("error scanning directory: %w", err)
	}

	// Get next batch of items to process
	items, err := w.getNextBatch(ctx)
	if err != nil {
		return fmt.Errorf("error getting next batch: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	// Process each item
	for _, item := range items {
		if err := w.processItem(ctx, item); err != nil {
			slog.ErrorContext(ctx, "Error processing item", "error", err, "path", item.Path)
			if err := w.updateItemStatus(ctx, item.ID, StatusError); err != nil {
				slog.ErrorContext(ctx, "Error updating item status", "error", err)
			}
			continue
		}

		// Delete the item from queue after successful processing
		if err := w.deleteItem(ctx, item.ID); err != nil {
			slog.ErrorContext(ctx, "Error deleting item from queue", "error", err)
		}
	}

	return nil
}

func (w *Watcher) processItem(ctx context.Context, item QueueItem) error {
	// Update status to running
	if err := w.updateItemStatus(ctx, item.ID, StatusRunning); err != nil {
		return err
	}

	// Create file info
	fileInfo := fileinfo.FileInfo{
		Path: item.Path,
		Size: uint64(item.Size),
	}

	// Post the file
	if err := w.postie.Post(ctx, []fileinfo.FileInfo{fileInfo}, w.watchFolder, w.outputFolder); err != nil {
		return err
	}

	// Delete the original file
	if err := os.Remove(item.Path); err != nil {
		return fmt.Errorf("error deleting original file: %w", err)
	}

	return nil
}

func (w *Watcher) Close() error {
	return w.db.Close()
}
