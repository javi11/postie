package apikey

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			key TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ'))
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestGenerateUnique(t *testing.T) {
	t.Parallel()
	a, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if a == b {
		t.Fatal("expected distinct keys, got same value twice")
	}
	if len(a) < 32 {
		t.Fatalf("expected non-trivial key length, got %d", len(a))
	}
}

func TestEnsureKey_GeneratesAndPersists(t *testing.T) {
	t.Parallel()
	store := NewSQLStore(newTestDB(t))
	ctx := context.Background()

	k1, err := EnsureKey(ctx, store)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if k1 == "" {
		t.Fatal("expected non-empty key")
	}

	k2, err := EnsureKey(ctx, store)
	if err != nil {
		t.Fatalf("ensure (second call): %v", err)
	}
	if k1 != k2 {
		t.Fatalf("EnsureKey should be idempotent, got %q vs %q", k1, k2)
	}
}

func TestRegenerateReplacesKey(t *testing.T) {
	t.Parallel()
	store := NewSQLStore(newTestDB(t))
	ctx := context.Background()

	first, err := EnsureKey(ctx, store)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	second, err := Regenerate(ctx, store)
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if first == second {
		t.Fatal("regenerate should produce a different key")
	}

	got, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != second {
		t.Fatalf("persisted key %q != regenerated %q", got, second)
	}
}

func TestSetEmptyKeyRejected(t *testing.T) {
	t.Parallel()
	store := NewSQLStore(newTestDB(t))
	if err := store.Set(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty key")
	}
}
