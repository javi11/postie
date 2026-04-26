// Package apikey manages the persistent HTTP API key used to authenticate
// callers of the gated upload endpoint. The key is stored in SQLite (single-row
// table) so it survives restarts and can be rotated without rewriting config.
package apikey

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
)

// Store reads and writes the single API key row.
type Store interface {
	// Get returns the stored key, or "" (no error) when no key has been
	// persisted yet.
	Get(ctx context.Context) (string, error)
	// Set upserts the API key into the single-row api_keys table.
	Set(ctx context.Context, key string) error
}

// SQLStore is the SQLite-backed implementation of Store.
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore wraps an open *sql.DB.
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

// Get returns the persisted key or an empty string if none exists.
func (s *SQLStore) Get(ctx context.Context) (string, error) {
	var key string
	err := s.db.QueryRowContext(ctx, `SELECT key FROM api_keys WHERE id = 1`).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read api key: %w", err)
	}
	return key, nil
}

// Set upserts the key, replacing any prior value.
func (s *SQLStore) Set(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("api key cannot be empty")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, key) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET key = excluded.key,
		                              created_at = strftime('%Y-%m-%dT%H:%M:%fZ')
	`, key)
	if err != nil {
		return fmt.Errorf("write api key: %w", err)
	}
	return nil
}

// Generate returns a fresh URL-safe random key (32 bytes → 43 chars).
func Generate() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// EnsureKey returns the existing key, or generates and persists a new one if
// none has been stored. Idempotent — safe to call on every startup.
func EnsureKey(ctx context.Context, s Store) (string, error) {
	existing, err := s.Get(ctx)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	key, err := Generate()
	if err != nil {
		return "", err
	}
	if err := s.Set(ctx, key); err != nil {
		return "", err
	}
	return key, nil
}

// Regenerate forces a new key, persists it, and returns it.
func Regenerate(ctx context.Context, s Store) (string, error) {
	key, err := Generate()
	if err != nil {
		return "", err
	}
	if err := s.Set(ctx, key); err != nil {
		return "", err
	}
	return key, nil
}
