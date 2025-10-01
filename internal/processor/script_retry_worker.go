package processor

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/queue"
)

// ScriptRetryWorker handles retrying failed post-upload script executions
type ScriptRetryWorker struct {
	queue               *queue.Queue
	scriptConfig        config.PostUploadScriptConfig
	ctx                 context.Context
	cancel              context.CancelFunc
	retryCheckInterval  time.Duration
}

// NewScriptRetryWorker creates a new script retry worker
func NewScriptRetryWorker(ctx context.Context, queue *queue.Queue, scriptConfig config.PostUploadScriptConfig) *ScriptRetryWorker {
	workerCtx, cancel := context.WithCancel(ctx)

	return &ScriptRetryWorker{
		queue:              queue,
		scriptConfig:       scriptConfig,
		ctx:                workerCtx,
		cancel:             cancel,
		retryCheckInterval: 1 * time.Minute, // Check for retries every minute
	}
}

// Start begins the retry worker loop
func (w *ScriptRetryWorker) Start() {
	if !w.scriptConfig.Enabled {
		slog.Info("Post-upload script retry worker not started (script execution disabled)")
		return
	}

	slog.Info("Starting post-upload script retry worker", "checkInterval", w.retryCheckInterval)

	go w.run()
}

// Stop stops the retry worker
func (w *ScriptRetryWorker) Stop() {
	slog.Info("Stopping post-upload script retry worker")
	w.cancel()
}

// run is the main worker loop
func (w *ScriptRetryWorker) run() {
	ticker := time.NewTicker(w.retryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			slog.Info("Script retry worker stopped")
			return
		case <-ticker.C:
			w.processRetries()
		}
	}
}

// processRetries checks for and processes pending script retries
func (w *ScriptRetryWorker) processRetries() {
	ctx := w.ctx

	// Get items that need retry (limit to 10 at a time to avoid overwhelming the system)
	items, err := w.queue.GetItemsForScriptRetry(ctx, 10)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get items for script retry", "error", err)
		return
	}

	if len(items) == 0 {
		return
	}

	slog.InfoContext(ctx, "Processing script retries", "count", len(items))

	for _, item := range items {
		// Execute the script for this item
		if err := w.executeScript(ctx, item); err != nil {
			slog.ErrorContext(ctx, "Script retry failed", "itemID", item.ID, "nzbPath", item.NzbPath, "error", err)
		}
	}
}

// executeScript executes the post-upload script for a specific item
func (w *ScriptRetryWorker) executeScript(ctx context.Context, item queue.CompletedItem) error {
	slog.InfoContext(ctx, "Retrying post-upload script", "itemID", item.ID, "nzbPath", item.NzbPath)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, w.scriptConfig.Timeout.ToDuration())
	defer cancel()

	// Replace {nzb_path} placeholder with actual NZB path
	command := strings.ReplaceAll(w.scriptConfig.Command, "{nzb_path}", item.NzbPath)

	// Parse command using appropriate shell for the OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", command)
	}
	cmd.Dir = filepath.Dir(item.NzbPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := fmt.Sprintf("script failed: %v, output: %s", err, string(output))
		slog.ErrorContext(ctx, "Script execution failed during retry", "itemID", item.ID, "error", err, "output", string(output))

		// Get current retry count from the item
		currentRetryCount := item.ScriptRetryCount
		newRetryCount := currentRetryCount + 1

		// Check if we've exceeded max retries
		if newRetryCount >= w.scriptConfig.MaxRetries {
			// Mark as permanently failed
			if updateErr := w.queue.MarkScriptFailed(ctx, item.ID, errorMsg); updateErr != nil {
				slog.ErrorContext(ctx, "Failed to mark script as permanently failed", "itemID", item.ID, "error", updateErr)
			}
			slog.WarnContext(ctx, "Script permanently failed after max retries", "itemID", item.ID, "retries", newRetryCount)
			return fmt.Errorf("script failed permanently after %d retries: %w", newRetryCount, err)
		}

		// Calculate next retry with exponential backoff
		baseDelay := w.scriptConfig.RetryDelay.ToDuration()
		backoffDelay := baseDelay * time.Duration(1<<newRetryCount) // Exponential: base * 2^retryCount
		nextRetry := time.Now().Add(backoffDelay)

		// Update status for next retry
		if updateErr := w.queue.UpdateScriptStatus(ctx, item.ID, "pending_retry", newRetryCount, errorMsg, &nextRetry); updateErr != nil {
			slog.ErrorContext(ctx, "Failed to update script status for retry", "itemID", item.ID, "error", updateErr)
		}

		slog.InfoContext(ctx, "Scheduled script retry", "itemID", item.ID, "retryCount", newRetryCount, "nextRetry", nextRetry)
		return fmt.Errorf("script retry failed, will retry in %v: %w", backoffDelay, err)
	}

	// Mark script as completed
	if err := w.queue.MarkScriptCompleted(ctx, item.ID); err != nil {
		slog.ErrorContext(ctx, "Failed to mark script as completed", "itemID", item.ID, "error", err)
		return err
	}

	slog.InfoContext(ctx, "Post-upload script executed successfully on retry", "itemID", item.ID, "output", string(output))
	return nil
}
