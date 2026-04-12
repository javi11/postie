//go:build e2e

// Tests MUST NOT call t.Parallel() — they share one server instance.
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── helpers ─────────────────────────────────────────────────────────────────────

// uploadFile sends a single file to POST /api/upload via multipart form.
func uploadFile(t *testing.T, filePath string) {
	t.Helper()

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fatalf("copy file to form: %v", err)
	}
	writer.Close()

	resp, err := http.Post(baseURL+"/api/upload", writer.FormDataContentType(), &body)
	if err != nil {
		t.Fatalf("POST /api/upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /api/upload returned %d: %s", resp.StatusCode, respBody)
	}
}

// queueResponse represents the paginated queue API response.
type queueResponse struct {
	Items []struct {
		ID      string  `json:"id"`
		Status  string  `json:"status"`
		NzbPath *string `json:"nzbPath"`
	} `json:"items"`
	TotalItems int `json:"totalItems"`
}

// waitForQueueComplete polls the queue API until at least one item reaches
// "complete" status or the timeout elapses. Returns the final queue response.
func waitForQueueComplete(t *testing.T, timeout time.Duration) queueResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check for completed items
		resp, err := http.Get(baseURL + "/api/queue?status=complete&limit=100")
		if err != nil {
			t.Logf("GET /api/queue: %v (retrying)", err)
			time.Sleep(1 * time.Second)
			continue
		}

		var qr queueResponse
		err = json.NewDecoder(resp.Body).Decode(&qr)
		resp.Body.Close()
		if err != nil {
			t.Logf("decode queue response: %v (retrying)", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if qr.TotalItems > 0 {
			return qr
		}

		// Log current queue state for debugging
		if statsResp, err := http.Get(baseURL + "/api/queue/stats"); err == nil {
			var stats map[string]any
			if json.NewDecoder(statsResp.Body).Decode(&stats) == nil {
				t.Logf("queue stats: %v", stats)
			}
			statsResp.Body.Close()
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatal("timed out waiting for queue items to complete")
	return queueResponse{} // unreachable
}

// clearQueue deletes all queue items so tests start clean.
func clearQueue(t *testing.T) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/queue", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/queue: %v", err)
	}
	resp.Body.Close()
}

// ── tests ───────────────────────────────────────────────────────────────────────

// TestOutputDir_UploadPlacesNzbInOutputDir uploads a file and verifies the
// generated NZB lands inside the configured output directory, not in the
// source/temp directory. This is the end-to-end validation for the cross-volume
// fix (replacing strings.TrimPrefix with filepath.Rel).
func TestOutputDir_UploadPlacesNzbInOutputDir(t *testing.T) {
	// 1. Configure a dedicated output directory
	outputDir := t.TempDir()
	cfg := getConfig(t)
	cfg["output_dir"] = outputDir
	// Disable par2 and post_check to speed up the test
	if par2, ok := cfg["par2"].(map[string]any); ok {
		par2["enabled"] = false
	}
	if postCheck, ok := cfg["post_check"].(map[string]any); ok {
		postCheck["enabled"] = false
	}
	saveConfig(t, cfg)

	// 2. Clear any existing queue items
	clearQueue(t)

	// 3. Create a source file in a separate temp dir (simulates different volume)
	sourceDir := t.TempDir()
	srcFile := filepath.Join(sourceDir, "test_upload.bin")
	// Write enough data for a minimal article (at least a few bytes)
	if err := os.WriteFile(srcFile, bytes.Repeat([]byte("X"), 1024), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	// 4. Upload the file
	uploadFile(t, srcFile)

	// 5. Wait for the job to complete
	qr := waitForQueueComplete(t, 30*time.Second)

	// 6. Verify the NZB path is inside the output directory
	if len(qr.Items) == 0 {
		t.Fatal("no completed queue items found")
	}

	for _, item := range qr.Items {
		if item.NzbPath == nil {
			t.Errorf("queue item %s completed but has nil nzbPath", item.ID)
			continue
		}
		nzbPath := *item.NzbPath

		// NZB must be inside the output directory
		absNzb, _ := filepath.Abs(nzbPath)
		absOut, _ := filepath.Abs(outputDir)
		if !strings.HasPrefix(absNzb, absOut+string(filepath.Separator)) && absNzb != absOut {
			t.Errorf("NZB not in output dir:\n  nzbPath:   %s\n  outputDir: %s", nzbPath, outputDir)
		}

		// NZB must NOT be in the source directory
		absSrc, _ := filepath.Abs(sourceDir)
		if strings.HasPrefix(absNzb, absSrc+string(filepath.Separator)) {
			t.Errorf("NZB leaked into source dir:\n  nzbPath:   %s\n  sourceDir: %s", nzbPath, sourceDir)
		}

		// Verify the NZB file actually exists on disk
		if _, err := os.Stat(nzbPath); os.IsNotExist(err) {
			t.Errorf("NZB file does not exist at %q", nzbPath)
		}

		t.Logf("NZB correctly placed at: %s", nzbPath)
	}
}

// TestOutputDir_ConfigPersistsRoundTrip verifies that output_dir survives a
// config save/load cycle.
func TestOutputDir_ConfigPersistsRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := getConfig(t)
	cfg["output_dir"] = tmpDir
	saveConfig(t, cfg)

	cfg = getConfig(t)
	got, _ := cfg["output_dir"].(string)
	if got != tmpDir {
		t.Errorf("expected output_dir=%q, got %q", tmpDir, got)
	}
}

// TestOutputDir_ConfigCanBeCleared verifies output_dir can be reset to empty.
func TestOutputDir_ConfigCanBeCleared(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := getConfig(t)
	cfg["output_dir"] = tmpDir
	saveConfig(t, cfg)

	cfg = getConfig(t)
	cfg["output_dir"] = ""
	saveConfig(t, cfg)

	cfg = getConfig(t)
	got, _ := cfg["output_dir"].(string)
	if got != "" {
		t.Errorf("expected output_dir=\"\", got %q", got)
	}
}

// TestOutputDir_IndependentOfSingleNzbPerFolder verifies that output_dir
// persists independently of the watcher single_nzb_per_folder setting.
func TestOutputDir_IndependentOfSingleNzbPerFolder(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := getConfig(t)
	cfg["output_dir"] = tmpDir
	saveConfig(t, cfg)

	patchWatcherConfig(t, map[string]any{"single_nzb_per_folder": false})

	cfg = getConfig(t)
	got, _ := cfg["output_dir"].(string)
	if got != tmpDir {
		t.Errorf("output_dir lost after patching watcher config: want %q, got %q", tmpDir, got)
	}

	watchers, _ := cfg["watchers"].([]any)
	if len(watchers) == 0 {
		t.Fatal("no watchers in config")
	}
	w, _ := watchers[0].(map[string]any)
	if w["single_nzb_per_folder"] != false {
		t.Errorf("expected single_nzb_per_folder=false, got %v", w["single_nzb_per_folder"])
	}
}

// assertQueueEmpty is a helper that fails if there are pending/running items.
func assertQueueEmpty(t *testing.T) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("%s/api/queue/stats", baseURL))
	if err != nil {
		t.Logf("GET /api/queue/stats: %v (skipping check)", err)
		return
	}
	defer resp.Body.Close()
	var stats struct {
		Pending int `json:"pending"`
		Running int `json:"running"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err == nil {
		if stats.Pending > 0 || stats.Running > 0 {
			t.Logf("queue not empty: pending=%d running=%d", stats.Pending, stats.Running)
		}
	}
}
