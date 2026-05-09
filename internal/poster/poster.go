package poster

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/javi11/nntppool/v4"
	"github.com/javi11/nxg"
	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/nzb"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/pausable"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/progress"
	"github.com/mnightingale/rapidyenc"
	concpool "github.com/sourcegraph/conc/pool"
)

var (
	// ErrPosterClosed is returned when attempting to post after the poster has been closed
	ErrPosterClosed = errors.New("poster is closed")
)

// bodyBufferPool provides reusable buffers for article body read-ahead to reduce GC pressure
var bodyBufferPool = sync.Pool{
	New: func() any {
		// Pre-allocate 768KB (slightly larger than default article size of 750KB)
		return make([]byte, 768*1024)
	},
}

// Poster defines the interface for posting articles to Usenet
type Poster interface {
	// Post posts files from a directory to Usenet
	Post(ctx context.Context, files []string, rootDir string, nzbGen nzb.NZBGenerator) error
	// PostWithRelativePaths posts files with custom display names (relative paths) for subjects
	// relativePaths maps absolute file path to the display name to use in the subject
	PostWithRelativePaths(ctx context.Context, files []string, rootDir string, nzbGen nzb.NZBGenerator, relativePaths map[string]string) error
	// PostWithCallback is the full-featured posting entry point. It returns as
	// soon as all uploads complete; the post-check propagation delay and STAT
	// verification run asynchronously on the poster's long-lived checkLoop.
	// completedItemID + onCheckExhausted let the caller persist failed-to-verify
	// articles to a deferred-check queue. Both may be empty/nil.
	PostWithCallback(ctx context.Context, files []string, rootDir string, nzbGen nzb.NZBGenerator, relativePaths map[string]string, completedItemID string, onCheckExhausted CheckExhaustedCallback) error
	// Stats returns posting statistics
	Stats() Stats
	// Close closes the poster
	Close()
}

// PostStatus represents the status of a post
type PostStatus int

const (
	PostStatusPending PostStatus = iota
	PostStatusPosted
	PostStatusVerified
	PostStatusFailed
	PostStatusCancelled
	PostStatusPosting
)

// CheckExhaustedCallback is invoked by the long-lived checkLoop when an
// uploaded file's articles cannot be verified after exhausting MaxRePost.
// It is called on a background goroutine, after Post() has already returned
// to the caller. The callback is responsible for persisting failed articles
// for deferred verification (e.g. via queue.AddPendingArticleChecks).
type CheckExhaustedCallback func(ctx context.Context, articles []FailedArticleInfo, totalArticles int, completedItemID string) error

// Post represents a file to be posted
type Post struct {
	FilePath string
	Articles []*article.Article
	Status   PostStatus
	Error    error
	Retries  int
	file     *os.File
	mu       sync.Mutex
	filesize int64
	wg       *sync.WaitGroup // per-call, drained when upload (not check) completes
	failed   *atomic.Int64
	progress progress.Progress
	// postedAt records when this file's upload finished. Used by checkLoop to
	// apply the propagation delay per-file with credit for time already elapsed.
	postedAt time.Time
	// errSink is the per-call channel that postLoop uses to surface fatal
	// upload errors back to the caller of Post(). Set by addPost. Never used
	// by checkLoop — check failures route through onCheckExhausted instead.
	errSink chan<- error
	// nzbGen is the per-call NZB generator. Set by addPost.
	nzbGen nzb.NZBGenerator
	// completedItemID identifies this post in the persistent queue. Used by
	// checkLoop to pass to onCheckExhausted on permanent verification failure.
	// Empty when the caller doesn't have a queue ID (e.g. CLI usage).
	completedItemID string
	// onCheckExhausted is invoked by checkLoop when articles exhaust MaxRePost.
	// May be nil — in that case checkLoop logs and drops the failure.
	onCheckExhausted CheckExhaustedCallback
	// retryParent links a retry post back to the original post's wg/failed/etc.
	// Retries do not increment the per-call wg (which counts only initial
	// uploads), so they need a way to find the original's metadata.
	retryParent *Post
}

// FailedArticleInfo contains information about an article that failed verification
// and should be deferred for later checking.
type FailedArticleInfo struct {
	MessageID string
	Groups    []string
}

// DeferredCheckError is a non-fatal error indicating some articles need deferred verification.
// The upload itself succeeded, but article verification (STAT check) failed after all immediate retries.
type DeferredCheckError struct {
	FailedArticles []FailedArticleInfo
	TotalArticles  int
}

func (e *DeferredCheckError) Error() string {
	return fmt.Sprintf("%d/%d articles deferred for later verification", len(e.FailedArticles), e.TotalArticles)
}

// articleWithBody holds an article with its pre-read body data for read-ahead buffering
type articleWithBody struct {
	article *article.Article
	body    []byte
	poolBuf []byte // original pooled buffer (may be larger than body)
}

// Stats tracks posting statistics
type Stats struct {
	ArticlesPosted  int64
	ArticlesChecked int64
	BytesPosted     int64
	ArticleErrors   int64
	StartTime       time.Time
	mu              sync.Mutex
}

