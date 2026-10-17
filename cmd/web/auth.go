package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/kipsilabs/postie/internal/webauth"
)

const sessionCookie = "postie_session"

type sessionContextKey struct{}

// sameOrigin deliberately ignores client supplied proxy headers. Proxies should
// preserve Host; HTTPS deployments terminate TLS here or set POSTIE_COOKIE_SECURE.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	scheme := "http"
	if r.TLS != nil || os.Getenv("POSTIE_COOKIE_SECURE") == "true" {
		scheme = "https"
	}
	return err == nil && parsed.Scheme == scheme && parsed.Host == r.Host && parsed.User == nil && parsed.Path == "" && parsed.RawQuery == "" && parsed.Fragment == ""
}

func (ws *WebServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ws.authRequired {
			if r.URL.Path == "/api/auth/user" && r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", "no-store")
				writeJSON(w, http.StatusOK, map[string]bool{"enabled": false})
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				http.NotFound(w, r)
				return
			}
			if r.URL.Path == "/login" || r.URL.Path == "/login.html" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			// Preserve existing unauthenticated deployments, including their CORS API.
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		// Integrations enforce their own API key, never a browser cookie. The Arr
		// exception is limited to POST on the actual receiver, not its admin APIs.
		if strings.HasPrefix(r.URL.Path, "/api/v1/") || (r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/arr/webhook/") && len(strings.Split(strings.TrimPrefix(r.URL.Path, "/api/arr/webhook/"), "/")) == 1) {
			next.ServeHTTP(w, r)
			return
		}
		if !sameOrigin(r) {
			http.Error(w, "Cross-origin request rejected", http.StatusForbidden)
			return
		}
		unsafe := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
		if unsafe && r.Header.Get("Origin") == "" && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "JSON or same-origin request required", http.StatusForbidden)
			return
		}
		// Static assets and the login shell contain no app data. Every other page
		// and API (including WebSocket) requires a valid, persisted session.
		if r.URL.Path == "/live" || r.URL.Path == "/login" || r.URL.Path == "/login.html" || strings.HasPrefix(r.URL.Path, "/_app/") || r.URL.Path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}
		if ws.auth == nil {
			http.Error(w, "Authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" || r.URL.Path == "/api/auth/registration-status" {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(sessionCookie)
		var claims *token.Claims
		if err == nil {
			claims, err = ws.auth.Validate(r.Context(), cookie.Value)
		}
		if err != nil || claims == nil {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, claims)))
	})
}

func (ws *WebServer) setSessionCookie(w http.ResponseWriter, r *http.Request, value string, maxAge int) {
	w.Header().Set("Cache-Control", "no-store")
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: value, Path: "/", HttpOnly: true, Secure: r.TLS != nil || os.Getenv("POSTIE_COOKIE_SECURE") == "true", SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}

func (ws *WebServer) handleRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	open, err := ws.auth.RegistrationOpen(r.Context())
	if err != nil {
		http.Error(w, "Authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]bool{"registration_enabled": open})
}

type credentials struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func readCredentials(w http.ResponseWriter, r *http.Request) (credentials, error) {
	var creds credentials
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return creds, errors.New("JSON required")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	err := decoder.Decode(&creds)
	if err == nil && (creds.Username == "" || creds.Password == "" || len(creds.Password) > 72) {
		err = errors.New("invalid credentials")
	}
	return creds, err
}

// authLimiter caps bcrypt work without trusting forwarded client addresses. Its
// map is bounded to prevent unbounded memory growth from unauthenticated traffic.
type authAttempt struct {
	start time.Time
	count int
}
type authLimiter struct {
	mu       sync.Mutex
	attempts map[string]authAttempt
}

func (l *authLimiter) allow(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.attempts == nil {
		l.attempts = make(map[string]authAttempt)
	}
	now := time.Now()
	for key, attempt := range l.attempts {
		if now.Sub(attempt.start) >= time.Minute {
			delete(l.attempts, key)
		}
	}
	attempt, exists := l.attempts[host]
	if !exists {
		if len(l.attempts) >= 1024 {
			return false
		}
		attempt.start = now
	}
	if attempt.count >= 10 {
		return false
	}
	attempt.count++
	l.attempts[host] = attempt
	return true
}

func (ws *WebServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !ws.authLimiter.allow(r) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Too many attempts", http.StatusTooManyRequests)
		return
	}
	creds, err := readCredentials(w, r)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	signed, err := ws.auth.Login(r.Context(), creds.Username, creds.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	ws.setSessionCookie(w, r, signed, int(webauth.SessionDuration.Seconds()))
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]string{"username": creds.Username})
}

func (ws *WebServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !ws.authLimiter.allow(r) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Too many attempts", http.StatusTooManyRequests)
		return
	}
	creds, err := readCredentials(w, r)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err = ws.auth.Register(r.Context(), creds.Username, creds.Email, creds.Password); err != nil {
		if errors.Is(err, webauth.ErrRegistrationClosed) {
			http.Error(w, "Registration disabled", http.StatusForbidden)
		} else {
			http.Error(w, "Registration failed: username must have 3–128 characters and password 12–72 bytes", http.StatusBadRequest)
		}
		return
	}
	signed, err := ws.auth.Login(r.Context(), strings.TrimSpace(creds.Username), creds.Password)
	if err != nil {
		http.Error(w, "Sign in to continue", http.StatusInternalServerError)
		return
	}
	ws.setSessionCookie(w, r, signed, int(webauth.SessionDuration.Seconds()))
	writeJSON(w, 201, map[string]string{"username": strings.TrimSpace(creds.Username)})
}

func (ws *WebServer) handleUser(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(sessionContextKey{}).(*token.Claims)
	writeJSON(w, 200, map[string]any{"enabled": true, "username": claims.Subject})
}

func (ws *WebServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(sessionContextKey{}).(*token.Claims)
	if err := ws.auth.Revoke(r.Context(), claims.ID); err != nil {
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}
	ws.wsHub.DisconnectSession(claims.ID)
	ws.setSessionCookie(w, r, "", -1)
	writeJSON(w, 200, map[string]bool{"success": true})
}
