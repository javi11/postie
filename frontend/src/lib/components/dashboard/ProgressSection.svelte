<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { isUploading, runningJobs } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { formatSpeed, formatTime, formatFileSize } from "$lib/utils";
import { ChartPie, CheckCircle, Play, X, Upload, Package, Check, Pause } from "lucide-svelte";
import { onMount, onDestroy } from "svelte";

// Use the generated types from Wails
let progressUpdateInterval: NodeJS.Timeout | null = null;
let isPaused = $state(false);

// Fetch progress data from API
async function fetchProgressData() {
  try {
    const [jobDetails, pausedStatus] = await Promise.all([
      apiClient.getRunningJobDetails(),
      apiClient.isProcessingPaused()
    ]);
    runningJobs.set(jobDetails);
    isPaused = pausedStatus;
  } catch (error) {
    console.error("Failed to fetch progress data:", error);
  }
}

// Setup periodic progress updates
onMount(() => {
  // Initial fetch
  fetchProgressData();
  
  // Update every 500ms for real-time progress
  progressUpdateInterval = setInterval(fetchProgressData, 500);
});

onDestroy(() => {
  if (progressUpdateInterval) {
    clearInterval(progressUpdateInterval);
    progressUpdateInterval = null;
  }
});

// Function to get icon for progress type
function getProgressIcon(type: string) {
  switch (type) {
    case "uploading":
      return Upload;
    case "par2_generation":
      return Package;
    case "checking":
      return Check;
    default:
      return Play;
  }
}

// Function to get color for progress type
function getProgressColor(type: string) {
  switch (type) {
    case "uploading":
      return "text-blue-500 bg-blue-500/10";
    case "par2_generation":
      return "text-green-500 bg-green-500/10";
    case "checking":
      return "text-yellow-500 bg-yellow-500/10";
    default:
      return "text-primary bg-primary/10";
  }
}

async function cancelJob(jobID: string) {
  try {
    await apiClient.cancelJob(jobID);

    // Immediately remove the job from running jobs store as a safety measure
    runningJobs.update((jobs) => {
      console.log("Force removing cancelled job from running jobs:", jobID);
      const index = jobs.findIndex((job) => job.id === jobID);
      if (index !== -1) {
        jobs.splice(index, 1);
      }
      
      return jobs;
    });

    toastStore.success(
      $t("common.messages.job_cancelled"),
      $t("common.messages.upload_cancelled_description"),
    );
  } catch (error) {
    console.error("Failed to cancel job:", error);
    toastStore.error($t("common.messages.failed_to_cancel"), String(error));
  }
}

async function cancelDirectUpload() {
  try {
    await apiClient.cancelUpload();
    toastStore.success(
      $t("common.messages.upload_cancelled"),
      $t("common.messages.upload_cancelled_description"),
    );
  } catch (error) {
    console.error("Failed to cancel upload:", error);
    toastStore.error(
      $t("common.messages.failed_to_cancel_upload"),
      String(error),
    );
  }
}

