package transferstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/database"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	db, err := database.New(ctx, config.DatabaseConfig{
		DatabaseType: "sqlite",
		DatabasePath: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.GetMigrationRunner().MigrateUp(); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	return New(db.DB)
}

func TestUpsertAndGetFile(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	posted := time.Unix(1700000000, 0).UTC()

	f := TransferFile{
		TransferID:    "tid-1",
		FileID:        "fid-1",
		ManifestPath:  "/m/tid-1/fid-1.jsonl.zst",
		SourcePath:    "/data/a.mkv",
		FileRole:      "original",
		ArticleCount:  42,
		UploadState:   StateUploaded,
		PostedAt:      &posted,
		CleanupPolicy: "delete_original",
	}
	if err := s.UpsertFile(ctx, f); err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	got, err := s.GetFile(ctx, "tid-1", "fid-1")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if got.ArticleCount != 42 || got.UploadState != StateUploaded || got.SourcePath != "/data/a.mkv" {
		t.Errorf("got %+v", got)
	}
	if got.PostedAt == nil || !got.PostedAt.Equal(posted) {
		t.Errorf("PostedAt = %v, want %v", got.PostedAt, posted)
	}
	if got.VerificationState != StatePlanned {
		t.Errorf("VerificationState default = %q, want %q", got.VerificationState, StatePlanned)
	}

	// Upsert again updates in place (no duplicate).
	f.ArticleCount = 99
	if err := s.UpsertFile(ctx, f); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	files, err := s.ListFilesByTransfer(ctx, "tid-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 1 || files[0].ArticleCount != 99 {
		t.Errorf("expected single updated row, got %+v", files)
	}
}

func TestSetStates(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.UpsertFile(ctx, TransferFile{TransferID: "t", FileID: "f", ManifestPath: "m", SourcePath: "s", FileRole: "original"})

	next := time.Now().Add(time.Hour).UTC()
	if err := s.SetVerificationState(ctx, "t", "f", StateVerifying, &next, "boom"); err != nil {
		t.Fatalf("SetVerificationState: %v", err)
	}
	got, _ := s.GetFile(ctx, "t", "f")
	if got.VerificationState != StateVerifying || got.LastError != "boom" {
		t.Errorf("got %+v", got)
	}
	if got.NextCheckAt == nil || got.NextCheckAt.Sub(next).Abs() > time.Second {
		t.Errorf("NextCheckAt = %v, want ~%v", got.NextCheckAt, next)
	}
}

func TestGetFileNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetFile(context.Background(), "nope", "nope")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestAddFailureIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	f := VerificationFailure{
		TransferID: "t", FileID: "f", ArticleIndex: 3,
		MessageID: "<a@p>", Groups: []string{"g1", "g2"},
	}
	if err := s.AddFailure(ctx, f); err != nil {
		t.Fatalf("AddFailure: %v", err)
	}
	// Duplicate message id ignored.
	if err := s.AddFailure(ctx, f); err != nil {
		t.Fatalf("AddFailure dup: %v", err)
	}
	n, err := s.CountFailures(ctx, "t", "f", "")
	if err != nil {
		t.Fatalf("CountFailures: %v", err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1 (idempotent insert)", n)
	}
}

