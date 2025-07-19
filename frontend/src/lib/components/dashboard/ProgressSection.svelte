<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { isUploading, progress } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { formatSpeed, formatTime } from "$lib/utils";
import { ChartPie, CheckCircle, Clock, Play, X } from "lucide-svelte";

$: jobs = Object.values($progress);

async function cancelJob(jobID: string) {
	try {
		await apiClient.cancelJob(jobID);

		// Immediately remove the job from progress store as a safety measure
		progress.update((jobs) => {
			console.log("Force removing cancelled job from progress:", jobID);
			const { [jobID]: _, ...rest } = jobs;
			return rest;
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
              ? 'bg-primary animate-pulse shadow-lg shadow-primary/50'
              : 'bg-base-content/40'}"
          ></div>
          <span class="text-sm font-medium text-base-content/80">
            {$isUploading ? $t("dashboard.progress.status.active") : $t("dashboard.progress.status.idle")}
          </span>
        </div>
      </div>
    </div>
  </div>

  {#if $isUploading}
    {#each jobs as job (job.jobID)}
      <div class="card bg-base-100 shadow-xl p-6 hover:shadow-2xl transition-all duration-200">
        <div class="space-y-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-full bg-primary/10">
                <Play class="w-5 h-5 text-primary" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-base-content">
                  {$t("dashboard.progress.job_title", { values: { jobId: job.jobID } })}
                </h3>
                <p class="text-sm text-base-content/70">Active upload in progress</p>
              </div>
            </div>
            <button
              type="button"
              onclick={() => cancelUpload(job.jobID)}
              class="btn btn-outline flex items-center gap-2 px-3 py-1.5 text-sm font-medium shadow-sm hover:shadow-md transition-all duration-200"
            >
              <CheckCircle class="w-4 h-4" />
              {$t("dashboard.progress.cancel_upload")}
            </button>
          </div>
          <!-- Overall Progress -->
          <div class="bg-gradient-to-r from-primary/5 to-secondary/5 border border-primary/20 rounded-xl p-4">
            <div class="flex justify-between items-center mb-3">
              <p class="text-sm font-medium text-primary">
                {$t("dashboard.progress.overall")}
              </p>
              <div class="text-right">
                <p class="text-2xl font-bold text-primary">
                  {Math.round(job.percentage)}%
                </p>
                <p class="text-xs text-base-content/60">
                  {job.completedFiles}/{job.totalFiles} files
                </p>
              </div>
            </div>
            <div class="w-full bg-base-300 rounded-full h-3">
              <div 
                class="bg-gradient-to-r from-primary to-secondary h-3 rounded-full transition-all duration-500 ease-out"
                style="width: {job.percentage}%"
              ></div>
            </div>
          </div>
          {#if job.currentFile}
            <!-- Current File Progress -->
            <div class="bg-base-100 rounded-xl border border-base-300 p-4">
              <div class="flex justify-between items-center mb-3">
                <p class="text-sm font-medium text-base-content/70">
                  {$t("dashboard.progress.current_file")}
                </p>
                <span class="text-sm font-semibold text-primary bg-primary/10 px-2 py-1 rounded-md">
                  {Math.round(job.currentFileProgress)}%
                </span>
              </div>
              <p class="text-sm text-base-content font-medium truncate mb-2" title={job.currentFile}>
                {job.currentFile}
              </p>
              <div class="w-full bg-base-300 rounded-full h-2">
                <div 
                  class="bg-primary h-2 rounded-full transition-all duration-300"
                  style="width: {job.currentFileProgress}%"
                ></div>
              </div>
            </div>
          {/if}
          <!-- Statistics Grid -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <!-- Elapsed Time -->
            <div class="bg-base-100 rounded-xl border border-base-300 p-4 shadow-sm hover:shadow-md transition-all duration-200">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-base-content/70">{$t("dashboard.progress.elapsed_time")}</p>
                  <p class="text-lg font-bold text-success mt-1">{formatTime(job.elapsedTime * 1000)}</p>
                </div>
                <div class="p-3 rounded-full bg-success/10">
                  <Clock class="w-5 h-5 text-success" />
                </div>
              </div>
            </div>
            <!-- Upload Speed -->
            <div class="bg-base-100 rounded-xl border border-base-300 p-4 shadow-sm hover:shadow-md transition-all duration-200">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-base-content/70">{$t("dashboard.progress.upload_speed")}</p>
                  <p class="text-lg font-bold text-info mt-1">{formatSpeed(job.speed * 1024)}</p>
                </div>
                <div class="p-3 rounded-full bg-info/10">
                  <Play class="w-5 h-5 text-info" />
                </div>
              </div>
            </div>
            <!-- Remaining Time -->
            <div class="bg-base-100 rounded-xl border border-base-300 p-4 shadow-sm hover:shadow-md transition-all duration-200">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-base-content/70">{$t("dashboard.progress.estimated_remaining")}</p>
                  <p class="text-lg font-bold text-warning mt-1">{formatTime(job.secondsLeft * 1000)}</p>
                </div>
                <div class="p-3 rounded-full bg-warning/10">
                  <Clock class="w-5 h-5 text-warning" />
                </div>
              </div>
            </div>
          </div>
          <!-- Status Information -->
          <div class="bg-base-200/50 rounded-xl p-4 space-y-3">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="flex justify-between items-center">
                <span class="text-sm text-base-content/70">
                  {$t("dashboard.progress.stage")}
                </span>
                <span class="text-sm font-medium text-base-content">
                  {job.stage}
                </span>
              </div>
              {#if job.details}
                <div class="flex justify-between items-center">
                  <span class="text-sm text-base-content/70">
                    {$t("dashboard.progress.details")}
                  </span>
                  <span class="text-sm font-medium text-base-content">
                    {job.details}
                  </span>
                </div>
              {/if}
            </div>
            <div class="flex justify-between items-center pt-2 border-t border-base-300">
              <span class="text-sm text-base-content/70">
                {$t("dashboard.progress.last_update")}
              </span>
              <span class="text-sm font-medium text-base-content">
                {new Date(job.lastUpdate * 1000).toLocaleTimeString()}
              </span>
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
