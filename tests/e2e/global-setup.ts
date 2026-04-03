/**
 * Playwright global setup — runs once before any tests start.
 *
 * Problem: the web server always validates NNTP server connectivity before
 * saving any config (both setup-wizard and POST /api/config paths). This
 * means real E2E tests cannot save config with a fake host.
 *
 * Solution: start a minimal fake NNTP TCP server on localhost that sends a
 * valid `200` welcome banner. The validation only needs a successful TCP
 * connect + NNTP greeting, not actual posting capability. The fake server
 * keeps running throughout the entire test session so that every patchConfig
 * call also passes validation.
 *
 * Teardown: the returned function is called by Playwright after all tests
 * complete and shuts down the fake NNTP server.
 */
import * as net from "net";
import { request } from "@playwright/test";

const BASE_URL = "http://localhost:8080";

/** Start a minimal fake NNTP server on a random loopback port. */
function startFakeNntpServer(): Promise<{ port: number; close: () => void }> {
  return new Promise((resolve, reject) => {
    const server = net.createServer((socket) => {
      // RFC 3977 §5.1: the server sends a greeting on connect.
      socket.write("200 Postie-test NNTP server ready\r\n");

      socket.on("data", (buf) => {
        const lines = buf.toString().split("\r\n");
        for (const line of lines) {
          const upper = line.trim().toUpperCase();
          if (!upper) continue;

          if (upper.startsWith("AUTHINFO USER")) {
            socket.write("381 Enter password\r\n");
          } else if (upper.startsWith("AUTHINFO PASS")) {
            socket.write("281 Authentication accepted\r\n");
          } else if (upper === "CAPABILITIES") {
            socket.write(
              "101 Capability list:\r\nVERSION 2\r\nREADER\r\nPOST\r\nDATE\r\n.\r\n"
            );
          } else if (upper === "DATE") {
            // RFC 3977 §7.1 — used by nntppool as a connectivity ping
            const now = new Date();
            const ts =
              String(now.getUTCFullYear()) +
              String(now.getUTCMonth() + 1).padStart(2, "0") +
              String(now.getUTCDate()).padStart(2, "0") +
              String(now.getUTCHours()).padStart(2, "0") +
              String(now.getUTCMinutes()).padStart(2, "0") +
              String(now.getUTCSeconds()).padStart(2, "0");
            socket.write(`111 ${ts}\r\n`);
          } else if (upper === "QUIT") {
            socket.write("205 closing connection\r\n");
            socket.end();
          } else {
            socket.write("500 Unknown command\r\n");
          }
        }
      });

      socket.on("error", () => {
        // Ignore socket errors — the test validation closes the connection
        // abruptly once it has received the greeting.
      });
    });

    server.listen(0, "127.0.0.1", () => {
      const addr = server.address() as net.AddressInfo;
      resolve({
        port: addr.port,
        close: () =>
          server.close(() => {
            /* ignore */
          }),
      });
    });

    server.on("error", reject);
  });
}

async function waitForServer(timeoutMs = 15_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const ctx = await request.newContext({ baseURL: BASE_URL });
      const resp = await ctx.get("/live");
      await ctx.dispose();
      if (resp.status() === 200) return;
    } catch {
      // server not ready yet
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error("Web server did not become ready within timeout");
}

export default async function globalSetup(): Promise<() => Promise<void>> {
  // The webServer option already polls /live, but wait here too for safety.
  await waitForServer();

  // Start the fake NNTP server before calling the setup API so the
  // connectivity check inside the server has something to connect to.
  const fakeNntp = await startFakeNntpServer();

  const ctx = await request.newContext({ baseURL: BASE_URL });

  try {
    // Check if the server is already configured (non-first-start).
    const statusResp = await ctx.get(`${BASE_URL}/api/status`);
    if (statusResp.ok()) {
      const status = await statusResp.json();
      if (!status.isFirstStart && status.configValid) {
        // Already configured — nothing to do, but keep the server running.
        return async () => fakeNntp.close();
      }
    }

    // Complete the setup wizard pointing at the loopback fake NNTP server.
    const resp = await ctx.post(`${BASE_URL}/api/setup/complete`, {
      data: {
        servers: [
          {
            host: "127.0.0.1",
            port: fakeNntp.port,
            username: "",
            password: "",
            ssl: false,
            maxConnections: 5,
            role: "upload",
          },
        ],
        outputDirectory: "/tmp",
        watchDirectory: "",
      },
      headers: { "Content-Type": "application/json" },
    });

    if (!resp.ok()) {
      const body = await resp.text();
      fakeNntp.close();
      throw new Error(
        `POST /api/setup/complete failed (${resp.status()}): ${body}`
      );
    }
  } finally {
    await ctx.dispose();
  }

  // Return teardown: shuts down the fake NNTP server after all tests finish.
  return async () => {
    fakeNntp.close();
  };
}
