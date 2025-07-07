<script lang="ts">
import { goto } from "$app/navigation";
import DashboardHeader from "$lib/components/dashboard/DashboardHeader.svelte";
import ProgressSection from "$lib/components/dashboard/ProgressSection.svelte";
import QueueSection from "$lib/components/dashboard/QueueSection.svelte";
import QueueStats from "$lib/components/dashboard/QueueStats.svelte";
import { t } from "$lib/i18n";
import { appStatus, progress } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import apiClient from "$lib/api/client";
import { PlusOutline } from "flowbite-svelte-icons";
import { onDestroy, onMount } from "svelte";

let needsConfiguration = false;
let criticalConfigError = false;
let isDragOver = false;
let dragCounter = 0;

onMount(async () => {
	// Initialize API client
	await apiClient.initialize();

	// Set up drag over detection for UI feedback
	window.addEventListener("dragenter", handleDragEnter);
	window.addEventListener("dragleave", handleDragLeave);
	window.addEventListener("dragover", handleDragOver);
	window.addEventListener("drop", handleDrop);

	// Listen for file drop events from backend
	await apiClient.on("file-drop-success", (count) => {
		// Hide overlay when files are successfully processed
		isDragOver = false;
		dragCounter = 0;
		toastStore.success(
			$t("common.common.success"),
			$t("common.messages.files_added_description"),
		);
	});

	await apiClient.on("file-drop-error", (error) => {
		// Hide overlay when there's an error
		isDragOver = false;
		dragCounter = 0;
		toastStore.error($t("common.common.error"), error);
	});

	// Listen for queue updates (from drag and drop or other sources)
	await apiClient.on("queue-updated", () => {
		// This event is emitted when files are added to queue via drag and drop
		// The QueueSection component should automatically refresh its data
		console.log("Queue updated via drag and drop");
	});

	// Listen for progress events
	await apiClient.on("progress", (data) => {
		progress.update((jobs) => {
			// For direct uploads without jobID, use a default key
			const jobKey = data.jobID || "direct-upload";
			
			// Remove job if not running (completed, cancelled, or error)
			if (!data.isRunning) {
				const { [jobKey]: _, ...rest } = jobs;
				return rest;
			}

			// Otherwise, update/add job
			// Ensure jobID is set for UI consistency
			const jobData = { ...data, jobID: jobKey };
			return { ...jobs, [jobKey]: jobData };
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

async function handleDrop(e: DragEvent) {
	e.preventDefault();
	console.log("Drop detected!", e.dataTransfer?.files);

	// Hide the overlay when files are dropped
	isDragOver = false;
	dragCounter = 0;

	// Handle file upload based on environment
	if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
		try {
			if (apiClient.environment === "web") {
				// In web mode, upload files directly via HTTP
				await apiClient.uploadFileList(e.dataTransfer.files);
				toastStore.success(
					$t("common.common.success"),
					$t("common.messages.files_added_description"),
				);
			}
			// In Wails mode, the backend OnFileDrop handler in main.go processes files automatically
		} catch (error) {
			console.error("File upload failed:", error);
			toastStore.error($t("common.common.error"), String(error));
		}
	}
}

async function handleUpload() {
	try {
		if (apiClient.environment === "wails") {
			// In Wails mode, use the existing upload flow
			await apiClient.uploadFiles();
		} else {
			// In web mode, trigger file input dialog
			const input = document.createElement("input");
			input.type = "file";
			input.multiple = true;
			input.onchange = async (e) => {
				const files = (e.target as HTMLInputElement).files;
				if (files && files.length > 0) {
					try {
						await apiClient.uploadFileList(files);
						toastStore.success(
							$t("common.common.success"),
							$t("common.messages.files_added_description"),
						);
					} catch (error) {
						console.error("File upload failed:", error);
						toastStore.error($t("common.common.error"), String(error));
					}
				}
			};
			input.click();
		}
	} catch (error) {
		console.error("Upload failed:", error);
		const errorMessage = String(error);

		if (errorMessage.includes("configuration required")) {
			toastStore.error(
				$t("common.common.error"),
				$t("common.messages.error_saving"),
			);
			// Navigate to settings
			if (apiClient.environment === "wails") {
				await apiClient.navigateToSettings();
			} else {
				goto("/settings");
			}
		} else if (errorMessage.includes("runtime not available")) {
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
