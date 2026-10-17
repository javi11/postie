package main

import (
	"context"
	"database/sql"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kipsilabs/postie/internal/database"
	"github.com/kipsilabs/postie/internal/webauth"
	_ "github.com/mattn/go-sqlite3"
)

func authenticatedServer(t *testing.T) (*WebServer, *http.Cookie) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "web.db"))
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
	service, err := webauth.New(db)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	ws := NewWebServer()
	ws.auth = service
	ws.authRequired = true
	if err := service.Register(context.Background(), "admin", "", "a-long-secret-password"); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	signed, err := service.Login(context.Background(), "admin", "a-long-secret-password")
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	return ws, &http.Cookie{Name: "postie_session", Value: signed}
}

func TestRealRouterProtectsWebAPIsAndWebSocket(t *testing.T) {
	ws, _ := authenticatedServer(t)
	for _, path := range []string{"/api/status", "/api/config", "/api/config/pending", "/api/filesystem/browse", "/api/logs/download", "/api/api-key", "/api/arr/instances", "/api/ws"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			ws.router.ServeHTTP(rr, req)
			if rr.Code != 401 {
				t.Fatalf("Equal check failed: %v %v", 401, rr.Code)
			}
		})
	}
	rr := httptest.NewRecorder()
	ws.router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/settings", nil))
	if rr.Code != 303 {
		t.Fatalf("Equal check failed: %v %v", 303, rr.Code)
	}
	if rr.Header().Get("Location") != "/login" {
		t.Fatalf("Equal check failed: %v %v", "/login", rr.Header().Get("Location"))
	}
}

