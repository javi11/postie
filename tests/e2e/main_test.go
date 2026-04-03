//go:build e2e

// Package e2e_test contains end-to-end tests that run against a real web server
// binary. Build and run with:
//
//	go build -o ./bin/postie-web ./cmd/web
//	go test -v -tags e2e -timeout 60s ./tests/e2e/...
//
// Tests MUST NOT call t.Parallel() — they share one server instance.
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var (
	baseURL  = "http://127.0.0.1:8080"
	fakeNntp *fakeNntpServer
)

// TestMain is the entry point for the E2E test suite.
// It uses a run() helper so that defer statements execute before os.Exit —
// os.Exit() bypasses defer, which would leave the web server process running.
func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	// 1. Isolated home directory (mirrors playwright.config.ts testHome)
	testHome, err := os.MkdirTemp("", "postie-e2e-*")
	if err != nil {
		log.Printf("create temp home: %v", err)
		return 1
	}
	defer os.RemoveAll(testHome)

	// 2. Start fake NNTP server (must outlive all tests — every POST /api/config
	// triggers validateServerConnections which dials the NNTP server)
	fakeNntp, err = startFakeNntpServer()
	if err != nil {
		log.Printf("start fake NNTP server: %v", err)
		return 1
	}
	defer fakeNntp.Close()

	// 3. Start the web server binary as a subprocess
	webCmd, err := startWebServer(testHome)
	if err != nil {
		log.Printf("start web server: %v", err)
		return 1
	}
	defer func() {
		if webCmd.Process != nil {
			webCmd.Process.Kill()
			webCmd.Wait() //nolint:errcheck
		}
	}()

	// 4. Wait for the server to be ready
	if err := waitForServer(baseURL); err != nil {
		log.Printf("web server did not become ready: %v", err)
		return 1
	}

	// 5. Complete setup wizard (skips if already configured)
	if err := completeSetupWizard(fakeNntp.port); err != nil {
		log.Printf("complete setup wizard: %v", err)
		return 1
	}

	return m.Run()
}

// startWebServer launches bin/postie-web with an isolated HOME directory.
func startWebServer(testHome string) (*exec.Cmd, error) {
	// Find the binary relative to the project root (two dirs up from tests/e2e/)
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	binaryPath := filepath.Join(projectRoot, "bin", "postie-web")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found at %s — run: go build -o ./bin/postie-web ./cmd/web", binaryPath)
	}

	env := append(os.Environ(),
		"HOME="+testHome,
		"XDG_CONFIG_HOME="+filepath.Join(testHome, ".config"),
		"XDG_DATA_HOME="+filepath.Join(testHome, ".local", "share"),
		"PORT=8080",
	)

	cmd := exec.Command(binaryPath, "--port", "8080")
	cmd.Env = env
	cmd.Stdout = os.Stderr // pipe server logs to stderr, not stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start web server: %w", err)
	}
	return cmd, nil
}

// waitForServer polls GET /live until it returns 200 or the timeout elapses.
func waitForServer(url string) error {
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/live")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("server did not respond within 15s")
}

// completeSetupWizard posts to /api/setup/complete if the server is in first-start mode.
func completeSetupWizard(nntpPort int) error {
	// Check current status — skip if already configured
	resp, err := http.Get(baseURL + "/api/status")
	if err == nil && resp.StatusCode == http.StatusOK {
		var status struct {
			IsFirstStart bool `json:"isFirstStart"`
			ConfigValid  bool `json:"configValid"`
		}
		if err2 := json.NewDecoder(resp.Body).Decode(&status); err2 == nil {
			resp.Body.Close()
			if !status.IsFirstStart && status.ConfigValid {
				return nil // already configured
			}
		} else {
			resp.Body.Close()
		}
	}

	payload := map[string]any{
		"servers": []map[string]any{
			{
				"host":           "127.0.0.1",
				"port":           nntpPort,
				"username":       "",
				"password":       "",
				"ssl":            false,
				"maxConnections": 5,
				"role":           "upload",
			},
		},
		"outputDirectory": "/tmp",
		"watchDirectory":  "",
	}
	body, _ := json.Marshal(payload)

	postResp, err := http.Post(
		baseURL+"/api/setup/complete",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("POST /api/setup/complete: %w", err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(postResp.Body)
		return fmt.Errorf("POST /api/setup/complete returned %d: %s", postResp.StatusCode, errBody)
	}
	return nil
}

// mustGetRawConfig fetches and decodes the full config from the server.
func mustGetRawConfig(t *testing.T) map[string]any {
	t.Helper()
	resp, err := http.Get(baseURL + "/api/config")
	if err != nil {
		t.Fatalf("GET /api/config: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config returned %d", resp.StatusCode)
	}
	var cfg map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode /api/config: %v", err)
	}
	return cfg
}
