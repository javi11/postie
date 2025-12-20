<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { uploadActions, uploadStore } from "$lib/stores/upload";
import { isUploading } from "$lib/stores/app";
import { AlertTriangle, Plus, X, Pause, Play, FolderOpen, FolderPlus } from "lucide-svelte";
import { onMount, onDestroy } from "svelte";
import FileExplorerModal from "./FileExplorerModal.svelte";

let { needsConfiguration, criticalConfigError, handleUpload }: {
  needsConfiguration: boolean;
  criticalConfigError: boolean;
  handleUpload: () => Promise<void>;
} = $props();

let isPaused = $state(false);
let isAutoPaused = $state(false);
let autoPauseReason = $state("");
let pauseCheckInterval: NodeJS.Timeout | null = null;
let showFileExplorer = $state(false);
let isWebMode = $state(false);

// Check pause status and auto-pause status (only update if changed to prevent unnecessary re-renders)
async function checkPauseStatus() {
  try {
    const newPaused = await apiClient.isProcessingPaused();
    const newAutoPaused = await apiClient.isProcessingAutoPaused();
    const newReason = await apiClient.getAutoPauseReason();

    if (isPaused !== newPaused) isPaused = newPaused;
    if (isAutoPaused !== newAutoPaused) isAutoPaused = newAutoPaused;
    if (autoPauseReason !== newReason) autoPauseReason = newReason;
  } catch (error) {
    console.error("Failed to check pause status:", error);
  }
}

// Check environment and setup periodic pause status checks
onMount(async () => {
  checkPauseStatus();
  pauseCheckInterval = setInterval(checkPauseStatus, 1000);
  
  // Check if we're in web mode
  await apiClient.initialize();
  isWebMode = apiClient.environment === "web";
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

function openFileExplorer() {
  showFileExplorer = true;
}

function closeFileExplorer() {
  showFileExplorer = false;
}

// Handle folder upload - works in both desktop (Wails) and web modes
async function handleFolderUpload() {
  try {
    if (apiClient.environment === "wails") {
      // Desktop mode: use native folder picker
      const folderPath = await apiClient.selectFolder();
      if (folderPath) {
        await apiClient.uploadFolder(folderPath);
        toastStore.success(
          $t("dashboard.header.folder_added"),
          $t("dashboard.header.folder_added_description")
        );
      }
    } else {
      // Web mode: use hidden input with webkitdirectory attribute
      const input = document.createElement("input");
      input.type = "file";
      // @ts-ignore - webkitdirectory is not in the type definitions
      input.webkitdirectory = true;
      // @ts-ignore
      input.directory = true;
      input.multiple = true;

      input.onchange = async () => {
        if (input.files && input.files.length > 0) {
          // In web mode, we need to send the files to the server
          const fileList = input.files;

          // Use the existing file upload mechanism
          uploadActions.startUpload(fileList);

          try {
            await apiClient.uploadFileList(
              input.files,
              (progress: number) => {
                uploadActions.updateTotalProgress(progress);
              },
              (xhr: XMLHttpRequest) => {
                uploadActions.setCurrentRequest(xhr);
              }
            );

            uploadActions.completeUpload();
            toastStore.success(
              $t("dashboard.header.folder_added"),
              $t("dashboard.header.folder_added_description")
            );
          } catch (error) {
            console.error("Folder upload failed:", error);
            toastStore.error($t("common.common.error"), String(error));
            uploadActions.cancelUpload();
          }
        }
      };

      input.click();
    }
  } catch (error) {
    console.error("Failed to upload folder:", error);
    toastStore.error($t("common.common.error"), String(error));
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

          <!-- Add Folder Button -->
          <button
            class="btn btn-secondary"
            onclick={handleFolderUpload}
            disabled={needsConfiguration}
            title={$t("dashboard.header.add_folder_tooltip")}
          >
            <FolderPlus class="w-4 h-4" />
            {$t("dashboard.header.add_folder")}
          </button>

          <!-- Import Files Button (Web mode only) -->
          {#if isWebMode}
            <button
              class="btn btn-secondary"
              onclick={openFileExplorer}
              disabled={needsConfiguration}
              title={$t("dashboard.header.import_files_tooltip")}
            >
              <FolderOpen class="w-4 h-4" />
              {$t("dashboard.header.import_files")}
            </button>
          {/if}
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
    
    {#if isAutoPaused && autoPauseReason}
      <div class="alert alert-warning mt-6">
        <Pause class="w-5 h-5" />
        <span>
          <span class="font-semibold">{$t("dashboard.alerts.auto_paused")}</span>
          {autoPauseReason}
        </span>
      </div>
    {/if}
  </div>
</div>

<!-- File Explorer Modal -->
<FileExplorerModal 
  isOpen={showFileExplorer} 
  onClose={closeFileExplorer} 
/>