func TestClaimDueFailuresLeasing(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Two due, one not yet due.
	_ = s.AddFailure(ctx, VerificationFailure{TransferID: "t", FileID: "f", MessageID: "<1>", NextAttemptAt: now.Add(-time.Minute)})
	_ = s.AddFailure(ctx, VerificationFailure{TransferID: "t", FileID: "f", MessageID: "<2>", NextAttemptAt: now.Add(-time.Minute)})
	_ = s.AddFailure(ctx, VerificationFailure{TransferID: "t", FileID: "f", MessageID: "<3>", NextAttemptAt: now.Add(time.Hour)})

	claimed, err := s.ClaimDueFailures(ctx, "worker-1", 5*time.Minute, 10, now)
	if err != nil {
		t.Fatalf("ClaimDueFailures: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed %d, want 2", len(claimed))
	}

	// A second claim immediately returns nothing (leases held, not expired).
	again, err := s.ClaimDueFailures(ctx, "worker-2", 5*time.Minute, 10, now)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("second claim got %d, want 0 (leases held)", len(again))
	}

	// After leases expire, the work is reclaimable.
	reclaimed, err := s.ReclaimExpiredLeases(ctx, now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("ReclaimExpiredLeases: %v", err)
	}
	if reclaimed != 2 {
		t.Errorf("reclaimed %d, want 2", reclaimed)
	}
	after, err := s.ClaimDueFailures(ctx, "worker-2", time.Minute, 10, now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("claim after reclaim: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("claim after reclaim got %d, want 2", len(after))
	}
}

func TestUpdateFailureAfterCheckResolves(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	_ = s.AddFailure(ctx, VerificationFailure{TransferID: "t", FileID: "f", MessageID: "<1>", NextAttemptAt: now.Add(-time.Minute)})

	claimed, err := s.ClaimDueFailures(ctx, "w", time.Minute, 10, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v len=%d", err, len(claimed))
	}

	upd := claimed[0]
	upd.State = FailureResolved
	upd.RepostCount = 1
	upd.NextAttemptAt = now
	if err := s.UpdateFailureAfterCheck(ctx, upd); err != nil {
		t.Fatalf("UpdateFailureAfterCheck: %v", err)
	}

	pending, err := s.CountFailures(ctx, "t", "f", FailurePending)
	if err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 0 {
		t.Errorf("pending = %d, want 0 (resolved)", pending)
	}
	resolved, _ := s.CountFailures(ctx, "t", "f", FailureResolved)
	if resolved != 1 {
		t.Errorf("resolved = %d, want 1", resolved)
	}
}

func TestMarkUploadedAndListDueFiles(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for _, fid := range []string{"f-past", "f-future"} {
		if err := s.UpsertFile(ctx, TransferFile{
			TransferID: "t", FileID: fid, ManifestPath: "m", SourcePath: "s", FileRole: "original", ArticleCount: 1,
		}); err != nil {
			t.Fatalf("UpsertFile %s: %v", fid, err)
		}
	}

	posted := now.Add(-time.Hour)
	if err := s.MarkUploaded(ctx, "t", "f-past", posted, now.Add(-time.Minute)); err != nil {
		t.Fatalf("MarkUploaded past: %v", err)
	}
	if err := s.MarkUploaded(ctx, "t", "f-future", posted, now.Add(time.Hour)); err != nil {
		t.Fatalf("MarkUploaded future: %v", err)
	}

	// Both should be in the uploaded state with posted_at set.
	pastFile, _ := s.GetFile(ctx, "t", "f-past")
	if pastFile.UploadState != StateUploaded || pastFile.VerificationState != StateUploaded {
		t.Errorf("f-past states = %q/%q, want uploaded/uploaded", pastFile.UploadState, pastFile.VerificationState)
	}
	if pastFile.PostedAt == nil || !pastFile.PostedAt.Equal(posted) {
		t.Errorf("f-past PostedAt = %v, want %v", pastFile.PostedAt, posted)
	}

	// Only the past-due file is returned by ListDueFiles.
	due, err := s.ListDueFiles(ctx, now, 10)
	if err != nil {
		t.Fatalf("ListDueFiles: %v", err)
	}
	if len(due) != 1 || due[0].FileID != "f-past" {
		t.Errorf("ListDueFiles returned %d files (want 1=f-past): %+v", len(due), due)
	}

	// After the future file becomes due, it is returned too.
	dueLater, _ := s.ListDueFiles(ctx, now.Add(2*time.Hour), 10)
	if len(dueLater) != 2 {
		t.Errorf("ListDueFiles later returned %d, want 2", len(dueLater))
	}
}
