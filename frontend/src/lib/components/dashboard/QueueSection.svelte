<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { QueueItem } from "$lib/types";
import { formatDate, formatFileSize, getStatusColor } from "$lib/utils";
import {
	Badge,
	Button,
	Card,
	Heading,
	P,
	Table,
	TableBody,
	TableBodyCell,
	TableBodyRow,
	TableHead,
	TableHeadCell,
} from "flowbite-svelte";
import {
	CheckCircleSolid,
	ChevronDoubleLeftOutline,
	ChevronDoubleRightOutline,
	ClockSolid,
	DownloadSolid,
	ExclamationCircleSolid,
	ListOutline,
	PlaySolid,
	RectangleListSolid,
	TrashBinSolid,
	XSolid,
} from "flowbite-svelte-icons";
import { onMount } from "svelte";

let queueItems: QueueItem[] = [];

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

onMount(async () => {
	// Initialize API client
	await apiClient.initialize();

	// Listen for queue updates
	await apiClient.on("queue-updated", () => {
		loadQueue();
	});

	// Load initial queue
	loadQueue();

	// Set up periodic refresh
	const interval = setInterval(loadQueue, 2000);

	return () => clearInterval(interval);
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

async function downloadNZB(id: string, fileName: string) {
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
			return ClockSolid;
		case "complete":
			return CheckCircleSolid;
		case "error":
			return ExclamationCircleSolid;
		default:
			return ClockSolid;
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
      <ListOutline class="w-6 h-6 text-white" />
    </div>
    <div class="flex-1">
      <Heading tag="h2" class="text-xl font-semibold text-gray-900 dark:text-white">
        {$t("dashboard.queue.title")}
      </Heading>
      <div class="flex items-center gap-3 mt-1">
        <div class="px-3 py-1.5 bg-blue-100 dark:bg-blue-900/30 rounded-full border border-blue-200 dark:border-blue-800">
          <span class="text-sm font-medium text-blue-800 dark:text-blue-200">
            {queueItems.length} {$t("dashboard.queue.items")}
          </span>
        </div>
        {#if queueItems.length > 0}
          <div class="flex items-center gap-2">
            <label for="items-per-page" class="text-sm text-gray-600 dark:text-gray-400">
              {$t("dashboard.queue.items_per_page")}:
            </label>
            <select
              id="items-per-page"
              class="px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
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
      <div class="w-16 h-16 mx-auto mb-4 p-4 rounded-full bg-gray-100 dark:bg-gray-800">
        <RectangleListSolid class="w-8 h-8 text-gray-400 dark:text-gray-600" />
      </div>
      <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
        {$t("dashboard.queue.no_items")}
      </h3>
      <p class="text-gray-600 dark:text-gray-400">
        {$t("dashboard.queue.no_items_description")}
      </p>
    </div>
  {:else}
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
        <div class="overflow-x-auto">
          <Table hoverable={true} striped={true} class="table-auto">
            <TableHead>
              <TableHeadCell>{$t("dashboard.queue.file")}</TableHeadCell>
              <TableHeadCell>{$t("dashboard.queue.size")}</TableHeadCell>
              <TableHeadCell>{$t("dashboard.queue.status")}</TableHeadCell>
              <TableHeadCell>{$t("dashboard.queue.created")}</TableHeadCell>
              <TableHeadCell class="text-right">{$t("dashboard.queue.actions")}</TableHeadCell>
            </TableHead>
            <TableBody class="divide-y">
              {#each currentPageItems as item (item.id)}
                <TableBodyRow>
                  <TableBodyCell>
                    <div class="max-w-xs">
                      <div
                        class="font-medium text-gray-900 dark:text-white truncate"
                        title={item.fileName}
                      >
                        {item.fileName}
                      </div>
                      <div
                        class="text-sm text-gray-500 dark:text-gray-400 truncate mt-1"
                        title={item.path}
                      >
                        {item.path}
                      </div>
                    </div>
                  </TableBodyCell>
                  <TableBodyCell>
                    <span
                      class="text-sm font-medium text-gray-900 dark:text-white"
                    >
                      {formatFileSize(item.size)}
                    </span>
                  </TableBodyCell>
                  <TableBodyCell>
                    <div class="flex items-center gap-2">
                      <svelte:component
                        this={getStatusIcon(item.status)}
                        class="w-4 h-4 text-{getStatusColor(item.status)}-600"
                      />
                      <Badge
                        color={getStatusColor(item.status)}
                        border={true}
                        class="capitalize"
                      >
                        {item.status}
                      </Badge>
                      {#if item.retryCount > 0}
                        <Badge color="gray" border={true}>
                          {$t("dashboard.queue.retry")} {item.retryCount}
                        </Badge>
                      {/if}
                      {#if item.status === "pending"}
                        <div class="flex items-center gap-1 ml-2">
                          <button
                            class="px-1 py-0.5 rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-xs"
                            title={$t("dashboard.queue.increase_priority")}
                            onclick={() => changePriority(item.id, item.priority + 1)}
                          >▲</button>
                          <span class="px-1 text-xs font-mono">{item.priority}</span>
                          <button
                            class="px-1 py-0.5 rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-xs"
                            title={$t("dashboard.queue.decrease_priority")}
                            onclick={() => changePriority(item.id, item.priority - 1)}
                            disabled={item.priority <= 0}
                          >▼</button>
                        </div>
                      {/if}
                    </div>
                    {#if item.errorMessage}
                      <div
                        class="text-xs text-red-600 dark:text-red-400 mt-2 p-2 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800"
                        title={item.errorMessage}
                      >
                        {item.errorMessage.length > 50
                          ? item.errorMessage.substring(0, 50) + "..."
                          : item.errorMessage}
                      </div>
                    {/if}
                  </TableBodyCell>
                  <TableBodyCell>
                    <div
                      class="text-sm font-medium text-gray-900 dark:text-white"
                    >
                      {formatDate(item.createdAt)}
                    </div>
                    {#if item.completedAt}
                      <div
                        class="text-xs text-gray-500 dark:text-gray-400 mt-1"
                      >
                        {$t("dashboard.queue.completed")}: {formatDate(item.completedAt)}
                      </div>
                    {/if}
                  </TableBodyCell>
                  <TableBodyCell class="text-right">
                    <div class="flex items-center justify-end space-x-2">
                      {#if item.status === "complete"}
                        <Button
                          size="xs"
                          class="cursor-pointer"
                          color="blue"
                          onclick={() => downloadNZB(item.id, item.fileName)}
                          title={$t("dashboard.queue.download_nzb")}
                        >
                          <DownloadSolid class="w-4 h-4" />
                        </Button>
                      {/if}
                      {#if item.status === "error"}
                        <Button
                          class="cursor-pointer"
                          size="xs"
                          color="yellow"
                          onclick={() => retryJob(item.id)}
                          title={$t("dashboard.queue.retry")}
                        >
                          <PlaySolid class="w-4 h-4" />
                        </Button>
                      {/if}

                      <Button
                        class="cursor-pointer"
                        size="xs"
                        color="red"
                        onclick={() => removeFromQueue(item.id)}
                        title={$t("dashboard.queue.remove_from_queue")}
                      >
                        <TrashBinSolid class="w-4 h-4" />
                      </Button>
                    </div>
                  </TableBodyCell>
                </TableBodyRow>
              {/each}
            </TableBody>
          </Table>
        </div>

        <!-- Pagination Controls -->
        {#if totalPages > 1}
        <div class="px-6 py-4 bg-gray-50 dark:bg-gray-700/50 border-t border-gray-200 dark:border-gray-600">
            <div class="flex items-center justify-between">
              <div class="text-sm text-gray-700 dark:text-gray-300">
                {$t("dashboard.queue.showing")} {startItem} {$t("dashboard.queue.to")} {endItem} {$t("dashboard.queue.of")} {queueItems.length} {$t("dashboard.queue.entries")}
              </div>
              
              <div class="flex items-center space-x-2">
                <Button
                  size="sm"
                  color="light"
                  disabled={currentPage === 1}
                  onclick={() => goToPage(currentPage - 1)}
                  class="flex items-center gap-1 cursor-pointer"
                >
                  <ChevronDoubleLeftOutline class="w-4 h-4" />
                  {$t("dashboard.queue.previous")}
                </Button>
                
                <div class="flex items-center space-x-1">
                  {#each Array.from({ length: Math.min(7, totalPages) }, (_, i) => {
                    if (totalPages <= 7) return i + 1;
                    if (currentPage <= 4) return i + 1;
                    if (currentPage >= totalPages - 3) return totalPages - 6 + i;
                    return currentPage - 3 + i;
                  }) as page}
                    <Button
                      size="sm"
                      color={currentPage === page ? "blue" : "light"}
                      onclick={() => goToPage(page)}
                      class="w-8 h-8 p-0 flex items-center justify-center cursor-pointer"
                    >
                      {page}
                    </Button>
                  {/each}
                  
                  {#if totalPages > 7 && currentPage < totalPages - 3}
                    <span class="text-gray-500 dark:text-gray-400">...</span>
                    <Button
                      size="sm"
                      color="light"
                      onclick={() => goToPage(totalPages)}
                      class="w-8 h-8 p-0 flex items-center justify-center cursor-pointer"
                    >
                      {totalPages}
                    </Button>
                  {/if}
                </div>
                
                <Button
                  size="sm"
                  color="light"
                  disabled={currentPage === totalPages}
                  onclick={() => goToPage(currentPage + 1)}
                  class="flex items-center gap-1 cursor-pointer"
                >
                  {$t("dashboard.queue.next")}
                  <ChevronDoubleRightOutline class="w-4 h-4" />
                </Button>
              </div>
            </div>
          </div>
        {/if}
    </div>
  {/if}
</div>
