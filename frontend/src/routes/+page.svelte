<script lang="ts">
import { goto } from "$app/navigation";
import DashboardHeader from "$lib/components/dashboard/DashboardHeader.svelte";
import ProgressSection from "$lib/components/dashboard/ProgressSection.svelte";
import QueueSection from "$lib/components/dashboard/QueueSection.svelte";
import QueueStats from "$lib/components/dashboard/QueueStats.svelte";
import { t } from "$lib/i18n";
import { appStatus, progress } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { waitForWailsRuntime } from "$lib/utils";
import * as App from "$lib/wailsjs/go/backend/App";
import { EventsOn } from "$lib/wailsjs/runtime/runtime";
import { PlusOutline } from "flowbite-svelte-icons";
import { onMount, onDestroy } from "svelte";

let needsConfiguration = false;
let criticalConfigError = false;
let isDragOver = false;
let dragCounter = 0;

onMount(async () => {
	// Wait for Wails runtime to be ready
	await waitForWailsRuntime();

	// Set up drag over detection for UI feedback only
	// The backend OnFileDrop in main.go handles the actual file processing
	window.addEventListener("dragenter", handleDragEnter);
	window.addEventListener("dragleave", handleDragLeave);
	window.addEventListener("dragover", handleDragOver);
	window.addEventListener("drop", handleDrop);

	// Listen for file drop events from backend
	EventsOn("file-drop-success", (count) => {
		// Hide overlay when files are successfully processed
		isDragOver = false;
		dragCounter = 0;
		toastStore.success(
			$t("common.common.success"),
			$t("common.messages.files_added_description"),
		);
	});

	EventsOn("file-drop-error", (error) => {
		// Hide overlay when there's an error
		isDragOver = false;
		dragCounter = 0;
		toastStore.error($t("common.common.error"), error);
	});

	// Listen for queue updates (from drag and drop or other sources)
	EventsOn("queue-updated", () => {
		// This event is emitted when files are added to queue via drag and drop
		// The QueueSection component should automatically refresh its data
		console.log("Queue updated via drag and drop");
	});

	// Listen for progress events
	EventsOn("progress", (data) => {
		progress.update((jobs) => {
			if (!data.jobID) return jobs;

			// Remove job if not running (completed, cancelled, or error)
			if (!data.isRunning) {
				console.log(
					"Removing job from progress:",
					data.jobID,
					"stage:",
					data.stage,
				);
				const { [data.jobID]: _, ...rest } = jobs;
				return rest;
			}

			// Otherwise, update/add job
			console.log(
				"Updating job progress:",
				data.jobID,
				"stage:",
				data.stage,
				"isRunning:",
				data.isRunning,
			);
			return { ...jobs, [data.jobID]: data };
		});
	});

	// Subscribe to app status
	const unsubscribe = appStatus.subscribe((status) => {
		needsConfiguration = status.needsConfiguration;
		criticalConfigError = status.criticalConfigError;

		// Redirect to settings if configuration is needed or there's a critical error
		if (needsConfiguration || criticalConfigError) {
			goto("/settings");
		}
	});

	return unsubscribe;
});

onDestroy(() => {
	// Clean up drag event listeners
	window.removeEventListener("dragenter", handleDragEnter);
	window.removeEventListener("dragleave", handleDragLeave);
	window.removeEventListener("dragover", handleDragOver);
	window.removeEventListener("drop", handleDrop);
});

function handleDragEnter(e: DragEvent) {
	e.preventDefault();
	if (e.dataTransfer?.types.includes("Files")) {
		dragCounter++;
		isDragOver = true;
	}
}

function handleDragLeave(e: DragEvent) {
	e.preventDefault();
	if (e.dataTransfer?.types.includes("Files")) {
		dragCounter--;
		if (dragCounter <= 0) {
			dragCounter = 0;
			isDragOver = false;
		}
	}
}

function handleDragOver(e: DragEvent) {
	e.preventDefault();
	// Keep the overlay visible while dragging
	if (e.dataTransfer?.types.includes("Files")) {
		isDragOver = true;
	}
}

function handleDrop(e: DragEvent) {
	e.preventDefault();
	console.log("Drop detected!", e.dataTransfer?.files);
	// Hide the overlay when files are dropped
	// The backend OnFileDrop handler in main.go will process the files
	isDragOver = false;
	dragCounter = 0;
}

async function handleUpload() {
	try {
		// Ensure runtime is ready before calling backend
		await waitForWailsRuntime();
		await App.UploadFiles();
	} catch (error) {
		console.error("Upload failed:", error);
		const errorMessage = String(error);

		if (errorMessage.includes("configuration required")) {
			toastStore.error(
				$t("common.common.error"),
				$t("common.messages.error_saving"),
			);
			// Navigate to settings using SvelteKit's navigation
			App.NavigateToSettings();
		} else if (errorMessage.includes("Wails runtime not available")) {
			toastStore.error($t("common.common.error"), $t("common.common.loading"));
		} else {
			toastStore.error($t("common.common.error"), errorMessage);
		}
	}
}
</script>

<svelte:head>
  <title>{$t('dashboard.title')} - Postie</title>
  <meta name="description" content="Manage your uploads and monitor progress" />
</svelte:head>

<div style="--wails-drop-target: drop">
  <!-- Drag and Drop Overlay -->
  {#if isDragOver}
    <div class="drag-overlay">
      <div class="drag-overlay-content">
        <div class="drag-icon">
          <PlusOutline class="w-16 h-16 text-white" />
        </div>
        <h2 class="text-2xl font-bold text-white mb-2">Drop Files Here</h2>
        <p class="text-white/80">Release to add files to queue</p>
      </div>
    </div>
  {/if}
	<div class="space-y-8 relative">
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
</div>
