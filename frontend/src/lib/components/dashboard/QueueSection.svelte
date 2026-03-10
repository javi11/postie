<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { formatDate, formatFileSize, getStatusBadgeClass, getStatusIconClass } from "$lib/utils";
import type { backend } from "$lib/wailsjs/go/models";
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
	List,
	Play,
	RotateCcw,
	Search,
	Trash2,
	Upload,
} from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let paginatedResult = $state<backend.PaginatedQueueResult | null>(null);
let initialLoad = $state(true);
let debounceTimer: ReturnType<typeof setTimeout> | undefined;

// Pagination state - controlled by server
let currentPage = $state(1);
let itemsPerPage = $state(10);
let sortBy = $state("created");
let sortOrder = $state("desc");
let statusFilter = $state(""); // "" = all, "pending", "complete", "error"
let searchQuery = $state("");

// Derived from server response
let queueItems = $derived(paginatedResult?.items ?? []);
let totalPages = $derived(paginatedResult?.totalPages ?? 0);
let totalItems = $derived(paginatedResult?.totalItems ?? 0);
let startItem = $derived(paginatedResult ? Math.max(1, (currentPage - 1) * itemsPerPage + 1) : 0);
let endItem = $derived(paginatedResult ? Math.min(currentPage * itemsPerPage, totalItems) : 0);
let hasNext = $derived(paginatedResult?.hasNext ?? false);
let hasPrev = $derived(paginatedResult?.hasPrev ?? false);

// Client-side search within current page
let filteredItems = $derived(
	searchQuery.trim()
		? queueItems.filter((item) =>
				item.fileName?.toLowerCase().includes(searchQuery.toLowerCase()),
			)
		: queueItems,
);

let intervalId: ReturnType<typeof setInterval> | undefined;
let pendingRemoveId = $state<string | null>(null);
let selectedIds = $state(new Set<string>());
let pendingBatchDelete = $state(false);
const allSelected = $derived(
	filteredItems.length > 0 && filteredItems.every((item) => selectedIds.has(item.id)),
);

function startPolling() {
	if (intervalId) return;
	intervalId = setInterval(loadQueue, 5000);
}

function stopPolling() {
	if (intervalId) {
		clearInterval(intervalId);
		intervalId = undefined;
	}
}

function handleVisibilityChange() {
	if (document.hidden) {
		stopPolling();
	} else {
		loadQueue();
		startPolling();
	}
}

onMount(() => {
	apiClient.on("queue-updated", () => {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(loadQueue, 100);
	});

	startPolling();
	document.addEventListener("visibilitychange", handleVisibilityChange);

	// Initial load
	loadQueue();
});