// Poster handles posting articles to Usenet
type poster struct {
	cfg         config.PostingConfig
	checkCfg    config.PostCheck
	uploadPool  pool.NNTPClient // Pool for posting articles
	verifyPool  pool.NNTPClient // Pool for article verification (may be same as uploadPool)
	stats       *Stats
	throttle    *Throttle
	jobProgress progress.JobProgress
	closed      atomic.Bool
	closeOnce   sync.Once

	// Long-lived loop infrastructure. postLoop and checkLoop are spawned once
	// in New() and shared by every Post() call, so the propagation check no
	// longer blocks Post()'s critical path.
	postQueue  chan *Post
	checkQueue chan *Post // nil when post-check is disabled
	loopCtx    context.Context
	loopCancel context.CancelFunc
	loopsWG    sync.WaitGroup
	checkOn    bool // captured at construction; whether checkLoop is running
	loopsOnce  sync.Once
}

// New creates a new poster using dependency injection for the connection pool manager
func New(ctx context.Context, cfg config.Config, poolManager pool.PoolManager, jobProgress progress.JobProgress) (Poster, error) {
	if poolManager == nil {
		return nil, fmt.Errorf("pool manager cannot be nil")
	}

	// Get the upload pool from the manager
	uploadPool := poolManager.GetUploadPool()
	if uploadPool == nil {
		return nil, fmt.Errorf("connection pool is not available")
	}

	// Get the verify pool (may be same as upload pool if no verify-role servers configured)
	verifyPool := poolManager.GetVerifyPool()
	if verifyPool == nil {
		verifyPool = uploadPool
	}

	stats := &Stats{
		StartTime: time.Now(),
	}

	postCheckCfg := cfg.GetPostCheckConfig()
	checkOn := postCheckCfg.Enabled != nil && *postCheckCfg.Enabled

	// Loop ctx is detached from the caller's ctx so the loops outlive any
	// individual Post() call. They are torn down in Close().
	loopCtx, loopCancel := context.WithCancel(context.Background())

	p := &poster{
		cfg:         cfg.GetPostingConfig(),
		checkCfg:    postCheckCfg,
		uploadPool:  uploadPool,
		verifyPool:  verifyPool,
		stats:       stats,
		jobProgress: jobProgress,
		postQueue:   make(chan *Post, 100),
		loopCtx:     loopCtx,
		loopCancel:  loopCancel,
		checkOn:     checkOn,
	}
	if checkOn {
		p.checkQueue = make(chan *Post, 100)
	}

	// Log post check configuration for debugging
	if postCheckCfg.Enabled != nil {
		slog.DebugContext(ctx, "Poster initialized",
			"post_check_enabled", *postCheckCfg.Enabled,
			"max_repost", postCheckCfg.MaxRePost,
			"retry_delay", postCheckCfg.RetryDelay)
	} else {
		slog.WarnContext(ctx, "PostCheck.Enabled is nil, will default to true")
	}

	throttleRate := p.cfg.ThrottleRate
	if throttleRate > 0 {
		p.throttle = NewThrottle(throttleRate, time.Second)
	}

	p.ensureLoops()
	if checkOn {
		slog.DebugContext(ctx, "Post check enabled - started checkLoop goroutine")
	} else {
		slog.InfoContext(ctx, "Post check disabled - skipping article verification")
	}

	return p, nil
}

// ensureLoops lazily initializes per-poster channels/ctx and spawns postLoop
// (and checkLoop, if enabled) exactly once. Called by New() and again by the
// post entry points so hand-constructed posters (used in unit tests) work
// without explicit setup.
func (p *poster) ensureLoops() {
	p.loopsOnce.Do(func() {
		if p.postQueue == nil {
			p.postQueue = make(chan *Post, 100)
		}
		if p.loopCtx == nil {
			p.loopCtx, p.loopCancel = context.WithCancel(context.Background())
		}
		// Honor an explicit checkOn even if checkCfg.Enabled is nil — supports
		// tests that pre-set the field. Otherwise derive from config.
		if !p.checkOn && p.checkCfg.Enabled != nil && *p.checkCfg.Enabled {
			p.checkOn = true
		}
		if p.checkOn && p.checkQueue == nil {
			p.checkQueue = make(chan *Post, 100)
		}

		p.loopsWG.Add(1)
		go func() {
			defer p.loopsWG.Done()
			p.postLoop()
		}()
		if p.checkOn {
			p.loopsWG.Add(1)
			go func() {
				defer p.loopsWG.Done()
				p.checkLoop()
			}()
		}
	})
}

func (p *poster) Close() {
	p.closeOnce.Do(func() {
		p.closed.Store(true)

		// If loops were never spawned (hand-constructed poster never used),
		// nothing to drain.
		if p.postQueue == nil {
			return
		}

		slog.Info("Poster closing - draining loops")

		// Closing postQueue signals postLoop to drain pending posts and exit.
		// postLoop closes checkQueue on its way out, which signals checkLoop.
		close(p.postQueue)

		// Wait for both loops with a soft timeout. If they don't exit within
		// 30s (e.g. hung NNTP STAT), cancel loopCtx to force them out.
		done := make(chan struct{})
		go func() {
			p.loopsWG.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			slog.Warn("Poster Close: loops did not drain within 30s, cancelling")
			if p.loopCancel != nil {
				p.loopCancel()
			}
			<-done
		}

		if p.loopCancel != nil {
			p.loopCancel()
		}
		slog.Info("Poster closed")
	})
}

