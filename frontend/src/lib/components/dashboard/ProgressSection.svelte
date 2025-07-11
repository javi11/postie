<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { isUploading, progress } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { formatSpeed, formatTime } from "$lib/utils";
import { Button, Card, Heading, P, Progressbar } from "flowbite-svelte";
import {
	ChartPieSolid,
	CheckCircleSolid,
	ClockSolid,
	CloseCircleSolid,
	PlaySolid,
} from "flowbite-svelte-icons";

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
      <ChartPieSolid class="w-6 h-6 text-white" />
    </div>
    <div>
      <Heading tag="h2" class="text-xl font-semibold text-gray-900 dark:text-white">
        {$t("dashboard.progress.title")}
      </Heading>
      <div class="flex items-center gap-3 mt-1">
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-50 dark:bg-gray-700">
          <div
            class="w-2 h-2 rounded-full transition-all duration-300 {$isUploading
              ? 'bg-blue-500 animate-pulse shadow-lg shadow-blue-500/50'
              : 'bg-gray-400'}"
          ></div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {$isUploading ? $t("dashboard.progress.status.active") : $t("dashboard.progress.status.idle")}
          </span>
        </div>
      </div>
    </div>
  </div>

  {#if $isUploading}
    {#each jobs as job (job.jobID)}
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm hover:shadow-md transition-all duration-200">
        <div class="space-y-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-full bg-blue-100 dark:bg-blue-900/20">
                <PlaySolid class="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {$t("dashboard.progress.job_title", { jobId: job.jobID })}
                </h3>
                <p class="text-sm text-gray-600 dark:text-gray-400">Active upload in progress</p>
              </div>
            </div>
            <Button
              size="sm"
              color="red"
              variant="outline"
              onclick={() => cancelUpload(job.jobID)}
              class="cursor-pointer flex items-center gap-2 px-3 py-1.5 text-sm font-medium shadow-sm hover:shadow-md transition-all duration-200"
            >
              <CloseCircleSolid class="w-4 h-4" />
              {$t("dashboard.progress.cancel_upload")}
            </Button>
          </div>
          <!-- Overall Progress -->
          <div class="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/10 dark:to-indigo-900/10 border border-blue-200 dark:border-blue-800 rounded-xl p-4">
            <div class="flex justify-between items-center mb-3">
              <p class="text-sm font-medium text-blue-800 dark:text-blue-200">
                {$t("dashboard.progress.overall")}
              </p>
              <div class="text-right">
                <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">
                  {Math.round(job.percentage)}%
                </p>
                <p class="text-xs text-gray-600 dark:text-gray-400">
                  {job.completedFiles}/{job.totalFiles} files
                </p>
              </div>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
              <div 
                class="bg-gradient-to-r from-blue-500 to-indigo-600 h-3 rounded-full transition-all duration-500 ease-out"
                style="width: {job.percentage}%"
              ></div>
            </div>
          </div>
          {#if job.currentFile}
            <!-- Current File Progress -->
            <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4">
              <div class="flex justify-between items-center mb-3">
                <p class="text-sm font-medium text-gray-600 dark:text-gray-400">
                  {$t("dashboard.progress.current_file")}
                </p>
                <span class="text-sm font-semibold text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30 px-2 py-1 rounded-md">
                  {Math.round(job.currentFileProgress)}%
                </span>
              </div>
              <p class="text-sm text-gray-900 dark:text-white font-medium truncate mb-2" title={job.currentFile}>
                {job.currentFile}
              </p>
              <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                <div 
                  class="bg-blue-500 h-2 rounded-full transition-all duration-300"
                  style="width: {job.currentFileProgress}%"
                ></div>
              </div>
            </div>
          {/if}
          <!-- Statistics Grid -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <!-- Elapsed Time -->
            <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm hover:shadow-md transition-all duration-200">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t("dashboard.progress.elapsed_time")}</p>
                  <p class="text-lg font-bold text-green-600 dark:text-green-400 mt-1">{formatTime(job.elapsedTime * 1000)}</p>
                </div>
                <div class="p-3 rounded-full bg-green-100 dark:bg-green-900/20">
                  <ClockSolid class="w-5 h-5 text-green-600 dark:text-green-400" />
                </div>
              </div>
            </div>
            <!-- Upload Speed -->
            <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm hover:shadow-md transition-all duration-200">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t("dashboard.progress.upload_speed")}</p>
                  <p class="text-lg font-bold text-blue-600 dark:text-blue-400 mt-1">{formatSpeed(job.speed * 1024)}</p>
                </div>
                <div class="p-3 rounded-full bg-blue-100 dark:bg-blue-900/20">
                  <PlaySolid class="w-5 h-5 text-blue-600 dark:text-blue-400" />
                </div>
              </div>
            </div>
            <!-- Remaining Time -->
            <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm hover:shadow-md transition-all duration-200">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{$t("dashboard.progress.estimated_remaining")}</p>
                  <p class="text-lg font-bold text-orange-600 dark:text-orange-400 mt-1">{formatTime(job.secondsLeft * 1000)}</p>
                </div>
                <div class="p-3 rounded-full bg-orange-100 dark:bg-orange-900/20">
                  <ClockSolid class="w-5 h-5 text-orange-600 dark:text-orange-400" />
                </div>
              </div>
            </div>
          </div>
          <!-- Status Information -->
          <div class="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-4 space-y-3">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="flex justify-between items-center">
                <span class="text-sm text-gray-600 dark:text-gray-400">
                  {$t("dashboard.progress.stage")}
                </span>
                <span class="text-sm font-medium text-gray-900 dark:text-white">
                  {job.stage}
                </span>
              </div>
              {#if job.details}
                <div class="flex justify-between items-center">
                  <span class="text-sm text-gray-600 dark:text-gray-400">
                    {$t("dashboard.progress.details")}
                  </span>
                  <span class="text-sm font-medium text-gray-900 dark:text-white">
                    {job.details}
                  </span>
                </div>
              {/if}
            </div>
            <div class="flex justify-between items-center pt-2 border-t border-gray-200 dark:border-gray-600">
              <span class="text-sm text-gray-600 dark:text-gray-400">
                {$t("dashboard.progress.last_update")}
              </span>
              <span class="text-sm font-medium text-gray-900 dark:text-white">
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
      <div class="w-16 h-16 mx-auto mb-4 p-4 rounded-full bg-gray-100 dark:bg-gray-800">
        <CheckCircleSolid class="w-8 h-8 text-gray-400 dark:text-gray-600" />
      </div>
      <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
        {$t("dashboard.progress.no_upload_title")}
      </h3>
      <p class="text-gray-600 dark:text-gray-400">
        {$t("dashboard.progress.no_upload_description")}
      </p>
    </div>
  {/if}
</div>
