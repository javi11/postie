//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuth_APIKeyIntegrationsWithoutBrowserSession(t *testing.T) {
	previous := getConfig(t)
	t.Cleanup(func() { saveConfig(t, previous) })
	patchConfig(t, "api", map[string]any{"enabled": true})

	resp, err := http.Get(baseURL + "/api/api-key")
	if err != nil {
		t.Fatal(err)
	}
	var key struct {
		Key string `json:"key"`
	}
	err = json.NewDecoder(resp.Body).Decode(&key)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || err != nil || key.Key == "" {
		t.Fatalf("authenticated key retrieval: status=%d error=%v", resp.StatusCode, err)
	}

	// This client intentionally has no cookie jar: integrations authenticate
	// using their API key even though the dashboard requires a login session.
	anonymous := &http.Client{Timeout: 15 * time.Second}
	request := func(method, path, apiKey string, payload []byte, expected int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
		}
		resp, err := anonymous.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != expected {
			t.Fatalf("%s %s: status=%d want=%d error=%v body=%s", method, path, resp.StatusCode, expected, err, body)
		}
		return body
	}

	request(http.MethodGet, "/api/api-key", key.Key, nil, http.StatusUnauthorized)
	request(http.MethodPost, "/api/v1/queue/upload", "", []byte(`{}`), http.StatusUnauthorized)
	request(http.MethodPost, "/api/v1/queue/upload", "incorrect", []byte(`{}`), http.StatusUnauthorized)

	root := t.TempDir()
	file := filepath.Join(root, "integration.bin")
	if err := os.WriteFile(file, bytes.Repeat([]byte("test payload"), 1024), 0600); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"file": file, "relative_path": root})
	if err != nil {
		t.Fatal(err)
	}
	body := request(http.MethodPost, "/api/v1/queue/upload", key.Key, payload, http.StatusAccepted)
	var result struct{ Status, File string }
	if err := json.Unmarshal(body, &result); err != nil || result.Status != "queued" || result.File != file {
		t.Fatalf("API-key upload response: %s error=%v", body, err)
	}
	// Remove the test job while its source still exists, so this shared server
	// has no pending work left behind for subsequent browser journeys.
	queueResp, err := http.Get(baseURL + "/api/queue?limit=100")
	if err != nil {
		t.Fatal(err)
	}
	var queue struct {
		Items []struct {
			ID       string `json:"id"`
			FilePath string `json:"path"`
		} `json:"items"`
	}
	err = json.NewDecoder(queueResp.Body).Decode(&queue)
	queueResp.Body.Close()
	if err != nil || queueResp.StatusCode != http.StatusOK {
		t.Fatalf("read queued job: status=%d error=%v", queueResp.StatusCode, err)
	}
	found := false
	for _, item := range queue.Items {
		if item.FilePath != file {
			continue
		}
		found = true
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/api/queue/"+url.PathEscape(item.ID), nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("remove test job: status=%d", resp.StatusCode)
		}
	}
	if !found {
		t.Fatal("accepted API-key upload did not appear in the queue")
	}

	ping := []byte(`{"eventType":"Test"}`)
	request(http.MethodPost, "/api/arr/webhook/test", "", ping, http.StatusUnauthorized)
	request(http.MethodPost, "/api/arr/webhook/test?apiKey=incorrect", "", ping, http.StatusUnauthorized)
	request(http.MethodPost, "/api/arr/webhook/test?apiKey="+url.QueryEscape(key.Key), "", ping, http.StatusOK)
}

// Exercise default-mode compatibility over TCP against the actual executable.
func TestAuth_DisabledServerRemainsAccessible(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = listener.Close()
	server, err := startWebServerMode(t.TempDir(), port, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Process.Kill(); _ = server.Wait() })
	address := "http://127.0.0.1:" + port
	if err := waitForServer(address); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, path := range []string{"/", "/setup", "/api/status", "/api/auth/user"} {
		response, err := client.Get(address + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("disabled auth GET %s: status=%d", path, response.StatusCode)
		}
	}
	socket, response, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:"+port+"/api/ws", nil)
	if response != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		t.Fatalf("disabled auth websocket: %v", err)
	}
	_ = socket.Close()
}
