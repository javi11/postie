import { afterEach, beforeEach, expect, mock, test } from "bun:test";
import { WebClient } from "./web-client";
mock.module("$app/environment", () => ({ browser: true }));
const { UnifiedClient } = await import("./client");

// A stalled proxy handshake leaves the browser socket CONNECTING indefinitely.
class ConnectingSocket {
	static CONNECTING = 0;
	static OPEN = 1;
	static CLOSED = 3;
	static instance: ConnectingSocket;
	readyState = ConnectingSocket.CONNECTING;
	onopen?: (event: unknown) => void;
	onmessage?: (event: { data: string }) => void;

	constructor(_url: string) {
		ConnectingSocket.instance = this;
	}
}

const originalWebSocket = globalThis.WebSocket;
const originalWindow = globalThis.window;
const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

beforeEach(() => {
	globalThis.WebSocket = ConnectingSocket as unknown as typeof WebSocket;
	globalThis.window = { location: { protocol: "http:", host: "localhost" } } as Window & typeof globalThis;
});

afterEach(() => {
	globalThis.WebSocket = originalWebSocket;
	globalThis.window = originalWindow;
});

test("delivers the first event immediately after the socket opens", async () => {
	const client = new WebClient();
	const received: unknown[] = [];
	const subscription = client.on("running-jobs-updated", (data) => received.push(data));
	try {
		ConnectingSocket.instance.readyState = ConnectingSocket.OPEN;
		ConnectingSocket.instance.onopen?.({});
		ConnectingSocket.instance.onmessage?.({
			data: JSON.stringify({ type: "running-jobs-updated", data: [{ id: "job-1" }] }),
		});
		expect(received).toEqual([[{ id: "job-1" }]]);
		client.off("running-jobs-updated");
		ConnectingSocket.instance.onmessage?.({
			data: JSON.stringify({ type: "running-jobs-updated", data: [{ id: "job-2" }] }),
		});
		expect(received).toEqual([[{ id: "job-1" }]]);
	} finally {
		ConnectingSocket.instance.readyState = ConnectingSocket.OPEN;
		await subscription;
	}
});

test("removing a dashboard listener before connection prevents it being restored on OPEN", async () => {
	const client = new UnifiedClient();
	const received: unknown[] = [];
	const listener = (data: unknown) => { received.push(data); };
	await client.on("running-jobs-updated", listener);
	await client.off("running-jobs-updated", listener);

	ConnectingSocket.instance.readyState = ConnectingSocket.OPEN;
	ConnectingSocket.instance.onopen?.({});
	// The old implementation deferred registration to its 100ms connection poll.
	await sleep(120);
	ConnectingSocket.instance.onmessage?.({
		data: JSON.stringify({ type: "running-jobs-updated", data: [{ id: "late-job" }] }),
	});
	expect(received).toEqual([]);
});
