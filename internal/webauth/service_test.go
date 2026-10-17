package webauth

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kipsilabs/postie/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "auth.db")+"?_busy_timeout=5000")
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if err := database.NewMigrationRunner(db).MigrateUp(); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	s, err := New(db)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	return s
}

func TestBootstrapLoginRevocationAndRestart(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	open, err := s.RegistrationOpen(ctx)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if !open {
		t.Fatalf("True check failed: %v", open)
	}
	if err := s.Register(ctx, "admin", "admin@example.com", "a-long-secret-password"); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if err := s.Register(ctx, "other", "", "another-long-password"); !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("registration should close after first administrator: %v", err)
	}
	_, err = s.Login(ctx, "admin", "incorrect")
	if !errors.Is(err, ErrCredentials) {
		t.Fatalf("ErrorIs check failed: %v %v", err, ErrCredentials)
	}
	jwt, err := s.Login(ctx, "admin@example.com", "a-long-secret-password")
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	claims, err := s.Validate(ctx, jwt)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if claims.Subject != "admin" {
		t.Fatalf("Equal check failed: %v %v", "admin", claims.Subject)
	}
	restarted, err := New(s.db)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	_, err = restarted.Validate(ctx, jwt)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if err := s.Revoke(ctx, claims.ID); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	_, err = restarted.Validate(ctx, jwt)
	if err == nil {
		t.Fatalf("Error check failed: %v", err)
	}
	_, err = s.Validate(ctx, jwt+"tampered")
	if err == nil {
		t.Fatalf("Error check failed: %v", err)
	}
}

func TestBootstrapAtomic(t *testing.T) {
	s := testService(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- s.Register(context.Background(), "admin", "", "a-long-secret-password") })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			if !errors.Is(err, ErrRegistrationClosed) {
				t.Fatalf("ErrorIs check failed: %v %v", err, ErrRegistrationClosed)
			}
		}
	}
	if successes != 1 {
		t.Fatalf("Equal check failed: %v %v", 1, successes)
	}
}
