<script lang="ts">
import { isUploading } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { t } from "$lib/i18n";
import * as App from "$lib/wailsjs/go/backend/App";
import { Alert, Button, Card, Heading, P } from "flowbite-svelte";
import {
	CirclePlusSolid,
	CloseCircleSolid,
	ExclamationCircleSolid,
	TrashBinSolid,
	UploadSolid,
} from "flowbite-svelte-icons";

export let needsConfiguration: boolean;
export let criticalConfigError: boolean;

async function addFilesToQueue() {
	try {
		await App.AddFilesToQueue();
		toastStore.success($t("common.messages.files_added"), $t("common.messages.files_added_description"));
	} catch (error) {
		console.error("Failed to add files to queue:", error);
	}
}

async function clearQueue() {
	try {
		await App.ClearQueue();
		toastStore.success(
			$t("common.messages.queue_cleared"),
			$t("common.messages.queue_cleared_description"),
		);
	} catch (error) {
		console.error("Failed to clear queue:", error);
		toastStore.error($t("common.messages.failed_to_clear_queue"), String(error));
	}
}

async function cancelUpload() {
	try {
		await App.CancelUpload();
		toastStore.success(
			$t("common.messages.upload_cancelled"),
			$t("common.messages.upload_cancelled_description"),
		);
	} catch (error) {
		console.error("Failed to cancel upload:", error);
		toastStore.error($t("common.messages.failed_to_cancel_upload"), String(error));
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
        {$t("dashboard.header.title")}
      </Heading>
      <P class="text-lg text-gray-600 dark:text-gray-400">
        {$t("dashboard.header.description")}
      </P>
    </div>

    <div class="flex flex-wrap gap-3">
      <Button
        color="primary"
        onclick={addFilesToQueue}
        disabled={needsConfiguration}
        class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm hover:shadow-md transition-all duration-200 border-gray-300 dark:border-gray-600"
      >
        <CirclePlusSolid class="w-4 h-4" />
        {$t("dashboard.header.add_files")}
      </Button>

      <Button
        color="red"
        variant="outline"
        onclick={clearQueue}
        class="cursor-pointer flex items-center gap-2 px-6 py-3 text-sm font-medium shadow-sm"
      >
        <TrashBinSolid class="w-4 h-4" />
        {$t("dashboard.header.clear_completed")}
      </Button>
    </div>
  </div>

  {#if criticalConfigError}
    <Alert color="red" class="mt-6">
      <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
      <span class="font-semibold">{$t("dashboard.alerts.config_error")}</span>
      {$t("dashboard.alerts.config_error_description", { settingsLink: `<a href="/settings" class="font-medium underline hover:no-underline transition-all">${$t("dashboard.alerts.settings_link")}</a>` })}
    </Alert>
  {:else if needsConfiguration}
    <Alert color="yellow" class="mt-6">
      <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
      <span class="font-semibold">{$t("dashboard.alerts.config_required")}</span>
      {$t("dashboard.alerts.config_required_description", { settingsLink: `<a href="/settings" class="font-medium underline hover:no-underline transition-all">${$t("dashboard.alerts.settings_link")}</a>` })}
    </Alert>
  {/if}
</Card>
