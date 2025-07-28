package poster

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressCallback defines the interface for progress notifications
type ProgressCallback func(stage string, current, total int64, details string, speed float64, secondsLeft float64, elapsedTime float64)

// ProgressUpdate represents a progress update event
type ProgressUpdate struct {
	bytesProcessed    int64
	articlesProcessed int64
	articleErrors     int64
	finished          bool
}

// ProgressManager handles progress tracking and display
type ProgressManager struct {
	bar            *progressbar.ProgressBar
	articlesTotal  int64
	callback       ProgressCallback
	description    string
	updateChan     chan ProgressUpdate
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	// Atomic counters for thread-safe updates
	bytesProcessed    int64
	articlesProcessed int64
	articleErrors     int64
}

// NewFileProgress creates a new progress bar for a file
func NewFileProgress(
	description string,
	totalBytes int64,
	articlesTotal int64,
) *ProgressManager {
	return NewFileProgressWithCallback(description, totalBytes, articlesTotal, nil)
}

// NewFileProgressWithCallback creates a new progress bar for a file with a callback
func NewFileProgressWithCallback(
	description string,
	totalBytes int64,
	articlesTotal int64,
	callback ProgressCallback,
) *ProgressManager {
	bar := progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			_, _ = fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionSetMaxDetailRow(1),
	)

	ctx, cancel := context.WithCancel(context.Background())
	pm := &ProgressManager{
		bar:           bar,
		articlesTotal: articlesTotal,
		callback:      callback,
		description:   description,
		updateChan:    make(chan ProgressUpdate, 100), // Buffered channel to prevent blocking
		cancel:        cancel,
	}

	// Start the progress update goroutine
	pm.wg.Add(1)
	go pm.progressUpdateWorker(ctx)

	return pm
}

// UpdateFileProgress updates the progress bar for a file (non-blocking)
func (pm *ProgressManager) UpdateFileProgress(bytesProcessed int64, articlesProcessed int64, articleErrors int64) {
	// Update atomic counters
	atomic.StoreInt64(&pm.bytesProcessed, bytesProcessed)
	atomic.StoreInt64(&pm.articlesProcessed, articlesProcessed)
	atomic.StoreInt64(&pm.articleErrors, articleErrors)

	// Send update to channel (non-blocking)
	update := ProgressUpdate{
		bytesProcessed:    bytesProcessed,
		articlesProcessed: articlesProcessed,
		articleErrors:     articleErrors,
		finished:          false,
	}

	// Non-blocking send to prevent goroutine slowdown
	select {
	case pm.updateChan <- update:
		// Update sent successfully
	default:
		// Channel is full, try to drain one old update and send new one
		select {
		case <-pm.updateChan: // Remove one old update
			select {
			case pm.updateChan <- update: // Try to send new update
			default: // Still full, skip
			}
		default: // Nothing to drain
		}
	}
}

// AddProgress atomically adds to the progress counters (thread-safe)
func (pm *ProgressManager) AddProgress(bytesProcessed int64, articlesProcessed int64, articleErrors int64) {
	// Atomically add to counters
	newBytes := atomic.AddInt64(&pm.bytesProcessed, bytesProcessed)
	newArticles := atomic.AddInt64(&pm.articlesProcessed, articlesProcessed)
	newErrors := atomic.AddInt64(&pm.articleErrors, articleErrors)

	// Send update to channel (non-blocking)
	update := ProgressUpdate{
		bytesProcessed:    newBytes,
		articlesProcessed: newArticles,
		articleErrors:     newErrors,
		finished:          false,
	}

	// Non-blocking send
	select {
	case pm.updateChan <- update:
	default:
		// Channel is full, try to drain one old update and send new one
		select {
		case <-pm.updateChan: // Remove one old update
			select {
			case pm.updateChan <- update: // Try to send new update
			default: // Still full, skip
			}
		default: // Nothing to drain
		}
	}
}

// progressUpdateWorker handles progress updates in a separate goroutine
func (pm *ProgressManager) progressUpdateWorker(ctx context.Context) {
	defer pm.wg.Done()

	// Throttle updates to prevent excessive UI updates
	ticker := time.NewTicker(25 * time.Millisecond) // Update at most every 25ms for better visibility
	defer ticker.Stop()

	var lastUpdate ProgressUpdate
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-pm.updateChan:
			lastUpdate = update
			// Always update immediately on new data
			pm.updateProgressBar(update)
			if update.finished {
				return
			}
		case <-ticker.C:
			// Throttled update using atomic values
			currentUpdate := ProgressUpdate{
				bytesProcessed:    atomic.LoadInt64(&pm.bytesProcessed),
				articlesProcessed: atomic.LoadInt64(&pm.articlesProcessed),
				articleErrors:     atomic.LoadInt64(&pm.articleErrors),
				finished:          false,
			}
			// Only update if values changed
			if currentUpdate.bytesProcessed != lastUpdate.bytesProcessed ||
				currentUpdate.articlesProcessed != lastUpdate.articlesProcessed ||
				currentUpdate.articleErrors != lastUpdate.articleErrors {
				pm.updateProgressBar(currentUpdate)
				lastUpdate = currentUpdate
			}
		}
	}
}

// updateProgressBar updates the actual progress bar and calls callback
func (pm *ProgressManager) updateProgressBar(update ProgressUpdate) {
	speed := pm.bar.State().KBsPerSecond
	details := fmt.Sprintf("Articles: %d/%d | Errors: %d",
		update.articlesProcessed,
		pm.articlesTotal,
		update.articleErrors,
	)

	_ = pm.bar.AddDetail(details)
	_ = pm.bar.Set64(update.bytesProcessed)

	// Notify callback if available
	if pm.callback != nil {
		// Calculate seconds left from progress bar state
		secondsLeft := pm.bar.State().SecondsLeft
		elapsedTime := pm.bar.State().SecondsSince
		stage := "uploading"
		if update.finished {
			stage = "completed"
		}
		pm.callback(stage, update.bytesProcessed, int64(pm.bar.GetMax64()), details, speed, secondsLeft, elapsedTime)
	}
}

// FinishFileProgress completes the progress bar for a file
func (pm *ProgressManager) FinishFileProgress() {
	// Send final update
	finalUpdate := ProgressUpdate{
		bytesProcessed:    atomic.LoadInt64(&pm.bytesProcessed),
		articlesProcessed: atomic.LoadInt64(&pm.articlesProcessed),
		articleErrors:     atomic.LoadInt64(&pm.articleErrors),
		finished:          true,
	}

	// Cancel the worker and wait for it to finish first
	pm.cancel()
	pm.wg.Wait()

	// Update the progress bar one final time with final values
	pm.updateProgressBar(finalUpdate)

	// Safely finish the progress bar
	if pm.bar != nil {
		_ = pm.bar.Finish()
	}
}
