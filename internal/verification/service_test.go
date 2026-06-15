package verification

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/database"
	"github.com/javi11/postie/internal/manifest"
	"github.com/javi11/postie/internal/transferstore"
)

// fakeStater returns "missing" for any message id in the missing set.
type fakeStater struct {
	mu      sync.Mutex
	missing map[string]bool
	calls   map[string]int
}

func newFakeStater(missing ...string) *fakeStater {
	m := make(map[string]bool)
	for _, id := range missing {
		m[id] = true
	}
	return &fakeStater{missing: m, calls: make(map[string]int)}
}

func (f *fakeStater) Stat(_ context.Context, messageID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[messageID]++
	if f.missing[messageID] {
		return errors.New("not found")
	}
	return nil
}

func (f *fakeStater) markPresent(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.missing, id)
}

// fakeReposter records reposts and can optionally mark them present afterwards.
type fakeReposter struct {
	mu       sync.Mutex
	reposted []string
	onRepost func(rec manifest.ArticleRecord)
	failWith error
}

func (f *fakeReposter) Repost(_ context.Context, rec manifest.ArticleRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return f.failWith
	}
	f.reposted = append(f.reposted, rec.MessageID)
	if f.onRepost != nil {
		f.onRepost(rec)
	}
	return nil
}

func (f *fakeReposter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reposted)
}

func newTestStore(t *testing.T) *transferstore.Store {
	t.Helper()
	ctx := context.Background()
	db, err := database.New(ctx, config.DatabaseConfig{
		DatabaseType: "sqlite",
		DatabasePath: filepath.Join(t.TempDir(), "v.db"),
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.GetMigrationRunner().MigrateUp(); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	return transferstore.New(db.DB)
}

// writeManifest creates a manifest with n articles "<m0>".."<m{n-1}>" and
// registers a transfer_files row pointing at it.
func writeManifest(t *testing.T, store *transferstore.Store, transferID, fileID string, n int) string {
	t.Helper()
	path := manifest.FilePath(t.TempDir(), transferID, fileID)
	w, err := manifest.NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for i := 0; i < n; i++ {
		if err := w.Write(manifest.ArticleRecord{
			Index:     i,
			MessageID: mid(i),
			Offset:    int64(i) * 1000,
			BodySize:  1000,
			Groups:    []string{"alt.binaries.test"},
		}); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.UpsertFile(context.Background(), transferstore.TransferFile{
		TransferID:   transferID,
		FileID:       fileID,
		ManifestPath: path,
		SourcePath:   "/data/x.bin",
		FileRole:     "original",
		ArticleCount: n,
		UploadState:  transferstore.StateUploaded,
	}); err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	return path
}

func mid(i int) string { return "<m" + string(rune('0'+i)) + ">" }

func TestVerifyFile_AllPresentMarksVerified(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 5)

	svc := New(store, newFakeStater(), &fakeReposter{}, Config{}, "w")
	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}

	got, _ := store.GetFile(ctx, "t", "f")
	if got.VerificationState != transferstore.StateVerified {
		t.Errorf("state = %q, want verified", got.VerificationState)
	}
	if n, _ := store.CountFailures(ctx, "t", "f", ""); n != 0 {
		t.Errorf("failures = %d, want 0", n)
	}
}

func TestVerifyFile_PersistsMisses(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 5)

	// m1 and m3 missing.
	svc := New(store, newFakeStater(mid(1), mid(3)), &fakeReposter{}, Config{}, "w")
	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}

	got, _ := store.GetFile(ctx, "t", "f")
	if got.VerificationState != transferstore.StateVerifying {
		t.Errorf("state = %q, want verifying", got.VerificationState)
	}
	if n, _ := store.CountFailures(ctx, "t", "f", transferstore.FailurePending); n != 2 {
		t.Errorf("pending failures = %d, want 2", n)
	}
}

func TestProcessDueFailures_RepostThenResolve(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 3)

	stater := newFakeStater(mid(1))
	// When re-posted, the article becomes present on the next STAT.
	reposter := &fakeReposter{onRepost: func(rec manifest.ArticleRecord) { stater.markPresent(rec.MessageID) }}

	cfg := Config{MaxReposts: 1, PropagationDelay: time.Millisecond, DeferredBackoff: time.Millisecond}
	svc := New(store, stater, reposter, cfg, "w")

	// Initial verification persists the miss for m1.
	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}

	now := time.Now().Add(time.Second) // make the failure due
	// First pass: re-posts m1 (RepostCount 0 -> 1), still pending.
	if _, err := svc.ProcessDueFailures(ctx, now); err != nil {
		t.Fatalf("ProcessDueFailures #1: %v", err)
	}
	if reposter.count() != 1 {
		t.Errorf("reposts = %d, want 1", reposter.count())
	}

	// Second pass (after the scheduled recheck is due): RepostCount now == MaxReposts,
	// so STAT-only recheck runs and finds it present -> resolved -> file verified.
	now = now.Add(time.Hour)
	if _, err := svc.ProcessDueFailures(ctx, now); err != nil {
		t.Fatalf("ProcessDueFailures #2: %v", err)
	}

	if n, _ := store.CountFailures(ctx, "t", "f", transferstore.FailurePending); n != 0 {
		t.Errorf("pending = %d, want 0", n)
	}
	got, _ := store.GetFile(ctx, "t", "f")
	if got.VerificationState != transferstore.StateVerified {
		t.Errorf("state = %q, want verified", got.VerificationState)
	}
}

