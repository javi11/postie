<script lang="ts">
import { goto } from "$app/navigation";
import apiClient from "$lib/api/client";
import SetupWizard from "$lib/components/setup/SetupWizard.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { backend } from "$lib/wailsjs/go/models";

let appStatus = $state<backend.AppStatus | null>(null);
let isLoading = $state(true);

// Check app status on mount
$effect(() => {
	async function checkAppStatus() {
		try {
			const status = await apiClient.getAppStatus();
			appStatus = status;

			// If not first start and not needing configuration, redirect to dashboard
			if (!status.isFirstStart && !status.needsConfiguration) {
				goto("/");
				return;
			}

			isLoading = false;
			return;
		} catch (error) {
			console.error("Failed to load app status:", error);
			// Assume setup is needed if we can't get status
			isLoading = false;
		}
	}
	checkAppStatus();
});

async function handleWizardComplete(
	data: backend.SetupWizardData,
): Promise<void> {
	try {
		await apiClient.setupWizardComplete(data);

		toastStore.success(
			$t("common.common.success"),
			$t("common.messages.configuration_saved_description"),
		);

		// Redirect to dashboard
		goto("/");
		return;
	} catch (error) {
		console.error("Setup wizard failed:", error);
		toastStore.error(
			$t("common.common.error"),
			$t("common.messages.configuration_save_failed"),
		);
	}
}

// handleWizardClose was removed since SetupWizard no longer has onclose prop
</script>

<svelte:head>
	<title>{$t("setup.title")} - Postie</title>
	<meta name="description" content="Set up Postie with your NNTP servers and directories" />
</svelte:head>

{#if isLoading}
	<div class="min-h-screen flex items-center justify-center p-4">
		<div class="text-center">
			<div class="w-16 h-16 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
			<p class="text-base-content/70">{$t("common.common.loading")}</p>
		</div>
	</div>
{:else}
	<SetupWizard 
		oncomplete={handleWizardComplete}
	/>
{/if}