<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { frontendLogs, type LogEntry } from "$lib/stores/logs";
import { ChevronDown, Pause, Play, RotateCcw } from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

type BackendLogEntry = {
	time: string;
	level: "INFO" | "WARN" | "ERROR" | "DEBUG";
	msg: string;
	[key: string]: unknown;
};

let backendLogs: LogEntry[] = [];
let fLogs: LogEntry[] = [];
let combinedLogs: LogEntry[] = [];

// Pagination state
let loadedLines = 0; // Total number of backend lines we've loaded
let loadingOlder = false;
let hasMoreLogs = true;
const LINES_PER_PAGE = 200;
let scrollDebounceTimer: ReturnType<typeof setTimeout> | undefined;

$: {
	combinedLogs = [...backendLogs, ...fLogs].sort(
		(a, b) => a.timestamp.getTime() - b.timestamp.getTime(),
	);
}

let loading = true;
let autoRefreshEnabled = false;
let intervalId: ReturnType<typeof setTimeout> | undefined;
let logContainer: HTMLDivElement | undefined;
let autoScroll = true;
let showFollowButton = false;

async function loadInitialLogs() {
	loading = true;
	try {
		// Load the most recent LINES_PER_PAGE lines (offset = 0)
		const rawLogs = await apiClient.getLogsPaginated(LINES_PER_PAGE, 0);
		const parsedLogs = parseLogLines(rawLogs);

		backendLogs = parsedLogs;
		loadedLines = parsedLogs.length;
		hasMoreLogs = parsedLogs.length === LINES_PER_PAGE;
	} catch (error) {
		backendLogs = [
			{
				timestamp: new Date(),
				level: "error",
				message: `Failed to load backend logs: ${error instanceof Error ? error.message : String(error)}`,
			},
		];
		hasMoreLogs = false;
	} finally {
		loading = false;
	}
}

async function loadOlderLogs() {
	if (loadingOlder || !hasMoreLogs) return;

	loadingOlder = true;
	try {
		// Load older logs starting from our current offset
		const rawLogs = await apiClient.getLogsPaginated(
			LINES_PER_PAGE,
			loadedLines,
		);
		const parsedLogs = parseLogLines(rawLogs);

		if (parsedLogs.length === 0) {
			hasMoreLogs = false;
			return;
		}

		// Remember current scroll position relative to the bottom
		const scrollFromBottom = logContainer
			? logContainer.scrollHeight -
				logContainer.scrollTop -
				logContainer.clientHeight
			: 0;

		// Prepend older logs to the beginning
		backendLogs = [...parsedLogs, ...backendLogs];
		loadedLines += parsedLogs.length;
		hasMoreLogs = parsedLogs.length === LINES_PER_PAGE;

		// Restore scroll position after the DOM updates
		requestAnimationFrame(() => {
			if (logContainer) {
				const newScrollTop =
					logContainer.scrollHeight -
					logContainer.clientHeight -
					scrollFromBottom;
				logContainer.scrollTop = Math.max(0, newScrollTop);
			}
		});
	} catch (error) {
		console.error("Failed to load older logs:", error);
		hasMoreLogs = false;
	} finally {
		loadingOlder = false;
	}
}

async function refreshRecentLogs() {
	if (loading) return;

	try {
		// Only refresh the most recent logs (same as initial load)
		const rawLogs = await apiClient.getLogsPaginated(LINES_PER_PAGE, 0);
		const parsedLogs = parseLogLines(rawLogs);

		// Find how many new logs we have
		const oldestCurrentLog =
			backendLogs.length > 0 ? backendLogs[backendLogs.length - 1] : null;
		let newLogsCount = 0;

		if (oldestCurrentLog) {
			// Count how many logs are newer than our newest current log
			newLogsCount = parsedLogs.findIndex(
				(log) =>
					log.timestamp.getTime() <= oldestCurrentLog.timestamp.getTime(),
			);
			if (newLogsCount === -1) {
				newLogsCount = parsedLogs.length;
			}
		} else {
			newLogsCount = parsedLogs.length;
		}

		if (newLogsCount > 0) {
			// Add only the new logs to the end
			const newLogs = parsedLogs.slice(0, newLogsCount);
			backendLogs = [...backendLogs, ...newLogs];
		}
	} catch (error) {
		console.error("Failed to refresh logs:", error);
	}
}

function parseLogLines(rawLogs: string): LogEntry[] {
	if (!rawLogs.trim()) return [];

	return rawLogs
		.split("\n")
		.filter((line) => line.trim() !== "")
		.map((line) => {
			try {
				const entry: BackendLogEntry = JSON.parse(line);
				return {
					timestamp: new Date(entry.time),
					level: entry.level.toLowerCase() as LogEntry["level"],
					message: entry.msg,
				};
			} catch (e) {
				return {
					timestamp: new Date(),
					level: "error",
					message: `Failed to parse log line: "${line}"`,
				};
			}
		});
}

frontendLogs.subscribe((logs) => {
	fLogs = logs;
});

