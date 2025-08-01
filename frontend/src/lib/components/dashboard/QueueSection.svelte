<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { formatDate, formatFileSize, getStatusColor } from "$lib/utils";
import type { backend } from "$lib/wailsjs/go/models";
import {
	AlertCircle,
	CheckCircle,
	ChevronLeft,
	ChevronRight,
	Clock,
	Download,
	FileText,
	List,
	Play,
	Trash2,
} from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let queueItems: backend.QueueItem[] = [];

// Pagination state
let currentPage = 1;
let itemsPerPage = 10;

// Computed properties for pagination
$: totalPages = Math.ceil(queueItems.length / itemsPerPage);
$: startIndex = (currentPage - 1) * itemsPerPage;
$: endIndex = startIndex + itemsPerPage;
$: currentPageItems = queueItems.slice(startIndex, endIndex);
$: startItem = queueItems.length === 0 ? 0 : startIndex + 1;
$: endItem = Math.min(endIndex, queueItems.length);

// Reset to first page when queue items change significantly
$: if (queueItems.length > 0 && currentPage > totalPages) {
	currentPage = 1;
}

let intervalId: ReturnType<typeof setInterval> | undefined;

onMount(() => {
	// Listen for queue updates
	apiClient.on("queue-updated", () => {
		loadQueue();
	});

  // Set up periodic refresh for queue
	intervalId = setInterval(() => {
		loadQueue();
	}, 5000);

	// Load initial queue
	loadQueue();
});

onDestroy(() => {
	// Clean up event listener
	apiClient.off("queue-updated");
  if (intervalId) {
    clearInterval(intervalId);
  }
});

async function loadQueue() {
	try {
		const items = await apiClient.getQueueItems();
		queueItems = items || [];
	} catch (error) {
		console.error("Failed to load queue:", error);
		toastStore.error($t("common.messages.failed_to_load_queue"), String(error));
	}
}

async function removeFromQueue(id: string) {
	try {
		await apiClient.removeFromQueue(id);

		queueItems = queueItems.filter((item) => item.id !== id);
		// Immediately refresh the queue to ensure UI updates
		await loadQueue();
		toastStore.success(
			$t("common.messages.item_removed"),
			$t("common.messages.item_removed_description"),
		);
	} catch (error) {
		console.error("Failed to remove item from queue:", error);
		toastStore.error(
			$t("common.messages.failed_to_remove_item"),
			String(error),
		);
	}
}

async function downloadNZB(id: string) {
	try {
		await apiClient.downloadNZB(id);
	} catch (error) {
		console.error("Failed to download NZB:", error);
		toastStore.error(
			$t("common.messages.failed_to_download_nzb"),
			String(error),
		);
	}
}

async function retryJob(id: string) {
	try {
		await apiClient.retryJob(id);
		await loadQueue();
		toastStore.success($t("common.messages.item_retried"));
	} catch (error) {
		console.error("Failed to retry job:", error);
		toastStore.error($t("common.messages.failed_to_retry_item"), String(error));
	}
}

async function changePriority(id: string, newPriority: number) {
	try {
		await apiClient.setQueueItemPriority(id, newPriority);
		await loadQueue();
	} catch (error) {
		console.error("Failed to update priority:", error);
		toastStore.error(
			$t("common.messages.failed_to_update_priority"),
			String(error),
		);
	}
}

function getStatusIcon(status: string) {
	switch (status) {
		case "pending":
			return Clock;
		case "complete":
			return CheckCircle;
		case "error":
			return AlertCircle;
		default:
			return Clock;
	}
}

function goToPage(page: number) {
	if (page >= 1 && page <= totalPages) {
		currentPage = page;
	}
}

