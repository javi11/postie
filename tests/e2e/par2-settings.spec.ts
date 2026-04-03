/**
 * E2E tests for PAR2 settings persistence.
 *
 * Verifies that the SkipIfPar2Exists and ParparBinaryPath fields can be saved
 * and reloaded correctly via the API — which is what the Settings UI uses.
 *
 * Regression tests for github.com/javi11/postie/issues/168 (skip PAR2 bug).
 */
import { test, expect } from "@playwright/test";
import { getConfig, patchConfig } from "./helpers";

test.describe("PAR2 settings — API persistence", () => {
  test("skip_if_par2_exists=true is persisted after save", async ({
    request,
  }) => {
    // Set it explicitly to true.
    await patchConfig(request, "par2", { skip_if_par2_exists: true });

    const cfg = await getConfig(request);
    expect(cfg.par2.skip_if_par2_exists).toBe(true);
  });

  test("skip_if_par2_exists=false is persisted after save", async ({
    request,
  }) => {
    await patchConfig(request, "par2", { skip_if_par2_exists: false });

    const cfg = await getConfig(request);
    expect(cfg.par2.skip_if_par2_exists).toBe(false);
  });

  test("parpar_binary_path is persisted after save", async ({ request }) => {
    const testPath = "/usr/local/bin/parpar";
    await patchConfig(request, "par2", { parpar_binary_path: testPath });

    const cfg = await getConfig(request);
    expect(cfg.par2.parpar_binary_path).toBe(testPath);
  });

  test("parpar_binary_path can be cleared", async ({ request }) => {
    // First set it.
    await patchConfig(request, "par2", {
      parpar_binary_path: "/some/path/parpar",
    });
    // Then clear it.
    await patchConfig(request, "par2", { parpar_binary_path: "" });

    const cfg = await getConfig(request);
    expect(cfg.par2.parpar_binary_path ?? "").toBe("");
  });

  test("par2 enabled flag is persisted", async ({ request }) => {
    await patchConfig(request, "par2", { enabled: false });
    let cfg = await getConfig(request);
    expect(cfg.par2.enabled).toBe(false);

    await patchConfig(request, "par2", { enabled: true });
    cfg = await getConfig(request);
    expect(cfg.par2.enabled).toBe(true);
  });
});

/** Navigate to settings and open the File Processing tab where PAR2 lives. */
async function openFileProcessingTab(page: import("@playwright/test").Page) {
  await page.goto("/settings");
  await page.waitForLoadState("networkidle");
  await page.getByRole("tab", { name: "File Processing" }).click();
  // Give the tab panel a moment to render.
  await page.waitForTimeout(200);
}

test.describe("PAR2 settings — UI presence", () => {
  test("settings page shows skip_if_par2_exists toggle", async ({ page }) => {
    await openFileProcessingTab(page);

    // The toggle has id="skip-if-par2-exists" in Par2Section.svelte
    const toggle = page.locator("#skip-if-par2-exists");
    await expect(toggle).toBeVisible();
  });

  test("settings page shows parpar_binary_path input", async ({ page }) => {
    await openFileProcessingTab(page);

    const input = page.locator("#parpar-binary-path");
    await expect(input).toBeVisible();
  });

  test("skip_if_par2_exists toggle reflects saved value (true)", async ({
    request,
    page,
  }) => {
    // Set via API.
    await patchConfig(request, "par2", { skip_if_par2_exists: true });

    // Navigate to settings → File Processing tab.
    await openFileProcessingTab(page);

    const toggle = page.locator("#skip-if-par2-exists");
    await expect(toggle).toBeChecked();
  });

  test("skip_if_par2_exists toggle reflects saved value (false)", async ({
    request,
    page,
  }) => {
    await patchConfig(request, "par2", { skip_if_par2_exists: false });

    await openFileProcessingTab(page);

    const toggle = page.locator("#skip-if-par2-exists");
    await expect(toggle).not.toBeChecked();
  });
});