func TestProcessDueFailures_ExhaustsAndMarksFailed(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 2)

	// m0 permanently missing; no reposts allowed -> straight to deferred checks.
	stater := newFakeStater(mid(0))
	cfg := Config{
		MaxReposts:        0,
		PropagationDelay:  time.Millisecond,
		DeferredBackoff:   time.Millisecond,
		MaxBackoff:        time.Millisecond,
		MaxDeferredChecks: 2,
	}
	svc := New(store, stater, &fakeReposter{}, cfg, "w")

	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}

	// Run enough cycles to exhaust deferred checks.
	now := time.Now()
	for i := 0; i < 5; i++ {
		now = now.Add(time.Hour)
		if _, err := svc.ProcessDueFailures(ctx, now); err != nil {
			t.Fatalf("ProcessDueFailures #%d: %v", i, err)
		}
	}

	got, _ := store.GetFile(ctx, "t", "f")
	if got.VerificationState != transferstore.StateVerificationFailed {
		t.Errorf("state = %q, want verification_failed", got.VerificationState)
	}
	if n, _ := store.CountFailures(ctx, "t", "f", transferstore.FailureFailed); n != 1 {
		t.Errorf("failed failures = %d, want 1", n)
	}
}

func TestProcessDueFailures_NothingDue(t *testing.T) {
	store := newTestStore(t)
	svc := New(store, newFakeStater(), &fakeReposter{}, Config{}, "w")
	n, err := svc.ProcessDueFailures(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("ProcessDueFailures: %v", err)
	}
	if n != 0 {
		t.Errorf("processed = %d, want 0", n)
	}
}

func TestVerifyDueFiles_VerifiesOnlyDueUploadedFiles(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	writeManifest(t, store, "t", "due", 3)
	writeManifest(t, store, "t", "notdue", 3)
	if err := store.MarkUploaded(ctx, "t", "due", now.Add(-time.Hour), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkUploaded(ctx, "t", "notdue", now.Add(-time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	svc := New(store, newFakeStater(), &fakeReposter{}, Config{}, "w")
	n, err := svc.VerifyDueFiles(ctx, now, 10)
	if err != nil {
		t.Fatalf("VerifyDueFiles: %v", err)
	}
	if n != 1 {
		t.Errorf("verified %d files, want 1 (only the due one)", n)
	}

	dueFile, _ := store.GetFile(ctx, "t", "due")
	if dueFile.VerificationState != transferstore.StateVerified {
		t.Errorf("due file state = %q, want verified", dueFile.VerificationState)
	}
	notDue, _ := store.GetFile(ctx, "t", "notdue")
	if notDue.VerificationState != transferstore.StateUploaded {
		t.Errorf("not-due file state = %q, want still uploaded", notDue.VerificationState)
	}
}

// fakeCleaner records which transfers cleanup was attempted for.
type fakeCleaner struct{ called []string }

func (c *fakeCleaner) CleanupTransfer(_ context.Context, transferID string) (bool, error) {
	c.called = append(c.called, transferID)
	return true, nil
}

func TestVerifyFile_TriggersCleanupWhenVerified(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 3)

	cleaner := &fakeCleaner{}
	svc := New(store, newFakeStater(), &fakeReposter{}, Config{}, "w")
	svc.SetCleaner(cleaner)

	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if len(cleaner.called) != 1 || cleaner.called[0] != "t" {
		t.Errorf("cleanup called for %v, want [t]", cleaner.called)
	}
}

func TestVerifyFile_NoCleanupWhenMissesRemain(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 3)

	cleaner := &fakeCleaner{}
	svc := New(store, newFakeStater(mid(1)), &fakeReposter{}, Config{}, "w")
	svc.SetCleaner(cleaner)

	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if len(cleaner.called) != 0 {
		t.Errorf("cleanup must not run while misses remain, got %v", cleaner.called)
	}
}