// Post posts files from a directory to Usenet
func (p *poster) Post(
	ctx context.Context,
	files []string,
	rootDir string,
	nzbGen nzb.NZBGenerator,
) error {
	return p.PostWithCallback(ctx, files, rootDir, nzbGen, nil, "", nil)
}

// PostWithRelativePaths posts files with custom display names (relative paths) for subjects.
// relativePaths maps absolute file path to the display name to use in the subject.
// If relativePaths is nil or a file is not found in the map, the filename is used.
func (p *poster) PostWithRelativePaths(
	ctx context.Context,
	files []string,
	rootDir string,
	nzbGen nzb.NZBGenerator,
	relativePaths map[string]string,
) error {
	return p.PostWithCallback(ctx, files, rootDir, nzbGen, relativePaths, "", nil)
}

// PostWithCallback submits files to the long-lived post pipeline and returns
// as soon as all uploads complete. Post-check verification runs asynchronously
// on the shared checkLoop and surfaces permanent failures via onCheckExhausted.
func (p *poster) PostWithCallback(
	ctx context.Context,
	files []string,
	rootDir string,
	nzbGen nzb.NZBGenerator,
	relativePaths map[string]string,
	completedItemID string,
	onCheckExhausted CheckExhaustedCallback,
) error {
	if p.closed.Load() {
		return ErrPosterClosed
	}

	if len(files) == 0 {
		return nil
	}

	// Lazy-init in case this poster was hand-constructed (tests).
	p.ensureLoops()

	var wg sync.WaitGroup
	var failedPosts atomic.Int64

	// errChan is per-call: postLoop reports fatal upload errors here. checkLoop
	// never writes to errChan — check failures route through onCheckExhausted.
	errChan := make(chan error, 4)

	wg.Add(len(files))
	added := 0
	for i, file := range files {
		select {
		case <-ctx.Done():
			// Drain wg for files we never enqueued.
			for j := added; j < len(files); j++ {
				wg.Done()
			}
			return ctx.Err()
		default:
		}

		displayName := ""
		if relativePaths != nil {
			displayName = relativePaths[file]
		}

		if err := p.addPost(ctx, file, displayName, i+1, len(files), &wg, &failedPosts, nzbGen, errChan, completedItemID, onCheckExhausted); err != nil {
			// addPost on its failure path may or may not have enqueued; if it
			// returned early before enqueuing, the wg slot it claimed is still
			// outstanding. Drain it here plus any remaining files.
			for j := added; j < len(files); j++ {
				wg.Done()
			}
			return fmt.Errorf("error adding file %s to posting queue: %w", file, err)
		}
		added++
	}

	// Wait for all uploads to complete, or for a fatal error.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		// Fatal upload error. The wg may not have drained; we leave the post
		// goroutines to handle their own cleanup via the per-Post errSink (the
		// caller's ctx is the source of truth for cancellation).
		return err
	case <-done:
	}

	if n := failedPosts.Load(); n > 0 {
		return fmt.Errorf("failed to post %d files", n)
	}
	return nil
}

