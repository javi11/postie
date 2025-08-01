<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { uploadActions, uploadStore } from "$lib/stores/upload";
import { isUploading } from "$lib/stores/app";
import { AlertTriangle, Plus, X, Pause, Play } from "lucide-svelte";
import { onMount, onDestroy } from "svelte";

let { needsConfiguration, criticalConfigError, handleUpload }: {
  needsConfiguration: boolean;
  criticalConfigError: boolean;
  handleUpload: () => Promise<void>;
} = $props();

let isPaused = $state(false);
let pauseCheckInterval: NodeJS.Timeout | null = null;

// Check pause status
async function checkPauseStatus() {
  try {
    isPaused = await apiClient.isProcessingPaused();
  } catch (error) {
    console.error("Failed to check pause status:", error);
  }
}

// Setup periodic pause status checks
onMount(() => {
  checkPauseStatus();
  pauseCheckInterval = setInterval(checkPauseStatus, 1000);
});

onDestroy(() => {
  if (pauseCheckInterval) {
    clearInterval(pauseCheckInterval);
    pauseCheckInterval = null;
  }
});

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

async function pauseProcessing() {
  try {
    await apiClient.pauseProcessing();
    isPaused = true;
    toastStore.success(
      $t("dashboard.progress.paused"),
      $t("dashboard.progress.paused_description")
    );
  } catch (error) {
    console.error("Failed to pause processing:", error);
    toastStore.error($t("common.messages.error"), String(error));
  }
}

async function resumeProcessing() {
  try {
    await apiClient.resumeProcessing();
    isPaused = false;
    toastStore.success(
      $t("dashboard.progress.resumed"),
      $t("dashboard.progress.resumed_description")
    );
  } catch (error) {
    console.error("Failed to resume processing:", error);
    toastStore.error($t("common.messages.error"), String(error));
  }  
}

</script>

<div class="card bg-base-100/60 backdrop-blur-sm border border-base-300/60 shadow-xl animate-fade-in max-w-full">
  <div class="card-body">
    <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6">
      <div class="space-y-2">
        <h1 class="text-3xl font-bold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">
          {$t("dashboard.header.title")}
        </h1>
        <p class="text-lg text-base-content/70">
          {$t("dashboard.header.description")}
        </p>
      </div>

      <div class="flex flex-wrap gap-3">
        {#if $uploadStore.isUploading}
          <!-- Upload Progress Indicator -->
          <div class="bg-base-100 rounded-lg border border-base-300 p-4 min-w-[300px]">
            <div class="flex items-center justify-between mb-2">
              <span class="text-sm font-medium">
                Uploading {$uploadStore.files.length} file{$uploadStore.files.length !== 1 ? 's' : ''}
              </span>
              <div class="flex items-center gap-2">
                <span class="text-sm text-base-content/70">
                  {Math.round($uploadStore.totalProgress)}%
                </span>
                <button
                  class="btn btn-error btn-xs w-6 h-6"
                  onclick={cancelCurrentUpload}
                >
                  <X class="w-5 h-5" />
                </button>
              </div>
            </div>
            
            <!-- Overall Progress Bar -->
            <div class="w-full bg-base-300 rounded-full h-2 mb-2">
              <div
                class="bg-primary h-2 rounded-full transition-all duration-300"
                style="width: {$uploadStore.totalProgress}%"
              ></div>
            </div>

            <!-- Current File Info -->
            {#if $uploadStore.files.length > 0}
              {@const currentFile = $uploadStore.files.find(f => f.status === 'uploading' || f.status === 'processing') || $uploadStore.files[0]}
              <div class="text-xs text-base-content/70 truncate">
                {currentFile.name} ({formatFileSize(currentFile.size)})
              </div>
            {/if}
          </div>
        {:else}
          <!-- Add Files Button -->
          <button
            class="btn btn-primary"
            onclick={handleUpload}
            disabled={needsConfiguration}
          >
            <Plus class="w-4 h-4" />
            {$t("dashboard.header.add_files")}
          </button>
        {/if}

        <!-- Pause/Resume Button - Show when there are queue items or active uploads -->
        <button
          type="button"
          onclick={isPaused ? resumeProcessing : pauseProcessing}
          class="btn {isPaused ? 'btn-warning' : 'btn-success'}"
          title={isPaused ? $t("dashboard.progress.resume_processing") : $t("dashboard.progress.pause_processing")}
        >
          {#if isPaused}
            <Play class="w-4 h-4" />
            {$t("dashboard.progress.resume_processing")}
          {:else}
            <Pause class="w-4 h-4" />
            {$t("dashboard.progress.pause_processing")}
          {/if}
        </button>
      </div>
    </div>

    {#if criticalConfigError}
      <div class="alert alert-error mt-6">
        <AlertTriangle class="w-5 h-5" />
        <span>
          <span class="font-semibold">{$t("dashboard.alerts.config_error")}</span>
          {@html $t("dashboard.alerts.config_error_description", { values: { settingsLink: `<a href="/settings" class="font-medium underline hover:no-underline transition-all">${$t("dashboard.alerts.settings_link")}</a>` } })}
        </span>
      </div>
    {:else if needsConfiguration}
      <div class="alert alert-warning mt-6">
        <AlertTriangle class="w-5 h-5" />
        <span>
          <span class="font-semibold">{$t("dashboard.alerts.config_required")}</span>
          {@html $t("dashboard.alerts.config_required_description", { values: { settingsLink: `<a href="/settings" class="font-medium underline hover:no-underline transition-all">${$t("dashboard.alerts.settings_link")}</a>` } })}
        </span>
      </div>
    {/if}
  </div>
</div>
