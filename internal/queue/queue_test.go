package queue

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/javi11/postie/internal/config"
	"github.com/javi11/postie/internal/database"
)

// newTestQueue creates an isolated Queue backed by a temp sqlite DB with all
// migrations applied. The DB is removed automatically when the test ends.
func newTestQueue(t *testing.T) *Queue {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ctx := context.Background()
	db, err := database.New(ctx, config.DatabaseConfig{
		DatabaseType: "sqlite",
		DatabasePath: dbPath,
	})
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.GetMigrationRunner().MigrateUp(); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}

	q, err := New(ctx, db)
	if err != nil {
		t.Fatalf("queue.New: %v", err)
	}
	t.Cleanup(func() { _ = q.Close() })
	return q
}

func countRows(t *testing.T, q *Queue, table, where string, args ...any) int {
	t.Helper()
	var n int
	query := "SELECT COUNT(*) FROM " + table
	if where != "" {
		query += " WHERE " + where
	}
	if err := q.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestRetryDoesNotLeakInProgressRows reproduces the duplicate-entries bug.
// Without ClearInProgress before ReaddJob, each retry leaks an in_progress
// row keyed by the previous goqite message ID, causing the same path to
// appear in two tables simultaneously (pending goqite + stale in_progress).
func TestRetryDoesNotLeakInProgressRows(t *testing.T) {
	q := newTestQueue(t)
	ctx := context.Background()

	const path = "/tmp/example.bin"
	if err := q.AddFile(ctx, path, 1234); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	// Simulate 3 receive→fail→retry cycles, mirroring processor.handleProcessingError.
	for i := range 3 {
		msg, job, err := q.ReceiveFile(ctx)
		if err != nil {
			t.Fatalf("ReceiveFile #%d: %v", i, err)
		}
		if msg == nil || job == nil {
			t.Fatalf("ReceiveFile #%d returned nil", i)
		}

		// Invariant: while processing, exactly one in_progress row for this path.
		if got := countRows(t, q, "in_progress_items", "path = ?", path); got != 1 {
			t.Fatalf("cycle %d: in_progress rows = %d, want 1", i, got)
		}

		// Simulate the retry path in handleProcessingError.
		if err := q.ClearInProgress(ctx, msg.ID); err != nil {
			t.Fatalf("ClearInProgress: %v", err)
		}
		job.RetryCount++
		if err := q.ReaddJob(ctx, job); err != nil {
			t.Fatalf("ReaddJob: %v", err)
		}

		// After clear+readd: exactly one pending goqite row, zero in_progress rows.
		if got := countRows(t, q, "in_progress_items", "path = ?", path); got != 0 {
			t.Fatalf("cycle %d: in_progress leak after ClearInProgress: %d rows", i, got)
		}
		if got := countRows(t, q, "goqite", "queue = 'file_jobs' AND json_extract(body, '$.path') = ?", path); got != 1 {
			t.Fatalf("cycle %d: pending goqite rows = %d, want 1", i, got)
		}
	}
}

// TestIsPathInQueueDuringReceive verifies the path is always visible to
// IsPathInQueue during ReceiveFile (insert-then-delete ordering).
func TestIsPathInQueueDuringReceive(t *testing.T) {
	q := newTestQueue(t)
	ctx := context.Background()

	const path = "/tmp/race.bin"
	if err := q.AddFile(ctx, path, 100); err != nil {
		t.Fatalf("AddFile: %v", err)
	}

	if ok, err := q.IsPathInQueue(path); err != nil || !ok {
		t.Fatalf("before receive: IsPathInQueue=%v err=%v, want true,nil", ok, err)
	}

	msg, _, err := q.ReceiveFile(ctx)
	if err != nil || msg == nil {
		t.Fatalf("ReceiveFile: msg=%v err=%v", msg, err)
	}

	if ok, err := q.IsPathInQueue(path); err != nil || !ok {
		t.Fatalf("after receive (in_progress): IsPathInQueue=%v err=%v, want true,nil", ok, err)
	}

	if err := q.ClearInProgress(ctx, msg.ID); err != nil {
		t.Fatalf("ClearInProgress: %v", err)
	}

	if ok, err := q.IsPathInQueue(path); err != nil || ok {
		t.Fatalf("after clear: IsPathInQueue=%v err=%v, want false,nil", ok, err)
	}
}
