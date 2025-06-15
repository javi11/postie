/**
 * Format file size in bytes to human readable format
 */
export function formatFileSize(bytes: number): string {
	const sizes = ["B", "KB", "MB", "GB", "TB"];
	if (bytes === 0) return "0 B";
	const i = Math.floor(Math.log(bytes) / Math.log(1024));
	return `${Math.round((bytes / 1024 ** i) * 100) / 100} ${sizes[i]}`;
}

/**
 * Format date string to localized format
 */
export function formatDate(dateString: string): string {
	return new Date(dateString).toLocaleString();
}

/**
 * Format time in milliseconds to human readable format (HH:MM:SS or MM:SS)
 */
export function formatTime(ms: number): string {
	if (ms <= 0) return "00:00";

	const totalSeconds = Math.floor(ms / 1000);
	const hours = Math.floor(totalSeconds / 3600);
	const minutes = Math.floor((totalSeconds % 3600) / 60);
	const seconds = totalSeconds % 60;

	if (hours > 0) {
		return `${hours.toString().padStart(2, "0")}:${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
	}
	return `${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
}

/**
 * Format upload/download speed in bytes per second to human readable format
 */
export function formatSpeed(bytesPerSecond: number): string {
	if (bytesPerSecond <= 0) return "0 MB/s";

	const mbPerSecond = bytesPerSecond / (1024 * 1024);

	if (mbPerSecond >= 1000) {
		return `${(mbPerSecond / 1000).toFixed(1)} GB/s`;
	}

	if (mbPerSecond >= 1) {
		return `${mbPerSecond.toFixed(1)} MB/s`;
	}

	const kbPerSecond = bytesPerSecond / 1024;
	return `${kbPerSecond.toFixed(1)} KB/s`;
}

/**
 * Calculate average speed from an array of speed measurements
 */
export function calculateAverageSpeed(speedHistory: number[]): number {
	if (speedHistory.length === 0) return 0;
	const sum = speedHistory.reduce((a, b) => a + b, 0);
	return sum / speedHistory.length;
}

/**
 * Get status color for badges based on status type
 */
export function getStatusColor(
	status: string,
): "yellow" | "green" | "red" | "gray" {
	switch (status) {
		case "pending":
			return "yellow";
		case "complete":
			return "green";
		case "error":
			return "red";
		default:
			return "gray";
	}
}

/**
 * Parse duration string to nanoseconds for configuration
 */
export function parseDuration(durationStr: string): number {
	if (!durationStr) return 0;

	const match = durationStr.match(/^(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)$/);
	if (!match) return 0;

	const value = Number.parseFloat(match[1]);
	const unit = match[2];

	switch (unit) {
		case "ns":
			return value;
		case "us":
		case "µs":
			return value * 1000;
		case "ms":
			return value * 1000000;
		case "s":
			return value * 1000000000;
		case "m":
			return value * 60 * 1000000000;
		case "h":
			return value * 3600 * 1000000000;
		default:
			return 0;
	}
}

/**
 * Debounce function to limit rapid function calls
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
	func: T,
	wait: number,
): (...args: Parameters<T>) => void {
	let timeout: number;
	return (...args: Parameters<T>) => {
		clearTimeout(timeout);
		timeout = setTimeout(() => func(...args), wait);
	};
}
