//go:build e2e

package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Both panels finish their initial snapshots normally. Then events stop while
// the browser still reports OPEN, as with a silently blackholed connection.
// Advancing Date.now ages the transport without slowing the browser's timers.
func TestDashboard_RefreshesAfterOpenSocketFallsSilent(t *testing.T) {
	ctx, cancel := newChromedpCtx(t)
	defer cancel()

	const fixture = `
window.__silence = { offset: 0, counts: {} };
const realNow = Date.now.bind(Date);
Date.now = () => realNow() + window.__silence.offset;
class SilentSocket {
  static OPEN = 1; static CONNECTING = 0; static CLOSING = 2; static CLOSED = 3;
  readyState = 1;
  constructor() { window.__silence.socket = this; }
}
window.WebSocket = SilentSocket;
const originalFetch = window.fetch.bind(window);
window.fetch = async (url, options) => {
  const path = new URL(url, location.href).pathname;
  if (path !== '/api/running-job-details' && path !== '/api/metrics/nntp-pool') {
    return originalFetch(url, options);
  }
  const state = window.__silence;
  state.counts[path] = (state.counts[path] || 0) + 1;
  const initial = state.counts[path] === 1;
  const data = path === '/api/running-job-details'
    ? (initial ? [{ id: 'silent-job', fileName: 'silent-upload.bin', path: '/silent-upload.bin', size: 1024, progress: [] }] : [])
    : { providers: [{ host: 'silent-provider', activeConnections: 1, maxConnections: 2,
        availableSlots: 1, totalErrors: 0, missing: 0, inflight: 0 }],
        uploadSpeed: initial ? 1048576 : 7340032, uploadAvgSpeed: 1048576,
        bytesUploaded: initial ? 1048576 : 7340032, elapsed: '1s', totalErrors: 0 };
  return new Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } });
};`

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(fixture).Do(ctx)
			return err
		}),
		chromedp.Navigate(baseURL+"/"),
		chromedp.Poll(`document.body.innerText.includes('silent-upload.bin') &&
          document.body.innerText.includes('silent-provider') && document.body.innerText.includes('1 MB/s')`, nil),
		// No completion event, pending initial fetch, socket close or reconnect.
		chromedp.Evaluate(`window.__silence.offset = 20000`, nil),
		chromedp.Poll(`!document.body.innerText.includes('silent-upload.bin') &&
          document.body.innerText.includes('No upload in progress') &&
          document.body.innerText.includes('7 MB/s') &&
          window.__silence.counts['/api/running-job-details'] > 1 &&
          window.__silence.counts['/api/metrics/nntp-pool'] > 1 &&
          window.__silence.socket.readyState === WebSocket.OPEN`, nil,
			chromedp.WithPollingTimeout(8*time.Second)),
	)
	if err != nil {
		var state string
		_ = chromedp.Run(ctx, chromedp.Evaluate(`JSON.stringify({ counts: window.__silence?.counts,
          socketState: window.__silence?.socket.readyState, text: document.body.innerText })`, &state))
		t.Fatalf("dashboard did not refresh after its OPEN socket fell silent: %v; browser state=%s", err, state)
	}
}