func TestRealRouterCSRFAndIntegrationExceptions(t *testing.T) {
	ws, cookie := authenticatedServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/processor/pause", nil)
	req.AddCookie(cookie)
	req.Header.Set("Origin", "http://attacker.test")
	rr := httptest.NewRecorder()
	ws.router.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatalf("Equal check failed: %v %v", 403, rr.Code)
	}
	// The v1 router must match before the generic /api router and enforce its
	// existing API-enabled/key contract, rather than a browser session.
	rr = httptest.NewRecorder()
	ws.router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/queue/upload", strings.NewReader(`{}`)))
	if rr.Code != 403 {
		t.Fatalf("Equal check failed: %v %v", 403, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "api disabled") {
		t.Fatalf("Contains check failed: %v %v", rr.Body.String(), "api disabled")
	}
	// Arr has its own key validation, and cannot acquire a browser session.
	rr = httptest.NewRecorder()
	ws.router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/arr/webhook/test", nil))
	if rr.Body.String() == "Authentication required\n" {
		t.Fatalf("NotEqual check failed: %v %v", "Authentication required\n", rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/ws", nil)
	req.AddCookie(cookie)
	req.Header.Set("Origin", "http://attacker.test")
	rr = httptest.NewRecorder()
	ws.router.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatalf("Equal check failed: %v %v", 403, rr.Code)
	}
}

func TestLoginCookieLogoutAndRateLimit(t *testing.T) {
	ws, _ := authenticatedServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"a-long-secret-password"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	ws.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, rr.Code)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Len check failed: %v %v", cookies, 1)
	}
	cookie := cookies[0]
	if !cookie.HttpOnly {
		t.Fatalf("True check failed: %v", cookie.HttpOnly)
	}
	if http.SameSiteStrictMode != cookie.SameSite {
		t.Fatalf("Equal check failed: %v %v", http.SameSiteStrictMode, cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Fatalf("Equal check failed: %v %v", "/", cookie.Path)
	}
	if strings.Contains(rr.Body.String(), cookie.Value) {
		t.Fatalf("NotContains check failed: %v %v", rr.Body.String(), cookie.Value)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/auth/user", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	ws.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, rr.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	ws.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, rr.Code)
	}
	if rr.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("Equal check failed: %v %v", -1, rr.Result().Cookies()[0].MaxAge)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/auth/user", nil)
	req.AddCookie(cookie)
	rr = httptest.NewRecorder()
	ws.router.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("Equal check failed: %v %v", 401, rr.Code)
	}
	for range 10 {
		req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"incorrect"}`))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		ws.router.ServeHTTP(rr, req)
	}
	if rr.Code != 429 {
		t.Fatalf("Equal check failed: %v %v", 429, rr.Code)
	}
	if rr.Header().Get("Retry-After") != "60" {
		t.Fatalf("Equal check failed: %v %v", "60", rr.Header().Get("Retry-After"))
	}
}

func TestLogoutDisconnectsExistingWebSocket(t *testing.T) {
	ws, cookie := authenticatedServer(t)
	server := httptest.NewServer(ws.router)
	t.Cleanup(server.Close)
	header := http.Header{"Cookie": []string{cookie.String()}, "Origin": []string{server.URL}}
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/ws", header)
	if response != nil {
		defer func() {
			if err := response.Body.Close(); err != nil {
				t.Fatal(err)
			}
		}()
	}
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ws.wsHub.EmitEvent("test", map[string]string{"value": "before logout"})
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if !strings.Contains(string(message), "before logout") {
		t.Fatalf("Contains check failed: %v %v", string(message), "before logout")
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/auth/logout", nil)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if response.StatusCode != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, response.StatusCode)
	}
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatalf("Error check failed: %v", err)
	}
	netErr, ok := err.(net.Error)
	if ok && netErr.Timeout() {
		t.Fatalf("False check failed: %v", ok && netErr.Timeout())
	}
}

func TestExpiredSessionClosesExistingWebSocket(t *testing.T) {
	ws, cookie := authenticatedServer(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws.wsHub.ServeSessionHTTP(w, r, "test-expiring-session", time.Now().Add(50*time.Millisecond))
	}))
	t.Cleanup(server.Close)
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), http.Header{"Cookie": []string{cookie.String()}})
	if response != nil {
		defer func() {
			if err := response.Body.Close(); err != nil {
				t.Fatal(err)
			}
		}()
	}
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatalf("Error check failed: %v", err)
	}
	netErr, ok := err.(net.Error)
	if ok && netErr.Timeout() {
		t.Fatalf("False check failed: %v", ok && netErr.Timeout())
	}
}

func TestSecureCookieAndBrowserOriginPolicy(t *testing.T) {
	ws, cookie := authenticatedServer(t)
	t.Setenv("POSTIE_COOKIE_SECURE", "true")
	login := httptest.NewRequest(http.MethodPost, "https://postie.example/api/auth/login", strings.NewReader(`{"username":"admin","password":"a-long-secret-password"}`))
	login.Header.Set("Content-Type", "application/json")
	login.Header.Set("Origin", "https://postie.example")
	rr := httptest.NewRecorder()
	ws.router.ServeHTTP(rr, login)
	if rr.Code != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, rr.Code)
	}
	if !rr.Result().Cookies()[0].Secure {
		t.Fatalf("True check failed: %v", rr.Result().Cookies()[0].Secure)
	}
	if rr.Result().Cookies()[0].Domain != "" {
		t.Fatalf("Empty check failed: %v", rr.Result().Cookies()[0].Domain)
	}
	for _, origin := range []string{"http://postie.example", "https://postie.example.attacker.test", "null", "https://attacker.test", "https://user@postie.example"} {
		req := httptest.NewRequest(http.MethodPost, "https://postie.example/api/auth/logout", nil)
		req.AddCookie(cookie)
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		ws.router.ServeHTTP(rr, req)
		if rr.Code != 403 {
			t.Fatalf("Equal check failed: %v %v", 403, rr.Code)
		}
	}
}

func TestSimpleFormRequestsCannotUseBrowserSession(t *testing.T) {
	ws, cookie := authenticatedServer(t)
	for _, contentType := range []string{"text/plain", "application/x-www-form-urlencoded", "multipart/form-data; boundary=csrf"} {
		req := httptest.NewRequest(http.MethodPost, "/api/processor/pause", strings.NewReader("csrf"))
		req.Header.Set("Content-Type", contentType)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		ws.router.ServeHTTP(rr, req)
		if rr.Code != 403 {
			t.Fatalf("Equal check failed: %v %v", 403, rr.Code)
		}
	}
}

func TestAuthenticationDisabledByDefault(t *testing.T) {
	ws := NewWebServer()
	for _, path := range []string{"/api/status", "/api/auth/user"} {
		rr := httptest.NewRecorder()
		ws.router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("Equal check failed: %v %v", http.StatusOK, rr.Code)
		}
		if path == "/api/auth/user" {
			if strings.TrimSpace(rr.Body.String()) != `{"enabled":false}` {
				t.Fatalf("JSONEq check failed: %v %v", `{"enabled":false}`, rr.Body.String())
			}
		}
	}
	for _, path := range []string{"/api/auth/register", "/api/auth/login", "/api/auth/logout"} {
		rr := httptest.NewRecorder()
		ws.router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("Equal check failed: %v %v", http.StatusNotFound, rr.Code)
		}
	}
	server := httptest.NewServer(ws.router)
	t.Cleanup(server.Close)
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/ws", nil)
	if response != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
}

func TestInvalidAuthEnvironmentFailsBeforeStartup(t *testing.T) {
	t.Setenv("POSTIE_AUTH_ENABLED", "typo")
	err := rootCmd.RunE(rootCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "POSTIE_AUTH_ENABLED") {
		t.Fatalf("invalid authentication setting must reject startup: %v", err)
	}
}
