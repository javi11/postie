import { derived, writable } from "svelte/store";

export interface AppStatus {
	needsConfiguration: boolean;
	criticalConfigError: boolean;
	[key: string]: unknown;
}

// Create app status store
export const appStatus = writable<AppStatus>({
	needsConfiguration: false,
	criticalConfigError: false,
});

// Create progress store (map of jobID -> progress object)
export interface JobProgress {
	jobID: string;
	currentFile: string;
	totalFiles: number;
	completedFiles: number;
	stage: string;
	details: string;
	isRunning: boolean;
	lastUpdate: number;
	percentage: number;
	currentFileProgress: number;
	elapsedTime?: number;
	speed?: number;
	secondsLeft?: number;
}

export const progress = writable<Record<string, JobProgress>>({});

// Create upload state store
export const isUploading = derived(progress, (progress) => {
	return Object.values(progress).some((job: JobProgress) => job.isRunning);
});

// Create settings store for save functionality
export const settingsSaveFunction = writable<(() => Promise<void>) | null>(
	null,
);
