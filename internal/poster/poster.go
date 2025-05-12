package poster

import (
	"context"
	"crypto/md5"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/javi11/nntppool"
	"github.com/javi11/nxg"
	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/nzb"
	"github.com/javi11/postie/internal/par2"
	"github.com/mnightingale/rapidyenc"
	"github.com/sourcegraph/conc/pool"
)

// Poster defines the interface for posting articles to Usenet
type Poster interface {
	// Post posts files from a directory to Usenet
	Post(ctx context.Context, files []string, rootDir string, outputDir string) error
	// GetStats returns posting statistics
	GetStats() Stats
}

// PostStatus represents the status of a post
type PostStatus int

const (
	PostStatusPending PostStatus = iota
	PostStatusPosted
	PostStatusVerified
	PostStatusFailed
)

// Post represents a file to be posted
type Post struct {
	FilePath string
	Articles []article.Article
	Status   PostStatus
	Error    error
	Retries  int
	file     *os.File
	mu       sync.Mutex
	filesize int64
	wg       *sync.WaitGroup
	failed   *int
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
	cfg      config.PostingConfig
	checkCfg config.PostCheck
	pool     nntppool.UsenetConnectionPool
	yenc     *rapidyenc.Encoder
	stats    *Stats
	throttle *Throttle
	nzbGen   nzb.NZBGenerator
}

// New creates a new poster
func New(ctx context.Context, cfg config.Config) (Poster, error) {
	pool, err := cfg.GetNNTPPool()
	if err != nil {
		return nil, fmt.Errorf("error getting NNTP pool: %w", err)
	}

	yenc := rapidyenc.NewEncoder()

	stats := &Stats{
		StartTime: time.Now(),
	}

	throttle := &Throttle{
		rate:     cfg.GetPostingConfig().ThrottleRate,
		interval: time.Second,
	}

	p := &poster{
		cfg:      cfg.GetPostingConfig(),
		checkCfg: cfg.GetPostCheckConfig(),
		pool:     pool,
		yenc:     yenc,
		stats:    stats,
		throttle: throttle,
		nzbGen:   nzb.NewGenerator(cfg.GetPostingConfig().ArticleSizeInBytes),
	}

	return p, nil
}

// Post posts files from a directory to Usenet
func (p *poster) Post(ctx context.Context, files []string, rootDir string, outputDir string) error {
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
	go p.postLoop(ctx, postQueue, checkQueue, errChan)

	// Start a goroutine to process checks
	go p.checkLoop(ctx, checkQueue, postQueue, errChan)

	wg.Add(len(files))
	for i, file := range files {
		if err := p.addPost(file, i+1, len(files), &wg, &failedPosts, postQueue); err != nil {
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

		// Generate single NZB file for all files
		firstFile := files[0]
		dirPath := filepath.Dir(firstFile)
		dirPath = strings.TrimPrefix(dirPath, rootDir)

		nzbPath := filepath.Join(outputDir, dirPath, filepath.Base(firstFile)+".nzb")
		if err := p.nzbGen.Generate("", nzbPath); err != nil {
			return fmt.Errorf("error generating NZB file: %w", err)
		}

		return nil
	}
}

// postLoop processes posts from the queue
func (p *poster) postLoop(ctx context.Context, postQueue chan *Post, checkQueue chan *Post, errChan chan<- error) {
	defer close(postQueue)
	defer close(checkQueue)

	for post := range postQueue {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			// Create a pool with error handling
			pool := pool.New().WithContext(ctx).WithMaxGoroutines(p.cfg.MaxWorkers).WithCancelOnError().WithFirstError()
			var bytesPosted int64
			articlesProcessed := 0
			articleErrors := 0

			// Create progress bar for this file
			progress := NewFileProgress(fmt.Sprintf("Uploading %s...", post.FilePath), post.filesize, int64(len(post.Articles)))

			// Submit all articles to the pool
			for _, art := range post.Articles {
				art := art // Create new variable for goroutine
				pool.Go(func(ctx context.Context) error {
					if err := p.postArticle(ctx, art, post.file); err != nil {
						return err
					}

					// Update progress
					post.mu.Lock()
					bytesPosted += int64(art.GetSize())
					articlesProcessed++

					progress.UpdateFileProgress(bytesPosted, int64(articlesProcessed), int64(articleErrors))

					// Add article to NZB generator
					p.nzbGen.AddArticle(art)
					post.mu.Unlock()

					return nil
				})
			}

			// Wait for all workers to complete and collect errors
			errors := pool.Wait()

			if errors != nil {
				post.mu.Lock()
				post.Status = PostStatusFailed
				post.Error = fmt.Errorf("failed to post articles: %v", errors)
				post.mu.Unlock()

				errChan <- fmt.Errorf("failed to post file %s after %d retries: %v", post.FilePath, p.cfg.MaxRetries, errors)
				return
			}

			// Move to check queue
			post.mu.Lock()
			post.Status = PostStatusPosted
			progress.FinishFileProgress()
			post.mu.Unlock()

			if p.checkCfg.Enabled {
				checkQueue <- post

				continue
			}

			// Close file
			if post.file != nil {
				_ = post.file.Close()
			}

			post.wg.Done()
		}
	}
}

