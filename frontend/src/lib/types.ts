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

// NNTP Pool Metrics Types
export interface NntpPoolMetrics {
	timestamp: string;
	uptime: number;
	activeConnections: number;
	downloadSpeed: number;
	uploadSpeed: number;
	commandSuccessRate: number;
	errorRate: number;
	totalAcquires: number;
	totalBytesDownloaded: number;
	totalBytesUploaded: number;
	totalArticlesRetrieved: number;
	averageAcquireWaitTime: number;
	totalErrors: number;
	providers: NntpProviderMetrics[];
}

export interface NntpProviderMetrics {
	host: string;
	username: string;
	state: string;
	totalConnections: number;
	maxConnections: number;
	acquiredConnections: number;
	idleConnections: number;
	totalBytesDownloaded: number;
	totalBytesUploaded: number;
	successRate: number;
	averageConnectionAge: number;
}
