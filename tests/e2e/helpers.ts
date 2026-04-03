/**
 * Shared helpers for postie E2E tests.
 */
import { APIRequestContext } from "@playwright/test";

/** Base URL for the API (matches playwright.config.ts baseURL). */
export const BASE_URL = "http://localhost:8080";

/**
 * Fetch the current config from the API.
 */
export async function getConfig(request: APIRequestContext) {
  const resp = await request.get(`${BASE_URL}/api/config`);
  if (!resp.ok()) {
    throw new Error(`GET /api/config failed: ${resp.status()}`);
  }
  return resp.json();
}

/**
 * Save config via the API and return the saved result.
 */
export async function saveConfig(
  request: APIRequestContext,
  configData: object
) {
  const resp = await request.post(`${BASE_URL}/api/config`, {
    data: configData,
    headers: { "Content-Type": "application/json" },
  });
  if (!resp.ok()) {
    const body = await resp.text();
    throw new Error(`POST /api/config failed (${resp.status()}): ${body}`);
  }
}

/**
 * Patch a specific top-level section of the config.
 * Fetches current config, merges the patch, then saves.
 */
export async function patchConfig(
  request: APIRequestContext,
  section: string,
  patch: object
) {
  const current = await getConfig(request);
  current[section] = { ...(current[section] ?? {}), ...patch };
  await saveConfig(request, current);
}
