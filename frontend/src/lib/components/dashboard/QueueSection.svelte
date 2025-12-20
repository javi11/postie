<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { formatDate, formatFileSize, getStatusColor } from "$lib/utils";
import type { backend } from "$lib/wailsjs/go/models";
// Using backend types for pagination
import {
	AlertCircle,
	ArrowDown,
	ArrowUp,
	ArrowUpDown,
	CheckCircle,
	ChevronLeft,
	ChevronRight,
	Clock,
	Download,
	FileText,
	Filter,
	List,
	Play,
	Trash2,
} from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let paginatedResult: backend.PaginatedQueueResult | null = null;
let initialLoad = true;
let debounceTimer: ReturnType<typeof setTimeout> | undefined;

// Pagination state - now controlled by server
let currentPage = 1;
let itemsPerPage = 10;
let sortBy = "created";
let sortOrder = "desc";
let statusFilter = ""; // "" = all, "pending", "complete", "error"

// Computed properties for pagination - now from server response
$: queueItems = paginatedResult?.items || [];
$: totalPages = paginatedResult?.totalPages || 0;
$: totalItems = paginatedResult?.totalItems || 0;
$: startItem = paginatedResult ? Math.max(1, (currentPage - 1) * itemsPerPage + 1) : 0;
$: endItem = paginatedResult ? Math.min(currentPage * itemsPerPage, totalItems) : 0;
$: hasNext = paginatedResult?.hasNext || false;
$: hasPrev = paginatedResult?.hasPrev || false;

let intervalId: ReturnType<typeof setInterval> | undefined;

// Reactive statements to reload queue when pagination parameters change
$: if (currentPage || itemsPerPage || sortBy || sortOrder || statusFilter !== undefined) {
	loadQueue();
}

onMount(() => {
	// Listen for queue updates with debouncing to prevent double-fetches
	apiClient.on("queue-updated", () => {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(loadQueue, 100);
	});

  // Set up periodic refresh for queue
	intervalId = setInterval(() => {
		loadQueue();
	}, 5000);

	// Load initial queue
	loadQueue();
});

onDestroy(() => {
	// Clean up event listener and timers
	apiClient.off("queue-updated");
	clearTimeout(debounceTimer);
	if (intervalId) {
		clearInterval(intervalId);
	}
});

async function loadQueue() {
	try {
		const result = await apiClient.getQueueItems({
			page: currentPage,
			limit: itemsPerPage,
			sortBy: sortBy,
			order: sortOrder,
			status: statusFilter,
		});
		paginatedResult = result;
	} catch (error) {
		console.error("Failed to load queue:", error);
		toastStore.error($t("common.messages.failed_to_load_queue"), String(error));
		paginatedResult = null;
	} finally {
		initialLoad = false;
	}
}

async function removeFromQueue(id: string) {
	try {
		await apiClient.removeFromQueue(id);

		// Refresh the queue to get updated data
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

function toggleSort(column: string) {
	if (sortBy === column) {
		sortOrder = sortOrder === "asc" ? "desc" : "asc";
	} else {
		sortBy = column;
		sortOrder = "desc";
	}
	currentPage = 1; // Reset to first page when changing sort
}

function changeStatusFilter(event: Event) {
	const target = event.target as HTMLSelectElement;
	statusFilter = target.value;
	currentPage = 1; // Reset to first page when changing filter
}

function getSortIcon(column: string) {
	if (sortBy !== column) {
		return ArrowUpDown;
	}
	return sortOrder === "asc" ? ArrowUp : ArrowDown;
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
      <div class="flex flex-wrap items-center gap-3 mt-1">
        <div class="badge badge-info">
          <span class="text-sm font-medium">
            {totalItems} {$t("dashboard.queue.items")}
          </span>
        </div>
        {#if !initialLoad}
          <!-- Status Filter -->
          <div class="flex items-center gap-2">
            <Filter class="w-4 h-4 text-base-content/70" />
            <select
              id="status-filter"
              class="select select-bordered select-sm"
              value={statusFilter}
              onchange={changeStatusFilter}
            >
              <option value="">{$t("dashboard.queue.filter_all")}</option>
              <option value="pending">{$t("dashboard.queue.filter_pending")}</option>
              <option value="complete">{$t("dashboard.queue.filter_complete")}</option>
              <option value="error">{$t("dashboard.queue.filter_error")}</option>
            </select>
          </div>
          <!-- Items per page -->
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

  {#if initialLoad}
    <!-- Loading State -->
    <div class="text-center py-12">
      <div class="loading loading-spinner loading-lg mb-4"></div>
      <h3 class="text-lg font-medium mb-2">Loading queue...</h3>
    </div>
  {:else if totalItems === 0}
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
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("filename")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.file")}
                    <svelte:component
                      this={getSortIcon("filename")}
                      class="w-4 h-4 {sortBy === 'filename' ? 'text-primary' : 'text-base-content/40'}"
                    />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("size")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.size")}
                    <svelte:component
                      this={getSortIcon("size")}
                      class="w-4 h-4 {sortBy === 'size' ? 'text-primary' : 'text-base-content/40'}"
                    />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("status")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.status")}
                    <svelte:component
                      this={getSortIcon("status")}
                      class="w-4 h-4 {sortBy === 'status' ? 'text-primary' : 'text-base-content/40'}"
                    />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("priority")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.priority")}
                    <svelte:component
                      this={getSortIcon("priority")}
                      class="w-4 h-4 {sortBy === 'priority' ? 'text-primary' : 'text-base-content/40'}"
                    />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("created")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.created")}
                    <svelte:component
                      this={getSortIcon("created")}
                      class="w-4 h-4 {sortBy === 'created' ? 'text-primary' : 'text-base-content/40'}"
                    />
                  </div>
                </th>
                <th class="text-right">{$t("dashboard.queue.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {#each queueItems as item (item.id)}
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
              {$t("dashboard.queue.showing")} {startItem} {$t("dashboard.queue.to")} {endItem} {$t("dashboard.queue.of")} {totalItems} {$t("dashboard.queue.entries")}
            </div>
            
            <div class="flex items-center space-x-2">
              <button
                class="btn btn-sm"
                disabled={!hasPrev}
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
                disabled={!hasNext}
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
