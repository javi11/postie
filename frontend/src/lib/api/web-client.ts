// Web API client for browser environment (replaces Wails bindings)

import type { backend, config, processor } from "$lib/wailsjs/go/models";

const API_BASE = "/api";

export class WebClient {
	private ws: WebSocket | null = null;
	private wsListeners: Map<string, (data: unknown) => void> = new Map();

	constructor() {
		this.initWebSocket();
	}

	// Wait for WebSocket to be connected
	async waitForConnection(): Promise<void> {
		return new Promise((resolve, reject) => {
			if (this.ws?.readyState === WebSocket.OPEN) {
				resolve();
				return;
			}

			const checkConnection = () => {
				if (this.ws?.readyState === WebSocket.OPEN) {
					resolve();
				} else if (
					this.ws?.readyState === WebSocket.CLOSED ||
					this.ws?.readyState === WebSocket.CLOSING
				) {
					reject(new Error("WebSocket connection failed"));
				} else {
					setTimeout(checkConnection, 100);
				}
			};

			checkConnection();
		});
	}

	private initWebSocket() {
		const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
		const wsUrl = `${protocol}//${window.location.host}/api/ws`;
		console.log("WebSocket URL:", wsUrl);
		console.log("Attempting WebSocket connection...");

		this.ws = new WebSocket(wsUrl);

		this.ws.onopen = (event) => {
			console.log("WebSocket connected successfully", event);
		};

		this.ws.onmessage = (event) => {
			try {
				const message = JSON.parse(event.data);

				// Check if this is a structured message with type and data
				if (message.type && message.data !== undefined) {
					// Dispatch to specific event listener
					const listener = this.wsListeners.get(message.type);
					if (listener) {
						listener(message.data);
					}
				} else {
					// Fallback: emit to all listeners for unstructured messages
					for (const listener of this.wsListeners.values()) {
						listener(message);
					}
				}
			} catch (error) {
				console.error("Error parsing WebSocket message:", error);
			}
		};

		this.ws.onclose = (event) => {
			console.log("WebSocket disconnected:", event.code, event.reason);
			console.log("Attempting to reconnect in 3 seconds...");
			setTimeout(() => this.initWebSocket(), 3000);
		};

		this.ws.onerror = (error) => {
			console.error("WebSocket error:", error);
			console.log("WebSocket state:", this.ws?.readyState);
		};
	}

	// Add event listener for real-time updates
	async on(event: string, callback: (data: unknown) => void): Promise<void> {
		// Wait for WebSocket connection before registering listener
		try {
			await this.waitForConnection();
		} catch (error) {
			console.error(
				"Failed to connect WebSocket before registering listener:",
				error,
			);
		}

		this.wsListeners.set(event, callback);
	}

	// Remove event listener
	off(event: string) {
		this.wsListeners.delete(event);
	}

