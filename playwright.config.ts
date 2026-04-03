import { defineConfig, devices } from "@playwright/test";
import * as os from "os";
import * as path from "path";
import * as fs from "fs";

// Isolated HOME so the web server writes its config to a temp directory,
// leaving the developer's real config untouched.
const testHome = fs.mkdtempSync(path.join(os.tmpdir(), "postie-e2e-"));

export default defineConfig({
  testDir: "./tests/e2e",
  globalSetup: "./tests/e2e/global-setup.ts",
  fullyParallel: false, // sequential — one server instance
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [
    ["list"],
    ["html", { outputFolder: "playwright-report", open: "never" }],
  ],
  use: {
    baseURL: "http://localhost:8080",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    // The binary must be pre-built: go build -o ./bin/postie-web ./cmd/web
    command: "./bin/postie-web --port 8080",
    url: "http://localhost:8080/live",
    reuseExistingServer: false,
    timeout: 20_000,
    env: {
      HOME: testHome,
      // On Linux CI use XDG_CONFIG_HOME for isolation.
      XDG_CONFIG_HOME: path.join(testHome, ".config"),
      XDG_DATA_HOME: path.join(testHome, ".local", "share"),
      // Disable any auto-start behaviour that could interfere.
      PORT: "8080",
    },
    stdout: "pipe",
    stderr: "pipe",
  },
});