function changeItemsPerPage(event: Event) {
	const target = event.target as HTMLSelectElement;
	itemsPerPage = Number.parseInt(target.value);
	currentPage = 1; // Reset to first page when changing items per page
}
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center gap-3 mb-6">
    <div class="p-2 rounded-lg bg-gradient-to-br from-purple-500 to-pink-600">
      <List class="w-6 h-6 text-white" />
    </div>
    <div class="flex-1">
      <h2 class="text-xl font-semibold">
        {$t("dashboard.queue.title")}
      </h2>
      <div class="flex items-center gap-3 mt-1">
        <div class="badge badge-info">
          <span class="text-sm font-medium">
            {queueItems.length} {$t("dashboard.queue.items")}
          </span>
        </div>
        {#if queueItems.length > 0}
          <div class="flex items-center gap-2">
            <label for="items-per-page" class="text-sm text-base-content/70">
              {$t("dashboard.queue.items_per_page")}:
            </label>
            <select
              id="items-per-page"
              class="select select-bordered select-sm"
              value={itemsPerPage}
              onchange={changeItemsPerPage}
            >
              <option value={5}>5</option>
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
            </select>
          </div>
        {/if}
      </div>
    </div>
  </div>

  {#if queueItems.length === 0}
    <!-- Empty State -->
    <div class="text-center py-12">
      <div class="w-16 h-16 mx-auto mb-4 p-4 rounded-full bg-base-200">
        <FileText class="w-8 h-8 text-base-content/50" />
      </div>
      <h3 class="text-lg font-medium mb-2">
        {$t("dashboard.queue.no_items")}
      </h3>
      <p class="text-base-content/70">
        {$t("dashboard.queue.no_items_description")}
      </p>
    </div>
  {:else}
    <div class="card bg-base-100 shadow-xl border border-base-300 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="table table-zebra">
            <thead>
              <tr>
                <th>{$t("dashboard.queue.file")}</th>
                <th>{$t("dashboard.queue.size")}</th>
                <th>{$t("dashboard.queue.status")}</th>
                <th>{$t("dashboard.queue.priority")}</th>
                <th>{$t("dashboard.queue.created")}</th>
                <th class="text-right">{$t("dashboard.queue.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {#each currentPageItems as item (item.id)}
                <tr>
                  <td>
                    <div class="max-w-xs">
                      <div class="font-medium truncate" title={item.fileName}>
                        {item.fileName}
                      </div>
                      <div class="text-sm text-base-content/70 truncate mt-1" title={item.path}>
                        {item.path}
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="text-sm font-medium">
                      {formatFileSize(item.size)}
                    </span>
                  </td>
                  <td>
                    <div class="flex items-center gap-2">
                      <svelte:component
                        this={getStatusIcon(item.status)}
                        class="w-4 h-4 text-{getStatusColor(item.status)}-600"
                      />
                      <div class="badge badge-{getStatusColor(item.status)} capitalize">
                        {item.status}
                      </div>
                      {#if item.retryCount > 0}
                        <div class="badge badge-warning badge-sm">
                          {$t("dashboard.queue.retry")} {item.retryCount}
                        </div>
                      {/if}
                    </div>
                    {#if item.errorMessage}
                      <div class="alert alert-error alert-sm mt-2" title={item.errorMessage}>
                        <span class="text-xs">
                          {item.errorMessage.length > 50
                            ? item.errorMessage.substring(0, 50) + "..."
                            : item.errorMessage}
                        </span>
                      </div>
                    {/if}
                  </td>
                  <td>
                    {#if item.status === "pending"}
                      <div class="flex items-center gap-1">
                        <button
                          class="btn btn-xs btn-success"
                          title={$t("dashboard.queue.increase_priority")}
                          onclick={() => changePriority(item.id, item.priority + 1)}
                        >▲</button>
                        <span class="px-2 py-1 text-xs font-mono bg-base-200 rounded min-w-[2rem] text-center">{item.priority}</span>
                        <button
                          class="btn btn-xs btn-outline"
                          title={$t("dashboard.queue.decrease_priority")}
                          onclick={() => changePriority(item.id, Math.max(0, item.priority - 1))}
                          disabled={item.priority <= 0}
                        >▼</button>
                      </div>
                    {:else}
                      <span class="text-xs text-base-content/50">—</span>
                    {/if}
                  </td>
                  <td>
                    <div class="text-sm font-medium">
                      {formatDate(item.createdAt)}
                    </div>
                    {#if item.completedAt}
                      <div class="text-xs text-base-content/70 mt-1">
                        {$t("dashboard.queue.completed")}: {formatDate(item.completedAt)}
                      </div>
                    {/if}
                  </td>
                  <td class="text-right">
                    <div class="flex items-center justify-end space-x-2">
                      {#if item.status === "complete"}
                        <button
                          class="btn btn-primary btn-xs"
                          onclick={() => downloadNZB(item.id)}
                          title={$t("dashboard.queue.download_nzb")}
                        >
                          <Download class="w-4 h-4" />
                        </button>
                      {/if}
                      {#if item.status === "error"}
                        <button
                          class="btn btn-warning btn-xs"
                          onclick={() => retryJob(item.id)}
                          title={$t("dashboard.queue.retry")}
                        >
                          <Play class="w-4 h-4" />
                        </button>
                      {/if}
                      <button
                        class="btn btn-error btn-xs"
                        onclick={() => removeFromQueue(item.id)}
                        title={$t("dashboard.queue.remove_from_queue")}
                      >
                        <Trash2 class="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Pagination Controls -->
        {#if totalPages > 1}
        <div class="card-actions justify-between p-4 bg-base-200 border-t border-base-300">
            <div class="text-sm text-base-content/70">
              {$t("dashboard.queue.showing")} {startItem} {$t("dashboard.queue.to")} {endItem} {$t("dashboard.queue.of")} {queueItems.length} {$t("dashboard.queue.entries")}
            </div>
            
            <div class="flex items-center space-x-2">
              <button
                class="btn btn-sm"
                disabled={currentPage === 1}
                onclick={() => goToPage(currentPage - 1)}
              >
                <ChevronLeft class="w-4 h-4" />
                {$t("dashboard.queue.previous")}
              </button>
              
              <div class="flex items-center space-x-1">
                {#each Array.from({ length: Math.min(7, totalPages) }, (_, i) => {
                  if (totalPages <= 7) return i + 1;
                  if (currentPage <= 4) return i + 1;
                  if (currentPage >= totalPages - 3) return totalPages - 6 + i;
                  return currentPage - 3 + i;
                }) as page}
                  <button
                    class="btn btn-sm {currentPage === page ? 'btn-primary' : 'btn-outline'}"
                    onclick={() => goToPage(page)}
                  >
                    {page}
                  </button>
                {/each}
                
                {#if totalPages > 7 && currentPage < totalPages - 3}
                  <span class="text-base-content/50">...</span>
                  <button
                    class="btn btn-sm btn-outline"
                    onclick={() => goToPage(totalPages)}
                  >
                    {totalPages}
                  </button>
                {/if}
              </div>
              
              <button
                class="btn btn-sm"
                disabled={currentPage === totalPages}
                onclick={() => goToPage(currentPage + 1)}
              >
                {$t("dashboard.queue.next")}
                <ChevronRight class="w-4 h-4" />
              </button>
            </div>
          </div>
        {/if}
    </div>
  {/if}
</div>
