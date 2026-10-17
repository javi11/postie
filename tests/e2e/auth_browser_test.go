//go:build e2e

package e2e_test

import (
	"net"
	"strings"
	"testing"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// A separate fresh server proves the actual browser's first-admin bootstrap,
// login form and logout behavior without altering the posting suite's session.
func TestBrowserAuthenticationLifecycle(t *testing.T) {
	requireChrome(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	server, err := startWebServerAt(t.TempDir(), port)
	if err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	t.Cleanup(func() {
		if server.Process != nil {
			_ = server.Process.Kill()
			_ = server.Wait()
		}
	})
	url := "http://127.0.0.1:" + port
	if err := waitForServer(url); err != nil {
		t.Fatalf("NoError check failed: %v", err)
	}
	ctx, cancel := newChromedpCtx(t)
	defer cancel()
	var location string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url+"/settings"),
		chromedp.WaitVisible("#username", chromedp.ByQuery),
		chromedp.Location(&location),
	); err != nil {
		t.Fatal(err)
	}
	if !(strings.HasSuffix(location, "/login")) {
		t.Fatalf("True check failed: %v", strings.HasSuffix(location, "/login"))
	}
	if err := chromedp.Run(ctx,
		chromedp.SendKeys("#username", "browser-admin", chromedp.ByQuery),
		chromedp.SendKeys("#password", "browser-long-secret-password", chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`[data-testid="sign-out"]`, chromedp.ByQuery),
		chromedp.Poll(`window.location.pathname !== '/login'`, nil),
	); err != nil {
		t.Fatal(err)
	}
	var status int
	await := func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }
	if err := chromedp.Run(ctx, chromedp.Evaluate(`fetch('/api/status').then(r=>r.status)`, &status, await)); err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, status)
	}
	// The setup wizard exposes logout, even before NNTP configuration exists.
	if err := chromedp.Run(ctx,
		chromedp.Click(`[data-testid="sign-out"]`, chromedp.ByQuery),
		chromedp.WaitVisible("#username", chromedp.ByQuery),
		chromedp.Poll(`window.location.pathname === '/login'`, nil),
		chromedp.Evaluate(`fetch('/api/status').then(r=>r.status)`, &status, await),
	); err != nil {
		t.Fatal(err)
	}
	if status != 401 {
		t.Fatalf("Equal check failed: %v %v", 401, status)
	}
	if err := chromedp.Run(ctx,
		chromedp.SendKeys("#username", "browser-admin", chromedp.ByQuery),
		chromedp.SendKeys("#password", "browser-long-secret-password", chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`[data-testid="sign-out"]`, chromedp.ByQuery),
		chromedp.Poll(`window.location.pathname !== '/login'`, nil),
		chromedp.Evaluate(`fetch('/api/status').then(r=>r.status)`, &status, await),
	); err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Fatalf("Equal check failed: %v %v", 200, status)
	}
}

func TestBrowserAuthenticationDisabled(t *testing.T) {
	requireChrome(t)
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
	ctx, cancel := newChromedpCtx(t)
	defer cancel()
	var signOut bool
	if err := chromedp.Run(ctx,
		chromedp.Navigate(address+"/"),
		chromedp.Poll(`window.location.pathname === '/setup' && document.querySelector('h1') !== null`, nil),
		chromedp.Evaluate(`document.querySelector('[data-testid="sign-out"]') !== null`, &signOut),
	); err != nil {
		t.Fatal(err)
	}
	if signOut {
		t.Fatal("disabled authentication shows sign out")
	}
}
