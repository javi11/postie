package poster

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/javi11/nntppool/v2"
	"github.com/javi11/nxg"
	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/nzb"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/pausable"
	"github.com/javi11/postie/internal/pool"
	"github.com/javi11/postie/internal/progress"
	concpool "github.com/sourcegraph/conc/pool"
)

var (
	// ErrPosterClosed is returned when attempting to post after the poster has been closed
	ErrPosterClosed = errors.New("poster is closed")
)

// bodyBufferPool provides reusable buffers for article body read-ahead to reduce GC pressure
var bodyBufferPool = sync.Pool{
	New: func() interface{} {
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
	wg       *sync.WaitGroup
	failed   *int
	progress progress.Progress
}

// articleWithBody holds an article with its pre-read body data for read-ahead buffering
type articleWithBody struct {
	article  *article.Article
	body     []byte
	poolBuf  []byte // original pooled buffer (may be larger than body)
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
	pool        nntppool.UsenetConnectionPool // Pool for posting articles
	checkPool   nntppool.UsenetConnectionPool // Pool for article verification (may be same as pool)
	stats       *Stats
	throttle    *Throttle
	jobProgress progress.JobProgress
	closed      atomic.Bool
	closeOnce   sync.Once
}

// New creates a new poster using dependency injection for the connection pool manager
func New(ctx context.Context, cfg config.Config, poolManager pool.PoolManager, jobProgress progress.JobProgress) (Poster, error) {
	if poolManager == nil {
		return nil, fmt.Errorf("pool manager cannot be nil")
	}

	// Get the posting pool from the manager
	nntpPool := poolManager.GetPool()
	if nntpPool == nil {
		return nil, fmt.Errorf("connection pool is not available")
	}

	// Get the check pool (may be same as posting pool if no check-only servers configured)
	checkPool := poolManager.GetCheckPool()
	if checkPool == nil {
		// Fall back to posting pool if check pool is not available
		checkPool = nntpPool
	}

	stats := &Stats{
		StartTime: time.Now(),
	}

	postCheckCfg := cfg.GetPostCheckConfig()
	p := &poster{
		cfg:         cfg.GetPostingConfig(),
		checkCfg:    postCheckCfg,
		pool:        nntpPool,
		checkPool:   checkPool,
		stats:       stats,
		jobProgress: jobProgress,
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

	return p, nil
}

func (p *poster) Close() {
	p.closeOnce.Do(func() {
		p.closed.Store(true)
		slog.Info("Poster closed - no new Post() calls will be accepted")
	})
}

// Post posts files from a directory to Usenet
func (p *poster) Post(
	ctx context.Context,
	files []string,
	rootDir string,
	nzbGen nzb.NZBGenerator,
) error {
	return p.PostWithRelativePaths(ctx, files, rootDir, nzbGen, nil)
}

// PostWithRelativePaths posts files with custom display names (relative paths) for subjects
// relativePaths maps absolute file path to the display name to use in the subject
// If relativePaths is nil or a file is not found in the map, the filename is used
func (p *poster) PostWithRelativePaths(
	ctx context.Context,
	files []string,
	rootDir string,
	nzbGen nzb.NZBGenerator,
	relativePaths map[string]string,
) error {
	// Check if poster has been closed
	if p.closed.Load() {
		return ErrPosterClosed
	}

	wg := sync.WaitGroup{}
	var failedPosts int

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create error channel to collect errors
	errChan := make(chan error, 1)

	// Create channels for post and check queues
	postQueue := make(chan *Post, 100)
	checkQueue := make(chan *Post, 100)

	// Track posts in flight (initial + retries) to know when all processing is complete
	var postsInFlight sync.WaitGroup

	// Start a goroutine to process posts
	go p.postLoop(ctx, postQueue, checkQueue, errChan, nzbGen, &postsInFlight)

	// Start a goroutine to process checks only if post check is enabled
	if p.checkCfg.Enabled != nil && *p.checkCfg.Enabled {
		go p.checkLoop(ctx, checkQueue, postQueue, errChan, nzbGen, &postsInFlight)
		slog.DebugContext(ctx, "Post check enabled - started checkLoop goroutine")
	} else {
		slog.InfoContext(ctx, "Post check disabled - skipping article verification")
	}

	wg.Add(len(files))
	for i, file := range files {
		// Check if context is canceled before adding more posts
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get the display name (relative path) for this file, or empty string to use filename
		displayName := ""
		if relativePaths != nil {
			displayName = relativePaths[file]
		}

		if err := p.addPost(ctx, file, displayName, i+1, len(files), &wg, &failedPosts, postQueue, nzbGen, &postsInFlight); err != nil {
			return fmt.Errorf("error adding file %s to posting queue: %w", file, err)
		}
	}

	// Close postQueue after all initial posts have been added
	// The checkLoop can still add retries back to the queue if needed
	close(postQueue)

	// Wait for all posts to complete or an error to occur
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		cancel() // Cancel the context to stop all operations
		return ctx.Err()
	case err := <-errChan:
		cancel() // Cancel the context to stop all operations
		return err
	case <-done:
		if failedPosts > 0 {
			return fmt.Errorf("failed to post %d files", failedPosts)
		}

		return nil
	}
}

// postLoop processes posts from the queue
func (p *poster) postLoop(ctx context.Context, postQueue chan *Post, checkQueue chan *Post, errChan chan<- error, nzbGen nzb.NZBGenerator, postsInFlight *sync.WaitGroup) {
	// Only close channels that this goroutine writes to
	defer close(checkQueue)

	numOfConnections := 0

	for _, pr := range p.pool.GetProvidersInfo() {
		numOfConnections += pr.MaxConnections
	}

	for post := range postQueue {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			// Check if we should pause before processing the post
			if err := pausable.CheckPause(ctx); err != nil {
				errChan <- err
				return
			}

			// Set post status to Posting
			post.mu.Lock()
			post.Status = PostStatusPosting
			post.mu.Unlock()

			// Create read-ahead channel (buffer 50 articles ahead to overlap I/O with network)
			readAheadChan := make(chan articleWithBody, 50)

			// Start read-ahead goroutine to pre-read article bodies
			go func() {
				defer close(readAheadChan)
				for _, art := range post.Articles {
					select {
					case <-ctx.Done():
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
							bodyBufferPool.Put(poolBuf[:cap(poolBuf)])
							slog.ErrorContext(ctx, "Error pre-reading article", "error", err, "offset", art.Offset)
							return
						}

						select {
						case readAheadChan <- articleWithBody{article: art, body: body, poolBuf: poolBuf}:
						case <-ctx.Done():
							// Return buffer to pool if context cancelled
							bodyBufferPool.Put(poolBuf[:cap(poolBuf)])
							return
						}
					}
				}
			}()

			// Create a pool with error handling - use all available connections
			pool := concpool.New().WithContext(ctx).WithMaxGoroutines(numOfConnections).WithCancelOnError().WithFirstError()

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
							bodyBufferPool.Put(poolBuf[:cap(poolBuf)])
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

			// Batch add completed articles to NZB generator (reduces lock contention)
			for _, art := range completedArticles {
				nzbGen.AddArticle(art)
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

				// Mark this post as done in the queue tracking
				postsInFlight.Done()

				if !errors.Is(errs, context.Canceled) {
					errChan <- fmt.Errorf("failed to post file %s after %d retries: %v", post.FilePath, p.cfg.MaxRetries, errs)
				}

				return
			}

			post.mu.Lock()
			post.Status = PostStatusPosted
			post.mu.Unlock()

			if p.checkCfg.Enabled != nil && *p.checkCfg.Enabled {
				checkQueue <- post

				continue
			}

			// Post complete without check - mark as done in queue tracking
			postsInFlight.Done()

			// Close file
			if post.file != nil {
				if err := post.file.Close(); err != nil {
					slog.WarnContext(ctx, "Error closing file handle", "error", err, "file", post.FilePath)
				}
			}

			post.wg.Done()
		}
	}
}

