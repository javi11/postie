import { writable } from "svelte/store";

export type LogEntry = {
	timestamp: Date;
	level: "log" | "info" | "warn" | "error" | "debug";
	message: string;
	attributes?: Record<string, unknown>;
};

export const frontendLogs = writable<LogEntry[]>([]);

function formatMessage(args: unknown[]): string {
	return args
		.map((arg) => {
			if (typeof arg === "string") {
				return arg;
			}
			if (arg instanceof Error) {
				return arg.message;
			}

			return JSON.stringify(arg, null, 2);
		})
		.join(" ");
}

export function setupConsoleInterceptor() {
	const originalConsole = { ...console };

	const intercept =
		(level: LogEntry["level"]) =>
		(...args: unknown[]) => {
			const message = formatMessage(args);
			const newLog: LogEntry = { timestamp: new Date(), level, message };

			frontendLogs.update((logs) => {
				const newLogs = [...logs, newLog];
				if (newLogs.length > 200) {
					// Limit log history
					return newLogs.slice(newLogs.length - 200);
				}
				return newLogs;
			});

			if (originalConsole[level]) {
				originalConsole[level](...args);
			} else {
				originalConsole.log(...args);
			}
		};

	console.log = intercept("log");
	console.info = intercept("info");
	console.warn = intercept("warn");
	console.error = intercept("error");
	console.debug = intercept("debug");
}