function cancelUpload(jobID: string) {
  if (jobID) {
    cancelJob(jobID);
  } else {
    cancelDirectUpload();
  }
}
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center gap-3 mb-6">
    <div class="p-2 rounded-lg bg-gradient-to-br from-green-500 to-blue-600">
      <ChartPie class="w-6 h-6 text-white" />
    </div>
    <div>
      <h2 class="text-xl font-semibold">
        {$t("dashboard.progress.title")}
      </h2>
      <div class="flex items-center gap-3 mt-1">
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-base-300/50">
          <div
            class="w-2 h-2 rounded-full transition-all duration-300 {$isUploading
              ? isPaused ? 'bg-warning animate-pulse shadow-lg shadow-warning/50' : 'bg-primary animate-pulse shadow-lg shadow-primary/50'
              : 'bg-base-content/40'}"
          ></div>
          <span class="text-sm font-medium text-base-content/80">
            {$isUploading 
              ? isPaused 
                ? $t("dashboard.progress.status.paused") 
                : $t("dashboard.progress.status.active") 
              : $t("dashboard.progress.status.idle")}
          </span>
        </div>
      </div>
    </div>
  </div>

  {#if $isUploading}
    <!-- Running Jobs with Progress -->
    {#each $runningJobs as job (job.id)}
      <div class="card bg-base-100 shadow-xl p-6 hover:shadow-2xl transition-all duration-200">
        <div class="space-y-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-full bg-primary/10">
                <Play class="w-5 h-5 text-primary" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-base-content">
                  {job.fileName}
                </h3>
              </div>
            </div>
            <button
              type="button"
              onclick={() => cancelUpload(job.id)}
              class="btn btn-outline btn-sm flex items-center gap-2"
            >
              <X class="w-4 h-4" />
              {$t("dashboard.progress.cancel_upload")}
            </button>
          </div>

          <!-- Individual Progress Indicators -->
          {#if job.progress.length > 0}
            <div class="space-y-4">
              <h4 class="text-md font-medium text-base-content">Active Tasks</h4>
              {#each job.progress as progressState}
                {@const IconComponent = getProgressIcon(progressState?.Type)}
                <div class="bg-base-100 rounded-xl border border-base-300 p-4">
                  <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center gap-3">
                      <div class="p-2 rounded-full {getProgressColor(progressState?.Type)}">
                        <IconComponent class="w-4 h-4" />
                      </div>
                      <div>
                        <div class="flex items-center gap-2">
                          <p class="text-sm font-medium text-base-content">
                            {progressState?.Description || progressState?.Type}
                          </p>
                          <!-- Status indicator based on IsStarted and IsPaused -->
                          <div class="flex items-center gap-1">
                            <div class="w-2 h-2 rounded-full {progressState?.IsPaused 
                              ? 'bg-orange-500 animate-pulse' 
                              : progressState?.IsStarted 
                              ? 'bg-green-500 animate-pulse' 
                              : 'bg-yellow-500'}"></div>
                            <span class="text-xs font-medium {progressState?.IsPaused 
                              ? 'text-orange-600' 
                              : progressState?.IsStarted 
                              ? 'text-green-600' 
                              : 'text-yellow-600'}">
                              {progressState?.IsPaused ? $t("dashboard.progress.task_status.paused") : progressState?.IsStarted ? $t("dashboard.progress.task_status.in_progress") : $t("dashboard.progress.task_status.pending")}
                            </span>
                          </div>
                        </div>
                        <p class="text-xs text-base-content/60 capitalize">
                          {progressState?.Type.replace('_', ' ')}
                        </p>
                      </div>
                    </div>
                    <div class="text-right">
                      <span class="text-sm font-semibold {progressState?.IsPaused ? 'text-orange-600 bg-orange-500/10' : 'text-primary bg-primary/10'} px-2 py-1 rounded-md">
                        {Math.round(progressState?.CurrentPercent * 100 || 0)}%
                      </span>
                    </div>
                  </div>
                  
                  <div class="w-full bg-base-300 rounded-full h-2 mb-3">
                    <div 
                      class="{progressState?.IsPaused ? 'bg-orange-500' : 'bg-primary'} h-2 rounded-full transition-all duration-300"
                      style="width: {progressState?.CurrentPercent * 100 || 0}%"
                    ></div>
                  </div>

                  <!-- Progress Stats -->
                  {#if !progressState?.IsPaused}
                    <div class="grid grid-cols-2 gap-4 text-xs text-base-content/70">
                      <div>
                        <span class="block">Elapsed</span>
                        <span class="font-medium text-base-content">{formatTime((progressState?.SecondsSince || 0) * 1000)}</span>
                      </div>
                      <div>
                        <span class="block">Remaining</span>
                        <span class="font-medium text-base-content">{formatTime((progressState?.SecondsLeft || 0) * 1000)}</span>
                      </div>
                      
                      <!-- Show speed for upload tasks -->
                      {#if (progressState.Type === "uploading" || progressState.Type === "checking") && progressState?.KBsPerSecond}
                        <div>
                          <span class="block">Speed</span>
                          <span class="font-medium text-base-content">{formatSpeed((progressState.KBsPerSecond || 0) * 1024)}</span>
                        </div>
                      {/if}
                      
                      <!-- Hide current/total for par2 generation, show as formatted bytes for uploads -->
                      {#if progressState.Type !== "par2_generation"}
                        <div>
                          <span class="block">Current</span>
                          <span class="font-medium text-base-content">
                              {formatFileSize(progressState.CurrentBytes)}
                          </span>
                        </div>
                        <div>
                          <span class="block">Total</span>
                          <span class="font-medium text-base-content">
                              {formatFileSize(progressState.Max)}
                          </span>
                        </div>
                      {/if}
                    </div>
                  {:else}
                    <!-- Show only current/total for paused tasks -->
                    {#if progressState.Type !== "par2_generation"}
                      <div class="grid grid-cols-2 gap-4 text-xs text-base-content/70">
                        <div>
                          <span class="block">Current</span>
                          <span class="font-medium text-base-content">
                              {formatFileSize(progressState.CurrentBytes)}
                          </span>
                        </div>
                        <div>
                          <span class="block">Total</span>
                          <span class="font-medium text-base-content">
                              {formatFileSize(progressState.Max)}
                          </span>
                        </div>
                      </div>
                    {/if}
                  {/if}
                </div>
              {/each}
            </div>
          {/if}

          <!-- Job Information -->
          <div class="bg-base-200/50 rounded-xl p-4 space-y-3">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="flex justify-between items-center">
                <span class="text-sm text-base-content/70">File Size</span>
                <span class="text-sm font-medium text-base-content">
                  {formatFileSize(job.size)}
                </span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-sm text-base-content/70">Path</span>
                <span class="text-sm font-medium text-base-content truncate" title="{job.path}">
                  {job.path.split('/').pop()}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    {/each}

  {:else}
    <!-- Empty State -->
    <div class="text-center py-12">
      <div class="w-16 h-16 mx-auto mb-4 p-4 rounded-full bg-base-300">
        <CheckCircle class="w-8 h-8 text-base-content/50" />
      </div>
      <h3 class="text-lg font-medium text-base-content mb-2">
        {$t("dashboard.progress.no_upload_title")}
      </h3>
      <p class="text-base-content/70">
        {$t("dashboard.progress.no_upload_description")}
      </p>
    </div>
  {/if}
</div>