import type { backend } from "$lib/wailsjs/go/models";
import { derived, writable } from "svelte/store";

// Create app status store
export const appStatus = writable<backend.AppStatus>({
	needsConfiguration: false,
	criticalConfigError: false,
	configPath: "",
	configValid: false,
	error: "",
	hasConfig: false,
	hasPostie: false,
	hasServers: false,
	isFirstStart: false,
	serverCount: 0,
	uploading: false,
	validServerCount: 0,
});

export const progress = writable<Record<string, backend.ProgressTracker>>({});

// Create upload state store
export const isUploading = derived(progress, (progress) => {
	return Object.values(progress).some(
		(job: backend.ProgressTracker) => job.isRunning,
	);
});

// Create settings store for save functionality
export const settingsSaveFunction = writable<(() => Promise<void>) | null>(
	null,
);

// Create advanced mode store for settings UI
export const advancedMode = writable<boolean>(false);
