// Package verification provides the durable, lease-based post-upload
// verification service for the process-wide upload architecture (issue 184).
//
// It is the primary post-check path: instead of holding a queue upload slot
// during propagation delay, completed transfers are recorded durably and this
// service verifies them in the background. It streams a file's manifest,
// issues STAT requests, and persists ONLY the articles that fail. Failed
// articles are then re-posted (reusing their exact Message-ID and headers from
// the manifest) or STAT-rechecked with backoff, using database leases so the
// work survives process restarts.
//
// The service is decoupled from the upload/NNTP layers through the Stater and
// Reposter interfaces so it can be unit-tested in isolation and wired into the
// processor later.
package verification

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/javi11/postie/internal/manifest"
	"github.com/javi11/postie/internal/transferstore"
)

// Stater reports whether an article exists on the server. A nil error means the
// article was found; any error means it is (still) missing.
type Stater interface {
	Stat(ctx context.Context, messageID string) error
}

// Reposter re-posts a single article described by its manifest record, reusing
// the record's Message-ID and headers so the NZB stays correct.
type Reposter interface {
	Repost(ctx context.Context, rec manifest.ArticleRecord) error
}

// Cleaner runs post-verification cleanup for a transfer once it is fully
// verified (deletes originals per policy, removes PAR2 and manifests). It is a
// no-op while the transfer is not yet terminal, and retains everything on
// failure.
type Cleaner interface {
	CleanupTransfer(ctx context.Context, transferID string) (bool, error)
}

// Config controls verification timing and concurrency. Zero values fall back to
// conservative defaults via withDefaults.
type Config struct {
	// MaxConcurrentChecks bounds simultaneous STAT requests. 0 = auto (16).
	MaxConcurrentChecks int
	// PropagationDelay is how long to wait after a (re)post before checking.
	PropagationDelay time.Duration
	// MaxReposts is the maximum number of times a missing article is re-posted
	// before falling back to STAT-only deferred checking.
	MaxReposts int
	// DeferredBackoff is the base delay between STAT-only rechecks; it grows
	// exponentially up to MaxBackoff.
	DeferredBackoff time.Duration
	MaxBackoff      time.Duration
	// MaxDeferredChecks caps STAT-only rechecks before the article (and its
	// file) is marked permanently failed.
	MaxDeferredChecks int
	// LeaseDuration is how long a claimed batch is owned before it can be
	// reclaimed by another worker after a crash.
	LeaseDuration time.Duration
	// BatchSize is the maximum number of failures claimed per cycle.
	BatchSize int
	// PollInterval is how often the Run loop checks for due files/failures.
	PollInterval time.Duration
}

func (c Config) withDefaults() Config {
	if c.MaxConcurrentChecks <= 0 {
		c.MaxConcurrentChecks = 16
	}
	if c.PropagationDelay <= 0 {
		c.PropagationDelay = 10 * time.Second
	}
	if c.MaxReposts < 0 {
		c.MaxReposts = 0
	}
	if c.DeferredBackoff <= 0 {
		c.DeferredBackoff = 5 * time.Minute
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = time.Hour
	}
	if c.MaxDeferredChecks <= 0 {
		c.MaxDeferredChecks = 5
	}
	if c.LeaseDuration <= 0 {
		c.LeaseDuration = 5 * time.Minute
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 500
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 15 * time.Second
	}
	return c
}

// Service runs durable verification against a transfer store.
type Service struct {
	store    *transferstore.Store
	stater   Stater
	reposter Reposter
	cfg      Config
	owner    string
	now      func() time.Time
	cleaner  Cleaner
}

// SetCleaner installs the post-verification cleaner invoked when a transfer
// becomes fully verified. Optional; nil disables cleanup.
func (s *Service) SetCleaner(c Cleaner) { s.cleaner = c }

// maybeCleanup attempts cleanup for a transfer (no-op if no cleaner or not yet
// fully verified).
func (s *Service) maybeCleanup(ctx context.Context, transferID string) {
	if s.cleaner == nil {
		return
	}
	if _, err := s.cleaner.CleanupTransfer(ctx, transferID); err != nil {
		slog.WarnContext(ctx, "verification: cleanup failed", "transfer", transferID, "error", err)
	}
}

// New creates a verification service. owner identifies this worker for lease
// ownership (e.g. a hostname+pid string).
func New(store *transferstore.Store, stater Stater, reposter Reposter, cfg Config, owner string) *Service {
	return &Service{
		store:    store,
		stater:   stater,
		reposter: reposter,
		cfg:      cfg.withDefaults(),
		owner:    owner,
		now:      time.Now,
	}
}