// checkLoop processes posts from the check queue
func (p *poster) checkLoop(ctx context.Context, checkQueue chan *Post, postQueue chan *Post, errChan chan<- error, nzbGen nzb.NZBGenerator, postsInFlight *sync.WaitGroup) {
	numOfConnections := 0

	// Use check pool's providers for connection count
	for _, pr := range p.checkPool.GetProvidersInfo() {
		numOfConnections += pr.MaxConnections
	}

	for post := range checkQueue {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			// Check if we should pause before processing the check
			if err := pausable.CheckPause(ctx); err != nil {
				errChan <- err
				return
			}
			// Create a pool with error handling - use all available CPU cores
			pool := concpool.New().WithContext(ctx).WithMaxGoroutines(numOfConnections).WithCancelOnError().WithFirstError()
			articlesChecked := 0
			articleErrors := 0
			var failedArticles []*article.Article
			var mu sync.Mutex

			post.progress = p.jobProgress.AddProgress(uuid.New(), fmt.Sprintf("%s (check)", filepath.Base(post.FilePath)), progress.ProgressTypeChecking, post.filesize)

			// Submit all articles to the pool
			for _, art := range post.Articles {
				pool.Go(func(ctx context.Context) error {
					if ctx.Err() != nil {
						return ctx.Err()
					}

					// Check for pause before checking article
					if err := pausable.CheckPause(ctx); err != nil {
						return err
					}

					if err := p.checkArticle(ctx, art); err != nil {
						// Track failed article
						mu.Lock()
						failedArticles = append(failedArticles, art)
						articleErrors++
						mu.Unlock()
						return err
					}

					// Update progress atomically (non-blocking)
					mu.Lock()
					articlesChecked++
					mu.Unlock()

					// Update progress if manager is available
					post.progress.UpdateProgress(int64(art.Size))
					return nil
				})
			}

			// Wait for all workers to complete and collect errors
			errors := pool.Wait()

			// If we have failed articles, handle them
			if len(failedArticles) > 0 {
				post.mu.Lock()
				post.Retries++
				post.mu.Unlock()

				// If we haven't exceeded max retries, add only failed articles back to queue
				if post.Retries < int(p.checkCfg.MaxRePost) {
					// Create a new post with only the failed articles
					failedPost := &Post{
						FilePath: post.FilePath,
						Articles: failedArticles,
						Status:   PostStatusPending,
						file:     post.file,
						filesize: post.filesize,
						wg:       post.wg,
						failed:   post.failed,
						Retries:  post.Retries,
						progress: p.jobProgress.AddProgress(uuid.New(), fmt.Sprintf("%s (retry)", filepath.Base(post.FilePath)), progress.ProgressTypeUploading, post.filesize),
					}

					slog.InfoContext(ctx,
						"Retrying failed articles",
						"file", post.FilePath,
						"attempt", post.Retries,
						"max_retries", p.checkCfg.MaxRePost,
					)

					// Track this retry in the queue before sending
					postsInFlight.Add(1)

					// Try to send retry to postQueue
					// Use select to handle closed channel gracefully
					select {
					case postQueue <- failedPost:
						// Retry sent successfully, continue to next post
						continue
					case <-ctx.Done():
						// Context canceled, stop processing
						// Decrement since we added above but didn't actually send
						postsInFlight.Done()
						slog.WarnContext(ctx, "Context canceled while trying to send retry", "file", post.FilePath)
						return
					default:
						// Channel is closed or full, cannot retry
						// Decrement since we added above but didn't actually send
						postsInFlight.Done()
						// Treat this as a failure
						slog.WarnContext(ctx, "Cannot send retry - postQueue unavailable", "file", post.FilePath)
						post.mu.Lock()
						post.Status = PostStatusFailed
						post.Error = fmt.Errorf("failed to queue retry - postQueue closed")
						post.mu.Unlock()

						if post.failed != nil {
							*post.failed++
						}

						errChan <- fmt.Errorf("failed to queue retry for file %s", post.FilePath)
						return
					}
				}

				// Increment failed posts counter if we've exceeded max retries
				post.mu.Lock()
				post.Status = PostStatusFailed
				post.Error = fmt.Errorf("failed to verify articles after %d retries", p.checkCfg.MaxRePost)
				post.mu.Unlock()

				// Mark this post as done in queue tracking - it failed permanently
				postsInFlight.Done()

				if post.failed != nil {
					*post.failed++
				}

				errChan <- fmt.Errorf("failed to verify file %s after %d retries", post.FilePath, p.checkCfg.MaxRePost)
				return
			} else if errors != nil {
				// This is a safety check - if we have errors but no failed articles, something went wrong
				post.mu.Lock()
				post.Status = PostStatusFailed
				post.Error = fmt.Errorf("verification failed but no articles were marked as failed: %v", errors)
				post.mu.Unlock()

				// Mark this post as done in queue tracking - it failed with unexpected error
				postsInFlight.Done()

				if post.failed != nil {
					*post.failed++
				}

				errChan <- fmt.Errorf("unexpected error verifying file %s: %v", post.FilePath, errors)
				return
			}

			// Mark as verified
			post.mu.Lock()
			post.Status = PostStatusVerified
			post.mu.Unlock()

			// Mark this post as done in queue tracking - verification successful
			postsInFlight.Done()

			p.jobProgress.FinishProgress(post.progress.GetID())

			// Close file
			if post.file != nil {
				if err := post.file.Close(); err != nil {
					slog.WarnContext(ctx, "Error closing file handle", "error", err, "file", post.FilePath)
				}
			}

			post.wg.Done()
		}
	}
}