// postLoop processes posts from the long-lived poster.postQueue. It reads
// per-call state (errSink, wg, failed, nzbGen) off the Post struct so a
// single shared loop can serve every Post() call without coupling.
func (p *poster) postLoop() {
	ctx := p.loopCtx
	// Close the checkQueue when this loop exits so checkLoop drains naturally.
	if p.checkQueue != nil {
		defer close(p.checkQueue)
	}

	numOfConnections := 0
	for _, pr := range p.uploadPool.Stats().Providers {
		numOfConnections += pr.MaxConnections
	}

	for post := range p.postQueue {
		select {
		case <-ctx.Done():
			p.failPostUpload(ctx, post, ctx.Err())
			// Drain the rest of the queue, failing each post, so callers' wg
			// counters decrement and Post() returns rather than hanging.
			for rest := range p.postQueue {
				p.failPostUpload(ctx, rest, ctx.Err())
			}
			return
		default:
			// Check if we should pause before processing the post
			if err := pausable.CheckPause(ctx); err != nil {
				p.failPostUpload(ctx, post, err)
				continue
			}

			// Set post status to Posting
			post.mu.Lock()
			post.Status = PostStatusPosting
			post.mu.Unlock()

			// Per-post context so the read-ahead goroutine always terminates
			// when this post's block exits, even if the parent ctx is still
			// alive (e.g. on the deferred-check non-fatal-error path).
			postCtx, postCancel := context.WithCancel(ctx)

			// Create read-ahead channel (buffer 50 articles ahead to overlap I/O with network)
			readAheadChan := make(chan articleWithBody, 50)

			// Start read-ahead goroutine to pre-read article bodies
			go func() {
				defer close(readAheadChan)
				for _, art := range post.Articles {
					select {
					case <-postCtx.Done():
						return
					default:
						// Get buffer from pool, resize if needed
						poolBuf := bodyBufferPool.Get().([]byte)
						if cap(poolBuf) < int(art.Size) {
							// Buffer too small, allocate a larger one (won't go back to pool)
							poolBuf = make([]byte, art.Size)
						}
						body := poolBuf[:art.Size]

						if _, err := post.file.ReadAt(body, art.Offset); err != nil {
							// Return buffer to pool on error
							bodyBufferPool.Put(poolBuf[:cap(poolBuf)]) //nolint:staticcheck // SA6002: slices have pointer semantics, no wrapper needed
							slog.ErrorContext(ctx, "Error pre-reading article", "error", err, "offset", art.Offset)
							return
						}

						select {
						case readAheadChan <- articleWithBody{article: art, body: body, poolBuf: poolBuf}:
						case <-postCtx.Done():
							// Return buffer to pool if context cancelled
							bodyBufferPool.Put(poolBuf[:cap(poolBuf)]) //nolint:staticcheck // SA6002: slices have pointer semantics, no wrapper needed
							return
						}
					}
				}
			}()

			// Create a pool with error handling - use all available connections
			// Note: We intentionally don't use WithCancelOnError() to prevent cascading failures
			// when a single article fails (e.g., TLS timeout). Other articles should continue.
			pool := concpool.New().WithContext(ctx).WithMaxGoroutines(numOfConnections).WithFirstError()

			// Collect completed articles for batch NZB addition (reduces lock contention)
			var completedArticles []*article.Article
			var completedMu sync.Mutex

			// Consume from read-ahead channel and submit to posting pool
			for artWithBody := range readAheadChan {
				art := artWithBody.article
				body := artWithBody.body
				poolBuf := artWithBody.poolBuf

				pool.Go(func(ctx context.Context) error {
					// Return buffer to pool when done (even on error)
					defer func() {
						if poolBuf != nil {
							bodyBufferPool.Put(poolBuf[:cap(poolBuf)]) //nolint:staticcheck // SA6002: slices have pointer semantics, no wrapper needed
						}
					}()

					if ctx.Err() != nil {
						return ctx.Err()
					}

					// Check for pause before processing article
					if err := pausable.CheckPause(ctx); err != nil {
						return err
					}

					if err := p.postArticleWithBody(ctx, art, body); err != nil {
						return err
					}

					// Update progress if manager is available
					post.progress.UpdateProgress(int64(art.Size))

					// Collect completed article for batch NZB addition
					completedMu.Lock()
					completedArticles = append(completedArticles, art)
					completedMu.Unlock()

					return nil
				})
			}

			// Wait for all workers to complete and collect errors
			errs := pool.Wait()

			// Read-ahead goroutine has finished (readAheadChan was drained above)
			// but cancel the per-post ctx so any straggler observes Done.
			postCancel()

			// Batch add completed articles to NZB generator (reduces lock contention)
			if post.nzbGen != nil {
				for _, art := range completedArticles {
					post.nzbGen.AddArticle(art)
				}
			}

			p.jobProgress.FinishProgress(post.progress.GetID())

			if errs != nil {
				post.mu.Lock()
				if errors.Is(errs, context.Canceled) {
					post.Status = PostStatusCancelled
					post.Error = fmt.Errorf("posting cancelled: %v", errs)
				} else {
					post.Status = PostStatusFailed
					post.Error = fmt.Errorf("failed to post articles: %v", errs)
				}
				post.mu.Unlock()

				// Close the underlying file so the descriptor isn't leaked on
				// failure. Long-running daemons that hit intermittent NNTP
				// errors otherwise exhaust the process fd ulimit and stall.
				if post.file != nil {
					if cerr := post.file.Close(); cerr != nil {
						slog.WarnContext(ctx, "Error closing file handle on post failure", "error", cerr, "file", post.FilePath)
					}
				}

				// Account the failure: bump the per-call failed counter and
				// report the error on the per-call errSink (best-effort, the
				// caller may have already returned). Then release the wg slot.
				if post.failed != nil && post.retryParent == nil {
					post.failed.Add(1)
				}
				if !errors.Is(errs, context.Canceled) && post.errSink != nil {
					select {
					case post.errSink <- fmt.Errorf("failed to post file %s after %d retries: %v", post.FilePath, p.cfg.MaxRetries, errs):
					default:
					}
				}
				if post.retryParent == nil && post.wg != nil {
					post.wg.Done()
				}
				continue
			}

			post.mu.Lock()
			post.Status = PostStatusPosted
			post.postedAt = time.Now()
			post.mu.Unlock()

			// Upload succeeded — release the per-call wg slot now (was
			// previously delayed until after check). For retries (retryParent
			// != nil) the original post already released the slot.
			if post.retryParent == nil && post.wg != nil {
				post.wg.Done()
			}

			if p.checkOn && p.checkQueue != nil {
				select {
				case p.checkQueue <- post:
				case <-ctx.Done():
					if post.file != nil {
						_ = post.file.Close()
					}
				}
				continue
			}

			// No check configured — close file and we're done.
			if post.file != nil {
				if err := post.file.Close(); err != nil {
					slog.WarnContext(ctx, "Error closing file handle", "error", err, "file", post.FilePath)
				}
			}
		}
	}
}

