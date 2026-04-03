/**
 * Smoke tests — verify the web server starts correctly and serves the UI.
 * These are the most basic E2E checks for issue #168 regression: if the server
 * crashes or the UI fails to load, all downstream feature tests are meaningless.
 */
import { test, expect } from "@playwright/test";

test.describe("Server smoke tests", () => {
  test("health endpoint responds 200", async ({ request }) => {
    const resp = await request.get("/live");
    expect(resp.status()).toBe(200);
  });

  test("GET /api/config returns a valid config object", async ({ request }) => {
    const resp = await request.get("/api/config");
    expect(resp.status()).toBe(200);
    const cfg = await resp.json();
    // Config must have the main sections.
    expect(cfg).toHaveProperty("par2");
    expect(cfg).toHaveProperty("posting");
  });

  test("main dashboard page loads without console errors", async ({ page }) => {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") errors.push(msg.text());
    });
    page.on("pageerror", (err) => errors.push(err.message));

    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Filter out known benign errors (e.g. websocket not yet connected).
    const fatal = errors.filter(
      (e) =>
        !e.includes("WebSocket") &&
        !e.includes("ws://") &&
        !e.includes("wss://")
    );
    expect(
      fatal,
      `Console errors on main page:\n${fatal.join("\n")}`
    ).toHaveLength(0);
  });

  test("settings page loads without console errors", async ({ page }) => {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") errors.push(msg.text());
    });
    page.on("pageerror", (err) => errors.push(err.message));

    await page.goto("/settings");
    await page.waitForLoadState("networkidle");

    const fatal = errors.filter(
      (e) =>
        !e.includes("WebSocket") &&
        !e.includes("ws://") &&
        !e.includes("wss://")
    );
    expect(
      fatal,
      `Console errors on settings page:\n${fatal.join("\n")}`
    ).toHaveLength(0);
  });
});
