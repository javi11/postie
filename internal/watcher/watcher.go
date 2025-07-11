package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/queue"
	"github.com/opencontainers/selinux/pkg/pwalkdir"
)

// ProcessorInterface defines the interface for checking running jobs
type ProcessorInterface interface {
	IsPathBeingProcessed(path string) bool
}

type Watcher struct {
	cfg          config.WatcherConfig
	queue        *queue.Queue
	processor    ProcessorInterface
	watchFolder  string
	eventEmitter func(eventName string, optionalData ...interface{})
}

func New(
	cfg config.WatcherConfig,
	queue *queue.Queue,
	processor ProcessorInterface,
	watchFolder string,
	eventEmitter func(eventName string, optionalData ...interface{}),
) *Watcher {
	return &Watcher{
		cfg:          cfg,
		queue:        queue,
		processor:    processor,
		watchFolder:  watchFolder,
		eventEmitter: eventEmitter,
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	slog.InfoContext(ctx, fmt.Sprintf("Starting directory watching %s with interval %s", w.watchFolder, w.cfg.CheckInterval))

	scanTicker := time.NewTicker(w.cfg.CheckInterval)
	defer scanTicker.Stop()

	// Start continuous directory scanning
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-scanTicker.C:
			if w.isWithinSchedule() {
				if err := w.scanDirectory(ctx); err != nil {
					slog.ErrorContext(ctx, "Error scanning directory", "error", err)
				}
			} else {
				slog.Info("Not within schedule, skipping scan")
			}
		}
	}
}

func (w *Watcher) isWithinSchedule() bool {
	if w.cfg.Schedule.StartTime == "" || w.cfg.Schedule.EndTime == "" {
		return true
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	startTime := w.cfg.Schedule.StartTime
	endTime := w.cfg.Schedule.EndTime

	// Handle crossing midnight
	if startTime <= endTime {
		return currentTime >= startTime && currentTime <= endTime
	} else {
		return currentTime >= startTime || currentTime <= endTime
	}
}

func (w *Watcher) scanDirectory(ctx context.Context) error {
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

		// Skip files that don't meet criteria
		if !w.shouldProcessFile(path, info) {
			return nil
		}

		// Check if file is currently being processed
		if w.processor != nil && w.processor.IsPathBeingProcessed(path) {
			slog.InfoContext(ctx, "File is currently being processed, ignoring", "path", path)
			return nil
		}

		// Add file to queue
		// If delete original file is enabled, skip duplicate check since files are deleted after processing
		if w.cfg.DeleteOriginalFile {
			err = w.queue.AddFileWithoutDuplicateCheck(ctx, path, info.Size())
		} else {
			err = w.queue.AddFile(ctx, path, info.Size())
		}

		if err != nil {
			slog.ErrorContext(ctx, "Error adding file to queue", "path", path, "error", err)
			return nil // Continue processing other files
		}

		slog.InfoContext(ctx, "Added file to queue", "path", filepath.Base(path), "size", info.Size())

		// Emit queue update event
		if w.eventEmitter != nil {
			w.eventEmitter("queue-updated")
		}

		return nil
	})
}

func (w *Watcher) shouldProcessFile(path string, info os.FileInfo) bool {
	// Check file size threshold
	if info.Size() < int64(w.cfg.MinFileSize) {
		return false
	}

	// Check ignore patterns
	for _, pattern := range w.cfg.IgnorePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			slog.Warn("Invalid pattern", "pattern", pattern, "error", err)
			continue
		}
		if matched {
			return false
		}
	}

	// Check size threshold
	if info.Size() < int64(w.cfg.SizeThreshold) {
		return false
	}

	return true
}

// TriggerScan triggers an immediate directory scan
func (w *Watcher) TriggerScan(ctx context.Context) {
	go func() {
		if w.isWithinSchedule() {
			if err := w.scanDirectory(ctx); err != nil {
				slog.ErrorContext(ctx, "Error in triggered directory scan", "error", err)
			}
		}
	}()
}

// GetQueueItems returns queue items via the queue
func (w *Watcher) GetQueueItems() ([]queue.QueueItem, error) {
	return w.queue.GetQueueItems()
}

// RemoveFromQueue removes an item from the queue via queue
func (w *Watcher) RemoveFromQueue(id string) error {
	return w.queue.RemoveFromQueue(id)
}

// ClearQueue removes all completed and failed items from the queue via queue
func (w *Watcher) ClearQueue() error {
	return w.queue.ClearQueue()
}

// GetQueueStats returns statistics about the queue via queue
func (w *Watcher) GetQueueStats() (map[string]interface{}, error) {
	return w.queue.GetQueueStats()
}

// Close does nothing for the simple watcher (queue is managed separately)
func (w *Watcher) Close() error {
	return nil
}
