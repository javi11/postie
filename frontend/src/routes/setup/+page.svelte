<script lang="ts">
import { goto } from "$app/navigation";
import { onMount } from "svelte";
import apiClient from "$lib/api/client";
import SetupWizard from "$lib/components/setup/SetupWizard.svelte";
import { toastStore } from "$lib/stores/toast";
import { t } from "$lib/i18n";

let appStatus = null;
let isLoading = true;

onMount(async () => {
	// Initialize API client
	await apiClient.initialize();
	
	// Check app status to see if setup is needed
	try {
		appStatus = await apiClient.getAppStatus();
		
		// If not first start and not needing configuration, redirect to dashboard
		if (!appStatus.isFirstStart && !appStatus.needsConfiguration) {
			goto("/");
			return;
		}
		
		isLoading = false;
	} catch (error) {
		console.error("Failed to load app status:", error);
		// Assume setup is needed if we can't get status
		isLoading = false;
	}
});

async function handleWizardComplete(event) {
	try {
		await apiClient.setupWizardComplete(event.detail);
		
		toastStore.success(
			$t("common.common.success"),
			$t("common.messages.configuration_saved_description")
		);
		
		// Redirect to dashboard
		goto("/");
	} catch (error) {
		console.error("Setup wizard failed:", error);
		toastStore.error(
			$t("common.common.error"),
			$t("common.messages.configuration_save_failed")
		);
	}
}

function handleWizardClose() {
	// For first start, don't allow closing without completing setup
	if (appStatus?.isFirstStart) {
		toastStore.warning(
			$t("common.common.warning"),
			$t("setup.messages.setupRequired")
		);
		return;
	}
	
	// Otherwise redirect to dashboard or settings
	goto(appStatus?.needsConfiguration ? "/settings" : "/");
}
</script>

<svelte:head>
	<title>{$t("setup.title")} - Postie</title>
	<meta name="description" content="Set up Postie with your NNTP servers and directories" />
</svelte:head>

{#if isLoading}
	<div class="min-h-screen flex items-center justify-center p-4">
		<div class="text-center">
			<div class="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
			<p class="text-gray-600 dark:text-gray-400">{$t("common.common.loading")}</p>
		</div>
	</div>
{:else}
	<SetupWizard 
		on:complete={handleWizardComplete}
		on:close={handleWizardClose}
	/>
{/if}