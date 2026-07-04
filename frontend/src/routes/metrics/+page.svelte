<script lang="ts">
import apiClient from "$lib/api/client";
import {
	EVENT_NNTP_POOL_METRICS_UPDATED,
	type NntpPoolMetricsEvent,
} from "$lib/api/events";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { onMount, onDestroy } from "svelte";
import { Activity, Upload, Server, AlertCircle, AlertTriangle, Gauge } from "lucide-svelte";
import type { backend } from "$lib/wailsjs/go/models";
import { formatElapsed } from "$lib/utils";

let metrics = $state<backend.NntpPoolMetrics | null>(null);
let loading = $state(true);
let error = $state<string | null>(null);
let transferMetrics = $state<backend.TransferRuntimeMetrics | null>(null);
let transferInterval: ReturnType<typeof setInterval> | undefined;

onMount(async () => {
	await loadMetrics();
	await apiClient.on(EVENT_NNTP_POOL_METRICS_UPDATED, applyMetricsEvent);
	// Transfer-runtime metrics have no push event; poll them every 5s.
	await loadTransferMetrics();
	transferInterval = setInterval(loadTransferMetrics, 5000);
});

onDestroy(() => {
	apiClient.off(EVENT_NNTP_POOL_METRICS_UPDATED, applyMetricsEvent);
	if (transferInterval) clearInterval(transferInterval);
});

async function loadTransferMetrics() {
	try {
		transferMetrics = await apiClient.getTransferRuntimeMetrics();
	} catch (err) {
		console.error("Failed to load transfer runtime metrics:", err);
	}
}

function applyMetricsEvent(data: unknown) {
	if (!data) return;
	metrics = data as NntpPoolMetricsEvent;
	error = null;
}

async function loadMetrics(showLoading = true) {
	try {
		if (showLoading) {
			loading = true;
		}
		error = null;

		metrics = await apiClient.getNntpPoolMetrics();
	} catch (err) {
		console.error("Failed to load NNTP pool metrics:", err);
		error = String(err);
		toastStore.error($t("common.common.error"), $t("metrics.error_loading"));
	} finally {
		if (showLoading) {
			loading = false;
		}
	}
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return "0 B";
	const k = 1024;
	const sizes = ["B", "KB", "MB", "GB", "TB"];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatSpeed(bytesPerSec: number): string {
	if (bytesPerSec === 0) return "0 B/s";
	return formatBytes(bytesPerSec) + "/s";
}

function formatNumber(num: number): string {
	return num.toLocaleString();
}
</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold text-base-content">{$t('metrics.title')}</h1>
			<p class="text-base-content/70 mt-1">{$t('metrics.description')}</p>
		</div>

		<div class="flex items-center gap-3">
			<!-- Auto-refresh indicator -->
			<div class="flex items-center gap-2 text-sm text-base-content/60">
				<Activity class="w-4 h-4 animate-pulse text-primary" />
				<span>{$t('metrics.auto_refresh')}</span>
			</div>

			<button
				class="btn btn-primary btn-sm"
				onclick={() => loadMetrics(true)}
				disabled={loading}
			>
				{loading ? $t('common.common.loading') : $t('metrics.refresh')}
			</button>
		</div>
	</div>

	<!-- Transfer runtime (process-wide upload engine + PAR2 scheduler) -->
	{#if transferMetrics && (transferMetrics.uploadWorkerCount > 0 || transferMetrics.par2Capacity > 0)}
		<div>
			<h2 class="text-xl font-semibold text-base-content mb-3 flex items-center gap-2">
				<Gauge class="w-5 h-5 text-primary" />
				{$t('metrics.transfer_runtime.title')}
			</h2>
			<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
				<div class="stat bg-base-100 rounded-lg shadow-sm">
					<div class="stat-title text-xs">{$t('metrics.transfer_runtime.upload_workers')}</div>
					<div class="stat-value text-2xl">{transferMetrics.uploadActiveWorkers}/{transferMetrics.uploadWorkerCount}</div>
					<div class="stat-desc">{transferMetrics.uploadQueuedWorkers} {$t('metrics.transfer_runtime.queued')}</div>
				</div>
				<div class="stat bg-base-100 rounded-lg shadow-sm">
					<div class="stat-title text-xs">{$t('metrics.transfer_runtime.buffer_memory')}</div>
					<div class="stat-value text-2xl">{formatBytes(transferMetrics.uploadReservedBytes)}</div>
					<div class="stat-desc">/ {formatBytes(transferMetrics.uploadBudgetBytes)}</div>
				</div>
				<div class="stat bg-base-100 rounded-lg shadow-sm">
					<div class="stat-title text-xs">{$t('metrics.transfer_runtime.par2_jobs')}</div>
					<div class="stat-value text-2xl">{transferMetrics.par2ActiveJobs}/{transferMetrics.par2Capacity}</div>
					<div class="stat-desc">{transferMetrics.par2QueuedJobs} {$t('metrics.transfer_runtime.queued')}</div>
				</div>
			</div>
		</div>
	{/if}

	{#if loading && !metrics}
		<div class="flex items-center justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error}
		<div class="alert alert-error">
			<AlertCircle class="w-5 h-5" />
			<div>
				<h3>{$t('metrics.error_title')}</h3>
				<div class="text-sm opacity-75">{error}</div>
			</div>
		</div>
	{:else if metrics}
		<!-- Overview Stats -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<!-- Active Connections -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-primary">
					<Server class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.active_connections')}</div>
				<div class="stat-value text-primary">{metrics.activeConnections}</div>
			</div>

			<!-- Average Speed -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-success">
					<Gauge class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.avg_speed')}</div>
				<div class="stat-value text-success text-2xl">{formatSpeed(metrics.avgSpeed)}</div>
			</div>

			<!-- Elapsed -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-info">
					<Activity class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.elapsed')}</div>
				<div class="stat-value text-info text-2xl">{formatElapsed(metrics.elapsed) || "—"}</div>
			</div>

			<!-- Total Errors -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-error">
					<AlertTriangle class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.total_errors')}</div>
				<div class="stat-value text-error">{formatNumber(metrics.totalErrors)}</div>
			</div>
		</div>

		<!-- NNTP Server Details -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body">
				<h2 class="card-title flex items-center gap-2">
					<Server class="w-5 h-5" />
					{$t('metrics.nntp_servers')} ({metrics.providers.length})
				</h2>

				<div class="overflow-x-auto">
					<table class="table table-zebra w-full">
						<thead>
							<tr>
								<th>{$t('metrics.server_host')}</th>
								<th>{$t('metrics.connections')}</th>
								<th>{$t('metrics.avg_speed')}</th>
								<th>{$t('metrics.missing')}</th>
								<th>{$t('metrics.ping_rtt')}</th>
								<th>{$t('metrics.errors')}</th>
							</tr>
						</thead>
						<tbody>
							{#each metrics.providers as provider}
								<tr>
									<td>
										<div class="font-semibold">{provider.name || provider.host}</div>
									</td>
									<td>
										<div class="text-sm">
											{provider.activeConnections}/{provider.maxConnections}
										</div>
									</td>
									<td>
										<div class="text-sm text-success">
											{formatSpeed(provider.avgSpeed)}
										</div>
									</td>
									<td>
										<div class="text-sm">
											{formatNumber(provider.missing)}
										</div>
									</td>
									<td>
										<div class="text-sm">
											{provider.pingRTT || "—"}
										</div>
									</td>
									<td>
										<div class="text-sm font-mono {provider.totalErrors > 0 ? 'text-error' : 'text-success'}">
											{formatNumber(provider.totalErrors)}
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}
</div>
