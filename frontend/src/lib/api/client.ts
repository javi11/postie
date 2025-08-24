// Unified API client that switches between Wails (desktop) and HTTP (web) modes

import { browser } from "$app/environment";
import type { backend, config, processor, watcher } from "$lib/wailsjs/go/models";
// Using backend types for pagination - remove temporary types import
import type { WebClient } from "./web-client";

type WailsApp = typeof import("$lib/wailsjs/go/backend/App");
type WailsRuntime = typeof import("$lib/wailsjs/runtime/runtime");

// Environment detection
function isWailsEnvironment(): boolean {
	if (!browser) return false;

	return !!(
		typeof window !== "undefined" &&
		"go" in window &&
		window.go &&
		typeof window.go === "object" &&
		"backend" in window.go &&
		window.go.backend &&
		"App" in (window.go as { backend: { App: unknown } }).backend
	);
}

function isWebEnvironment(): boolean {
	if (!browser) return false;

	// Check if we're in a web environment (not Wails)
	return !isWailsEnvironment() && typeof fetch !== "undefined";
}

// Lazy imports to avoid bundling issues
let wailsClient: {
	App: WailsApp;
	Runtime: WailsRuntime;
} | null = null;
let webClient: WebClient | null = null;

async function getWailsClient(): Promise<{
	App: WailsApp;
	Runtime: WailsRuntime;
}> {
	if (!wailsClient) {
		const [AppModule, RuntimeModule] = await Promise.all([
			import("$lib/wailsjs/go/backend/App"),
			import("$lib/wailsjs/runtime/runtime"),
		]);
		wailsClient = { App: AppModule, Runtime: RuntimeModule };
	}

	return wailsClient;
}

async function getWebClient() {
	if (!webClient) {
		const { getWebClient: getWebClientFn } = await import("./web-client");
		webClient = getWebClientFn();
	}
	return webClient;
}

// Event handling abstraction
type EventCallback = (data: unknown) => void;

const eventListeners = new Map<string, Set<EventCallback>>();

// Unified API interface
export class UnifiedClient {
	private _isReady = false;
	private _environment: "wails" | "web" | "unknown" = "unknown";

	async initialize(): Promise<void> {
		if (this._isReady) return;

		if (!browser) {
			this._environment = "unknown";
			return;
		}

		if (isWailsEnvironment()) {
			this._environment = "wails";
			// Wait for Wails runtime to be ready
			await this.waitForWailsRuntime();
		} else if (isWebEnvironment()) {
			this._environment = "web";
			// Initialize web client
			await getWebClient();
		} else {
			this._environment = "unknown";
			console.warn("Unable to detect environment, some features may not work");
		}

		console.log(`API Client initialized in ${this._environment} mode`);

		this._isReady = true;
	}

	get environment(): "wails" | "web" | "unknown" {
		return this._environment;
	}

	get isReady(): boolean {
		return this._isReady;
	}

	private async waitForWailsRuntime(): Promise<void> {
		const maxAttempts = 50; // Max 5 seconds (50 * 100ms)
		let attempts = 0;

		while (attempts < maxAttempts) {
			if (isWailsEnvironment()) {
				return;
			}
			await new Promise((resolve) => setTimeout(resolve, 100));
			attempts++;
		}

		throw new Error("Wails runtime not available after timeout");
	}

