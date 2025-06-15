<script lang="ts">
import { toastStore } from "$lib/stores/toast";
import type { QueueItem } from "$lib/types";
import { formatDate, formatFileSize, getStatusColor } from "$lib/utils";
import * as App from "$lib/wailsjs/go/backend/App";
import { EventsOn } from "$lib/wailsjs/runtime/runtime";
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
	ClockSolid,
	DownloadSolid,
	ExclamationCircleSolid,
	PlaySolid,
	TrashBinSolid,
	XSolid,
} from "flowbite-svelte-icons";
import { onMount } from "svelte";
import { SetQueueItemPriority } from "$lib/wailsjs/go/backend/App";

let queueItems: QueueItem[] = [];

onMount(() => {
	// Listen for queue updates
	EventsOn("queue-updated", () => {
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
		const items = await App.GetQueueItems();
		queueItems = items || [];
	} catch (error) {
		console.error("Failed to load queue:", error);
		toastStore.error("Failed to load queue", String(error));
	}
}

async function removeFromQueue(id: string) {
	try {
		await App.RemoveFromQueue(id);

		queueItems = queueItems.filter((item) => item.id !== id);
		// Immediately refresh the queue to ensure UI updates
		await loadQueue();
		toastStore.success("Item removed", "Item has been removed from the queue");
	} catch (error) {
		console.error("Failed to remove item from queue:", error);
		toastStore.error("Failed to remove item", String(error));
	}
}

async function downloadNZB(id: string, fileName: string) {
	try {
		await App.DownloadNZB(id);
		toastStore.success(
			"NZB Downloaded",
			`NZB file for ${fileName} has been saved`,
		);
	} catch (error) {
		console.error("Failed to download NZB:", error);
		toastStore.error("Failed to download NZB", String(error));
	}
}

async function changePriority(id: string, newPriority: number) {
	try {
		await SetQueueItemPriority(id, newPriority);
		await loadQueue();
	} catch (error) {
		console.error("Failed to update priority:", error);
		toastStore.error("Failed to update priority", String(error));
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
</script>

<Card
  class="max-w-full p-5 backdrop-blur-sm bg-white/60 dark:bg-gray-800/60 border border-gray-200/60 dark:border-gray-700/60 shadow-lg shadow-gray-900/5 dark:shadow-gray-900/20"
>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <Heading
        tag="h2"
        class="text-xl font-semibold text-gray-900 dark:text-white"
      >
        Upload Queue
      </Heading>
      <div
        class="px-3 py-1.5 bg-blue-100 dark:bg-blue-900/30 rounded-full border border-blue-200 dark:border-blue-800"
      >
        <span class="text-sm font-medium text-blue-800 dark:text-blue-200">
          {queueItems.length} items
        </span>
      </div>
    </div>

    {#if queueItems.length === 0}
      <div class="text-center py-16">
        <div
          class="w-20 h-20 mx-auto mb-6 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center"
        >
          <svg
            class="w-10 h-10 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            ></path>
          </svg>
        </div>
        <P class="text-gray-600 dark:text-gray-400 text-lg mb-2 font-medium">
          No items in queue
        </P>
        <P class="text-gray-500 dark:text-gray-500 text-sm">
          Add files to get started with your uploads
        </P>
      </div>
    {:else}
      <div
        class="bg-white/40 dark:bg-gray-800/40 rounded-lg border border-gray-200/40 dark:border-gray-700/40 overflow-hidden"
      >
        <div class="overflow-x-auto">
          <Table hoverable={true} striped={true} class="table-auto">
            <TableHead>
              <TableHeadCell>File</TableHeadCell>
              <TableHeadCell>Size</TableHeadCell>
              <TableHeadCell>Status</TableHeadCell>
              <TableHeadCell>Created</TableHeadCell>
              <TableHeadCell class="text-right">Actions</TableHeadCell>
            </TableHead>
            <TableBody class="divide-y">
              {#each queueItems as item (item.id)}
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
                          Retry {item.retryCount}
                        </Badge>
                      {/if}
                      {#if item.status === "pending"}
                        <div class="flex items-center gap-1 ml-2">
                          <button
                            class="px-1 py-0.5 rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-xs"
                            title="Increase priority"
                            on:click={() => changePriority(item.id, item.priority + 1)}
                          >▲</button>
                          <span class="px-1 text-xs font-mono">{item.priority}</span>
                          <button
                            class="px-1 py-0.5 rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-xs"
                            title="Decrease priority"
                            on:click={() => changePriority(item.id, item.priority - 1)}
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
                        Completed: {formatDate(item.completedAt)}
                      </div>
                    {/if}
                  </TableBodyCell>
                  <TableBodyCell class="text-right">
                    <div class="flex justify-end gap-2">
                      {#if item.status === "complete" && item.nzbPath}
                        <Button
                          size="xs"
                          color="blue"
                          onclick={() => downloadNZB(item.id, item.fileName)}
                          class="cursor-pointer flex items-center gap-1"
                        >
                          <DownloadSolid class="w-3 h-3" />
                          NZB
                        </Button>
                      {:else}
                        <Button
                          size="xs"
                          color="red"
                          variant="outline"
                          onclick={() => removeFromQueue(item.id)}
                          class="cursor-pointer flex items-center gap-1"
                        >
                          <TrashBinSolid class="w-3 h-3" />
                          Remove
                        </Button>
                      {/if}
                    </div>
                  </TableBodyCell>
                </TableBodyRow>
              {/each}
            </TableBody>
          </Table>
        </div>
      </div>
    {/if}
  </div>
</Card>
