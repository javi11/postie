<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { onMount, onDestroy } from "svelte";
import { Activity, Upload, Server, AlertCircle, FileText } from "lucide-svelte";
  import type { backend } from "$lib/wailsjs/go/models";

let metrics = $state<backend.NntpPoolMetrics | null>(null);
let loading = $state(true);
let error = $state<string | null>(null);
let refreshInterval = $state<NodeJS.Timeout | null>(null);

// Auto-refresh every 5 seconds
const REFRESH_INTERVAL = 5000;

onMount(async () => {
	await loadMetrics();
	startAutoRefresh();
});

onDestroy(() => {
	stopAutoRefresh();
});

function startAutoRefresh() {
	refreshInterval = setInterval(async () => {
		await loadMetrics(false); // Don't show loading state on auto-refresh
	}, REFRESH_INTERVAL);
}

function stopAutoRefresh() {
	if (refreshInterval) {
		clearInterval(refreshInterval);
		refreshInterval = null;
	}
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

function getProviderStatusColor(state: string): string {
	switch (state.toLowerCase()) {
		case "healthy":
		case "active":
		case "connected":
			return "badge-success";
		case "connecting":
		case "idle":
			return "badge-warning";
		case "error":
		case "failed":
		case "disconnected":
			return "badge-error";
		default:
			return "badge-neutral";
	}
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
				<Activity class="w-4 h-4 {refreshInterval ? 'animate-pulse text-primary' : ''}" />
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
		<!-- Posting Overview Stats -->
		<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
			<!-- Total Data Posted -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-success">
					<Upload class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.total_data_posted')}</div>
				<div class="stat-value text-success">{formatBytes(metrics.totalBytesUploaded)}</div>
			</div>

			<!-- Articles Posted -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-info">
					<FileText class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.articles_posted')}</div>
				<div class="stat-value text-info">{metrics.totalArticlesPosted.toLocaleString()}</div>
			</div>

			<!-- Total Errors -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-error">
					<AlertCircle class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.total_errors')}</div>
				<div class="stat-value text-error">{metrics.totalErrors.toLocaleString()}</div>
			</div>
		</div>

		<!-- Upload Statistics Summary -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body">
				<h2 class="card-title flex items-center gap-2">
					<Upload class="w-5 h-5" />
					{$t('metrics.upload_statistics')}
				</h2>

				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
					<div class="stat">
						<div class="stat-title">{$t('metrics.total_data_posted')}</div>
						<div class="stat-value text-success text-2xl">{formatBytes(metrics.totalBytesUploaded)}</div>
					</div>
					<div class="stat">
						<div class="stat-title">{$t('metrics.articles_posted')}</div>
						<div class="stat-value text-info text-2xl">{metrics.totalArticlesPosted.toLocaleString()}</div>
					</div>
					<div class="stat">
						<div class="stat-title">{$t('metrics.total_errors')}</div>
						<div class="stat-value text-error text-2xl">{metrics.totalErrors.toLocaleString()}</div>
					</div>
					<div class="stat">
						<div class="stat-title">{$t('metrics.average_article_size')}</div>
						<div class="stat-value text-base-content text-2xl">
							{metrics.totalArticlesPosted > 0
								? formatBytes(Math.round(metrics.totalBytesUploaded / metrics.totalArticlesPosted))
								: "0 B"
							}
						</div>
					</div>
				</div>
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
								<th>{$t('metrics.status')}</th>
								<th>{$t('metrics.connections')}</th>
								<th>{$t('metrics.data_posted')}</th>
								<th>{$t('metrics.articles_posted')}</th>
								<th>{$t('metrics.total_errors')}</th>
							</tr>
						</thead>
						<tbody>
							{#each metrics.providers as provider}
								<tr>
									<td>
										<div class="font-semibold">{provider.host}</div>
									</td>
									<td>
										<div class="badge {getProviderStatusColor(provider.state)} badge-sm">
											{provider.state}
										</div>
									</td>
									<td>
										<div class="text-sm">
											{provider.activeConnections}/{provider.maxConnections}
										</div>
									</td>
									<td>
										<div class="text-sm text-success">
											{formatBytes(provider.totalBytesUploaded)}
										</div>
									</td>
									<td>
										<div class="text-sm text-info">
											{provider.totalArticlesPosted.toLocaleString()}
										</div>
									</td>
									<td>
										<div class="text-sm text-error">
											{provider.totalErrors.toLocaleString()}
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