package webauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
)

// Every token below uses a real, persisted login session. Re-signing modified
// claims isolates claim validation from the signature and session checks.
func TestValidateRejectsInvalidSignedClaims(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	if err := s.Register(ctx, "admin", "", "a-long-secret-password"); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	signed, err := s.Login(ctx, "admin", "a-long-secret-password")
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}

	cases := []struct {
		name   string
		modify func(*token.Claims)
	}{
		{"issuer", func(c *token.Claims) { c.Issuer = "altmount" }},
		{"audience", func(c *token.Claims) { c.Audience = jwt.ClaimStrings{"altmount-api"} }},
		{"missing audience", func(c *token.Claims) { c.Audience = nil }},
		{"expired", func(c *token.Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }},
		{"missing expiry", func(c *token.Claims) { c.ExpiresAt = nil }},
		{"not yet valid", func(c *token.Claims) { c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour)) }},
		{"subject", func(c *token.Claims) { c.Subject = "other" }},
		{"missing user", func(c *token.Claims) { c.User = nil }},
		{"user identity", func(c *token.Claims) { c.User.ID = "other" }},
		{"unknown session", func(c *token.Claims) { c.ID = "unregistered-session" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := s.Validate(ctx, signed)
			if err != nil {
				t.Fatalf("NoError check failed: %v", err)
			}
			tc.modify(claims)
			changed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.secret))
			if err != nil {
				t.Fatalf("NoError check failed: %v", err)
			}
			_, err = s.Validate(ctx, changed)
			if !errors.Is(err, ErrCredentials) {
				t.Fatalf("ErrorIs check failed: %v %v", err, ErrCredentials)
			}
		})
	}

	t.Run("different signing algorithm", func(t *testing.T) {
		claims, err := s.Validate(ctx, signed)
		if err != nil {
			t.Fatalf("NoError check failed: %v", err)
		}
		changed, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(s.secret))
		if err != nil {
			t.Fatalf("NoError check failed: %v", err)
		}
		_, err = s.Validate(ctx, changed)
		if !errors.Is(err, ErrCredentials) {
			t.Fatalf("ErrorIs check failed: %v %v", err, ErrCredentials)
		}
	})
	t.Run("wrong signing key", func(t *testing.T) {
		claims, err := s.Validate(ctx, signed)
		if err != nil {
			t.Fatalf("NoError check failed: %v", err)
		}
		changed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("different-secret"))
		if err != nil {
			t.Fatalf("NoError check failed: %v", err)
		}
		_, err = s.Validate(ctx, changed)
		if !errors.Is(err, ErrCredentials) {
			t.Fatalf("ErrorIs check failed: %v %v", err, ErrCredentials)
		}
	})
	t.Run("database session expired", func(t *testing.T) {
		claims, err := s.Validate(ctx, signed)
		if err != nil {
			t.Fatalf("NoError check failed: %v", err)
		}
		_, err = s.db.ExecContext(ctx, "UPDATE web_sessions SET expires_at=? WHERE id=?", time.Now().Add(-time.Minute).Unix(), claims.ID)
		if err != nil {
			t.Fatalf("NoError check failed: %v", err)
		}
		_, err = s.Validate(ctx, signed)
		if !errors.Is(err, ErrCredentials) {
			t.Fatalf("ErrorIs check failed: %v %v", err, ErrCredentials)
		}
	})
}
