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

// Provider status and metrics types
export interface NntpProviderMetrics {
	host: string;
	username: string;
	state: string;
	totalConnections: number;
	maxConnections: number;
	acquiredConnections: number;
	idleConnections: number;
	totalBytesUploaded: number;
	totalArticlesPosted: number;
	successRate: number;
	averageConnectionAge: number;
}

export interface NntpPoolMetrics {
	timestamp: string;
	uptime: number;
	activeConnections: number;
	uploadSpeed: number;
	commandSuccessRate: number;
	errorRate: number;
	totalAcquires: number;
	totalBytesUploaded: number;
	totalArticlesRetrieved: number;
	totalArticlesPosted: number;
	averageAcquireWaitTime: number;
	totalErrors: number;
	providers: NntpProviderMetrics[];
}