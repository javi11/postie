<script lang="ts">
import DashboardHeader from "$lib/components/dashboard/DashboardHeader.svelte";
import ProgressSection from "$lib/components/dashboard/ProgressSection.svelte";
import QueueSection from "$lib/components/dashboard/QueueSection.svelte";
import QueueStats from "$lib/components/dashboard/QueueStats.svelte";
import { appStatus, progress } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import * as App from "$lib/wailsjs/go/backend/App";
import { EventsOn } from "$lib/wailsjs/runtime/runtime";
import { onMount } from "svelte";

let needsConfiguration = false;
let criticalConfigError = false;

onMount(async () => {
	// Listen for progress events
	EventsOn("progress", (data) => {
		progress.update((jobs) => {
			if (!data.jobID) return jobs;
			// Remove job if not running
			if (!data.isRunning) {
				const { [data.jobID]: _, ...rest } = jobs;
				return rest;
			}
			// Otherwise, update/add job
			return { ...jobs, [data.jobID]: data };
		});
	});

	// Subscribe to app status
	const unsubscribe = appStatus.subscribe((status) => {
		needsConfiguration = status.needsConfiguration;
		criticalConfigError = status.criticalConfigError;
	});

	return unsubscribe;
});

async function handleUpload() {
	try {
		await App.UploadFiles();
	} catch (error) {
		console.error("Upload failed:", error);
		const errorMessage = String(error);

		if (errorMessage.includes("configuration required")) {
			toastStore.error(
				"Configuration Required",
				"Please configure at least one server before uploading files.",
			);
			// Navigate to settings using SvelteKit's navigation
			window.location.href = "/settings";
		} else {
			toastStore.error("Upload failed", errorMessage);
		}
	}
}
</script>

<svelte:head>
  <title>Dashboard - Postie</title>
  <meta name="description" content="Manage your uploads and monitor progress" />
</svelte:head>

<div class="space-y-8">
  <DashboardHeader {needsConfiguration} {criticalConfigError} {handleUpload} />

  <div
    class="flex flex-col gap-8"
    class:pointer-events-none={needsConfiguration || criticalConfigError}
    class:opacity-50={needsConfiguration || criticalConfigError}
  >
    <!-- Main Content Area -->
    <div class="space-y-8">
      <ProgressSection />
      <QueueSection />
    </div>

    <!-- Sidebar -->
    <div class="space-y-8">
      <QueueStats />
    </div>
  </div>
</div>
