// Package webauth adapts AltMount's direct admin authentication to Postie's
// HTTP server. Wails RPC and CLI operation do not use this service.
package webauth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrRegistrationClosed = errors.New("registration is disabled")
	ErrCredentials        = errors.New("invalid credentials")
)

const SessionDuration = 24 * time.Hour

// Service persists the single administrator, signing key and revocable sessions.
type Service struct {
	db     *sql.DB
	tokens *token.Service
	secret string
}

func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// New loads a persistent random signing secret. Concurrent initializers converge
// on the same database row; a restart does not invalidate valid sessions.
func New(db *sql.DB) (*Service, error) {
	if db == nil {
		return nil, errors.New("authentication database unavailable")
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(`INSERT OR IGNORE INTO web_auth_secret(id,secret) VALUES(1,?)`, secret); err != nil {
		return nil, fmt.Errorf("initialize signing secret: %w", err)
	}
	if err = db.QueryRow(`SELECT secret FROM web_auth_secret WHERE id=1`).Scan(&secret); err != nil {
		return nil, fmt.Errorf("read signing secret: %w", err)
	}
	return &Service{db: db, secret: secret, tokens: token.NewService(token.Opts{SecretReader: token.SecretFunc(func(string) (string, error) { return secret, nil }), Issuer: "postie", TokenDuration: SessionDuration})}, nil
}

func (s *Service) RegistrationOpen(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM web_admin`).Scan(&count); err != nil {
		return false, fmt.Errorf("read admin: %w", err)
	}
	return count == 0, nil
}

func (s *Service) Register(ctx context.Context, username, email, password string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if len(username) < 3 || len(username) > 128 || len(email) > 254 || len(password) < 12 || len(password) > 72 {
		return errors.New("username must have 3–128 characters and password 12–72 bytes")
	}
	open, err := s.RegistrationOpen(ctx)
	if err != nil {
		return err
	}
	if !open {
		return ErrRegistrationClosed
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO web_admin(id,username,email,password_hash) VALUES(1,?,?,?)`, username, email, string(hash))
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	if count != 1 {
		return ErrRegistrationClosed
	}
	return nil
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	var stored, email, hash string
	err := s.db.QueryRowContext(ctx, `SELECT username,email,password_hash FROM web_admin WHERE id=1`).Scan(&stored, &email, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrCredentials
	}
	if err != nil {
		return "", fmt.Errorf("read admin: %w", err)
	}
	// Always verify the hash, including unknown users, to avoid an account oracle.
	hashErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if hashErr != nil || (username != stored && (email == "" || username != email)) {
		return "", ErrCredentials
	}
	return s.Issue(ctx, stored)
}

func (s *Service) Issue(ctx context.Context, username string) (string, error) {
	id, err := randomSecret()
	if err != nil {
		return "", err
	}
	now := time.Now()
	expiry := now.Add(SessionDuration)
	claims := token.Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "postie", Subject: username, Audience: jwt.ClaimStrings{"postie-api"}, ExpiresAt: jwt.NewNumericDate(expiry), IssuedAt: jwt.NewNumericDate(now), ID: id}, User: &token.User{ID: username, Name: username}}
	signed, err := s.tokens.Token(claims)
	if err != nil {
		return "", fmt.Errorf("sign session: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM web_sessions WHERE expires_at<=?`, now.Unix())
	if err != nil {
		return "", fmt.Errorf("clean expired sessions: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO web_sessions(id,expires_at) VALUES(?,?)`, id, expiry.Unix())
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return signed, nil
}

// Validate additionally pins HS256, issuer, audience and expiration (AltMount's
// token representation is reused), and checks that logout has not revoked it.
func (s *Service) Validate(ctx context.Context, signed string) (*token.Claims, error) {
	claims := &token.Claims{}
	_, err := jwt.ParseWithClaims(signed, claims, func(*jwt.Token) (any, error) { return []byte(s.secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("postie"), jwt.WithAudience("postie-api"), jwt.WithExpirationRequired())
	if err != nil {
		return nil, ErrCredentials
	}
	var username string
	err = s.db.QueryRowContext(ctx, `SELECT a.username FROM web_sessions s CROSS JOIN web_admin a WHERE s.id=? AND s.expires_at>? AND a.id=1`, claims.ID, time.Now().Unix()).Scan(&username)
	if err != nil || username != claims.Subject || claims.User == nil || claims.User.ID != username {
		return nil, ErrCredentials
	}
	return claims, nil
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM web_sessions WHERE id=?`, id); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
