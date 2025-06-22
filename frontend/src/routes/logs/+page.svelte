<script lang="ts">
import { GetLogs } from "$lib/wailsjs/go/backend/App";
import { onDestroy, onMount } from "svelte";
import { Card, Button, Spinner } from "flowbite-svelte";
import { t } from "$lib/i18n";
import {
	ArrowsRepeatOutline,
	PauseOutline,
	PlayOutline,
} from "flowbite-svelte-icons";
import { frontendLogs, type LogEntry } from "$lib/stores/logs";

type BackendLogEntry = {
	time: string;
	level: "INFO" | "WARN" | "ERROR" | "DEBUG";
	msg: string;
	[key: string]: unknown;
};

let backendLogs: LogEntry[] = [];
let fLogs: LogEntry[] = [];
let combinedLogs: LogEntry[] = [];

$: {
	combinedLogs = [...backendLogs, ...fLogs].sort(
		(a, b) => a.timestamp.getTime() - b.timestamp.getTime(),
	);
}

let loading = true;
let autoRefreshEnabled = false;
let intervalId: ReturnType<typeof setInterval> | undefined;

async function loadLogs() {
	loading = true;
	try {
		const rawLogs = await GetLogs();
		const parsedLogs: LogEntry[] = rawLogs
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
		backendLogs = parsedLogs;
	} catch (error) {
		backendLogs = [
			{
				timestamp: new Date(),
				level: "error",
				message: `Failed to load backend logs: ${error instanceof Error ? error.message : String(error)}`,
			},
		];
	} finally {
		loading = false;
	}
}

frontendLogs.subscribe((logs) => {
	fLogs = logs;
});

function startAutoRefresh() {
	if (autoRefreshEnabled) return;
	autoRefreshEnabled = true;
	loadLogs();
	intervalId = setInterval(loadLogs, 2000); // 2 seconds
}

function stopAutoRefresh() {
	if (!autoRefreshEnabled) return;
	autoRefreshEnabled = false;
	if (intervalId) {
		clearInterval(intervalId);
		intervalId = undefined;
	}
}

onMount(() => {
	loadLogs();
	startAutoRefresh();
});

onDestroy(() => {
	stopAutoRefresh();
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
			return "text-gray-400";
		default:
			return "text-gray-200";
	}
}
</script>

<div class="w-full space-y-4">
	<Card class="flex min-w-full flex-col p-10">
		<div class="mb-4 flex items-center justify-between">
			<h5 class="text-xl font-bold leading-none text-gray-900 dark:text-white">
				{$t('common.nav.logs')}
			</h5>
			<div class="flex items-center space-x-2">
				<Button
					class="cursor-pointer"
					onclick={autoRefreshEnabled ? stopAutoRefresh : startAutoRefresh}
				>
					{#if autoRefreshEnabled}
						<PauseOutline class="mr-2 h-4 w-4" />
						{$t('common.common.stop_auto_refresh')}
					{:else}
						<PlayOutline class="mr-2 h-4 w-4" />
						{$t('common.common.start_auto_refresh')}
					{/if}
				</Button>
				<Button class="cursor-pointer" onclick={loadLogs} disabled={loading || autoRefreshEnabled}>
					<ArrowsRepeatOutline class="mr-2 h-4 w-4" />
					{$t('common.common.refresh')}
				</Button>
			</div>
		</div>
		<div class="max-h-screen overflow-auto rounded-lg bg-gray-800 p-4 font-mono">
            {#if loading}
                <div class="flex items-center justify-center text-white">
                    <Spinner class="mr-2 w-8 h-8" />
                    {$t('common.common.loading')}
                </div>
            {:else}
                {#each combinedLogs as log (log.timestamp.getTime() + log.message)}
                    <div class="flex items-start gap-2">
                        <span class="w-48 flex-shrink-0 text-gray-500">
                            {log.timestamp.toLocaleTimeString()}
                        </span>
                        <span class="w-16 flex-shrink-0 font-bold uppercase"
                            class:text-red-400={log.level === 'error'}
                            class:text-yellow-400={log.level === 'warn'}
                            class:text-blue-400={log.level === 'info'}
                            class:text-purple-400={log.level === 'debug'}
                            class:text-gray-400={log.level === 'log'}
                        >
                            [{log.level}]
                        </span>
                        <pre class="whitespace-pre-wrap text-gray-200">{log.message}</pre>
                    </div>
                {/each}
            {/if}
		</div>
	</Card>
</div> 