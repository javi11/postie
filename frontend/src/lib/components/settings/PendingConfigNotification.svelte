<script lang="ts">
import { onDestroy, onMount } from 'svelte';
import { t } from '$lib/i18n';
import { toastStore } from '$lib/stores/toast';
import apiClient from '$lib/api/client';

let pendingStatus = $state<Record<string, unknown>>({});
let loading = $state(false);

// Check pending config status on mount and periodically
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
		pendingStatus = await apiClient.getPendingConfigStatus();
	} catch (error) {
		console.error('Failed to check pending config status:', error);
	}
}

function handlePendingEvent(data: unknown) {
	if (typeof data === 'object' && data !== null) {
		pendingStatus = { ...data as Record<string, unknown> };
	}
}

function handleAppliedEvent(data: unknown) {
	if (typeof data === 'object' && data !== null) {
		pendingStatus = { ...data as Record<string, unknown> };
		toastStore.success(
			$t('settings.pending_config.applied_title'),
			$t('settings.pending_config.applied_message')
		);
	}
}

async function applyNow() {
	if (loading) return;
	
	try {
		loading = true;
		await apiClient.applyPendingConfig();
		
		// Immediately update the pending status after successful apply
		await checkPendingStatus();
		
		toastStore.success(
			$t('settings.pending_config.apply_success'),
			$t('settings.pending_config.apply_success_message')
		);
	} catch (error) {
		console.error('Failed to apply pending config:', error);
		toastStore.error(
			$t('settings.pending_config.apply_error'),
			String(error)
		);
	} finally {
		loading = false;
	}
}

async function discardChanges() {
	if (loading) return;
	
	try {
		loading = true;
		await apiClient.discardPendingConfig();
		
		// Immediately update the pending status after successful discard
		await checkPendingStatus();
		
		toastStore.info(
			$t('settings.pending_config.discard_success'),
			$t('settings.pending_config.discard_message')
		);
	} catch (error) {
		console.error('Failed to discard pending config:', error);
		toastStore.error(
			$t('settings.pending_config.discard_error'),
			String(error)
		);
	} finally {
		loading = false;
	}
}

// Derived state
let hasPending = $derived(Boolean(pendingStatus.hasPendingConfig));
let canApplyNow = $derived(Boolean(pendingStatus.canApplyNow));
let message = $derived(pendingStatus.message as string);
let hasError = $derived(Boolean(pendingStatus.error));
let errorMessage = $derived(pendingStatus.error as string);
</script>

{#if hasPending}
	<div class="alert {hasError ? 'alert-error' : 'alert-info'} animate-fade-in border-l-4 {hasError ? 'border-l-error' : 'border-l-info'}">
		<div class="flex items-center gap-3">
			{#if hasError}
				<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			{:else}
				<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
				</svg>
			{/if}
			
			<div class="flex-1">
				<h3 class="font-semibold">
					{#if hasError}
						{$t('settings.pending_config.error_title')}
					{:else}
						<span class="flex items-center gap-2">
							{$t('settings.pending_config.pending_title')}
							<span class="badge badge-accent badge-sm">EDITING</span>
						</span>
					{/if}
				</h3>
				<p class="text-sm opacity-90">
					{#if hasError}
						{errorMessage}
					{:else}
						<strong>You are viewing and editing unsaved changes.</strong> {message}
					{/if}
				</p>
			</div>
			
			<div class="flex gap-2">
				{#if canApplyNow && !hasError}
					<button 
						class="btn btn-sm btn-primary"
						onclick={applyNow}
						disabled={loading}
					>
						{loading ? $t('common.applying') : $t('settings.pending_config.apply_now')}
					</button>
				{/if}
				
				<button 
					class="btn btn-sm btn-ghost"
					onclick={discardChanges}
					disabled={loading}
				>
					{loading ? $t('common.discarding') : $t('settings.pending_config.discard')}
				</button>
			</div>
		</div>
	</div>
{/if}