// failPostUpload is invoked when postLoop cannot upload a post (e.g. ctx
// cancellation drained from the queue before processing). It releases the
// per-call wg slot so Post() returns rather than hanging.
func (p *poster) failPostUpload(ctx context.Context, post *Post, cause error) {
	if post == nil {
		return
	}
	if post.file != nil {
		_ = post.file.Close()
	}
	if post.failed != nil && post.retryParent == nil {
		post.failed.Add(1)
	}
	if post.errSink != nil && !errors.Is(cause, context.Canceled) {
		select {
		case post.errSink <- fmt.Errorf("post %s aborted: %w", post.FilePath, cause):
		default:
		}
	}
	if post.retryParent == nil && post.wg != nil {
		post.wg.Done()
	}
	_ = ctx // accepted for symmetry with future logging; unused otherwise
}

// checkLoop processes posts from the long-lived poster.checkQueue. It is
// fully decoupled from any individual Post() call: by the time a post arrives
// here, postLoop has already released the per-call wg slot, so checkLoop
// failures cannot block the caller. Permanent verification failures route
// through post.onCheckExhausted (set per-call via Post struct) instead of an
// error channel.
func (p *poster) checkLoop() {
	ctx := p.loopCtx

	numOfConnections := 0
	for _, pr := range p.verifyPool.Stats().Providers {
		numOfConnections += pr.MaxConnections
	}

	deferredEnabled := p.checkCfg.DeferredCheckDelay.ToDuration() > 0

	for post := range p.checkQueue {
		select {
		case <-ctx.Done():
			// Drain remaining posts so file descriptors close cleanly.
			if post.file != nil {
				_ = post.file.Close()
			}
			for rest := range p.checkQueue {
				if rest.file != nil {
					_ = rest.file.Close()
				}
			}
			return
		default:
			if err := pausable.CheckPause(ctx); err != nil {
				slog.WarnContext(ctx, "checkLoop paused error", "error", err)
				if post.file != nil {
					_ = post.file.Close()
				}
				continue
			}

			// Create the progress task immediately so the user sees "checking" status
			// during the RetryDelay wait, preventing the queue item from appearing stuck.
			post.progress = p.jobProgress.AddProgress(uuid.New(), fmt.Sprintf("%s (check)", filepath.Base(post.FilePath)), progress.ProgressTypeChecking, post.filesize)

			// Wait for this file's articles to propagate to the verify server before
			// checking. We sleep only the remainder of RetryDelay that hasn't already
			// elapsed since this file finished posting — files that sat in the queue
			// long enough incur no extra wait, while files checked right after upload
			// still get the full propagation grace period.
			if delay := p.checkCfg.RetryDelay.ToDuration(); delay > 0 {
				post.mu.Lock()
				postedAt := post.postedAt
				post.mu.Unlock()

				var remaining time.Duration
				if postedAt.IsZero() {
					// Defensive: postedAt should always be set by postLoop, but if
					// it isn't (e.g. unusual code paths) wait the full delay.
					remaining = delay
				} else {
					remaining = delay - time.Since(postedAt)
				}

				if remaining > 0 {
					post.progress.SetWaitDeadline(time.Now().Add(remaining))
					select {
					case <-ctx.Done():
						post.progress.SetWaitDeadline(time.Time{})
						if post.file != nil {
							_ = post.file.Close()
						}
						return
					case <-time.After(remaining):
					}
					post.progress.SetWaitDeadline(time.Time{})
				}
			}

			// Create a pool with error handling - use all available connections.
			// Don't WithCancelOnError(): per-article failures should not abort
			// the rest; we collect failures and retry/defer at file granularity.
			pool := concpool.New().WithContext(ctx).WithMaxGoroutines(numOfConnections).WithFirstError()
			var failedArticles []*article.Article
			var mu sync.Mutex

			for _, art := range post.Articles {
				pool.Go(func(ctx context.Context) error {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if err := pausable.CheckPause(ctx); err != nil {
						return err
					}
					if err := p.checkArticle(ctx, art); err != nil {
						mu.Lock()
						failedArticles = append(failedArticles, art)
						mu.Unlock()
						return err
					}
					post.progress.UpdateProgress(int64(art.Size))
					return nil
				})
			}
			poolErr := pool.Wait()

			if len(failedArticles) > 0 {
				post.mu.Lock()
				post.Retries++
				retryCount := post.Retries
				post.mu.Unlock()

				// If we still have retries left, re-post the failed articles.
				if retryCount < int(p.checkCfg.MaxRePost) {
					// Refresh article headers before re-posting (preserve MessageID;
					// the NZB generator keys entries by (filename, messageID)).
					for _, art := range failedArticles {
						if art.From == "" {
							if defaultFrom := p.cfg.PostHeaders.DefaultFrom; defaultFrom != "" {
								art.From = defaultFrom
							} else if from, err := article.GenerateFrom(); err == nil {
								art.From = from
							}
						}
						if art.Subject == "" {
							art.Subject = article.GenerateRandomSubject()
						}
						if len(art.Groups) == 0 {
							slog.WarnContext(ctx, "Retried article has no newsgroups", "messageID", art.MessageID)
						}
					}

					// Build the retry post. retryParent links it back to the
					// original so postLoop skips wg/failed bookkeeping (already
					// accounted for on the original).
					failedPost := &Post{
						FilePath:         post.FilePath,
						Articles:         failedArticles,
						Status:           PostStatusPending,
						file:             post.file,
						filesize:         post.filesize,
						wg:               nil, // retries don't gate the per-call wg
						failed:           nil,
						Retries:          retryCount,
						progress:         p.jobProgress.AddProgress(uuid.New(), fmt.Sprintf("%s (retry)", filepath.Base(post.FilePath)), progress.ProgressTypeUploading, post.filesize),
						errSink:          nil, // Post() has already returned; no caller to surface to
						nzbGen:           post.nzbGen,
						completedItemID:  post.completedItemID,
						onCheckExhausted: post.onCheckExhausted,
						retryParent:      post,
					}

					slog.InfoContext(ctx, "Retrying failed articles",
						"file", post.FilePath,
						"attempt", retryCount,
						"max_retries", p.checkCfg.MaxRePost,
					)

					select {
					case p.postQueue <- failedPost:
					case <-ctx.Done():
						slog.WarnContext(ctx, "Context canceled while trying to send retry", "file", post.FilePath)
						if post.file != nil {
							_ = post.file.Close()
						}
						return
					}
					continue
				}

				// Max retries exhausted.
				p.jobProgress.FinishProgress(post.progress.GetID())
				if post.file != nil {
					if err := post.file.Close(); err != nil {
						slog.WarnContext(ctx, "Error closing file handle on verify failure", "error", err, "file", post.FilePath)
					}
				}

				// Build the failed-article info list for downstream handling.
				infos := make([]FailedArticleInfo, 0, len(failedArticles))
				for _, art := range failedArticles {
					infos = append(infos, FailedArticleInfo{
						MessageID: art.MessageID,
						Groups:    art.Groups,
					})
				}

				if deferredEnabled && post.onCheckExhausted != nil {
					slog.InfoContext(ctx, "Articles deferred for later verification",
						"file", post.FilePath,
						"deferred_count", len(infos),
						"retries_exhausted", p.checkCfg.MaxRePost,
					)
					post.mu.Lock()
					post.Status = PostStatusVerified // optimistic; deferred worker updates later
					post.mu.Unlock()
					if err := post.onCheckExhausted(ctx, infos, len(post.Articles), post.completedItemID); err != nil {
						slog.ErrorContext(ctx, "onCheckExhausted callback failed",
							"file", post.FilePath, "error", err)
					}
				} else {
					// No deferred path or no callback: log and drop. Post() has
					// already returned to the caller; we cannot retroactively
					// turn this into a synchronous error.
					post.mu.Lock()
					post.Status = PostStatusFailed
					post.Error = fmt.Errorf("failed to verify articles after %d retries", p.checkCfg.MaxRePost)
					post.mu.Unlock()
					slog.WarnContext(ctx, "Articles failed verification (no deferred handler)",
						"file", post.FilePath,
						"failed_count", len(infos),
						"retries_exhausted", p.checkCfg.MaxRePost,
					)
				}
				continue
			} else if poolErr != nil {
				// Pool error with no per-article failures collected — context
				// cancellation or a checkArticle bug. Log and move on.
				slog.WarnContext(ctx, "checkLoop pool error with no failed articles",
					"file", post.FilePath, "error", poolErr)
				if post.file != nil {
					_ = post.file.Close()
				}
				continue
			}

			// All articles verified.
			post.mu.Lock()
			post.Status = PostStatusVerified
			post.mu.Unlock()
			p.jobProgress.FinishProgress(post.progress.GetID())
			if post.file != nil {
				if err := post.file.Close(); err != nil {
					slog.WarnContext(ctx, "Error closing file handle", "error", err, "file", post.FilePath)
				}
			}
		}
	}
}

