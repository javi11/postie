<script lang="ts">
import { onDestroy, onMount } from 'svelte';
import { t } from '$lib/i18n';
import apiClient from '$lib/api/client';

let hasPending = $state(false);

// Check pending config status on mount and listen for changes
onMount(async () => {
	await checkPendingStatus();
	
	// Listen for pending config events
	await apiClient.on('config-pending', handlePendingEvent);
	await apiClient.on('config-applied', handleAppliedEvent);
	await apiClient.on('config-updated', handleAppliedEvent);
});

onDestroy(() => {
	apiClient.off('config-pending', handlePendingEvent);
	apiClient.off('config-applied', handleAppliedEvent);
	apiClient.off('config-updated', handleAppliedEvent);
});

async function checkPendingStatus() {
	try {
		hasPending = await apiClient.hasPendingConfigChanges();
	} catch (error) {
		console.error('Failed to check pending config status:', error);
	}
}

function handlePendingEvent(data: unknown) {
	if (typeof data === 'object' && data !== null) {
		const status = data as Record<string, unknown>;
		hasPending = Boolean(status.hasPendingConfig);
	}
}

function handleAppliedEvent() {
	hasPending = false;
}
</script>

{#if hasPending}
	<div class="flex items-center gap-2 px-3 py-1 bg-info/20 border border-info/30 rounded-full">
		<div class="w-2 h-2 bg-info rounded-full animate-pulse"></div>
		<span class="text-sm font-medium text-info">
			{$t('settings.config_status.editing_pending')}
		</span>
	</div>
{/if}