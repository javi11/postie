package verification

import (
	"context"
	"testing"
	"time"

	"github.com/kipsilabs/postie/internal/manifest"
	"github.com/kipsilabs/postie/internal/transferstore"
)

// A slow repost must still receive the full propagation delay after it
// finishes, even when it takes longer than that delay to upload.
func TestProcessDueFailures_PropagationDelayStartsAfterRepost(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 1)

	now := time.Now().UTC().Truncate(time.Second)
	delay := 10 * time.Second
	stater := newFakeStater(mid(0))
	reposter := &fakeReposter{onRepost: func(manifest.ArticleRecord) {
		now = now.Add(time.Minute) // time spent uploading the missing article
	}}
	svc := New(store, stater, reposter, Config{
		MaxReposts: 1, PropagationDelay: delay, MaxDeferredChecks: 1,
	}, "w")
	svc.now = func() time.Time { return now }
	tf, err := store.GetFile(ctx, "t", "f")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatal(err)
	}
	now = now.Add(delay)
	if n, err := svc.ProcessDueFailures(ctx, now); err != nil || n != 1 {
		t.Fatalf("repost cycle: processed=%d, err=%v", n, err)
	}
	if reposter.count() != 1 {
		t.Fatal("missing article was not reposted")
	}

	// Propagation has not finished yet. Checking now would exhaust the final
	// deferred attempt and permanently fail an article still propagating.
	now = now.Add(delay - time.Second)
	if n, err := svc.ProcessDueFailures(ctx, now); err != nil || n != 0 {
		t.Fatalf("checked before repost propagation delay elapsed: processed=%d, err=%v", n, err)
	}
	stater.markPresent(mid(0))
	now = now.Add(time.Second)
	if n, err := svc.ProcessDueFailures(ctx, now); err != nil || n != 1 {
		t.Fatalf("propagated cycle: processed=%d, err=%v", n, err)
	}
	got, err := store.GetFile(ctx, "t", "f")
	if err != nil {
		t.Fatal(err)
	}
	if got.VerificationState != transferstore.StateVerified {
		t.Fatalf("state=%q, want verified", got.VerificationState)
	}
}

// Regression: files could go terminal before the processor linked the
// completed item; finalizeTransfer then ran destructive cleanup with an empty
// item id (originals/manifests deleted while the queue still showed
// pending_verification). The finalize must defer until the link exists and be
// picked up by the reconcile pass afterwards.
func TestFinalizeTransfer_DefersUntilItemLinked(t *testing.T) {
	store, db := newTestStoreWithDB(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 2)

	cleaner := &fakeCleaner{}
	svc := New(store, newFakeStater(), &fakeReposter{}, Config{}, "w")
	svc.SetCleaner(cleaner)

	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}

	got, _ := store.GetFile(ctx, "t", "f")
	if got.VerificationState != transferstore.StateVerified {
		t.Fatalf("file state = %q, want verified", got.VerificationState)
	}
	if len(cleaner.called) != 0 {
		t.Fatalf("cleanup ran before the completed item was linked: %v", cleaner.called)
	}

	// The processor links the item after Post() returns; the reconcile pass
	// (startup + periodic) must then finalize the deferred transfer.
	insertCompletedItem(t, store, db, "t", "ci-late")
	if err := svc.ReconcileStuck(ctx); err != nil {
		t.Fatalf("ReconcileStuck: %v", err)
	}
	if status := completedItemStatus(t, db, "ci-late"); status != "verified" {
		t.Errorf("completed item status = %q, want verified", status)
	}
	if len(cleaner.called) != 1 || cleaner.called[0] != "t" {
		t.Errorf("cleanup after linking = %v, want [t]", cleaner.called)
	}
}

// Regression: with the verify pool shared with uploads, the busy-gate skipped
// every cycle while uploads had queued workers — on a continuously busy server
// verification (and repost recovery) never ran. After maxBusySkips consecutive
// deferrals a full cycle must run anyway.
func TestRunOnce_BusyGateForcesCycleAfterMaxSkips(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 2)
	past := time.Now().Add(-time.Hour)
	if err := store.MarkUploaded(ctx, "t", "f", past, past); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}

	svc := New(store, newFakeStater(), &fakeReposter{}, Config{}, "w")
	svc.SetBusyCheck(func() bool { return true })

	for range maxBusySkips {
		svc.runOnce(ctx)
	}
	got, _ := store.GetFile(ctx, "t", "f")
	if got.VerificationState == transferstore.StateVerified {
		t.Fatal("file verified while the busy-gate should still be deferring")
	}

	// Next cycle exceeds maxBusySkips and must run despite busy=true.
	svc.runOnce(ctx)
	got, _ = store.GetFile(ctx, "t", "f")
	if got.VerificationState != transferstore.StateVerified {
		t.Errorf("file state = %q, want verified after forced cycle", got.VerificationState)
	}
}

// Regression: a transient STAT error (timeout, dropped connection) was treated
// as "article missing" and consumed repost budget, re-uploading data that
// already existed and eventually marking files verification_failed during
// provider flakiness. A check error must reschedule without consuming budget.
func TestProcessDueFailures_StatErrorDoesNotConsumeBudget(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	writeManifest(t, store, "t", "f", 3)

	stater := newFakeStater(mid(1))
	reposter := &fakeReposter{}
	cfg := Config{MaxReposts: 1, PropagationDelay: time.Millisecond, DeferredBackoff: time.Minute}
	svc := New(store, stater, reposter, cfg, "w")

	tf, _ := store.GetFile(ctx, "t", "f")
	if err := svc.VerifyFile(ctx, tf); err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if n, _ := store.CountFailures(ctx, "t", "f", transferstore.FailurePending); n != 1 {
		t.Fatalf("pending after sweep = %d, want 1", n)
	}

	// The confirming STAT now errors: no repost, failure stays pending.
	stater.markErrored(mid(1))
	now := time.Now().Add(time.Second)
	if _, err := svc.ProcessDueFailures(ctx, now); err != nil {
		t.Fatalf("ProcessDueFailures (errored): %v", err)
	}
	if reposter.count() != 0 {
		t.Fatalf("reposts after stat error = %d, want 0 (budget must not be consumed)", reposter.count())
	}
	if n, _ := store.CountFailures(ctx, "t", "f", transferstore.FailurePending); n != 1 {
		t.Fatalf("pending after stat error = %d, want 1", n)
	}

	// Once the check works again and confirms the miss, the full repost budget
	// must still be available.
	stater.clearErrored(mid(1))
	now = now.Add(2 * time.Minute) // past the error reschedule backoff
	if _, err := svc.ProcessDueFailures(ctx, now); err != nil {
		t.Fatalf("ProcessDueFailures (recovered): %v", err)
	}
	if reposter.count() != 1 {
		t.Errorf("reposts after recovery = %d, want 1", reposter.count())
	}
}
