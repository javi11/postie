//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/chromedp/chromedp"
)

func TestSmoke_HealthEndpoint(t *testing.T) {
	resp, err := http.Get(baseURL + "/live")
	if err != nil {
		t.Fatalf("GET /live: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSmoke_ConfigHasPar2AndPosting(t *testing.T) {
	cfg := getConfig(t)
	if _, ok := cfg["par2"]; !ok {
		t.Error("config missing 'par2' section")
	}
	if _, ok := cfg["posting"]; !ok {
		t.Error("config missing 'posting' section")
	}
}

// collectConsoleErrors is the JS snippet that intercepts console.error
// and stores messages (excluding benign WebSocket noise) in window.__e2eErrors.
const collectConsoleErrors = `
	window.__e2eErrors = [];
	const _orig = console.error;
	console.error = (...args) => {
		const msg = args.join(' ');
		if (!msg.includes('WebSocket') && !msg.includes('ws://')) {
			window.__e2eErrors.push(msg);
		}
		_orig.apply(console, args);
	};
`

func TestSmoke_DashboardLoadsWithoutErrors(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	var errors []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/"),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(collectConsoleErrors, nil),
		chromedp.Sleep(500),
		chromedp.Evaluate(`window.__e2eErrors`, &errors),
	)
	if err != nil {
		t.Fatalf("chromedp run: %v", err)
	}
	for _, e := range errors {
		t.Errorf("console error on dashboard: %s", e)
	}
}

func TestSmoke_SettingsLoadsWithoutErrors(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	var errors []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/settings"),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(collectConsoleErrors, nil),
		chromedp.Sleep(500),
		chromedp.Evaluate(`window.__e2eErrors`, &errors),
	)
	if err != nil {
		t.Fatalf("chromedp run: %v", err)
	}
	for _, e := range errors {
		t.Errorf("console error on settings: %s", e)
	}
}
