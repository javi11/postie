<script lang="ts">
import { t } from "$lib/i18n";
import type { QueueStats } from "$lib/types";
import apiClient from "$lib/api/client";
import { Badge, Card, Heading } from "flowbite-svelte";
import {
	ClockSolid,
	PlaySolid,
	CheckCircleSolid,
	ExclamationCircleSolid,
	RectangleListSolid,
	ChartPieSolid,
} from "flowbite-svelte-icons";
import { onMount, onDestroy } from "svelte";

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
      <ChartPieSolid class="w-6 h-6 text-white" />
    </div>
    <div>
      <Heading tag="h2" class="text-xl font-semibold text-gray-900 dark:text-white">
        {$t('dashboard.stats.queue_stats.title')}
      </Heading>
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
          <RectangleListSolid class="w-6 h-6 text-gray-600 dark:text-gray-400" />
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
          <ClockSolid class="w-6 h-6 text-amber-600 dark:text-amber-400" />
        </div>
      </div>
      {#if queueStats.pending > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-amber-500 rounded-full animate-pulse mr-2"></div>
          <span class="text-xs text-amber-600 dark:text-amber-400">{$t('dashboard.stats.queue_stats.waiting_to_process')}</span>
        </div>
      {/if}
    </div>

    <!-- Running -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm hover:shadow-md transition-all duration-200">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t('dashboard.stats.running')}</p>
          <p class="text-2xl font-bold text-blue-600 dark:text-blue-400 mt-1">{queueStats.running}</p>
        </div>
        <div class="p-3 rounded-full bg-blue-100 dark:bg-blue-900/20">
          <PlaySolid class="w-6 h-6 text-blue-600 dark:text-blue-400" />
        </div>
      </div>
      {#if queueStats.running > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-blue-500 rounded-full animate-pulse mr-2"></div>
          <span class="text-xs text-blue-600 dark:text-blue-400">{$t('dashboard.stats.queue_stats.currently_processing')}</span>
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
          <CheckCircleSolid class="w-6 h-6 text-green-600 dark:text-green-400" />
        </div>
      </div>
      {#if queueStats.complete > 0}
        <div class="mt-3 flex items-center">
          <div class="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
          <span class="text-xs text-green-600 dark:text-green-400">{$t('dashboard.stats.queue_stats.successfully_finished')}</span>
        </div>
      {/if}
    </div>
  </div>

  <!-- Error Section (only show if there are errors) -->
  {#if queueStats.error > 0}
    <div class="bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800 rounded-xl p-6">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-full bg-red-100 dark:bg-red-900/20">
            <ExclamationCircleSolid class="w-5 h-5 text-red-600 dark:text-red-400" />
          </div>
          <div>
            <p class="text-sm font-medium text-red-800 dark:text-red-200">{$t('dashboard.stats.queue_stats.errors_detected')}</p>
            <p class="text-xs text-red-600 dark:text-red-400">{$t('dashboard.stats.queue_stats.failed_to_process')}</p>
          </div>
        </div>
        <div class="text-right">
          <p class="text-2xl font-bold text-red-600 dark:text-red-400">{queueStats.error}</p>
          <p class="text-xs text-red-500 dark:text-red-500">{$t('dashboard.stats.queue_stats.failed_items')}</p>
        </div>
      </div>
    </div>
  {/if}

  <!-- Progress Overview -->
  {#if queueStats.total > 0}
    <div class="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/10 dark:to-indigo-900/10 border border-blue-200 dark:border-blue-800 rounded-xl p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{$t('dashboard.stats.queue_stats.queue_progress')}</h3>
          <p class="text-sm text-gray-600 dark:text-gray-400">{$t('dashboard.stats.queue_stats.overall_completion')}</p>
        </div>
        <div class="text-right">
          <p class="text-3xl font-bold text-blue-600 dark:text-blue-400">
            {Math.round((queueStats.complete / queueStats.total) * 100)}%
          </p>
          <p class="text-sm text-gray-600 dark:text-gray-400">{$t('dashboard.stats.complete')}</p>
        </div>
      </div>
      
      <!-- Progress Bar -->
      <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3 mb-4">
        <div 
          class="bg-gradient-to-r from-blue-500 to-indigo-600 h-3 rounded-full transition-all duration-500 ease-out"
          style="width: {queueStats.total > 0 ? (queueStats.complete / queueStats.total) * 100 : 0}%"
        ></div>
      </div>

      <!-- Summary Stats -->
      <div class="grid grid-cols-3 gap-4 text-center">
        <div>
          <p class="text-xl font-bold text-gray-900 dark:text-white">{queueStats.complete}</p>
          <p class="text-xs text-gray-600 dark:text-gray-400">{$t('dashboard.stats.completed')}</p>
        </div>
        <div>
          <p class="text-xl font-bold text-amber-600 dark:text-amber-400">{queueStats.pending + queueStats.running}</p>
          <p class="text-xs text-gray-600 dark:text-gray-400">{$t('dashboard.stats.in_progress')}</p>
        </div>
        <div>
          <p class="text-xl font-bold text-red-600 dark:text-red-400">{queueStats.error}</p>
          <p class="text-xs text-gray-600 dark:text-gray-400">{$t('dashboard.stats.failed')}</p>
        </div>
      </div>
    </div>
  {:else}
    <!-- Empty State -->
    <div class="text-center py-12">
      <div class="w-16 h-16 mx-auto mb-4 p-4 rounded-full bg-gray-100 dark:bg-gray-800">
        <RectangleListSolid class="w-8 h-8 text-gray-400 dark:text-gray-600" />
      </div>
      <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('dashboard.stats.queue_stats.no_items')}</h3>
      <p class="text-gray-600 dark:text-gray-400">{$t('dashboard.stats.queue_stats.upload_files_prompt')}</p>
    </div>
  {/if}
</div>
