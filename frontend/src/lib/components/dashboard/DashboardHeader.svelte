<script lang="ts">
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { uploadStore, uploadActions } from "$lib/stores/upload";
import apiClient from "$lib/api/client";
import { Alert, Button, Card, Heading, P } from "flowbite-svelte";
import {
	CirclePlusSolid,
	CloseCircleSolid,
	ExclamationCircleSolid,
	TrashBinSolid,
} from "flowbite-svelte-icons";

export let needsConfiguration: boolean;
export let criticalConfigError: boolean;
export let handleUpload: () => Promise<void>;

function formatFileSize(bytes: number): string {
	if (bytes === 0) return "0 Bytes";
	const k = 1024;
	const sizes = ["Bytes", "KB", "MB", "GB"];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${Number.parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
}

async function cancelCurrentUpload() {
	try {
		// Cancel the upload (this will abort the XMLHttpRequest)
		uploadActions.cancelUpload();

		// Also try to cancel on server side if possible
		try {
			await apiClient.cancelUpload();
		} catch (serverError) {
			// Server cancel may fail, but that's okay since we already cancelled client-side
			console.warn("Server-side cancel failed:", serverError);
		}

		toastStore.success(
			$t("common.messages.upload_cancelled"),
			"Upload has been cancelled",
		);
	} catch (error) {
		console.error("Failed to cancel upload:", error);
		toastStore.error($t("common.common.error"), String(error));
	}
}

async function clearQueue() {
	try {
		await apiClient.clearQueue();
		toastStore.success(
			$t("common.messages.queue_cleared"),
			$t("common.messages.queue_cleared_description"),
		);
	} catch (error) {
		console.error("Failed to clear queue:", error);
		toastStore.error(
			$t("common.messages.failed_to_clear_queue"),
			String(error),
		);
	}
}

</script>

<Card
  class="max-w-full p-5 backdrop-blur-sm bg-white/60 dark:bg-gray-800/60 border border-gray-200/60 dark:border-gray-700/60 shadow-lg shadow-gray-900/5 dark:shadow-gray-900/20"
>
  <div
    class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6"
  >
    <div class="space-y-2">
      <Heading
        tag="h1"
        class="text-3xl font-bold bg-gradient-to-r from-gray-900 to-gray-600 dark:from-white dark:to-gray-300 bg-clip-text text-transparent"
      >
        {$t("dashboard.header.title")}
      </Heading>
      <P class="text-lg text-gray-600 dark:text-gray-400">
        {$t("dashboard.header.description")}
      </P>
    </div>

    <div class="flex flex-wrap gap-3">
      {#if $uploadStore.isUploading}
        <!-- Upload Progress Indicator -->
        <div class="bg-white rounded-lg border border-gray-200 p-4 min-w-[300px]">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-medium text-gray-900">
              Uploading {$uploadStore.files.length} file{$uploadStore.files.length !== 1 ? 's' : ''}
            </span>
            <div class="flex items-center gap-2">
              <span class="text-sm text-gray-500">
                {Math.round($uploadStore.totalProgress)}%
              </span>
              <Button
                size="xs"
                color="red"
                onclick={cancelCurrentUpload}
                class="cursor-pointer h-6 w-6 flex items-center justify-center"
              >
                <CloseCircleSolid class="w-3 h-3" />
              </Button>
            </div>
          </div>
          
          <!-- Overall Progress Bar -->
          <div class="w-full bg-gray-200 rounded-full h-2 mb-2">
            <div
              class="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style="width: {$uploadStore.totalProgress}%"
            ></div>
          </div>

          <!-- Current File Info -->
          {#if $uploadStore.files.length > 0}
            {@const currentFile = $uploadStore.files.find(f => f.status === 'uploading' || f.status === 'processing') || $uploadStore.files[0]}
            <div class="text-xs text-gray-600 truncate">
              {currentFile.name} ({formatFileSize(currentFile.size)})
            </div>
          {/if}
        </div>
      {:else}
        <!-- Add Files Button -->
        <Button
          color="primary"
          onclick={handleUpload}
          disabled={needsConfiguration}
          class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm hover:shadow-md transition-all duration-200 border-gray-300 dark:border-gray-600"
        >
          <CirclePlusSolid class="w-4 h-4" />
          {$t("dashboard.header.add_files")}
        </Button>
      {/if}

      <Button
        color="red"
        variant="outline"
        onclick={clearQueue}
        class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm"
      >
        <TrashBinSolid class="w-4 h-4" />
        {$t("dashboard.header.clear_completed")}
      </Button>
    </div>
  </div>

  {#if criticalConfigError}
    <Alert color="red" class="mt-6">
      <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
      <span class="font-semibold">{$t("dashboard.alerts.config_error")}</span>
      {$t("dashboard.alerts.config_error_description", { settingsLink: `<a href="/settings" class="font-medium underline hover:no-underline transition-all">${$t("dashboard.alerts.settings_link")}</a>` })}
    </Alert>
  {:else if needsConfiguration}
    <Alert color="yellow" class="mt-6">
      <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
      <span class="font-semibold">{$t("dashboard.alerts.config_required")}</span>
      {$t("dashboard.alerts.config_required_description", { settingsLink: `<a href="/settings" class="font-medium underline hover:no-underline transition-all">${$t("dashboard.alerts.settings_link")}</a>` })}
    </Alert>
  {/if}
</Card>
