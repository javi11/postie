<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import type { QueueStats } from "$lib/types";
import {
	AlertTriangle,
	CheckCircle,
	Clock,
	List,
	PieChart,
	Play,
} from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let queueStats: QueueStats = {
	total: 0,
	pending: 0,
	running: 0,
	complete: 0,
	error: 0,
};

let interval: NodeJS.Timeout | null = null;

onMount(async () => {
	// Listen for queue updates
	await apiClient.on("queue-updated", () => {
		loadQueueStats();
	});

	// Load initial stats
	loadQueueStats();

	// Set up periodic refresh
	interval = setInterval(loadQueueStats, 5000);
});

onDestroy(async () => {
	// Clean up event listener
	await apiClient.off("queue-updated");

	// Clear interval
	if (interval) {
		clearInterval(interval);
		interval = null;
	}
});

async function loadQueueStats() {
	try {
		const stats = await apiClient.getQueueStats();
		queueStats = stats as QueueStats;
	} catch (error) {
		console.error("Failed to load queue stats:", error);
	}
}
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center gap-3 mb-6">
    <div class="p-2 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600">
      <PieChart class="w-6 h-6 text-white" />
    </div>
    <div>
      <h2 class="text-xl font-semibold">
        {$t('dashboard.stats.queue_stats.title')}
      </h2>
      <p class="text-sm text-gray-500 dark:text-gray-400">{$t('dashboard.stats.queue_stats.overview')}</p>
    </div>
  </div>

  <!-- Main Stats Grid -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Total -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t('dashboard.stats.queue_stats.total_items')}</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white mt-1">{queueStats.total}</p>
        </div>
        <div class="p-3 rounded-full bg-gray-100 dark:bg-gray-700">
          <List class="w-6 h-6 text-gray-600 dark:text-gray-400" />
        </div>
      </div>
    </div>

    <!-- Pending -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t('dashboard.stats.pending')}</p>
          <p class="text-2xl font-bold text-amber-600 dark:text-amber-400 mt-1">{queueStats.pending}</p>
        </div>
        <div class="p-3 rounded-full bg-amber-100 dark:bg-amber-900/20">
          <Clock class="w-6 h-6 text-amber-600 dark:text-amber-400" />
        </div>
      </div>
      {#if queueStats.pending > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-amber-500 rounded-full animate-pulse mr-2"></div>
          <span class="text-xs text-amber-600 dark:text-amber-400">{$t('dashboard.stats.queue_stats.waiting_to_process')}</span>
        </div>
      {/if}
    </div>

    <!-- Complete -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t('dashboard.stats.complete')}</p>
          <p class="text-2xl font-bold text-green-600 dark:text-green-400 mt-1">{queueStats.complete}</p>
        </div>
        <div class="p-3 rounded-full bg-green-100 dark:bg-green-900/20">
          <CheckCircle class="w-6 h-6 text-green-600 dark:text-green-400" />
        </div>
      </div>
      {#if queueStats.complete > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
          <span class="text-xs text-green-600 dark:text-green-400">{$t('dashboard.stats.queue_stats.successfully_finished')}</span>
        </div>
      {/if}
    </div>

     <!-- Error Section -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t('dashboard.stats.errors')}</p>
          <p class="text-2xl font-bold text-red-600 dark:text-red-400 mt-1">{queueStats.error}</p>
        </div>
        <div class="p-3 rounded-full bg-red-100 dark:bg-red-900/20">
          <AlertTriangle class="w-6 h-6 text-red-600 dark:text-red-400" />
        </div>
      </div>
      {#if queueStats.error > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-red-500 rounded-full mr-2"></div>
          <span class="text-xs text-red-600 dark:text-red-400">{$t('dashboard.stats.queue_stats.failed_to_process')}</span>
        </div>
      {/if}
    </div>
  </div>
</div>