// addPost adds a file to the posting queue.
// displayName is the name to use in the subject (e.g., "Folder/subfolder/file.mp4").
// If displayName is empty, the filename is used.
func (p *poster) addPost(ctx context.Context, filePath string, displayName string, fileNumber int, totalFiles int, wg *sync.WaitGroup, failedPosts *atomic.Int64, nzbGen nzb.NZBGenerator, errSink chan<- error, completedItemID string, onCheckExhausted CheckExhaustedCallback) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("error getting file info: %w", err)
	}

	// Skip empty files - they have no segments to post
	if fileInfo.Size() == 0 {
		_ = file.Close()
		slog.WarnContext(ctx, "Skipping empty file", "path", filePath)
		return nil
	}

	// Calculate number of segments
	segmentSize := p.cfg.ArticleSizeInBytes
	numSegments := int((fileInfo.Size() + int64(segmentSize) - 1) / int64(segmentSize))
	nxgHeader := nxg.GenerateNXGHeader(int64(numSegments), 0)

	groups := make([]string, 0)

	switch p.cfg.GroupPolicy {
	case config.GroupPolicyEachFile:
		randomGroup := p.cfg.Groups[rand.Intn(len(p.cfg.Groups))]
		groups = append(groups, randomGroup.Name)
	case config.GroupPolicyAll:
		groups = make([]string, 0)
		for _, group := range p.cfg.Groups {
			groups = append(groups, group.Name)
		}
	}

	from, err := article.GenerateFrom()
	if err != nil {
		return fmt.Errorf("error generating from header: %w", err)
	}

	customHeaders := make(map[string]string)
	if len(p.cfg.PostHeaders.CustomHeaders) > 0 {
		for _, v := range p.cfg.PostHeaders.CustomHeaders {
			customHeaders[v.Name] = v.Value
		}
	}

	partType := nxg.PartTypeData
	if par2.IsPar2File(filePath) {
		partType = nxg.PartTypePar2
	}

	// Create articles for each segment
	articles := make([]*article.Article, 0, numSegments)
	for i := range numSegments {
		offset := int64(i) * int64(segmentSize)
		size := int64(segmentSize)
		if offset+size > fileInfo.Size() {
			size = fileInfo.Size() - offset
		}

		partNumber := i + 1

		messageID := ""
		if p.cfg.MessageIDFormat == config.MessageIDFormatRandom {
			msgID, err := article.GenerateMessageID()
			if err != nil {
				return fmt.Errorf("error generating message ID: %w", err)
			}

			messageID = msgID
		} else {
			msgID, err := nxgHeader.GenerateSegmentID(partType, int64(partNumber))
			if err != nil {
				return fmt.Errorf("error generating message ID: %w", err)
			}

			messageID = msgID
		}

		// Use displayName if provided, otherwise fall back to filename
		fileName := filepath.Base(filePath)
		subjectName := fileName
		if displayName != "" {
			subjectName = displayName
		}
		subject := article.GenerateSubject(fileNumber, totalFiles, subjectName, partNumber, numSegments)
		originalSubject := subject

		var fName string

		obfuscationPolicy := p.cfg.ObfuscationPolicy
		if par2.IsPar2File(filePath) {
			obfuscationPolicy = p.cfg.Par2ObfuscationPolicy
		}

		switch obfuscationPolicy {
		case config.ObfuscationPolicyNone:
			fName = fileName
		case config.ObfuscationPolicyFull:
			fName = article.GenerateRandomFilename()
			subject = article.GenerateRandomSubject()
		default:
			hasher := md5.New()
			_, _ = fmt.Fprintf(hasher, "%s%d", fileName, partNumber)
			fName = fmt.Sprintf("%x", hasher.Sum(nil))

			if p.cfg.MessageIDFormat == config.MessageIDFormatRandom {
				hasher := md5.New()
				_, _ = hasher.Write([]byte(subject))
				subject = fmt.Sprintf("%x", hasher.Sum(nil))
			} else {
				subject, err = nxgHeader.GetObfuscatedSubject(partType, int64(partNumber))
				if err != nil {
					return fmt.Errorf("error generating obfuscated subject: %w", err)
				}
			}
		}

		defaultFrom := p.cfg.PostHeaders.DefaultFrom
		if defaultFrom != "" {
			from = defaultFrom
		} else if p.cfg.ObfuscationPolicy == config.ObfuscationPolicyFull {
			from, err = article.GenerateFrom()
			if err != nil {
				return fmt.Errorf("error generating from header: %w", err)
			}
		}

		var xNxgHeader string
		if p.cfg.PostHeaders.AddNXGHeader &&
			p.cfg.ObfuscationPolicy != config.ObfuscationPolicyFull &&
			p.cfg.MessageIDFormat != config.MessageIDFormatNXG {
			xNxgHeader, err = nxgHeader.GetXNxgHeader(
				int64(fileNumber),
				int64(totalFiles),
				fileName,
				partType,
				fileInfo.Size(),
			)
			if err != nil {
				return fmt.Errorf("error generating xNxg header: %w", err)
			}
		}

		var date *time.Time
		if p.cfg.ObfuscationPolicy == config.ObfuscationPolicyFull {
			rd := article.RandomDateWithinLast6Hours()
			date = &rd
		}

		art := article.New(
			messageID,
			subject,
			originalSubject,
			from,
			groups,
			partNumber,
			numSegments,
			fileInfo.Size(),
			fName,
			fileNumber,
			fileName,
			customHeaders,
		)

		if date != nil {
			art.Date = *date
		}

		if xNxgHeader != "" {
			art.XNxgHeader = xNxgHeader
		}

		art.Offset = offset
		art.Size = uint64(size)

		articles = append(articles, art)
	}

	post := &Post{
		FilePath:         filePath,
		Articles:         articles,
		Status:           PostStatusPending,
		file:             file,
		filesize:         fileInfo.Size(),
		wg:               wg,
		failed:           failedPosts,
		progress:         p.jobProgress.AddProgress(uuid.New(), filepath.Base(filePath), progress.ProgressTypeUploading, fileInfo.Size()),
		errSink:          errSink,
		nzbGen:           nzbGen,
		completedItemID:  completedItemID,
		onCheckExhausted: onCheckExhausted,
	}

	// Send to the poster-lifetime postQueue. If the poster has been Closed
	// concurrently, the queue will be closed and the send will panic — guard
	// with a closed-check.
	if p.closed.Load() {
		_ = file.Close()
		return ErrPosterClosed
	}

	select {
	case p.postQueue <- post:
		return nil
	case <-ctx.Done():
		if err := file.Close(); err != nil {
			slog.WarnContext(ctx, "Error closing file after context cancellation", "error", err, "file", filePath)
		}
		return ctx.Err()
	}
}

