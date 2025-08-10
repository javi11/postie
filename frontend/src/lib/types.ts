export interface Par2DownloadStatus {
	status: "downloading" | "completed" | "error";
	message: string;
}

export interface ProgressStatus {
	currentFile: string;
	totalFiles: number;
	completedFiles: number;
	stage: string;
	details: string;
	isRunning: boolean;
	lastUpdate: number;
	percentage: number;
	currentFileProgress: number;
	jobID: string;
	totalBytes: number;
	transferredBytes: number;
	currentFileBytes: number;
	speed: number;
	secondsLeft: number;
	elapsedTime: number;
}
