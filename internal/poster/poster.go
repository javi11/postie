package poster

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/javi11/nntppool"
	"github.com/javi11/nxg"
	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/nzb"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/progress"
	"github.com/sourcegraph/conc/pool"
)

// Poster defines the interface for posting articles to Usenet
type Poster interface {
	// Post posts files from a directory to Usenet
	Post(ctx context.Context, files []string, rootDir string, nzbGen nzb.NZBGenerator) error
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
	pool        nntppool.UsenetConnectionPool
	stats       *Stats
	throttle    *Throttle
	jobProgress progress.JobProgress
}

// New creates a new poster
func New(ctx context.Context, cfg config.Config, jobProgress progress.JobProgress) (Poster, error) {
	pool, err := cfg.GetNNTPPool()
	if err != nil {
		return nil, fmt.Errorf("error getting NNTP pool: %w", err)
	}

	stats := &Stats{
		StartTime: time.Now(),
	}

	p := &poster{
		cfg:         cfg.GetPostingConfig(),
		checkCfg:    cfg.GetPostCheckConfig(),
		pool:        pool,
		stats:       stats,
		jobProgress: jobProgress,
	}

	throttleRate := p.cfg.ThrottleRate
	if throttleRate > 0 {
		p.throttle = NewThrottle(throttleRate, time.Second)
	}

	return p, nil
}

func (p *poster) Close() {
	slog.Info("Closing poster")

	if p.pool != nil {
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic during pool quit", "panic", r)
				}
				close(done)
			}()
			p.pool.Quit()
		}()

		// Use longer timeout on Windows due to slower networking cleanup
		timeout := 5 * time.Second
		if runtime.GOOS == "windows" {
			timeout = 10 * time.Second
		}

		select {
		case <-done:
			// Quit completed successfully
			slog.Debug("Connection pool closed successfully")
		case <-time.After(timeout):
			// Timeout occurred, log warning but continue
			slog.Warn("Pool quit timed out, forcing close",
				"timeout_seconds", timeout.Seconds(),
				"os", runtime.GOOS)
		}
		p.pool = nil
	}
}