onDestroy(() => {
	apiClient.off("queue-updated");
	clearTimeout(debounceTimer);
	stopPolling();
	document.removeEventListener("visibilitychange", handleVisibilityChange);
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
	pendingRemoveId = id;
}

async function confirmRemove() {
	if (!pendingRemoveId) return;
	const id = pendingRemoveId;
	pendingRemoveId = null;

	try {
		await apiClient.removeFromQueue(id);
		await loadQueue();
		toastStore.success(
			$t("common.messages.item_removed"),
			$t("common.messages.item_removed_description"),
		);
	} catch (error) {
		console.error("Failed to remove item from queue:", error);
		toastStore.error($t("common.messages.failed_to_remove_item"), String(error));
	}
}

function cancelRemove() {
	pendingRemoveId = null;
}

function toggleSelectAll() {
	if (allSelected) {
		selectedIds = new Set();
	} else {
		selectedIds = new Set(filteredItems.map((i) => i.id));
	}
}

function toggleItem(id: string) {
	const next = new Set(selectedIds);
	if (next.has(id)) next.delete(id);
	else next.add(id);
	selectedIds = next;
}

async function batchDeleteSelected() {
	pendingBatchDelete = true;
}

async function confirmBatchDelete() {
	const ids = [...selectedIds];
	let successCount = 0;
	for (const id of ids) {
		try {
			await apiClient.removeFromQueue(id);
			successCount++;
		} catch (e) {
			console.error("Failed to remove", id, e);
		}
	}
	selectedIds = new Set();
	pendingBatchDelete = false;
	toastStore.success($t("dashboard.queue.batch_deleted", { values: { count: successCount } }));
	await loadQueue();
}

async function downloadNZB(id: string) {
	try {
		await apiClient.downloadNZB(id);
	} catch (error) {
		console.error("Failed to download NZB:", error);
		toastStore.error($t("common.messages.failed_to_download_nzb"), String(error));
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
		toastStore.error($t("common.messages.failed_to_update_priority"), String(error));
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
		selectedIds = new Set();
		loadQueue();
	}
}

function changeItemsPerPage(event: Event) {
	const target = event.target as HTMLSelectElement;
	itemsPerPage = Number.parseInt(target.value);
	currentPage = 1;
	selectedIds = new Set();
	loadQueue();
}

function toggleSort(column: string) {
	if (sortBy === column) {
		sortOrder = sortOrder === "asc" ? "desc" : "asc";
	} else {
		sortBy = column;
		sortOrder = "desc";
	}
	currentPage = 1;
	selectedIds = new Set();
	loadQueue();
}

function setStatusFilter(value: string) {
	statusFilter = value;
	currentPage = 1;
	selectedIds = new Set();
	loadQueue();
}

async function clearCompleted() {
	try {
		await apiClient.clearQueue();
		await loadQueue();
		toastStore.success(
			$t("common.messages.queue_cleared"),
			$t("common.messages.queue_cleared_description"),
		);
	} catch (error) {
		console.error("Failed to clear queue:", error);
		toastStore.error($t("common.messages.failed_to_clear_queue"), String(error));
	}
}

async function retryAllFailed() {
	// Fetch all error items across all pages (not just current page)
	let allFailedItems: backend.QueueItem[] = [];
	try {
		const result = await apiClient.getQueueItems({
			page: 1,
			limit: 9999,
			sortBy: "created",
			order: "desc",
			status: "error",
		});
		allFailedItems = result?.items ?? [];
	} catch {
		// Fall back to current page items
		allFailedItems = queueItems.filter((item) => item.status === "error");
	}

	if (allFailedItems.length === 0) return;

	let succeeded = 0;
	for (const item of allFailedItems) {
		try {
			await apiClient.retryJob(item.id);
			succeeded++;
		} catch (error) {
			console.error("Failed to retry item:", item.id, error);
		}
	}

	await loadQueue();
	if (succeeded > 0) {
		toastStore.success($t("dashboard.queue.bulk_retry_success", { values: { count: succeeded } }));
	}
}

function getSortIcon(column: string) {
	if (sortBy !== column) {
		return ArrowUpDown;
	}
	return sortOrder === "asc" ? ArrowUp : ArrowDown;
}

let FilenameIcon = $derived(getSortIcon("filename"));
let SizeIcon = $derived(getSortIcon("size"));
let StatusSortIcon = $derived(getSortIcon("status"));
let PriorityIcon = $derived(getSortIcon("priority"));
let CreatedIcon = $derived(getSortIcon("created"));
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center gap-3 mb-4">
    <div class="p-2 rounded-lg bg-gradient-to-br from-secondary to-accent">
      <List class="w-6 h-6 text-secondary-content" />
    </div>
    <div class="flex-1 min-w-0">
      <h2 class="text-xl font-semibold">
        {$t("dashboard.queue.title")}
      </h2>
      <div class="flex items-center gap-2 mt-1">
        <div class="badge badge-info">
          <span class="text-sm font-medium">
            {totalItems} {$t("dashboard.queue.items")}
          </span>
        </div>
      </div>
    </div>
  </div>

  {#if !initialLoad}
    <!-- Filter Tabs + Search + Controls row -->
    <div class="flex flex-wrap items-center gap-3">
      <!-- Status filter tabs -->
      <div class="join">
        <button
          type="button"
          class="btn btn-sm join-item {statusFilter === '' ? 'btn-primary' : 'btn-ghost'}"
          onclick={() => setStatusFilter('')}
        >
          {$t("dashboard.queue.filter_all")}
        </button>
        <button
          type="button"
          class="btn btn-sm join-item {statusFilter === 'pending' ? 'btn-primary' : 'btn-ghost'}"
          onclick={() => setStatusFilter('pending')}
        >
          {$t("dashboard.queue.filter_pending")}
        </button>
        <button
          type="button"
          class="btn btn-sm join-item {statusFilter === 'complete' ? 'btn-primary' : 'btn-ghost'}"
          onclick={() => setStatusFilter('complete')}
        >
          {$t("dashboard.queue.filter_complete")}
        </button>
        <button
          type="button"
          class="btn btn-sm join-item {statusFilter === 'error' ? 'btn-error' : 'btn-ghost'}"
          onclick={() => setStatusFilter('error')}
        >
          {$t("dashboard.queue.filter_error")}
        </button>
      </div>

      <!-- Search input -->
      <div class="relative">
        <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-base-content/50 pointer-events-none" />
        <input
          type="search"
          class="input input-bordered input-sm pl-8 w-44"
          placeholder={$t("dashboard.queue.search_placeholder")}
          bind:value={searchQuery}
        />
      </div>
      {#if searchQuery.trim()}
        <span class="text-xs text-base-content/50 italic">
          {filteredItems.length}/{queueItems.length} — {$t("dashboard.queue.search_within_page")}
        </span>
      {/if}

      <!-- Items per page -->
      <div class="flex items-center gap-2 ml-auto">
        <label for="items-per-page" class="text-sm text-base-content/70 hidden sm:block">
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

      <!-- Bulk Actions -->
      {#if totalItems > 0}
        <div class="flex items-center gap-2">
          {#if selectedIds.size > 0}
            <button
              class="btn btn-xs btn-error gap-1"
              onclick={batchDeleteSelected}
            >
              <Trash2 class="w-3 h-3" />
              {$t("dashboard.queue.delete_selected", { values: { count: selectedIds.size } })}
            </button>
          {/if}
          <button
            class="btn btn-xs btn-warning gap-1"
            onclick={retryAllFailed}
            title={$t("dashboard.queue.retry_all_failed")}
          >
            <RotateCcw class="w-3 h-3" />
            {$t("dashboard.queue.retry_all_failed")}
          </button>
          <button
            class="btn btn-xs btn-ghost gap-1"
            onclick={clearCompleted}
            title={$t("dashboard.header.clear_completed")}
          >
            <Trash2 class="w-3 h-3" />
            {$t("dashboard.header.clear_completed")}
          </button>
        </div>
      {/if}
    </div>
  {/if}

  {#if initialLoad}
    <!-- Loading Skeleton -->
    <div class="card bg-base-100 shadow-xl border border-base-300 overflow-hidden">
      <div class="p-4 space-y-3">
        {#each Array(5) as _}
          <div class="flex items-center gap-4">
            <div class="skeleton h-4 w-48"></div>
            <div class="skeleton h-4 w-16"></div>
            <div class="skeleton h-6 w-20 rounded-full"></div>
            <div class="skeleton h-4 w-12"></div>
            <div class="skeleton h-4 w-24"></div>
            <div class="flex-1"></div>
            <div class="skeleton h-8 w-8 rounded"></div>
          </div>
        {/each}
      </div>
    </div>
  {:else if totalItems === 0}
    <!-- Empty State -->
    <div class="text-center py-12">
      <div class="w-16 h-16 mx-auto mb-4 p-4 rounded-full bg-base-200">
        <Upload class="w-8 h-8 text-base-content/50" />
      </div>
      <h3 class="text-lg font-medium mb-2">
        {$t("dashboard.queue.no_items")}
      </h3>
      <p class="text-base-content/70 mb-4">
        {$t("dashboard.queue.no_items_description")}
      </p>
      <p class="text-sm text-base-content/50">
        {$t("dashboard.queue.empty_hint")}
      </p>
    </div>
  {:else}
    {#snippet verificationBadge(item: backend.QueueItem)}
      {#if item.status === "complete" && item.verificationStatus === "pending_verification"}
        <div class="badge badge-warning badge-sm gap-1 shrink-0">
          <span class="loading loading-spinner loading-xs"></span>
          {$t("dashboard.queue.verification.pending")}
        </div>
      {:else if item.status === "complete" && item.verificationStatus === "verification_failed"}
        <div class="badge badge-error badge-sm shrink-0">
          {$t("dashboard.queue.verification.failed")}
        </div>
      {/if}
    {/snippet}

    <!-- Mobile card layout -->
    <div class="md:hidden space-y-3">
      {#each filteredItems as item (item.id)}
        <div class="card bg-base-100 shadow-sm border border-base-300 p-4">
          <div class="flex items-start justify-between gap-2">
            <div class="flex items-start gap-2 min-w-0 flex-1">
              <input
                type="checkbox"
                class="checkbox checkbox-sm mt-0.5 shrink-0"
                checked={selectedIds.has(item.id)}
                onclick={() => toggleItem(item.id)}
              />
              <div class="min-w-0 flex-1">
                <div class="font-medium truncate" title={item.fileName}>{item.fileName}</div>
                <div class="text-xs text-base-content/60 mt-0.5">{formatFileSize(item.size)}</div>
              </div>
            </div>
            <div class="flex flex-col items-end gap-1 shrink-0">
              <div class="badge {getStatusBadgeClass(item.status)} capitalize">{item.status}</div>
              {@render verificationBadge(item)}
            </div>
          </div>
          {#if item.errorMessage}
            <p class="text-xs text-error mt-2 break-words leading-snug">
              {item.errorMessage.length > 100
                ? item.errorMessage.substring(0, 100) + "…"
                : item.errorMessage}
            </p>
          {/if}
          <div class="flex items-center justify-between mt-3 pt-2 border-t border-base-200">
            <div class="text-xs text-base-content/60">{formatDate(item.createdAt)}</div>
            <div class="flex items-center gap-1">
              {#if item.priority > 0}
                <span class="badge badge-outline badge-xs">P{item.priority}</span>
              {/if}
              {#if item.status === "complete"}
                <button class="btn btn-primary btn-xs" onclick={() => downloadNZB(item.id)} aria-label={$t("dashboard.queue.download_nzb")}>
                  <Download class="w-3 h-3" />
                </button>
              {/if}
              {#if item.status === "error"}
                <button class="btn btn-warning btn-xs relative" onclick={() => retryJob(item.id)} aria-label={$t("dashboard.queue.retry")}>
                  <Play class="w-3 h-3" />
                  {#if item.retryCount > 0}
                    <sup class="absolute -top-1 -right-1 text-[9px] leading-none bg-warning-content text-warning rounded-full w-3.5 h-3.5 flex items-center justify-center font-bold">{item.retryCount}</sup>
                  {/if}
                </button>
              {/if}
              <button class="btn btn-error btn-xs" onclick={() => removeFromQueue(item.id)} aria-label={$t("dashboard.queue.remove_from_queue")}>
                <Trash2 class="w-3 h-3" />
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>

    <!-- Desktop table layout -->
    <div class="card bg-base-100 shadow-xl border border-base-300 overflow-hidden hidden md:block">
        <div class="overflow-x-auto">
          <table class="table table-zebra">
            <thead>
              <tr>
                <th class="w-8">
                  <input
                    type="checkbox"
                    class="checkbox checkbox-sm"
                    checked={allSelected}
                    onclick={toggleSelectAll}
                  />
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("filename")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.file")}
                    <FilenameIcon class="w-4 h-4 {sortBy === 'filename' ? 'text-primary' : 'text-base-content/40'}" />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("size")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.size")}
                    <SizeIcon class="w-4 h-4 {sortBy === 'size' ? 'text-primary' : 'text-base-content/40'}" />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("status")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.status")}
                    <StatusSortIcon class="w-4 h-4 {sortBy === 'status' ? 'text-primary' : 'text-base-content/40'}" />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("priority")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.priority")}
                    <PriorityIcon class="w-4 h-4 {sortBy === 'priority' ? 'text-primary' : 'text-base-content/40'}" />
                  </div>
                </th>
                <th
                  class="cursor-pointer hover:bg-base-200 select-none"
                  onclick={() => toggleSort("created")}
                >
                  <div class="flex items-center gap-1">
                    {$t("dashboard.queue.created")}
                    <CreatedIcon class="w-4 h-4 {sortBy === 'created' ? 'text-primary' : 'text-base-content/40'}" />
                  </div>
                </th>
                <th class="text-right">{$t("dashboard.queue.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {#each filteredItems as item (item.id)}
                <tr>
                  <td>
                    <input
                      type="checkbox"
                      class="checkbox checkbox-sm"
                      checked={selectedIds.has(item.id)}
                      onclick={() => toggleItem(item.id)}
                    />
                  </td>
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
                    <div class="flex items-center gap-2 flex-wrap">
                      {#if item.status === "complete"}
                        <CheckCircle class="w-4 h-4 {getStatusIconClass(item.status)}" />
                      {:else if item.status === "error"}
                        <AlertCircle class="w-4 h-4 {getStatusIconClass(item.status)}" />
                      {:else}
                        <Clock class="w-4 h-4 {getStatusIconClass(item.status)}" />
                      {/if}
                      <div class="badge {getStatusBadgeClass(item.status)} capitalize">
                        {item.status}
                      </div>
                      {@render verificationBadge(item)}
                    </div>
                    {#if item.errorMessage}
                      <p class="text-xs text-error mt-1.5 max-w-xs break-words leading-snug">
                        {item.errorMessage.length > 120
                          ? item.errorMessage.substring(0, 120) + "…"
                          : item.errorMessage}
                      </p>
                    {/if}
                  </td>
                  <td>
                    {#if item.status === "pending"}
                      <div class="flex items-center gap-1">
                        <button
                          class="btn btn-xs btn-success"
                          title={$t("dashboard.queue.increase_priority")}
                          aria-label={$t("dashboard.queue.increase_priority")}
                          onclick={() => changePriority(item.id, item.priority + 1)}
                        >▲</button>
                        <span class="px-2 py-1 text-xs font-mono bg-base-200 rounded min-w-[2rem] text-center">{item.priority}</span>
                        <button
                          class="btn btn-xs btn-outline"
                          title={$t("dashboard.queue.decrease_priority")}
                          aria-label={$t("dashboard.queue.decrease_priority")}
                          onclick={() => changePriority(item.id, Math.max(0, item.priority - 1))}
                          disabled={item.priority <= 0}
                        >▼</button>
                      </div>
                    {:else}
                      <span class="text-sm text-base-content/50">{item.priority}</span>
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
                          aria-label={$t("dashboard.queue.download_nzb")}
                        >
                          <Download class="w-4 h-4" />
                        </button>
                      {/if}
                      {#if item.status === "error"}
                        <button
                          class="btn btn-warning btn-xs relative"
                          onclick={() => retryJob(item.id)}
                          title={$t("dashboard.queue.retry")}
                          aria-label={$t("dashboard.queue.retry")}
                        >
                          <Play class="w-4 h-4" />
                          {#if item.retryCount > 0}
                            <sup class="absolute -top-1 -right-1 text-[9px] leading-none bg-warning-content text-warning rounded-full w-3.5 h-3.5 flex items-center justify-center font-bold">{item.retryCount}</sup>
                          {/if}
                        </button>
                      {/if}
                      <button
                        class="btn btn-error btn-xs"
                        onclick={() => removeFromQueue(item.id)}
                        title={$t("dashboard.queue.remove_from_queue")}
                        aria-label={$t("dashboard.queue.remove_from_queue")}
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
    </div>

    <!-- Pagination Controls (shared between mobile/desktop) -->
    {#if totalPages > 1}
      <div class="flex flex-col sm:flex-row justify-between items-center gap-3 p-4 bg-base-200 rounded-lg mt-3">
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
            <span class="hidden sm:inline">{$t("dashboard.queue.previous")}</span>
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
            <span class="hidden sm:inline">{$t("dashboard.queue.next")}</span>
            <ChevronRight class="w-4 h-4" />
          </button>
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Batch delete confirmation dialog -->
{#if pendingBatchDelete}
  <div class="modal modal-open" role="dialog">
    <div class="modal-box max-w-sm">
      <h3 class="text-lg font-bold">{$t("common.actions.confirm")}</h3>
      <p class="py-4">{$t("dashboard.queue.confirm_delete_selected", { values: { count: selectedIds.size } })}</p>
      <div class="modal-action">
        <button class="btn btn-ghost" onclick={() => pendingBatchDelete = false}>{$t("common.actions.cancel")}</button>
        <button class="btn btn-error" onclick={confirmBatchDelete}>{$t("common.actions.delete")}</button>
      </div>
    </div>
    <button aria-label={$t("common.actions.cancel")} class="modal-backdrop" onclick={() => pendingBatchDelete = false}></button>
  </div>
{/if}

<!-- Remove confirmation dialog -->
{#if pendingRemoveId}
  <div class="modal modal-open">
    <div class="modal-box max-w-sm">
      <h3 class="text-lg font-bold">{$t("common.actions.confirm")}</h3>
      <p class="py-4">{$t("common.messages.confirm_delete")}</p>
      <div class="modal-action">
        <button class="btn btn-ghost" onclick={cancelRemove}>{$t("common.actions.cancel")}</button>
        <button class="btn btn-error" onclick={confirmRemove}>{$t("common.actions.delete")}</button>
      </div>
    </div>
    <button aria-label={$t("common.actions.cancel")} class="modal-backdrop" onclick={cancelRemove}></button>
  </div>
{/if}