	async get<T>(endpoint: string): Promise<T> {
		const response = await fetch(`${API_BASE}${endpoint}`);
		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}
		return response.json();
	}

	async post<T>(endpoint: string, data?: unknown): Promise<T> {
		const response = await fetch(`${API_BASE}${endpoint}`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: data ? JSON.stringify(data) : undefined,
		});

		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}

		// Handle empty responses
		const text = await response.text();
		return text ? JSON.parse(text) : ({} as T);
	}

	async delete<T>(endpoint: string): Promise<T> {
		const response = await fetch(`${API_BASE}${endpoint}`, {
			method: "DELETE",
		});

		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}

		const text = await response.text();
		return text ? JSON.parse(text) : ({} as T);
	}

	// App methods
	async getStatus(): Promise<backend.AppStatus> {
		return this.get<backend.AppStatus>("/status");
	}

	async getConfig(): Promise<config.ConfigData> {
		return this.get<config.ConfigData>("/config");
	}

	async saveConfig(config: config.ConfigData): Promise<void> {
		return this.post<void>("/config", config);
	}

	// Queue methods
	async getQueueItems(): Promise<backend.QueueItem[]> {
		return this.get<backend.QueueItem[]>("/queue");
	}

	async getQueueStats(): Promise<backend.QueueStats> {
		return this.get<backend.QueueStats>("/queue/stats");
	}

	async retryJob(id: string): Promise<void> {
		return this.post<void>(`/queue/${id}/retry`);
	}

	async cancelJob(id: string): Promise<void> {
		return this.delete<void>(`/queue/${id}/cancel`);
	}

	// Processor methods
	async getProcessorStatus(): Promise<backend.ProcessorStatus> {
		return this.get<backend.ProcessorStatus>("/processor/status");
	}

	async getRunningJobs(): Promise<processor.RunningJobItem[]> {
		return this.get<processor.RunningJobItem[]>("/running-jobs");
	}

	async getProgress(): Promise<backend.ProgressTracker> {
		return this.get<backend.ProgressTracker>("/progress");
	}

	// Logs
	async getLogs(limit?: number, offset?: number): Promise<string> {
		const params = new URLSearchParams();
		if (limit) params.append("limit", limit.toString());
		if (offset) params.append("offset", offset.toString());

		const endpoint = `/logs${params.toString() ? `?${params.toString()}` : ""}`;
		const response = await fetch(`${API_BASE}${endpoint}`);

		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}

		return response.text();
	}

	// Queue management
	async removeFromQueue(id: string): Promise<void> {
		return this.delete<void>(`/queue/${id}`);
	}

	async setQueueItemPriority(id: string, priority: number): Promise<void> {
		return this.post<void>(`/queue/${id}/priority`, { priority });
	}

	async clearQueue(): Promise<void> {
		return this.delete<void>("/queue");
	}

	async addFilesToQueue(): Promise<void> {
		// This triggers a file picker dialog in the browser
		// The actual file selection is handled by the frontend
		return this.post<void>("/queue/add-files");
	}

	// Upload management
	async uploadFiles(
		files: FileList,
		onProgress?: (progress: number) => void,
		setRequest?: (xhr: XMLHttpRequest) => void,
	): Promise<void> {
		const formData = new FormData();
		for (let i = 0; i < files.length; i++) {
			formData.append("files", files[i]);
		}

		return new Promise((resolve, reject) => {
			const xhr = new XMLHttpRequest();

			// Store the request reference for cancellation
			if (setRequest) {
				setRequest(xhr);
			}

			// Track upload progress
			if (onProgress) {
				xhr.upload.addEventListener("progress", (event) => {
					if (event.lengthComputable) {
						const progress = (event.loaded / event.total) * 100;
						onProgress(progress);
					}
				});
			}

			xhr.addEventListener("load", () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve();
				} else {
					reject(new Error(`HTTP error! status: ${xhr.status}`));
				}
			});

			xhr.addEventListener("error", () => {
				reject(new Error("Upload failed"));
			});

			xhr.addEventListener("abort", () => {
				reject(new Error("Upload cancelled"));
			});

			xhr.open("POST", `${API_BASE}/upload`);
			xhr.send(formData);
		});
	}

	async cancelUpload(): Promise<void> {
		return this.post<void>("/upload/cancel");
	}

	// NZB operations
	async downloadNZB(id: string): Promise<void> {
		const response = await fetch(`${API_BASE}/nzb/${id}/download`);

		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}

		// Create a blob from the response and trigger download
		const blob = await response.blob();
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = `${id}.nzb`;
		document.body.appendChild(a);
		a.click();
		window.URL.revokeObjectURL(url);
		document.body.removeChild(a);
	}

	// Setup Wizard
	async validateNNTPServer(
		serverData: backend.ServerData,
	): Promise<backend.ValidationResult> {
		return this.post<backend.ValidationResult>("/validate-server", serverData);
	}

	async setupWizardComplete(
		wizardData: backend.SetupWizardData,
	): Promise<void> {
		return this.post<void>("/setup/complete", wizardData);
	}
}

// Singleton instance
let webClientInstance: WebClient | null = null;

export function getWebClient(): WebClient {
	if (!webClientInstance) {
		console.log("Initializing web client");
		webClientInstance = new WebClient();
	}
	return webClientInstance;
}