function startAutoRefresh() {
	if (autoRefreshEnabled) return;
	autoRefreshEnabled = true;
	const refresh = async () => {
		await refreshRecentLogs();
		if (autoRefreshEnabled) {
			intervalId = setTimeout(refresh, 2000); // 2 seconds
		}
	};
	refresh();
}

function stopAutoRefresh() {
	if (!autoRefreshEnabled) return;
	autoRefreshEnabled = false;
	if (intervalId) {
		clearTimeout(intervalId);
		intervalId = undefined;
	}
}

onMount(() => {
	loadInitialLogs().then(() => {
		startAutoRefresh();
		followLogs();
	});
});

onDestroy(() => {
	stopAutoRefresh();
	if (scrollDebounceTimer) {
		clearTimeout(scrollDebounceTimer);
	}
});

function getLevelColor(level: LogEntry["level"]) {
	switch (level) {
		case "error":
			return "text-red-400";
		case "warn":
			return "text-yellow-400";
		case "info":
			return "text-blue-400";
		case "debug":
			return "text-base-content/50";
		default:
			return "text-base-content/70";
	}
}

function handleScroll() {
	if (!logContainer) return;

	const atBottom =
		logContainer.scrollTop + logContainer.clientHeight >=
		logContainer.scrollHeight - 10;
	const atTop = logContainer.scrollTop <= 50; // Increased threshold for better UX

	autoScroll = atBottom;
	showFollowButton = !atBottom;

	// Debounce loading older logs to prevent rapid API calls
	if (atTop && hasMoreLogs && !loadingOlder) {
		if (scrollDebounceTimer) {
			clearTimeout(scrollDebounceTimer);
		}
		scrollDebounceTimer = setTimeout(() => {
			loadOlderLogs();
		}, 100); // 100ms debounce
	}
}

function followLogs() {
	autoScroll = true;
	showFollowButton = false;
	if (logContainer) {
		logContainer.scrollTo({
			top: logContainer.scrollHeight,
			behavior: "smooth",
		});
	}
}

function handleRefresh() {
	// Reset pagination and reload from the beginning
	loadedLines = 0;
	hasMoreLogs = true;
	backendLogs = [];
	loadInitialLogs().then(() => {
		followLogs();
	});
}

$: if (autoScroll && logContainer) {
	// scroll to bottom after logs update
	logContainer.scrollTo({ top: logContainer.scrollHeight });
	showFollowButton = false;
}
</script>

<div class="w-full space-y-4">
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body">
			<div class="flex items-center justify-between mb-4">
				<h2 class="card-title text-xl">
					{$t('common.nav.logs')}
				</h2>
				<div class="flex items-center space-x-2">
					<span class="text-sm text-base-content/70">
						{combinedLogs.length} logs loaded
						{#if hasMoreLogs}(more available){/if}
					</span>
					<button
						class="btn btn-sm"
						onclick={autoRefreshEnabled ? stopAutoRefresh : startAutoRefresh}
					>
						{#if autoRefreshEnabled}
							<Pause class="w-4 h-4" />
							{$t('common.common.stop_auto_refresh')}
						{:else}
							<Play class="w-4 h-4" />
							{$t('common.common.start_auto_refresh')}
						{/if}
					</button>
					<button 
						class="btn btn-sm" 
						onclick={handleRefresh} 
						disabled={loading || autoRefreshEnabled}
					>
						<RotateCcw class="w-4 h-4" />
						{$t('common.common.refresh')}
					</button>
				</div>
			</div>
			<div class="relative">
				{#if showFollowButton}
					<button
						class="btn btn-circle btn-primary absolute right-4 top-4 z-10"
						onclick={followLogs}
						title="Follow logs"
					>
						<ChevronDown class="w-5 h-5" />
					</button>
				{/if}
				
				{#if loadingOlder}
					<div class="absolute left-1/2 top-4 z-10 -translate-x-1/2 badge badge-neutral">
						<span class="loading loading-spinner loading-xs mr-2"></span>
						Loading older logs...
					</div>
				{/if}
				
				<div
					bind:this={logContainer}
					class="h-[400px] overflow-y-auto rounded-lg bg-base-300 p-2 font-mono"
					onscroll={handleScroll}
				>
					{#if !hasMoreLogs && backendLogs.length > 0}
						<div class="text-center text-base-content/50 text-sm py-2 border-b border-base-content/20 mb-2">
							— Beginning of logs —
						</div>
					{/if}
					
					{#each combinedLogs as log, i (i)}
						<div class="flex items-start gap-2">
							<span class="w-48 flex-shrink-0 text-base-content/50">
								{log.timestamp.toLocaleTimeString()}
							</span>
							<span
								class="w-16 flex-shrink-0 font-bold uppercase {getLevelColor(log.level)}"
							>
								[{log.level}]
							</span>
							<span class="min-w-0 flex-1 whitespace-pre-wrap">{log.message}</span>
						</div>
					{/each}
					
					{#if loading}
						<div class="flex items-center justify-center py-4">
							<span class="loading loading-spinner loading-md mr-2"></span>
							Loading logs...
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>