// postArticle posts an article to Usenet (reads body from file)
func (p *poster) postArticle(ctx context.Context, article *article.Article, file *os.File) error {
	// Check if we should pause before posting
	if err := pausable.CheckPause(ctx); err != nil {
		return err
	}

	// Read article body
	body := make([]byte, article.Size)
	if _, err := file.ReadAt(body, article.Offset); err != nil {
		return fmt.Errorf("error reading article body: %w", err)
	}

	return p.postArticleWithBody(ctx, article, body)
}

// postArticleWithBody posts an article with pre-read body data
func (p *poster) postArticleWithBody(ctx context.Context, art *article.Article, body []byte) error {
	// Check if we should pause before posting
	if err := pausable.CheckPause(ctx); err != nil {
		return err
	}

	// Build PostHeaders for nntppool v4
	headers := nntppool.PostHeaders{
		From:       art.From,
		Subject:    art.Subject,
		Newsgroups: art.Groups,
		MessageID:  fmt.Sprintf("<%s>", art.MessageID),
		Date:       art.Date.UTC(),
		Extra:      make(map[string][]string),
	}

	// Add custom headers
	if art.CustomHeaders != nil {
		for k, v := range art.CustomHeaders {
			headers.Extra[k] = []string{v}
		}
	}

	// Add X-Nxg header if present
	if art.XNxgHeader != "" {
		headers.Extra["X-Nxg"] = []string{art.XNxgHeader}
	}

	// Build yEnc metadata
	meta := rapidyenc.Meta{
		FileName:   art.FileName,
		FileSize:   art.FileSize,
		PartNumber: int64(art.PartNumber),
		TotalParts: int64(art.TotalParts),
		Offset:     int64(art.Offset),
		PartSize:   int64(art.Size),
	}

	// Post article with timeout to prevent indefinite TLS hangs
	postCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Retry once on broken pipe: the pool may have returned a stale connection that
	// the server silently closed. After the broken pipe error, the pool discards that
	// connection; the retry picks a fresh one. bytes.NewReader(body) is cheap to
	// recreate so the body is fully re-readable on each attempt.
	var lastErr error
	for attempt := range 2 {
		if attempt > 0 {
			slog.WarnContext(ctx, "Retrying article post after broken pipe (stale pooled connection)",
				"messageID", art.MessageID)
		}

		_, lastErr = p.uploadPool.PostYenc(postCtx, headers, bytes.NewReader(body), meta)
		if lastErr == nil {
			break
		}

		if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) {
			return context.Canceled
		}

		if !isBrokenPipe(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return fmt.Errorf("error posting article: %w", lastErr)
	}

	// Apply throttling after posting
	if p.throttle != nil {
		p.throttle.Wait(int64(art.Size))
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.ArticlesPosted++
	p.stats.BytesPosted += int64(art.Size)
	p.stats.mu.Unlock()

	return nil
}

// isBrokenPipe reports whether err represents a broken-pipe / connection-reset
// condition, indicating that the remote server closed the TCP connection before
// the client finished writing (typically a stale idle pooled connection).
func isBrokenPipe(err error) bool {
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}

// checkArticle checks if an article exists using the check pool
func (p *poster) checkArticle(ctx context.Context, art *article.Article) error {
	// Check if we should pause before checking
	if err := pausable.CheckPause(ctx); err != nil {
		return err
	}

	// Use the dedicated verify pool for article verification
	// v4 Stat only takes messageID (no groups parameter)
	_, err := p.verifyPool.Stat(ctx, art.MessageID)
	if err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.ArticlesChecked++
	p.stats.mu.Unlock()

	return nil
}

// Stats returns posting statistics
func (p *poster) Stats() Stats {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	return Stats{
		ArticlesPosted:  p.stats.ArticlesPosted,
		ArticlesChecked: p.stats.ArticlesChecked,
		BytesPosted:     p.stats.BytesPosted,
		ArticleErrors:   p.stats.ArticleErrors,
		StartTime:       p.stats.StartTime,
	}
}
