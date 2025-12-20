<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import type { backend } from "$lib/wailsjs/go/models";
import {
	AlertTriangle,
	ChartPie,
	CheckCircle,
	Clock,
	List,
	Play,
} from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let queueStats: backend.QueueStats = {
	total: 0,
	pending: 0,
	running: 0,
	complete: 0,
	error: 0,
};

let intervalId: ReturnType<typeof setInterval> | undefined;
let debounceTimer: ReturnType<typeof setTimeout> | undefined;

onMount(async () => {
	// Listen for queue updates with debouncing to prevent double-fetches
	await apiClient.on("queue-updated", () => {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(loadQueueStats, 100);
	});

  // Set up polling to refresh stats every 10 seconds
	intervalId = setInterval(() => {
		loadQueueStats();
	}, 10000);

	// Load initial stats
	loadQueueStats();
});

onDestroy(async () => {
	// Clean up event listener and timers
	await apiClient.off("queue-updated");
	clearTimeout(debounceTimer);
	if (intervalId) {
		clearInterval(intervalId);
	}
});

async function loadQueueStats() {
	try {
		queueStats = await apiClient.getQueueStats();
	} catch (error) {
		console.error("Failed to load queue stats:", error);
	}
}
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center gap-3 mb-6">
    <div class="p-2 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600">
      <ChartPie class="w-6 h-6 text-white" />
    </div>
    <div>
      <h2 class="text-xl font-semibold">
        {$t('dashboard.stats.queue_stats.title')}
      </h2>
      <p class="text-sm text-base-content/60">{$t('dashboard.stats.queue_stats.overview')}</p>
    </div>
  </div>

  <!-- Main Stats Grid -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Total -->
    <div class="card bg-base-100 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-base-content/70">{$t('dashboard.stats.queue_stats.total_items')}</p>
          <p class="text-2xl font-bold text-base-content mt-1">{queueStats.total}</p>
        </div>
        <div class="p-3 rounded-full bg-base-300">
          <List class="w-6 h-6 text-base-content/70" />
        </div>
      </div>
    </div>

    <!-- Pending -->
    <div class="card bg-base-100 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-base-content/70">{$t('dashboard.stats.pending')}</p>
          <p class="text-2xl font-bold text-warning mt-1">{queueStats.pending}</p>
        </div>
        <div class="p-3 rounded-full bg-warning/10">
          <Clock class="w-6 h-6 text-warning" />
        </div>
      </div>
      {#if queueStats.pending > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-amber-500 rounded-full animate-pulse mr-2"></div>
          <span class="text-xs text-warning">{$t('dashboard.stats.queue_stats.waiting_to_process')}</span>
        </div>
      {/if}
    </div>

    <!-- Complete -->
    <div class="card bg-base-100 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-base-content/70">{$t('dashboard.stats.complete')}</p>
          <p class="text-2xl font-bold text-success mt-1">{queueStats.complete}</p>
        </div>
        <div class="p-3 rounded-full bg-success/10">
          <CheckCircle class="w-6 h-6 text-success" />
        </div>
      </div>
      {#if queueStats.complete > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
          <span class="text-xs text-success">{$t('dashboard.stats.queue_stats.successfully_finished')}</span>
        </div>
      {/if}
    </div>

     <!-- Error Section -->
    <div class="card bg-base-100 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-base-content/70">{$t('dashboard.stats.errors')}</p>
          <p class="text-2xl font-bold text-error mt-1">{queueStats.error}</p>
        </div>
        <div class="p-3 rounded-full bg-error/10">
          <AlertTriangle class="w-6 h-6 text-error" />
        </div>
      </div>
      {#if queueStats.error > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-red-500 rounded-full mr-2"></div>
          <span class="text-xs text-error">{$t('dashboard.stats.queue_stats.failed_to_process')}</span>
        </div>
      {/if}
    </div>
  </div>
</div>