// Run drives verification until ctx is cancelled: on each tick it reclaims
// expired leases, runs the first verification check for any due files, then
// processes due verification failures (re-posts/rechecks). It is intended to be
// started once as a background goroutine, owned by the processor.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		s.runOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runOnce performs a single verification cycle.
func (s *Service) runOnce(ctx context.Context) {
	now := s.now()
	if _, err := s.store.ReclaimExpiredLeases(ctx, now); err != nil {
		slog.WarnContext(ctx, "verification: reclaim leases failed", "error", err)
	}
	if _, err := s.VerifyDueFiles(ctx, now, s.cfg.BatchSize); err != nil {
		slog.WarnContext(ctx, "verification: verify due files failed", "error", err)
	}
	if _, err := s.ProcessDueFailures(ctx, now); err != nil {
		slog.WarnContext(ctx, "verification: process due failures failed", "error", err)
	}
}

// VerifyDueFiles runs the first verification check for every file whose check is
// due (verification_state=uploaded, next_check_at<=now), up to limit. A failure
// verifying one file is logged and does not abort the batch. Returns the number
// of files verified.
func (s *Service) VerifyDueFiles(ctx context.Context, now time.Time, limit int) (int, error) {
	due, err := s.store.ListDueFiles(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, tf := range due {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if err := s.VerifyFile(ctx, tf); err != nil {
			slog.WarnContext(ctx, "verification: verify file failed",
				"transfer", tf.TransferID, "file", tf.FileID, "error", err)
			continue
		}
		processed++
	}
	return processed, nil
}

// VerifyFile streams a transfer file's manifest, STATs every article with
// bounded concurrency, and persists only the misses. If no article is missing
// the file is marked verified; otherwise it is marked verifying and the misses
// are left for ProcessDueFailures to re-post/recheck.
func (s *Service) VerifyFile(ctx context.Context, tf transferstore.TransferFile) error {
	r, err := manifest.OpenReader(tf.ManifestPath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	sem := make(chan struct{}, s.cfg.MaxConcurrentChecks)
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		missing  []manifest.ArticleRecord
		statErrs int
	)

	for {
		rec, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			wg.Wait()
			return err
		}

		if err := ctx.Err(); err != nil {
			wg.Wait()
			return err
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(rec manifest.ArticleRecord) {
			defer wg.Done()
			defer func() { <-sem }()
			if statErr := s.stater.Stat(ctx, rec.MessageID); statErr != nil {
				mu.Lock()
				missing = append(missing, rec)
				statErrs++
				mu.Unlock()
			}
		}(rec)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return err
	}

	if len(missing) == 0 {
		if err := s.store.SetVerificationState(ctx, tf.TransferID, tf.FileID, transferstore.StateVerified, nil, ""); err != nil {
			return err
		}
		s.maybeCleanup(ctx, tf.TransferID)
		return nil
	}

	// Persist misses for durable re-posting/rechecking.
	next := s.now().Add(s.cfg.PropagationDelay)
	for _, rec := range missing {
		if err := s.store.AddFailure(ctx, transferstore.VerificationFailure{
			TransferID:    tf.TransferID,
			FileID:        tf.FileID,
			ArticleIndex:  rec.Index,
			MessageID:     rec.MessageID,
			Groups:        rec.Groups,
			State:         transferstore.FailurePending,
			NextAttemptAt: next,
		}); err != nil {
			return err
		}
	}
	return s.store.SetVerificationState(ctx, tf.TransferID, tf.FileID, transferstore.StateVerifying, &next, "")
}

// ProcessDueFailures claims a batch of due verification failures and handles
// each: re-post (while under MaxReposts) or STAT-only recheck with exponential
// backoff. Returns the number of failures processed. Resolving the last failure
// for a file flips that file to verified; exhausting retries flips it to
// verification_failed.
func (s *Service) ProcessDueFailures(ctx context.Context, now time.Time) (int, error) {
	claimed, err := s.store.ClaimDueFailures(ctx, s.owner, s.cfg.LeaseDuration, s.cfg.BatchSize, now)
	if err != nil {
		return 0, err
	}
	if len(claimed) == 0 {
		return 0, nil
	}

	// Resolve manifest records for failures that still need a re-post, grouped
	// by file so each manifest is streamed at most once per batch.
	records := s.resolveRecords(ctx, claimed)

	touchedFiles := make(map[[2]string]bool)
	for _, f := range claimed {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		s.handleFailure(ctx, f, records, now)
		touchedFiles[[2]string{f.TransferID, f.FileID}] = true
	}

	// Reconcile each touched file's verification state.
	for key := range touchedFiles {
		s.reconcileFileState(ctx, key[0], key[1])
	}

	return len(claimed), nil
}

// handleFailure processes a single claimed failure, updating its row.
func (s *Service) handleFailure(ctx context.Context, f transferstore.VerificationFailure, records map[recordKey]manifest.ArticleRecord, now time.Time) {
	if f.RepostCount < s.cfg.MaxReposts {
		rec, ok := records[recordKey{f.TransferID, f.FileID, f.ArticleIndex}]
		if !ok {
			// No manifest record (e.g. legacy STAT-only failure): fall through
			// to a STAT recheck instead.
			s.statRecheck(ctx, f, now)
			return
		}
		if err := s.reposter.Repost(ctx, rec); err != nil {
			// Re-post failed: keep pending, short backoff, record error.
			f.NextAttemptAt = now.Add(s.cfg.PropagationDelay)
			f.LastError = "repost failed: " + err.Error()
			_ = s.store.UpdateFailureAfterCheck(ctx, f)
			return
		}
		// Re-posted: schedule a recheck after propagation delay.
		f.RepostCount++
		f.State = transferstore.FailurePending
		f.NextAttemptAt = now.Add(s.cfg.PropagationDelay)
		f.LastError = ""
		_ = s.store.UpdateFailureAfterCheck(ctx, f)
		return
	}
	s.statRecheck(ctx, f, now)
}

// statRecheck issues a STAT and either resolves the failure, marks it
// permanently failed, or defers it with exponential backoff.
func (s *Service) statRecheck(ctx context.Context, f transferstore.VerificationFailure, now time.Time) {
	if statErr := s.stater.Stat(ctx, f.MessageID); statErr == nil {
		f.State = transferstore.FailureResolved
		f.LastError = ""
		f.NextAttemptAt = now
		_ = s.store.UpdateFailureAfterCheck(ctx, f)
		return
	}

	f.DeferredCount++
	if f.DeferredCount >= s.cfg.MaxDeferredChecks {
		f.State = transferstore.FailureFailed
		f.NextAttemptAt = now
		f.LastError = "exhausted deferred rechecks"
		_ = s.store.UpdateFailureAfterCheck(ctx, f)
		return
	}

	backoff := s.cfg.DeferredBackoff << min(f.DeferredCount-1, 16)
	if backoff > s.cfg.MaxBackoff || backoff <= 0 {
		backoff = s.cfg.MaxBackoff
	}
	f.State = transferstore.FailurePending
	f.NextAttemptAt = now.Add(backoff)
	f.LastError = "article still missing"
	_ = s.store.UpdateFailureAfterCheck(ctx, f)
}

// reconcileFileState flips a file to verified when no pending/reposted failures
// remain, or to verification_failed when only permanently-failed ones do.
func (s *Service) reconcileFileState(ctx context.Context, transferID, fileID string) {
	outstanding, err := s.store.CountFailures(ctx, transferID, fileID, transferstore.FailurePending)
	if err != nil {
		slog.WarnContext(ctx, "verification: count pending failed", "error", err)
		return
	}
	if outstanding > 0 {
		return // still working
	}
	failed, err := s.store.CountFailures(ctx, transferID, fileID, transferstore.FailureFailed)
	if err != nil {
		slog.WarnContext(ctx, "verification: count failed failed", "error", err)
		return
	}
	if failed > 0 {
		_ = s.store.SetVerificationState(ctx, transferID, fileID, transferstore.StateVerificationFailed, nil, "verification failed for some articles")
		return
	}
	_ = s.store.SetVerificationState(ctx, transferID, fileID, transferstore.StateVerified, nil, "")
	s.maybeCleanup(ctx, transferID)
}

type recordKey struct {
	transferID string
	fileID     string
	index      int
}

// resolveRecords streams the manifest of each file that has a re-postable
// failure and returns the manifest records keyed by (transfer,file,index).
func (s *Service) resolveRecords(ctx context.Context, failures []transferstore.VerificationFailure) map[recordKey]manifest.ArticleRecord {
	out := make(map[recordKey]manifest.ArticleRecord)

	// Which indices does each file need (only for failures still eligible to
	// re-post)?
	needed := make(map[[2]string]map[int]bool)
	for _, f := range failures {
		if f.RepostCount >= s.cfg.MaxReposts || f.ArticleIndex < 0 {
			continue
		}
		key := [2]string{f.TransferID, f.FileID}
		if needed[key] == nil {
			needed[key] = make(map[int]bool)
		}
		needed[key][f.ArticleIndex] = true
	}

	for key, indices := range needed {
		tf, err := s.store.GetFile(ctx, key[0], key[1])
		if err != nil {
			slog.WarnContext(ctx, "verification: load transfer file failed", "transfer", key[0], "file", key[1], "error", err)
			continue
		}
		r, err := manifest.OpenReader(tf.ManifestPath)
		if err != nil {
			slog.WarnContext(ctx, "verification: open manifest failed", "path", tf.ManifestPath, "error", err)
			continue
		}
		remaining := len(indices)
		for remaining > 0 {
			rec, err := r.Next()
			if errors.Is(err, io.EOF) || err != nil {
				break
			}
			if indices[rec.Index] {
				out[recordKey{key[0], key[1], rec.Index}] = rec
				remaining--
			}
		}
		_ = r.Close()
	}
	return out
}