// Post posts files from a directory to Usenet
func (p *poster) Post(
	ctx context.Context,
	files []string,
	rootDir string,
	nzbGen nzb.NZBGenerator,
) error {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "Panic in poster.Post",
				"panic", r,
				"files", len(files),
				"rootDir", rootDir,
				"os", runtime.GOOS)
		}
	}()

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

	// Start a goroutine to process posts
	go p.postLoop(ctx, postQueue, checkQueue, errChan, nzbGen)

	// Start a goroutine to process checks
	go p.checkLoop(ctx, checkQueue, postQueue, errChan, nzbGen)

	wg.Add(len(files))
	for i, file := range files {
		if err := p.addPost(file, i+1, len(files), &wg, &failedPosts, postQueue, nzbGen); err != nil {
			return fmt.Errorf("error adding file %s to posting queue: %w", file, err)
		}
	}

	// Wait for all posts to complete or an error to occur
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
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
func (p *poster) postLoop(ctx context.Context, postQueue chan *Post, checkQueue chan *Post, errChan chan<- error, nzbGen nzb.NZBGenerator) {
	defer close(postQueue)
	defer close(checkQueue)

	for post := range postQueue {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:

			// Set post status to Posting
			post.mu.Lock()
			post.Status = PostStatusPosting
			post.mu.Unlock()

			// Create a pool with error handling - use all available CPU cores
			pool := pool.New().WithContext(ctx).WithMaxGoroutines(runtime.NumCPU()).WithCancelOnError().WithFirstError()

			var combinedHash string

			// Submit all articles to the pool
			for _, art := range post.Articles {
				combinedHash += art.Hash

				pool.Go(func(ctx context.Context) error {
					if err := p.postArticle(ctx, art, post.file); err != nil {
						return err
					}

					// Update progress if manager is available
					post.progress.UpdateProgress(int64(art.Size))

					// Add article to NZB generator (still needs mutex for thread safety)
					post.mu.Lock()
					nzbGen.AddArticle(art)
					post.mu.Unlock()

					return nil
				})
			}

			// Wait for all workers to complete and collect errors
			errs := pool.Wait()

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

				errChan <- fmt.Errorf("failed to post file %s after %d retries: %v", post.FilePath, p.cfg.MaxRetries, errs)
				return
			}

			// Calculate file hash by merging all article hashes
			post.mu.Lock()
			post.Status = PostStatusPosted

			post.mu.Unlock()

			// Add file hash to NZB generator
			fileHash := CalculateHash([]byte(combinedHash))
			nzbGen.AddFileHash(post.Articles[0].OriginalName, fileHash)

			if *p.checkCfg.Enabled {
				checkQueue <- post

				continue
			}

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
func (p *poster) checkLoop(ctx context.Context, checkQueue chan *Post, postQueue chan *Post, errChan chan<- error, nzbGen nzb.NZBGenerator) {
	for post := range checkQueue {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			// Create a pool with error handling - use all available CPU cores
			pool := pool.New().WithContext(ctx).WithMaxGoroutines(runtime.NumCPU()).WithCancelOnError().WithFirstError()
			articlesChecked := 0
			articleErrors := 0
			var failedArticles []*article.Article
			var mu sync.Mutex

			// Submit all articles to the pool
			for _, art := range post.Articles {
				pool.Go(func(ctx context.Context) error {
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
					//current := articlesChecked
					//errors := articleErrors
					mu.Unlock()

					//progress.UpdateFileProgress(0, int64(current), int64(errors))
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

					postQueue <- failedPost
					continue
				}

				// Increment failed posts counter if we've exceeded max retries
				post.mu.Lock()
				post.Status = PostStatusFailed
				post.Error = fmt.Errorf("failed to verify articles after %d retries", p.checkCfg.MaxRePost)
				post.mu.Unlock()

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
func (p *poster) addPost(filePath string, fileNumber int, totalFiles int, wg *sync.WaitGroup, failedPosts *int, postQueue chan<- *Post, nzbGen nzb.NZBGenerator) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("error getting file info: %w", err)
	}

	// Calculate number of segments
	segmentSize := p.cfg.ArticleSizeInBytes
	numSegments := int((fileInfo.Size() + int64(segmentSize) - 1) / int64(segmentSize))
	nxgHeader := nxg.GenerateNXGHeader(int64(numSegments), 0)

	groups := make([]string, 0)

	switch p.cfg.GroupPolicy {
	case config.GroupPolicyEachFile:
		randomGroup := p.cfg.Groups[rand.Intn(len(p.cfg.Groups))]
		groups = append(groups, randomGroup)
	case config.GroupPolicyAll:
		groups = p.cfg.Groups
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

		fileName := filepath.Base(filePath)
		subject := article.GenerateSubject(fileNumber, totalFiles, fileName, partNumber, numSegments)
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

	postQueue <- post
	return nil
}

// postArticle posts an article to Usenet
func (p *poster) postArticle(ctx context.Context, article *article.Article, file *os.File) error {
	// Read article body
	body := make([]byte, article.Size)
	if _, err := file.ReadAt(body, article.Offset); err != nil {
		return fmt.Errorf("error reading article body: %w", err)
	}

	// Calculate and set hash for the article
	articleHash := CalculateHash(body)
	article.Hash = articleHash

	// Create article
	buff, err := article.Encode(body)
	if err != nil {
		return fmt.Errorf("error encoding article: %w", err)
	}

	// Post article
	if err := p.pool.Post(ctx, buff); err != nil {
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

// checkArticle checks if an article exists
func (p *poster) checkArticle(ctx context.Context, art *article.Article) error {
	_, err := p.pool.Stat(ctx, art.MessageID, art.Groups)
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

func CalculateHash(buff []byte) string {
	hash := sha256.New()
	hash.Write(buff[:])
	hashInBytes := hash.Sum(nil)

	return hex.EncodeToString(hashInBytes)
}
