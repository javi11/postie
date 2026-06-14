package poster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/javi11/nntppool/v4"
	"github.com/mnightingale/rapidyenc"

	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/manifest"
	"github.com/javi11/postie/internal/pool"
)

// postYenc builds the NNTP headers + yEnc metadata for art and posts body,
// retrying up to 3 times on stale pooled connections. It is the shared posting
// primitive used by both the normal upload path (postArticleWithBody) and the
// durable re-post path (Reposter.Repost); extracting it keeps the exact same
// Message-ID and header handling for re-posts.
func postYenc(ctx context.Context, uploadPool pool.NNTPClient, throttle *Throttle, stats *Stats, art *article.Article, body []byte) error {
	headers := nntppool.PostHeaders{
		From:       art.From,
		Subject:    art.Subject,
		Newsgroups: art.Groups,
		MessageID:  fmt.Sprintf("<%s>", art.MessageID),
		Date:       art.Date.UTC(),
		Extra:      make(map[string][]string),
	}

	if art.CustomHeaders != nil {
		for k, v := range art.CustomHeaders {
			headers.Extra[k] = []string{v}
		}
	}
	if art.XNxgHeader != "" {
		headers.Extra["X-Nxg"] = []string{art.XNxgHeader}
	}

	meta := rapidyenc.Meta{
		FileName:   art.FileName,
		FileSize:   art.FileSize,
		PartNumber: int64(art.PartNumber),
		TotalParts: int64(art.TotalParts),
		Offset:     int64(art.Offset),
		PartSize:   int64(art.Size),
	}

	// Post article with timeout to prevent indefinite TLS hangs.
	postCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Retry on stale pooled connection (see isStaleConnError). bytes.NewReader
	// is cheap to recreate so the body is fully re-readable on each attempt.
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			slog.WarnContext(ctx, "Retrying article post after stale pooled connection",
				"messageID", art.MessageID, "attempt", attempt, "prevErr", lastErr.Error())
		}

		_, lastErr = uploadPool.PostYenc(postCtx, headers, bytes.NewReader(body), meta)
		if lastErr == nil {
			break
		}
		if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) {
			return context.Canceled
		}
		if !isStaleConnError(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return fmt.Errorf("error posting article: %w", lastErr)
	}

	if throttle != nil {
		throttle.Wait(int64(art.Size))
	}

	if stats != nil {
		stats.mu.Lock()
		stats.ArticlesPosted++
		stats.BytesPosted += int64(art.Size)
		stats.mu.Unlock()
	}

	return nil
}

// articleFromRecord reconstructs an article from a manifest record so it can be
// re-posted byte-for-byte with its original Message-ID and headers.
func articleFromRecord(rec manifest.ArticleRecord) *article.Article {
	return &article.Article{
		MessageID:       rec.MessageID,
		Subject:         rec.Subject,
		OriginalSubject: rec.OriginalSubject,
		From:            rec.From,
		Groups:          rec.Groups,
		PartNumber:      rec.PartNumber,
		TotalParts:      rec.TotalParts,
		FileName:        rec.FileName,
		Date:            rec.Date,
		Offset:          rec.Offset,
		Size:            rec.BodySize,
		FileSize:        rec.FileSize,
		CustomHeaders:   rec.CustomHeaders,
		XNxgHeader:      rec.XNxgHeader,
	}
}

// Reposter re-posts individual articles during durable verification, reading
// each body from its original source file and posting through the shared upload
// pool and engine. It is process-wide (owned by the transfer runtime), distinct
// from the per-job poster, and implements verification.Reposter.
type Reposter struct {
	uploadPool pool.NNTPClient
	engine     *Engine
	throttle   *Throttle
	stats      *Stats
}

// NewReposter creates a Reposter using the shared upload pool and engine. If
// throttleRate > 0 the same byte/sec throttle as normal uploads is applied.
func NewReposter(uploadPool pool.NNTPClient, engine *Engine, throttleRate int64) *Reposter {
	var throttle *Throttle
	if throttleRate > 0 {
		throttle = NewThrottle(throttleRate, time.Second)
	}
	return &Reposter{
		uploadPool: uploadPool,
		engine:     engine,
		throttle:   throttle,
		stats:      &Stats{StartTime: time.Now()},
	}
}

// Repost re-posts a single article from its manifest record. The body is read
// from rec.SourcePath at rec.Offset, and the post reuses rec's Message-ID and
// headers so the NZB remains correct. It acquires an engine worker slot and
// buffer reservation so re-posts share the process-wide resource limits.
func (r *Reposter) Repost(ctx context.Context, rec manifest.ArticleRecord) error {
	if r.uploadPool == nil {
		return errors.New("reposter has no upload pool")
	}

	f, err := os.Open(rec.SourcePath)
	if err != nil {
		return fmt.Errorf("repost: open source %q: %w", rec.SourcePath, err)
	}
	defer func() { _ = f.Close() }()

	reserve := r.engine.PerArticleBytes()
	if err := r.engine.ReserveBuffer(ctx, reserve); err != nil {
		return err
	}
	defer r.engine.ReleaseBuffer(reserve)

	body := make([]byte, rec.BodySize)
	if _, err := f.ReadAt(body, rec.Offset); err != nil {
		return fmt.Errorf("repost: read body at offset %d: %w", rec.Offset, err)
	}

	if err := r.engine.AcquireWorker(ctx); err != nil {
		return err
	}
	defer r.engine.ReleaseWorker()

	return postYenc(ctx, r.uploadPool, r.throttle, r.stats, articleFromRecord(rec), body)
}

// Stats returns a snapshot of re-post statistics.
func (r *Reposter) Stats() Stats {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()
	return Stats{
		ArticlesPosted: r.stats.ArticlesPosted,
		BytesPosted:    r.stats.BytesPosted,
		ArticleErrors:  r.stats.ArticleErrors,
		StartTime:      r.stats.StartTime,
	}
}
