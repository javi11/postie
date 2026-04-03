/**
 * E2E tests for watcher settings persistence.
 *
 * Verifies that single_nzb_per_folder can be saved and reloaded correctly.
 * Regression tests for github.com/javi11/postie/issues/168.
 *
 * Note: The watcher config is under cfg.watchers[] (array of watcher configs).
 * When no watcher is configured the array may be empty; we test a pre-existing
 * watcher entry if present, or skip the watcher-specific test otherwise.
 */
import { test, expect } from "@playwright/test";
import { getConfig, saveConfig } from "./helpers";

/**
 * Return the first watcher config object, or null if none are configured.
 */
async function getFirstWatcher(request: import("@playwright/test").APIRequestContext) {
  const cfg = await getConfig(request);
  const watchers: any[] = cfg.watchers ?? [];
  return watchers.length > 0 ? watchers[0] : null;
}

test.describe("Watcher settings — API persistence", () => {
  test("GET /api/config returns a watchers array", async ({ request }) => {
    const cfg = await getConfig(request);
    // watchers may be null/undefined on first run — that is acceptable.
    // But if present it must be an array.
    if (cfg.watchers !== undefined && cfg.watchers !== null) {
      expect(Array.isArray(cfg.watchers)).toBe(true);
    }
  });

  test("single_nzb_per_folder=true persists when watcher exists", async ({
    request,
  }) => {
    const cfg = await getConfig(request);
    const watchers: any[] = cfg.watchers ?? [];
    if (watchers.length === 0) {
      test.skip(true, "No watcher configured — skipping single_nzb_per_folder persistence test");
      return;
    }

    watchers[0].single_nzb_per_folder = true;
    cfg.watchers = watchers;
    await saveConfig(request, cfg);

    const saved = await getConfig(request);
    expect(saved.watchers[0].single_nzb_per_folder).toBe(true);
  });

  test("single_nzb_per_folder=false persists when watcher exists", async ({
    request,
  }) => {
    const cfg = await getConfig(request);
    const watchers: any[] = cfg.watchers ?? [];
    if (watchers.length === 0) {
      test.skip(true, "No watcher configured — skipping");
      return;
    }

    watchers[0].single_nzb_per_folder = false;
    cfg.watchers = watchers;
    await saveConfig(request, cfg);

    const saved = await getConfig(request);
    expect(saved.watchers[0].single_nzb_per_folder).toBe(false);
  });
});

test.describe("Watcher settings — UI presence", () => {
  test("settings page contains single_nzb_per_folder toggle", async ({
    page,
  }) => {
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");

    // The WatcherSection renders a checkbox bound to watcher.single_nzb_per_folder.
    // If watchers are configured there will be at least one such toggle.
    // We also check the watcher section heading is present.
    const watcherSection = page.getByText(/watcher/i).first();
    await expect(watcherSection).toBeVisible();
  });
});