// checkLoop processes posts from the check queue
func (p *poster) checkLoop(ctx context.Context, checkQueue chan *Post, postQueue chan *Post, errChan chan<- error) {
	for post := range checkQueue {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
			// Create a pool with error handling
			pool := pool.New().WithContext(ctx).WithMaxGoroutines(p.cfg.MaxWorkers).WithCancelOnError().WithFirstError()
			articlesChecked := 0
			articleErrors := 0

			// Create progress bar for this file
			progress := NewFileProgress(fmt.Sprintf("Verifying %s...", post.FilePath), post.filesize, int64(len(post.Articles)))

			// Submit all articles to the pool
			for _, art := range post.Articles {
				art := art // Create new variable for goroutine
				pool.Go(func(ctx context.Context) error {
					if err := p.checkArticle(ctx, art); err != nil {
						return err
					}

					// Update progress
					post.mu.Lock()
					articlesChecked++
					post.mu.Unlock()

					progress.UpdateFileProgress(0, int64(articlesChecked), int64(articleErrors))
					return nil
				})
			}

			// Wait for all workers to complete and collect errors
			errors := pool.Wait()

			if errors != nil {
				post.mu.Lock()
				post.Status = PostStatusFailed
				post.Error = fmt.Errorf("failed to verify articles: %v", errors)
				post.Retries++
				post.mu.Unlock()

				// If we haven't exceeded max retries, add back to queue
				if post.Retries < int(p.checkCfg.MaxRePost) {
					postQueue <- post
					continue
				}

				// Increment failed posts counter
				if post.failed != nil {
					*post.failed++
				}

				errChan <- fmt.Errorf("failed to verify file %s after %d retries: %v", post.FilePath, p.cfg.MaxRetries, errors)
				return
			}

			// Mark as verified
			post.mu.Lock()
			post.Status = PostStatusVerified
			post.mu.Unlock()

			progress.FinishFileProgress()

			// Close file
			if post.file != nil {
				_ = post.file.Close()
			}

			post.wg.Done()
		}
	}
}

// addPost adds a file to the posting queue
func (p *poster) addPost(filePath string, fileNumber int, totalFiles int, wg *sync.WaitGroup, failedPosts *int, postQueue chan<- *Post) error {
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
	articles := make([]article.Article, 0, numSegments)
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
		subject := article.GenerateSubject(fileNumber, totalFiles, fileName, int(fileInfo.Size()), partNumber, numSegments)
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
		if p.cfg.PostHeaders.AddNGXHeader &&
			p.cfg.ObfuscationPolicy != config.ObfuscationPolicyFull &&
			p.cfg.MessageIDFormat != config.MessageIDFormatNGX {
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
			rd := article.GetRandomDateWithinLast6Hours()
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
			art.SetDate(*date)
		}

		if xNxgHeader != "" {
			art.SetXNxgHeader(xNxgHeader)
		}

		art.SetOffset(offset)
		art.SetSize(uint64(size))

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
	}

	postQueue <- post
	return nil
}

// postArticle posts an article to Usenet
func (p *poster) postArticle(ctx context.Context, article article.Article, file *os.File) error {
	// Read article body
	body := make([]byte, article.GetSize())
	if _, err := file.ReadAt(body, article.GetOffset()); err != nil {
		return fmt.Errorf("error reading article body: %w", err)
	}

	// Create article
	art, err := article.EncodeBytes(p.yenc, body)
	if err != nil {
		return fmt.Errorf("error encoding article: %w", err)
	}

	// Post article
	if err := p.pool.Post(ctx, art); err != nil {
		return fmt.Errorf("error posting article: %w", err)
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.ArticlesPosted++
	p.stats.BytesPosted += int64(article.GetSize())
	p.stats.mu.Unlock()

	return nil
}

// checkArticle checks if an article exists
func (p *poster) checkArticle(ctx context.Context, art article.Article) error {
	_, err := p.pool.Stat(ctx, art.GetMessageID(), art.GetGroups())
	if err != nil {
		return fmt.Errorf("article not found: %w", err)
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.ArticlesChecked++
	p.stats.mu.Unlock()

	return nil
}

// GetStats returns posting statistics
func (p *poster) GetStats() Stats {
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
