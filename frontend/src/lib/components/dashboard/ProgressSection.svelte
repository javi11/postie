<script lang="ts">
import { isUploading, progress } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { formatSpeed, formatTime } from "$lib/utils";
import { CancelJob, CancelUpload } from "$lib/wailsjs/go/backend/App";
import { Button, Card, Heading, P, Progressbar } from "flowbite-svelte";
import { CloseCircleSolid } from "flowbite-svelte-icons";

$: jobs = Object.values($progress);

async function cancelJob(jobID: string) {
	try {
		await CancelJob(jobID);
		toastStore.success(
			"Job cancelled",
			"Upload has been cancelled successfully",
		);
	} catch (error) {
		console.error("Failed to cancel job:", error);
		toastStore.error("Failed to cancel", String(error));
	}
}

async function cancelDirectUpload() {
	try {
		await CancelUpload();
		toastStore.success(
			"Upload cancelled",
			"Upload has been cancelled successfully",
		);
	} catch (error) {
		console.error("Failed to cancel upload:", error);
		toastStore.error("Failed to cancel upload", String(error));
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

<Card
  class="max-w-full p-5 backdrop-blur-sm bg-white/60 dark:bg-gray-800/60 border border-gray-200/60 dark:border-gray-700/60 shadow-lg shadow-gray-900/5 dark:shadow-gray-900/20"
>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <Heading
        tag="h2"
        class="text-xl font-semibold text-gray-900 dark:text-white"
      >
        Upload Progress
      </Heading>
      <div class="flex items-center gap-3">
        <div
          class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-50 dark:bg-gray-700"
        >
          <div
            class="w-2 h-2 rounded-full transition-all duration-300 {$isUploading
              ? 'bg-blue-500 animate-pulse shadow-lg shadow-blue-500/50'
              : 'bg-gray-400'}"
          ></div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {$isUploading ? "Active" : "Idle"}
          </span>
        </div>
      </div>
    </div>

    {#if $isUploading}
      {#each jobs as job (job.jobID)}
        <Card
          class="max-w-full p-5 mb-6 backdrop-blur-sm bg-white/60 dark:bg-gray-800/60 border border-gray-200/60 dark:border-gray-700/60 shadow-lg shadow-gray-900/5 dark:shadow-gray-900/20"
        >
          <div class="space-y-6">
            <div class="flex items-center justify-between">
              <Heading
                tag="h2"
                class="text-xl font-semibold text-gray-900 dark:text-white"
              >
                Upload Progress (Job {job.jobID})
              </Heading>
              <Button
                size="sm"
                color="red"
                variant="outline"
                onclick={() => cancelUpload(job.jobID)}
                class="cursor-pointer flex items-center gap-2 px-3 py-1.5 text-sm font-medium shadow-sm hover:shadow-md transition-all duration-200"
              >
                <CloseCircleSolid class="w-4 h-4" />
                Cancel Upload
              </Button>
            </div>
            <!-- Overall Progress -->
            <div class="space-y-3">
              <div class="flex justify-between items-center">
                <P class="text-sm font-medium text-gray-800 dark:text-gray-200"
                  >Overall Progress</P
                >
                <span
                  class="text-sm font-semibold text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded-md"
                >
                  {job.completedFiles}/{job.totalFiles} files ({Math.round(
                    job.percentage
                  )}%)
                </span>
              </div>
              <Progressbar
                progress={job.percentage}
                color="blue"
                size="h-3"
                class="w-full"
              />
            </div>
            {#if job.currentFile}
              <!-- Current File Progress -->
              <div
                class="space-y-3 p-4 bg-blue-50/50 dark:bg-blue-900/10 rounded-lg border border-blue-200/50 dark:border-blue-800/50"
              >
                <div class="flex justify-between items-center">
                  <P
                    class="text-sm font-medium text-blue-800 dark:text-blue-200"
                    >Current File</P
                  >
                  <span
                    class="text-sm font-semibold text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30 px-2 py-1 rounded-md"
                  >
                    {Math.round(job.currentFileProgress)}%
                  </span>
                </div>
                <P
                  class="text-sm text-blue-700 dark:text-blue-300 font-medium truncate"
                  title={job.currentFile}>{job.currentFile}</P
                >
                <Progressbar
                  progress={job.currentFileProgress}
                  color="blue"
                  size="h-2"
                  class="w-full"
                />
              </div>
            {/if}
            <!-- Timing Information -->
            {#if job.elapsedTime > 0}
              <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div
                  class="flex justify-between items-center p-3 bg-green-50 dark:bg-green-900/10 rounded-lg border border-green-200/50 dark:border-green-800/50"
                >
                  <span
                    class="text-sm text-green-600 dark:text-green-400 font-medium"
                    >Elapsed Time:</span
                  >
                  <span
                    class="text-sm font-semibold text-green-700 dark:text-green-300 bg-green-100 dark:bg-green-900/30 px-2 py-1 rounded-md"
                    >{formatTime(job.elapsedTime * 1000)}</span
                  >
                </div>
                {#if job.speed > 0}
                  <div
                    class="flex justify-between items-center p-3 bg-blue-50 dark:bg-blue-900/10 rounded-lg border border-blue-200/50 dark:border-blue-800/50"
                  >
                    <span
                      class="text-sm text-blue-600 dark:text-blue-400 font-medium"
                      >Upload Speed:</span
                    >
                    <span
                      class="text-sm font-semibold text-blue-700 dark:text-blue-300 bg-blue-100 dark:bg-blue-900/30 px-2 py-1 rounded-md"
                      >{formatSpeed(job.speed * 1024)}</span
                    >
                  </div>
                {/if}
                {#if job.secondsLeft > 0}
                  <div
                    class="flex justify-between items-center p-3 bg-orange-50 dark:bg-orange-900/10 rounded-lg border border-orange-200/50 dark:border-orange-800/50"
                  >
                    <span
                      class="text-sm text-orange-600 dark:text-orange-400 font-medium"
                      >Est. Remaining:</span
                    >
                    <span
                      class="text-sm font-semibold text-orange-700 dark:text-orange-300 bg-orange-100 dark:bg-orange-900/30 px-2 py-1 rounded-md"
                      >{formatTime(job.secondsLeft * 1000)}</span
                    >
                  </div>
                {/if}
              </div>
            {/if}
            <!-- Status Information -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div
                class="flex justify-between items-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
              >
                <span class="text-sm text-gray-600 dark:text-gray-400"
                  >Stage:</span
                >
                <span class="text-sm font-medium text-gray-900 dark:text-white"
                  >{job.stage}</span
                >
              </div>
              {#if job.details}
                <div
                  class="flex justify-between items-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
                >
                  <span class="text-sm text-gray-600 dark:text-gray-400"
                    >Details:</span
                  >
                  <span
                    class="text-sm font-medium text-gray-900 dark:text-white"
                    >{job.details}</span
                  >
                </div>
              {/if}
            </div>
            <div
              class="flex justify-between items-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
            >
              <span class="text-sm text-gray-600 dark:text-gray-400"
                >Last Update:</span
              >
              <span class="text-sm font-medium text-gray-900 dark:text-white"
                >{new Date(job.lastUpdate * 1000).toLocaleTimeString()}</span
              >
            </div>
          </div>
        </Card>
      {/each}
    {:else}
      <div class="text-center py-12">
        <div
          class="w-16 h-16 mx-auto mb-4 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center"
        >
          <svg
            class="w-8 h-8 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
            ></path>
          </svg>
        </div>
        <P class="text-gray-600 dark:text-gray-400 text-lg mb-2 font-medium">
          No upload in progress
        </P>
        <P class="text-gray-500 dark:text-gray-500 text-sm">
          Click "Start Upload" to begin processing the queue
        </P>
      </div>
    {/if}
  </div>
</Card>
