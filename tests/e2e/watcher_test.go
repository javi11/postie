//go:build e2e

// Tests MUST NOT call t.Parallel() — they share one server instance.
package e2e_test

import (
	"testing"

	"github.com/chromedp/chromedp"
)

// ── API tests (net/http only) ────────────────────────────────────────────────

func TestWatcherAPI_ConfigReturnsWatchersArray(t *testing.T) {
	cfg := getConfig(t)
	watchers, ok := cfg["watchers"].([]any)
	if !ok {
		t.Fatalf("config.watchers is not an array, got %T", cfg["watchers"])
	}
	if len(watchers) == 0 {
		t.Error("expected at least one watcher in config.watchers")
	}
}

func TestWatcherAPI_SingleNzbPerFolderTruePersists(t *testing.T) {
	patchWatcherConfig(t, map[string]any{"single_nzb_per_folder": true})

	cfg := getConfig(t)
	watchers, _ := cfg["watchers"].([]any)
	if len(watchers) == 0 {
		t.Fatal("no watchers in config")
	}
	w, _ := watchers[0].(map[string]any)
	if w["single_nzb_per_folder"] != true {
		t.Errorf("expected single_nzb_per_folder=true, got %v", w["single_nzb_per_folder"])
	}
}

func TestWatcherAPI_SingleNzbPerFolderFalsePersists(t *testing.T) {
	patchWatcherConfig(t, map[string]any{"single_nzb_per_folder": false})

	cfg := getConfig(t)
	watchers, _ := cfg["watchers"].([]any)
	if len(watchers) == 0 {
		t.Fatal("no watchers in config")
	}
	w, _ := watchers[0].(map[string]any)
	if w["single_nzb_per_folder"] != false {
		t.Errorf("expected single_nzb_per_folder=false, got %v", w["single_nzb_per_folder"])
	}
}

// ── UI tests (chromedp) ──────────────────────────────────────────────────────

func TestWatcherUI_ToggleVisible(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	err := chromedp.Run(ctx,
		openSettingsTab(ctx, "Automation"),
		chromedp.WaitVisible("#single-nzb-per-folder-0", chromedp.ByID),
	)
	if err != nil {
		t.Fatalf("single-nzb-per-folder toggle not visible: %v", err)
	}
}
