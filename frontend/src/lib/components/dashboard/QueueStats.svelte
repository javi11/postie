<script lang="ts">
import { t } from "$lib/i18n";
import type { QueueStats } from "$lib/types";
import apiClient from "$lib/api/client";
import { Badge, Card, Heading } from "flowbite-svelte";
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

<Card
  class="max-w-full p-5 backdrop-blur-sm bg-white/60 dark:bg-gray-800/60 border border-gray-200/60 dark:border-gray-700/60 shadow-lg shadow-gray-900/5 dark:shadow-gray-900/20"
>
  <div class="space-y-6">
    <Heading
      tag="h2"
      class="text-xl font-semibold text-gray-900 dark:text-white"
          >
        {$t('dashboard.stats.queue_stats.title')}
      </Heading>

    <div class="grid grid-cols-2 gap-4">
      <!-- Total -->
      <Card
        class="text-center group hover:scale-105 transition-transform duration-200 bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-700 dark:to-gray-800 border-gray-200 dark:border-gray-600 group-hover:shadow-md"
      >
        <div class="text-3xl font-bold text-gray-900 dark:text-white mb-1">
          {queueStats.total}
        </div>
        <div class="text-sm font-medium text-gray-600 dark:text-gray-400">
          {$t('dashboard.stats.queue_stats.total')}
        </div>
      </Card>

      <!-- Pending -->
      <Card
        class="text-center group hover:scale-105 transition-transform duration-200 bg-gradient-to-br from-yellow-50 to-yellow-100 dark:from-yellow-900/20 dark:to-yellow-800/20 border-yellow-200 dark:border-yellow-700 group-hover:shadow-md"
      >
        <div
          class="text-3xl font-bold text-yellow-800 dark:text-yellow-200 mb-1"
        >
          {queueStats.pending}
        </div>
        <div class="text-sm font-medium text-yellow-700 dark:text-yellow-400">
          {$t('dashboard.stats.pending')}
        </div>
      </Card>

      <!-- Running -->
      <Card
        class="text-center group hover:scale-105 transition-transform duration-200 bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-900/20 dark:to-blue-800/20 border-blue-200 dark:border-blue-700 group-hover:shadow-md"
      >
        <div class="text-3xl font-bold text-blue-800 dark:text-blue-200 mb-1">
          {queueStats.running}
        </div>
        <div class="text-sm font-medium text-blue-700 dark:text-blue-400">
          {$t('dashboard.stats.queue_stats.running')}
        </div>
      </Card>

      <!-- Complete -->
      <Card
        class="text-center group hover:scale-105 transition-transform duration-200 bg-gradient-to-br from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20 border-green-200 dark:border-green-700 group-hover:shadow-md"
      >
        <div class="text-3xl font-bold text-green-800 dark:text-green-200 mb-1">
          {queueStats.complete}
        </div>
        <div class="text-sm font-medium text-green-700 dark:text-green-400">
          {$t('dashboard.stats.queue_stats.complete')}
        </div>
      </Card>

      <!-- Errors (only show if there are errors) -->
      {#if queueStats.error > 0}
        <div class="col-span-2">
          <Card
            class="text-center group hover:scale-105 transition-transform duration-200 bg-gradient-to-br from-red-50 to-red-100 dark:from-red-900/20 dark:to-red-800/20 border-red-200 dark:border-red-700 group-hover:shadow-md"
          >
            <div class="text-3xl font-bold text-red-800 dark:text-red-200 mb-1">
              {queueStats.error}
            </div>
            <div class="text-sm font-medium text-red-700 dark:text-red-400">
              {$t('dashboard.stats.errors')}
            </div>
          </Card>
        </div>
      {/if}
    </div>

    {#if queueStats.total > 0}
      <div class="pt-4 border-t border-gray-200/60 dark:border-gray-700/60">
        <Card
          class="bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/10 dark:to-emerald-900/10 border-green-200 dark:border-green-800"
        >
          <div class="flex justify-between items-center">
            <span class="text-sm font-medium text-green-700 dark:text-green-300"
              >{$t('dashboard.stats.queue_stats.success_rate')}</span
            >
            <div class="flex items-center gap-2">
              <div class="w-2 h-2 bg-green-500 rounded-full"></div>
              <span
                class="text-lg font-bold text-green-800 dark:text-green-200"
              >
                {Math.round((queueStats.complete / queueStats.total) * 100)}%
              </span>
            </div>
          </div>
        </Card>
      </div>
    {/if}
  </div>
</Card>