// addPost adds a file to the posting queue
// displayName is the name to use in the subject (e.g., "Folder/subfolder/file.mp4")
// If displayName is empty, the filename is used
func (p *poster) addPost(ctx context.Context, filePath string, displayName string, fileNumber int, totalFiles int, wg *sync.WaitGroup, failedPosts *int, postQueue chan<- *Post, nzbGen nzb.NZBGenerator, postsInFlight *sync.WaitGroup) error {
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
	for i := 0; i < numSegments; i++ {
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
		FilePath: filePath,
		Articles: articles,
		Status:   PostStatusPending,
		file:     file,
		filesize: fileInfo.Size(),
		wg:       wg,
		failed:   failedPosts,
		progress: p.jobProgress.AddProgress(uuid.New(), filepath.Base(filePath), progress.ProgressTypeUploading, fileInfo.Size()),
	}

	// Track this post as in-flight until it's sent to the queue
	postsInFlight.Add(1)

	// Use select to safely send to channel and handle context cancellation
	select {
	case postQueue <- post:
		// Successfully sent to queue
		return nil
	case <-ctx.Done():
		// Context canceled, decrement counter since we didn't send
		postsInFlight.Done()
		// Close the file and return error
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
func (p *poster) postArticleWithBody(ctx context.Context, article *article.Article, body []byte) error {
	// Check if we should pause before posting
	if err := pausable.CheckPause(ctx); err != nil {
		return err
	}

	// Create article (with buffer pooling)
	buff, cleanup, err := article.Encode(body)
	if err != nil {
		return fmt.Errorf("error encoding article: %w", err)
	}
	defer cleanup() // Return buffer to pool when done

	// Post article
	if err := p.pool.Post(ctx, buff); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}

		return fmt.Errorf("error posting article: %w", err)
	}

	// Apply throttling after posting
	if p.throttle != nil {
		p.throttle.Wait(int64(article.Size))
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.ArticlesPosted++
	p.stats.BytesPosted += int64(article.Size)
	p.stats.mu.Unlock()

	return nil
}

// checkArticle checks if an article exists using the check pool
func (p *poster) checkArticle(ctx context.Context, art *article.Article) error {
	// Check if we should pause before checking
	if err := pausable.CheckPause(ctx); err != nil {
		return err
	}

	// Use the dedicated check pool for article verification
	_, err := p.checkPool.Stat(ctx, art.MessageID, art.Groups)
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
