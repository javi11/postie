//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Hold the first dashboard's HTTP snapshots across a client-side navigation.
// The socket is already OPEN, so this exercises component lifetime rather than
// delayed WebSocket registration (#281). All other requests use the real server.
const dashboardNavigationFixture = `
window.__dashboard = { counts: {}, releases: [] };
class DashboardSocket {
  static OPEN = 1; static CONNECTING = 0; static CLOSING = 2; static CLOSED = 3;
  readyState = 1;
  constructor() { window.__dashboard.socket = this; }
}
window.WebSocket = DashboardSocket;
const originalFetch = window.fetch.bind(window);
window.fetch = async (url, options) => {
  const path = new URL(url, location.href).pathname;
  if (path !== '/api/running-job-details' && path !== '/api/metrics/nntp-pool') {
    return originalFetch(url, options);
  }
  const fixture = window.__dashboard;
  fixture.counts[path] = (fixture.counts[path] || 0) + 1;
  if (fixture.counts[path] === 1) {
    await new Promise(resolve => fixture.releases.push(resolve));
  }
  const data = path === '/api/running-job-details' ? [] : fixture.metrics(0);
  return new Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } });
};
window.__dashboard.metrics = bytes => ({
  providers: [{ host: 'navigation-provider', name: 'navigation-provider', activeConnections: 1,
    maxConnections: 2, availableSlots: 1, totalErrors: 0, missing: 0, inflight: 10 }],
  uploadSpeed: 0, uploadAvgSpeed: 0, bytesUploaded: bytes, elapsed: '1s', totalErrors: 0
});
window.__dashboard.emit = name => {
  const socket = window.__dashboard.socket;
  socket.onmessage({ data: JSON.stringify({ type: 'running-jobs-updated', data: [
    { id: 'navigation-job', fileName: name, path: '/' + name, size: 1024, progress: [] }
  ] }) });
  socket.onmessage({ data: JSON.stringify({ type: 'nntp-pool-metrics-updated',
    data: window.__dashboard.metrics(name === 'before-release.bin' ? 1048576 : 7340032) }) });
};
`

func dashboardStage(name string, actions ...chromedp.Action) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if err := (chromedp.Tasks(actions)).Do(ctx); err != nil {
			var state string
			_ = chromedp.Evaluate(`JSON.stringify({path: location.pathname, counts: window.__dashboard?.counts, held: window.__dashboard?.releases.length, text: document.body.innerText})`, &state).Do(ctx)
			return fmt.Errorf("%s: %w; browser state=%s", name, err, state)
		}
		return nil
	})
}

func TestDashboard_RemainsLiveAfterLeavingPendingInitialFetch(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(dashboardNavigationFixture).Do(ctx)
			return err
		}),
		// Exercise listener readiness under a slower browser, as on CI.
		emulation.SetCPUThrottlingRate(6),
		chromedp.Navigate(baseURL+"/"),
		dashboardStage("hold initial snapshots", chromedp.Poll(`window.__dashboard.releases.length === 2`, nil)),
		// Use navigation buttons so Svelte remounts panels without reloading JS.
		dashboardStage("click Logs", chromedp.Click(`button[title="Logs"]`, chromedp.ByQuery)),
		dashboardStage("navigate to logs", chromedp.Poll(`location.pathname === '/logs'`, nil)),
		dashboardStage("click Dashboard", chromedp.Click(`button[title="Dashboard"]`, chromedp.ByQuery)),
		dashboardStage("remount dashboard", chromedp.Poll(`window.__dashboard.counts['/api/running-job-details'] === 2 &&
          window.__dashboard.counts['/api/metrics/nntp-pool'] === 2 &&
          document.body.innerText.includes('navigation-provider')`, nil)),
		// Provider rendering does not imply Progress has finished its separate
		// /processor/paused request and registered its listener. Probe until
		// both panels consume events before releasing the abandoned fetches.
		dashboardStage("consume readiness probes before releasing snapshots", chromedp.Poll(`(() => {
          const ready = document.body.innerText.includes('before-release.bin') &&
            document.body.innerText.includes('1 MB');
          if (!ready) window.__dashboard.emit('before-release.bin');
          return ready;
        })()`, nil)),
		// Let the abandoned fetches complete, including their onMount continuations.
		dashboardStage("release abandoned snapshots", chromedp.Evaluate(`Promise.all(window.__dashboard.releases.map(release => release()))
          .then(() => new Promise(resolve => setTimeout(resolve, 200)))`, nil,
			func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) })),
		dashboardStage("emit event after releasing snapshots", chromedp.Evaluate(`window.__dashboard.emit('after-release.bin')`, nil)),
	)
	if err != nil {
		t.Fatalf("exercise dashboard navigation: %v", err)
	}
	var progressLive, providerLive bool
	if err := chromedp.Run(ctx,
		chromedp.Poll(`document.body.innerText.includes('after-release.bin') &&
          document.body.innerText.includes('7 MB')`, nil, chromedp.WithPollingTimeout(2*time.Second)),
	); err != nil {
		_ = chromedp.Run(ctx,
			chromedp.Evaluate(`document.body.innerText.includes('after-release.bin')`, &progressLive),
			chromedp.Evaluate(`document.body.innerText.includes('7 MB')`, &providerLive),
		)
		t.Fatalf("dashboard stopped consuming events after abandoned fetches resolved: progress live=%v, provider live=%v (%v)", progressLive, providerLive, err)
	}
}
