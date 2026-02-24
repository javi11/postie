package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/queue"
)

// PostCheckRetryWorker handles deferred article verification via STAT checks.
// When immediate post-check verification fails after all retries, articles are
// stored in the database and this worker periodically rechecks them with
// exponential backoff.
type PostCheckRetryWorker struct {
	queue         *queue.Queue
	checkPool     pool.NNTPClient
	cfg           config.PostCheck
	ctx           context.Context
	cancel        context.CancelFunc
	checkInterval time.Duration
	initialDelay  time.Duration
	maxBackoff    time.Duration
	maxRetries    int
}

// NewPostCheckRetryWorker creates a new post check retry worker
func NewPostCheckRetryWorker(
	ctx context.Context,
	q *queue.Queue,
	checkPool pool.NNTPClient,
	cfg config.PostCheck,
) *PostCheckRetryWorker {
	workerCtx, cancel := context.WithCancel(ctx)

	checkInterval := cfg.DeferredCheckInterval.ToDuration()
	if checkInterval <= 0 {
		checkInterval = 2 * time.Minute
	}

	initialDelay := cfg.DeferredCheckDelay.ToDuration()
	if initialDelay <= 0 {
		initialDelay = 5 * time.Minute
	}

	maxBackoff := cfg.DeferredMaxBackoff.ToDuration()
	if maxBackoff <= 0 {
		maxBackoff = 1 * time.Hour
	}

	maxRetries := cfg.DeferredMaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}

	return &PostCheckRetryWorker{
		queue:         q,
		checkPool:     checkPool,
		cfg:           cfg,
		ctx:           workerCtx,
		cancel:        cancel,
		checkInterval: checkInterval,
		initialDelay:  initialDelay,
		maxBackoff:    maxBackoff,
		maxRetries:    maxRetries,
	}
}

// Start begins the retry worker loop
func (w *PostCheckRetryWorker) Start() {
	if w.cfg.Enabled == nil || !*w.cfg.Enabled {
		slog.Info("Post check retry worker not started (post check disabled)")
		return
	}

	slog.Info("Starting post check retry worker",
		"checkInterval", w.checkInterval,
		"initialDelay", w.initialDelay,
		"maxBackoff", w.maxBackoff,
		"maxRetries", w.maxRetries)

	go w.run()
}

// Stop stops the retry worker
func (w *PostCheckRetryWorker) Stop() {
	slog.Info("Stopping post check retry worker")
	w.cancel()
}

// run is the main worker loop
func (w *PostCheckRetryWorker) run() {
	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			slog.Info("Post check retry worker stopped")
			return
		case <-ticker.C:
			w.processRetries()
		}
	}
}

// processRetries checks for and processes pending article verifications
func (w *PostCheckRetryWorker) processRetries() {
	ctx := w.ctx

	// Get articles that need checking (limit to 50 at a time)
	articles, err := w.queue.GetArticlesForCheck(ctx, 50)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get articles for deferred check", "error", err)
		return
	}

	if len(articles) == 0 {
		return
	}

	slog.InfoContext(ctx, "Processing deferred article checks", "count", len(articles))

	// Track completed items that need status updates
	completedItems := make(map[string]bool)

	for _, article := range articles {
		if ctx.Err() != nil {
			return
		}

		completedItems[article.CompletedItemID] = true

		// Parse groups from JSON
		var groups []string
		if err := json.Unmarshal([]byte(article.Groups), &groups); err != nil {
			slog.ErrorContext(ctx, "Failed to parse groups JSON", "error", err, "articleID", article.ID)
			if markErr := w.queue.MarkArticleCheckFailed(ctx, article.ID); markErr != nil {
				slog.ErrorContext(ctx, "Failed to mark article as failed", "error", markErr)
			}
			continue
		}

		// Run STAT check
		verified := w.checkArticle(ctx, article.MessageID, groups)

		if verified {
			if err := w.queue.MarkArticleVerified(ctx, article.ID); err != nil {
				slog.ErrorContext(ctx, "Failed to mark article as verified", "error", err, "articleID", article.ID)
			} else {
				slog.DebugContext(ctx, "Article verified on deferred check",
					"messageID", article.MessageID, "retryCount", article.RetryCount)
			}
			continue
		}

		// Check if we should retry
		newRetryCount := article.RetryCount + 1
		if newRetryCount >= w.maxRetries {
			// Max retries exhausted - mark as failed
			if err := w.queue.MarkArticleCheckFailed(ctx, article.ID); err != nil {
				slog.ErrorContext(ctx, "Failed to mark article as failed", "error", err, "articleID", article.ID)
			}
			slog.WarnContext(ctx, "Article verification permanently failed",
				"messageID", article.MessageID,
				"retries", newRetryCount,
				"maxRetries", w.maxRetries)
			continue
		}

		// Schedule next retry with exponential backoff
		backoff := w.calculateBackoff(newRetryCount)
		nextRetry := time.Now().Add(backoff)

		if err := w.queue.UpdateArticleCheckRetry(ctx, article.ID, newRetryCount, nextRetry); err != nil {
			slog.ErrorContext(ctx, "Failed to update article check retry", "error", err, "articleID", article.ID)
		} else {
			slog.DebugContext(ctx, "Scheduled article recheck",
				"messageID", article.MessageID,
				"retryCount", newRetryCount,
				"nextRetry", nextRetry,
				"backoff", backoff)
		}
	}

	// Update completed item verification statuses
	for completedItemID := range completedItems {
		w.updateCompletedItemStatus(ctx, completedItemID)
	}
}

// checkArticle verifies if an article exists on the server via STAT command
func (w *PostCheckRetryWorker) checkArticle(ctx context.Context, messageID string, groups []string) bool {
	// v4 Stat only takes messageID (no groups parameter)
	_, err := w.checkPool.Stat(ctx, messageID)
	return err == nil
}

// calculateBackoff calculates the exponential backoff delay
func (w *PostCheckRetryWorker) calculateBackoff(retryCount int) time.Duration {
	// Exponential backoff: initialDelay * 2^retryCount
	backoff := w.initialDelay * time.Duration(1<<retryCount)

	// Cap at max backoff
	if backoff > w.maxBackoff {
		return w.maxBackoff
	}

	return backoff
}

// updateCompletedItemStatus checks all articles for a completed item and updates its verification status
func (w *PostCheckRetryWorker) updateCompletedItemStatus(ctx context.Context, completedItemID string) {
	total, pending, failed, err := w.queue.GetPendingCheckCountForItem(ctx, completedItemID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get pending check counts", "error", err, "completedItemID", completedItemID)
		return
	}

	// If there are still pending checks, don't update status yet
	if pending > 0 {
		return
	}

	// All checks are done - determine final status
	var status string
	if failed > 0 {
		status = "verification_failed"
		slog.WarnContext(ctx, "Completed item verification failed",
			"completedItemID", completedItemID,
			"total", total, "failed", failed)
	} else {
		status = "verified"
		slog.InfoContext(ctx, "All deferred articles verified successfully",
			"completedItemID", completedItemID, "total", total)
	}

	if err := w.queue.UpdateCompletedItemVerificationStatus(ctx, completedItemID, status); err != nil {
		slog.ErrorContext(ctx, "Failed to update completed item verification status",
			"error", err, "completedItemID", completedItemID)
	}
}

// GetFailureReason returns a human-readable reason for why retries stopped
func (w *PostCheckRetryWorker) GetFailureReason(retryCount int) string {
	if retryCount >= w.maxRetries {
		return fmt.Sprintf("exceeded max retries of %d", w.maxRetries)
	}
	return "unknown reason"
}
