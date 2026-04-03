//go:build e2e

// Tests MUST NOT call t.Parallel() — they share one server instance.
package e2e_test

import (
	"testing"

	"github.com/chromedp/chromedp"
)

// ── API tests (net/http only) ────────────────────────────────────────────────

func TestPar2API_SkipIfPar2ExistsTruePersists(t *testing.T) {
	patchConfig(t, "par2", map[string]any{"skip_if_par2_exists": true})

	cfg := getConfig(t)
	par2, _ := cfg["par2"].(map[string]any)
	if par2["skip_if_par2_exists"] != true {
		t.Errorf("expected skip_if_par2_exists=true, got %v", par2["skip_if_par2_exists"])
	}
}

func TestPar2API_SkipIfPar2ExistsFalsePersists(t *testing.T) {
	patchConfig(t, "par2", map[string]any{"skip_if_par2_exists": false})

	cfg := getConfig(t)
	par2, _ := cfg["par2"].(map[string]any)
	if par2["skip_if_par2_exists"] != false {
		t.Errorf("expected skip_if_par2_exists=false, got %v", par2["skip_if_par2_exists"])
	}
}

func TestPar2API_ParparBinaryPathPersists(t *testing.T) {
	const testPath = "/usr/local/bin/parpar"
	patchConfig(t, "par2", map[string]any{"parpar_binary_path": testPath})

	cfg := getConfig(t)
	par2, _ := cfg["par2"].(map[string]any)
	if par2["parpar_binary_path"] != testPath {
		t.Errorf("expected parpar_binary_path=%q, got %v", testPath, par2["parpar_binary_path"])
	}
}

func TestPar2API_ParparBinaryPathCanBeCleared(t *testing.T) {
	patchConfig(t, "par2", map[string]any{"parpar_binary_path": "/some/path/parpar"})
	patchConfig(t, "par2", map[string]any{"parpar_binary_path": ""})

	cfg := getConfig(t)
	par2, _ := cfg["par2"].(map[string]any)
	got, _ := par2["parpar_binary_path"].(string)
	if got != "" {
		t.Errorf("expected parpar_binary_path=\"\", got %q", got)
	}
}

func TestPar2API_EnabledFlagPersists(t *testing.T) {
	patchConfig(t, "par2", map[string]any{"enabled": false})
	cfg := getConfig(t)
	par2, _ := cfg["par2"].(map[string]any)
	if par2["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", par2["enabled"])
	}

	patchConfig(t, "par2", map[string]any{"enabled": true})
	cfg = getConfig(t)
	par2, _ = cfg["par2"].(map[string]any)
	if par2["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", par2["enabled"])
	}
}

// ── UI tests (chromedp) ──────────────────────────────────────────────────────

func TestPar2UI_SkipToggleVisible(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	err := chromedp.Run(ctx,
		openSettingsTab(ctx, "File Processing"),
		chromedp.WaitVisible("#skip-if-par2-exists", chromedp.ByID),
	)
	if err != nil {
		t.Fatalf("skip-if-par2-exists not visible: %v", err)
	}
}

func TestPar2UI_ParparPathInputVisible(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	err := chromedp.Run(ctx,
		openSettingsTab(ctx, "File Processing"),
		chromedp.WaitVisible("#parpar-binary-path", chromedp.ByID),
	)
	if err != nil {
		t.Fatalf("parpar-binary-path not visible: %v", err)
	}
}

func TestPar2UI_ToggleReflectsTrueValue(t *testing.T) {
	patchConfig(t, "par2", map[string]any{"skip_if_par2_exists": true})

	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	var checked bool
	err := chromedp.Run(ctx,
		openSettingsTab(ctx, "File Processing"),
		chromedp.WaitVisible("#skip-if-par2-exists", chromedp.ByID),
		chromedp.Evaluate(`document.getElementById('skip-if-par2-exists').checked`, &checked),
	)
	if err != nil {
		t.Fatalf("chromedp run: %v", err)
	}
	if !checked {
		t.Error("expected #skip-if-par2-exists to be checked (skip_if_par2_exists=true)")
	}
}

func TestPar2UI_ToggleReflectsFalseValue(t *testing.T) {
	patchConfig(t, "par2", map[string]any{"skip_if_par2_exists": false})

	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	var checked bool
	err := chromedp.Run(ctx,
		openSettingsTab(ctx, "File Processing"),
		chromedp.WaitVisible("#skip-if-par2-exists", chromedp.ByID),
		chromedp.Evaluate(`document.getElementById('skip-if-par2-exists').checked`, &checked),
	)
	if err != nil {
		t.Fatalf("chromedp run: %v", err)
	}
	if checked {
		t.Error("expected #skip-if-par2-exists to be unchecked (skip_if_par2_exists=false)")
	}
}