	// App Status
	async getAppStatus(): Promise<backend.AppStatus> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetAppStatus();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getStatus();
		}

		throw new Error("No client available");
	}

	// Configuration
	async getConfig(): Promise<config.ConfigData> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetConfig()
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getConfig();
		}

		throw new Error("No client available");
	}

	async saveConfig(config: config.ConfigData): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.SaveConfig(
				config as unknown as import("$lib/wailsjs/go/models").config.ConfigData,
			);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.saveConfig(config);
		}
	}

	// Queue Management
	async getQueueItems(): Promise<backend.QueueItem[]> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetQueueItems();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getQueueItems();
		}

		throw new Error("No client available");
	}

	async getQueueItemsPaginated(params: backend.PaginationParams): Promise<backend.PaginatedQueueResult> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetQueueItemsPaginated(params);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getQueueItemsPaginated(params);
		}

		throw new Error("No client available");
	}

	async getQueueStats(): Promise<backend.QueueStats> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetQueueStats();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getQueueStats();
		}

		throw new Error("No client available");
	}

	async retryJob(id: string): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.RetryJob(id);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.retryJob(id);
		}

		throw new Error("No client available");
	}

	async cancelJob(id: string): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.CancelJob(id);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.cancelJob(id);
		}
	}

	async removeFromQueue(id: string): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.RemoveFromQueue(id);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.removeFromQueue(id);
		}
	}

	async setQueueItemPriority(id: string, priority: number): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.SetQueueItemPriority(id, priority);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.setQueueItemPriority(id, priority);
		}
	}

	// Processing
	async getProcessorStatus(): Promise<backend.ProcessorStatus> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetProcessorStatus();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getProcessorStatus();
		}

		throw new Error("No client available");
	}

	async getWatcherStatus(): Promise<watcher.WatcherStatusInfo> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetWatcherStatus();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getWatcherStatus();
		}

		throw new Error("No client available");
	}

	async getRunningJobDetails(): Promise<processor.RunningJobDetails[]> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetRunningJobsDetails();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getRunningJobDetails();
		}

		throw new Error("No client available");
	}

	async pauseProcessing(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.PauseProcessing();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.pauseProcessing();
		}

		throw new Error("No client available");
	}

	async resumeProcessing(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.ResumeProcessing();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.resumeProcessing();
		}

		throw new Error("No client available");
	}

	async isProcessingPaused(): Promise<boolean> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.IsProcessingPaused();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.isProcessingPaused();
		}

		throw new Error("No client available");
	}

	async isProcessingAutoPaused(): Promise<boolean> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.IsProcessingAutoPaused();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.isProcessingAutoPaused();
		}

		throw new Error("No client available");
	}

	async getAutoPauseReason(): Promise<string> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetAutoPauseReason();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getAutoPauseReason();
		}

		throw new Error("No client available");
	}

	async uploadFiles(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.UploadFiles();
		}

		if (this._environment === "web") {
			// Web upload is handled via file input and drag/drop
			throw new Error("Use uploadFileList for web uploads");
		}
	}

	async uploadFileList(
		files: FileList,
		onProgress?: (progress: number) => void,
		setRequest?: (xhr: XMLHttpRequest) => void,
	): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			// For Wails, we'd need to save files temporarily and call HandleDroppedFiles
			// This is more complex and would require file system access
			throw new Error("File upload from FileList not supported in Wails mode");
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.uploadFiles(files, onProgress, setRequest);
		}

		throw new Error("No client available");
	}

	// Logs
	async getLogs(): Promise<string> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetLogs();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getLogs();
		}

		throw new Error("No client available");
	}

	async getLogsPaginated(limit: number, offset: number): Promise<string> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetLogsPaginated(limit, offset);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getLogs(limit, offset);
		}

		throw new Error("No client available");
	}

	// Event Handling
	async on(event: string, callback: EventCallback): Promise<void> {
		await this.initialize();

		// Store callback for cleanup
		if (!eventListeners.has(event)) {
			eventListeners.set(event, new Set());
		}
		const listeners = eventListeners.get(event);
		if (listeners) {
			listeners.add(callback);
		}

		if (this._environment === "wails") {
			const client = await getWailsClient();
			client.Runtime.EventsOn(event, callback);
		} else if (this._environment === "web") {
			const client = await getWebClient();
			client.on(event, callback);
		}
	}

	async off(event: string, callback?: EventCallback): Promise<void> {
		await this.initialize();

		if (callback) {
			eventListeners.get(event)?.delete(callback);
		} else {
			eventListeners.get(event)?.clear();
		}

		if (this._environment === "wails") {
			// Wails doesn't have EventsOff, events are cleaned up automatically
		} else if (this._environment === "web") {
			const client = await getWebClient();
			client.off(event);
		}
	}

	// Queue Actions
	async clearQueue(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.ClearQueue();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.clearQueue();
		}

		throw new Error("No client available");
	}

	async addFilesToQueue(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.AddFilesToQueue();
		}

		if (this._environment === "web") {
			// In web mode, we handle file selection via HTML input
			// This method serves as a trigger for the file picker
			const client = await getWebClient();
			return client.addFilesToQueue();
		}

		throw new Error("No client available");
	}

	async cancelUpload(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.CancelUpload();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.cancelUpload();
		}

		throw new Error("No client available");
	}

	// NZB operations
	async downloadNZB(id: string): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.DownloadNZB(id);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.downloadNZB(id);
		}

		throw new Error("No client available");
	}

	// Navigation (Wails-specific)
	async navigateToSettings(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.NavigateToSettings();
		}
		// In web mode, navigation is handled by SvelteKit
	}

	async navigateToDashboard(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.NavigateToDashboard();
		}
		// In web mode, navigation is handled by SvelteKit
	}

	// Directory Selection
	async selectTempDirectory(): Promise<string> {
		await this.initialize();

		if (this._environment === "wails") {
			const { SelectTempDirectory } = await import(
				"$lib/wailsjs/go/backend/App"
			);
			return SelectTempDirectory();
		}

		if (this._environment === "web") {
			// In web mode, directory selection would need to be handled differently
			// For now, return empty string to indicate no selection
			return "";
		}

		throw new Error("No client available");
	}

	// Setup Wizard
	async validateNNTPServer(
		serverData: backend.ServerData,
	): Promise<backend.ValidationResult> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.ValidateNNTPServer(serverData);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.validateNNTPServer(serverData);
		}

		throw new Error("No client available");
	}

	async testProviderConnectivity(
		serverData: backend.ServerData,
	): Promise<backend.ValidationResult> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.TestProviderConnectivity(serverData);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.testProviderConnectivity(serverData);
		}

		throw new Error("No client available");
	}

	async setupWizardComplete(
		wizardData: backend.SetupWizardData,
	): Promise<void> {
		console.log("Completing setup wizard with data:", wizardData);
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.SetupWizardComplete(wizardData);
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.setupWizardComplete(wizardData);
		}

		throw new Error("No client available");
	}

	async getAppliedConfig(): Promise<config.ConfigData> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetAppliedConfig();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getAppliedConfig();
		}

		throw new Error("No client available");
	}

	// Pending Config Management
	async hasPendingConfigChanges(): Promise<boolean> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.HasPendingConfigChanges();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.hasPendingConfigChanges();
		}

		throw new Error("No client available");
	}

	async getPendingConfigStatus(): Promise<Record<string, unknown>> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetPendingConfigStatus();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getPendingConfigStatus();
		}

		throw new Error("No client available");
	}

	async applyPendingConfig(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.ApplyPendingConfig();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.applyPendingConfig();
		}

		throw new Error("No client available");
	}

	async discardPendingConfig(): Promise<void> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.DiscardPendingConfig();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.discardPendingConfig();
		}

		throw new Error("No client available");
	}

	async selectOutputDirectory(): Promise<string> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.SelectOutputDirectory();
		}

		throw new Error("No client available");
	}

	// NNTP Pool Metrics
	async getNntpPoolMetrics(): Promise<backend.NntpPoolMetrics> {
		await this.initialize();

		if (this._environment === "wails") {
			const client = await getWailsClient();
			return client.App.GetNntpPoolMetrics();
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.getNntpPoolMetrics();
		}

		throw new Error("No client available");
	}

	// Filesystem operations (web-only)
	async browseFilesystem(path: string): Promise<{
		path: string;
		items: Array<{
			name: string;
			path: string;
			isDir: boolean;
			size: number;
			modTime: string;
		}>;
	}> {
		await this.initialize();

		if (this._environment === "wails") {
			throw new Error("Filesystem browsing not supported in Wails mode");
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.browseFilesystem(path);
		}

		throw new Error("No client available");
	}

	async importFiles(filePaths: string[]): Promise<{
		success: boolean;
		importedCount: number;
		message: string;
	}> {
		await this.initialize();

		if (this._environment === "wails") {
			throw new Error("File import not supported in Wails mode");
		}

		if (this._environment === "web") {
			const client = await getWebClient();
			return client.importFiles(filePaths);
		}

		throw new Error("No client available");
	}
}

// Export singleton instance
export const apiClient = new UnifiedClient();
export default apiClient;
