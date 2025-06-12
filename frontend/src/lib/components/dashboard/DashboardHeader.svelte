<script lang="ts">
  import { Button, Heading, P, Alert, Card } from "flowbite-svelte";
  import {
    UploadSolid,
    CirclePlusSolid,
    TrashBinSolid,
    ExclamationCircleSolid,
    CloseCircleSolid,
  } from "flowbite-svelte-icons";
  import { isUploading } from "$lib/stores/app";
  import * as App from "$lib/wailsjs/go/main/App";
  import { toastStore } from "$lib/stores/toast";

  export let needsConfiguration: boolean;
  export let criticalConfigError: boolean;
  export let handleUpload: () => Promise<void>;

  async function addFilesToQueue() {
    try {
      await App.AddFilesToQueue();
      toastStore.success("Files added", "Files have been added to the queue");
    } catch (error) {
      console.error("Failed to add files to queue:", error);
    }
  }

  async function clearQueue() {
    try {
      await App.ClearQueue();
      toastStore.success(
        "Queue cleared",
        "Completed and failed items have been removed"
      );
    } catch (error) {
      console.error("Failed to clear queue:", error);
      toastStore.error("Failed to clear queue", String(error));
    }
  }

  async function cancelUpload() {
    try {
      await App.CancelUpload();
      toastStore.success(
        "Upload cancelled",
        "Upload has been cancelled successfully"
      );
    } catch (error) {
      console.error("Failed to cancel upload:", error);
      toastStore.error("Failed to cancel upload", String(error));
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
        Upload Dashboard
      </Heading>
      <P class="text-lg text-gray-600 dark:text-gray-400">
        Manage your file uploads and monitor progress
      </P>
    </div>

    <div class="flex flex-wrap gap-3">
      <Button
        color="alternative"
        onclick={addFilesToQueue}
        disabled={needsConfiguration}
        class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm hover:shadow-md transition-all duration-200 border-gray-300 dark:border-gray-600"
      >
        <CirclePlusSolid class="w-4 h-4" />
        Add Files
      </Button>

      <Button
        color={"primary"}
        onclick={handleUpload}
        disabled={needsConfiguration || $isUploading}
        class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm"
      >
        <UploadSolid class="w-4 h-4" />
        Start Upload
      </Button>

      <Button
        color="red"
        variant="outline"
        onclick={clearQueue}
        class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm"
      >
        <TrashBinSolid class="w-4 h-4" />
        Clear Completed
      </Button>
    </div>
  </div>

  {#if criticalConfigError}
    <Alert color="red" class="mt-6">
      <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
      <span class="font-semibold">Configuration Error</span>
      There was an error with your server configuration (e.g., invalid hostname,
      connection failure). Please check your
      <a
        href="/settings"
        class="font-medium underline hover:no-underline transition-all"
        >Settings</a
      >
      to fix the configuration.
    </Alert>
  {:else if needsConfiguration}
    <Alert color="yellow" class="mt-6">
      <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
      <span class="font-semibold">Configuration Required</span>
      Please configure at least one server in the
      <a
        href="/settings"
        class="font-medium underline hover:no-underline transition-all"
        >Settings</a
      >
      page before uploading files.
    </Alert>
  {/if}
</Card>
