package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
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
	cfg           config.WatcherConfig
	queue         queue.QueueInterface
	processor     ProcessorInterface
	watchFolder   string
	fileSizeCache map[string]fileCacheEntry
	cacheMutex    sync.RWMutex
}

type fileCacheEntry struct {
	size      int64
	timestamp time.Time
}

func New(
	cfg config.WatcherConfig,
	q queue.QueueInterface,
	processor ProcessorInterface,
	watchFolder string,
) *Watcher {
	return &Watcher{
		cfg:           cfg,
		queue:         q,
		processor:     processor,
		watchFolder:   watchFolder,
		fileSizeCache: make(map[string]fileCacheEntry),
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	slog.InfoContext(ctx, fmt.Sprintf("Starting directory watching %s with interval %v", w.watchFolder, w.cfg.CheckInterval))

	scanTicker := time.NewTicker(w.cfg.CheckInterval.ToDuration())
	defer scanTicker.Stop()

	// Cache cleanup ticker (runs every hour)
	cacheCleanupTicker := time.NewTicker(1 * time.Hour)
	defer cacheCleanupTicker.Stop()

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
		case <-cacheCleanupTicker.C:
			w.cleanupOldCacheEntries()
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

		// Add file to queue with duplicate checking
		// Always check for duplicates to prevent queue pollution, regardless of DeleteOriginalFile setting
		err = w.queue.AddFile(ctx, path, info.Size())

		if err != nil {
			slog.ErrorContext(ctx, "Error adding file to queue", "path", path, "error", err)
			return nil // Continue processing other files
		}

		slog.InfoContext(ctx, "Added file to queue", "path", filepath.Base(path), "size", info.Size())

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

	// Check if file is stable (not being written to)
	if !w.isFileStable(path, info) {
		return false
	}

	return true
}

// isFileStable checks if a file is stable (not being written to) using multiple methods
func (w *Watcher) isFileStable(path string, info os.FileInfo) bool {
	// Method 1: Check if file modification time is older than 2 seconds
	// This is the most reliable method for detecting actively written files
	if time.Since(info.ModTime()) < 2*time.Second {
		return false
	}

	// Method 2: Try to open the file exclusively to check if it's being used
	// This detects files that are open for writing by other processes
	if !w.canOpenFileExclusively(path) {
		return false
	}

	// Method 3: Check file size stability by comparing current size with cached size
	// This detects files that are still growing
	if !w.isFileSizeStable(path, info.Size()) {
		return false
	}

	return true
}

// canOpenFileExclusively attempts to open the file in exclusive mode to detect if it's in use
func (w *Watcher) canOpenFileExclusively(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		// If we can't open the file, assume it's being used
		return false
	}
	file.Close()
	return true
}

// isFileSizeStable checks if file size has remained constant
func (w *Watcher) isFileSizeStable(path string, currentSize int64) bool {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()

	cachedEntry, exists := w.fileSizeCache[path]
	w.fileSizeCache[path] = fileCacheEntry{
		size:      currentSize,
		timestamp: time.Now(),
	}

	// If this is the first time we see this file, it's not stable yet
	if !exists {
		return false
	}

	// If size changed, file is not stable
	if cachedEntry.size != currentSize {
		return false
	}

	return true
}

// cleanupOldCacheEntries removes cache entries older than 24 hours to prevent memory leaks
func (w *Watcher) cleanupOldCacheEntries() {
	w.cacheMutex.Lock()
	defer w.cacheMutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for path, entry := range w.fileSizeCache {
		if entry.timestamp.Before(cutoff) {
			delete(w.fileSizeCache, path)
		}
	}
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
