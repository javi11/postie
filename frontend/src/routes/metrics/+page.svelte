<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { onMount, onDestroy } from "svelte";
import { Activity, Upload, Server, Zap, Clock, AlertCircle, FileText, TrendingUp, Archive } from "lucide-svelte";
  import type { backend } from "$lib/wailsjs/go/models";

let metrics = $state<backend.NntpPoolMetrics | null>(null);
let loading = $state(true);
let error = $state<string | null>(null);
let refreshInterval = $state<NodeJS.Timeout | null>(null);
let selectedPeriod = $state<'current' | 'daily' | 'weekly'>('current');

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
	// Only auto-refresh for current metrics, not compressed historical data
	if (selectedPeriod === 'current') {
		refreshInterval = setInterval(async () => {
			await loadMetrics(false); // Don't show loading state on auto-refresh
		}, REFRESH_INTERVAL);
	}
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
		
		const allMetrics = await apiClient.getNntpPoolMetrics();
		
		// Filter metrics based on selected period
		if (selectedPeriod === 'daily' && allMetrics.dailyMetrics?.length > 0) {
			// Show daily compressed metrics - take the latest daily data
			const latestDaily = allMetrics.dailyMetrics[allMetrics.dailyMetrics.length - 1];
			metrics = {
				...allMetrics,
				// Override current metrics with compressed daily averages
				uploadSpeed: latestDaily.averageUploadSpeed,
				commandSuccessRate: latestDaily.averageSuccessRate,
				totalBytesUploaded: latestDaily.totalBytesUploaded,
				totalArticlesPosted: latestDaily.totalArticlesPosted,
				totalErrors: latestDaily.totalErrors,
				activeConnections: Math.round(latestDaily.averageConnections),
			};
		} else if (selectedPeriod === 'weekly' && allMetrics.weeklyMetrics?.length > 0) {
			// Show weekly compressed metrics - take the latest weekly data
			const latestWeekly = allMetrics.weeklyMetrics[allMetrics.weeklyMetrics.length - 1];
			metrics = {
				...allMetrics,
				// Override current metrics with compressed weekly averages
				uploadSpeed: latestWeekly.averageUploadSpeed,
				commandSuccessRate: latestWeekly.averageSuccessRate,
				totalBytesUploaded: latestWeekly.totalBytesUploaded,
				totalArticlesPosted: latestWeekly.totalArticlesPosted,
				totalErrors: latestWeekly.totalErrors,
				activeConnections: Math.round(latestWeekly.averageConnections),
			};
		} else {
			// Show real-time current metrics
			metrics = allMetrics;
		}
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

function formatSpeed(bytesPerSecond: number): string {
	return formatBytes(bytesPerSecond) + "/s";
}

function formatDuration(seconds: number): string {
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	const secs = Math.floor(seconds % 60);
	
	if (hours > 0) {
		return `${hours}h ${minutes}m ${secs}s`;
	} else if (minutes > 0) {
		return `${minutes}m ${secs}s`;
	} else {
		return `${secs}s`;
	}
}

function formatPercentage(value: number): string {
	return (value).toFixed(1) + "%";
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

function handlePeriodChange() {
	// Stop any existing refresh
	stopAutoRefresh();
	
	// Restart refresh only for current metrics
	if (selectedPeriod === 'current') {
		startAutoRefresh();
	}
	
	// Load metrics for the new period
	loadMetrics(true);
}

</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold text-base-content">{$t('metrics.title')}</h1>
			<p class="text-base-content/70 mt-1">{$t('metrics.description')}</p>
		</div>
		
		<div class="flex flex-col sm:flex-row items-start sm:items-center gap-3">
			<!-- Period Selection -->
			<div class="form-control">
				<label for="period-select" class="label">
					<span class="label-text text-sm">{$t('metrics.time_period')}</span>
				</label>
				<select 
					id="period-select"
					class="select select-bordered select-sm w-full sm:w-40"
					bind:value={selectedPeriod}
					onchange={handlePeriodChange}
				>
					<option value="current">
						{$t('metrics.periods.current')}
					</option>
					<option value="daily">
						{$t('metrics.periods.daily')}
					</option>
					<option value="weekly">
						{$t('metrics.periods.weekly')}
					</option>
				</select>
			</div>
			
			<div class="flex items-center gap-3">
				<!-- Auto-refresh indicator (only for current metrics) -->
				{#if selectedPeriod === 'current'}
					<div class="flex items-center gap-2 text-sm text-base-content/60">
						<Activity class="w-4 h-4 {refreshInterval ? 'animate-pulse text-primary' : ''}" />
						<span>{$t('metrics.auto_refresh')}</span>
					</div>
				{:else}
					<div class="flex items-center gap-2 text-sm text-base-content/60">
						<Archive class="w-4 h-4 text-info" />
						<span>{$t('metrics.compressed_data')}</span>
					</div>
				{/if}
				
				<button 
					class="btn btn-primary btn-sm" 
					onclick={() => loadMetrics(true)}
					disabled={loading}
				>
					{loading ? $t('common.common.loading') : $t('metrics.refresh')}
				</button>
			</div>
		</div>
	</div>

	<!-- Compressed Metrics Info Banner -->
	{#if selectedPeriod !== 'current'}
		{@const hasCompressedData = selectedPeriod === 'daily' ? (metrics?.dailyMetrics?.length || 0) > 0 : (metrics?.weeklyMetrics?.length || 0) > 0}
		<div class="alert {hasCompressedData ? 'alert-info' : 'alert-warning'}">
			<div class="flex items-start gap-3">
				<TrendingUp class="w-6 h-6 flex-shrink-0 mt-0.5" />
				<div class="flex-1">
					<h3 class="font-medium">
						{hasCompressedData 
							? $t('metrics.compressed_metrics_title')
							: $t('metrics.no_compressed_data_title')
						}
					</h3>
					<p class="text-sm mt-1 opacity-90">
						{#if hasCompressedData}
							{selectedPeriod === 'daily' 
								? $t('metrics.daily_metrics_description')
								: $t('metrics.weekly_metrics_description')
							}
						{:else}
							{selectedPeriod === 'daily'
								? $t('metrics.no_daily_data_description')  
								: $t('metrics.no_weekly_data_description')
							}
						{/if}
					</p>
					{#if hasCompressedData}
						<div class="mt-2 text-sm">
							<span class="badge badge-info badge-sm">
								{$t('metrics.data_source')}: nntppool v1.3.1+
							</span>
						</div>
					{:else}
						<div class="mt-2 text-sm">
							<span class="badge badge-warning badge-sm">
								{$t('metrics.showing_current_data')}
							</span>
						</div>
					{/if}
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
		<!-- Posting Overview Stats -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<!-- Active Connections -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-primary">
					<Server class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.active_connections')}</div>
				<div class="stat-value text-primary">{metrics.activeConnections}</div>
			</div>

			<!-- Upload Speed -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-success">
					<Upload class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.upload_speed')}</div>
				<div class="stat-value text-success">{formatSpeed(metrics.uploadSpeed)}</div>
			</div>

			<!-- Articles Posted -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-info">
					<FileText class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.articles_posted')}</div>
				<div class="stat-value text-info">{metrics.totalArticlesPosted.toLocaleString()}</div>
			</div>

			<!-- Pool Uptime -->
			<div class="stat bg-base-100 rounded-lg shadow-sm">
				<div class="stat-figure text-secondary">
					<Clock class="w-8 h-8" />
				</div>
				<div class="stat-title">{$t('metrics.pool_uptime')}</div>
				<div class="stat-value text-secondary">{formatDuration(metrics.uptime)}</div>
			</div>
		</div>

		<!-- Posting Performance Metrics -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Posting Success & Error Rates -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body">
					<h2 class="card-title flex items-center gap-2">
						<Zap class="w-5 h-5" />
						{$t('metrics.posting_performance')}
					</h2>
					
					<div class="grid grid-cols-2 gap-4">
						<div class="stat">
							<div class="stat-title">{$t('metrics.posting_success_rate')}</div>
							<div class="stat-value text-success">{formatPercentage(metrics.commandSuccessRate)}</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('metrics.posting_error_rate')}</div>
							<div class="stat-value text-error">{formatPercentage(metrics.errorRate)}</div>
						</div>
					</div>
					
					<div class="mt-4 space-y-2">
						<div class="flex justify-between text-sm">
							<span>{$t('metrics.connection_acquires')}</span>
							<span class="font-mono">{metrics.totalAcquires.toLocaleString()}</span>
						</div>
						<div class="flex justify-between text-sm">
							<span>{$t('metrics.articles_posted')}</span>
							<span class="font-mono">{metrics.totalArticlesPosted.toLocaleString()}</span>
						</div>
						<div class="flex justify-between text-sm">
							<span>{$t('metrics.avg_connection_wait')}</span>
							<span class="font-mono">{metrics.averageAcquireWaitTime.toFixed(2)}ms</span>
						</div>
						<div class="flex justify-between text-sm">
							<span>{$t('metrics.posting_errors')}</span>
							<span class="font-mono text-error">{metrics.totalErrors.toLocaleString()}</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Upload Statistics -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body">
					<h2 class="card-title flex items-center gap-2">
						<Upload class="w-5 h-5" />
						{$t('metrics.upload_statistics')}
					</h2>
					
					<div class="space-y-4">
						<div class="stat">
							<div class="stat-title">{$t('metrics.total_data_posted')}</div>
							<div class="stat-value text-success">{formatBytes(metrics.totalBytesUploaded)}</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('metrics.current_upload_speed')}</div>
							<div class="stat-value text-info">{formatSpeed(metrics.uploadSpeed)}</div>
						</div>
					</div>
					
					<div class="mt-4 space-y-2">
						<div class="flex justify-between text-sm">
							<span>{$t('metrics.average_article_size')}</span>
							<span class="font-mono">
								{metrics.totalArticlesPosted > 0 
									? formatBytes(Math.round(metrics.totalBytesUploaded / metrics.totalArticlesPosted))
									: "0 B"
								}
							</span>
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
								<th>{$t('metrics.posting_success_rate')}</th>
								<th>{$t('metrics.connection_age')}</th>
							</tr>
						</thead>
						<tbody>
							{#each metrics.providers as provider}
								<tr>
									<td>
										<div>
											<div class="font-semibold">{provider.host}</div>
											<div class="text-sm text-base-content/60">{provider.username}</div>
										</div>
									</td>
									<td>
										<div class="badge {getProviderStatusColor(provider.state)} badge-sm">
											{provider.state}
										</div>
									</td>
									<td>
										<div class="text-sm">
											<div>{provider.acquiredConnections}/{provider.maxConnections} active</div>
											<div class="text-base-content/60">{provider.idleConnections} idle</div>
										</div>
									</td>
									<td>
										<div class="text-sm">
											<div class="text-success">↑ {formatBytes(provider.totalBytesUploaded)}</div>
											<div class="text-base-content/60 text-xs">Data posted to server</div>
										</div>
									</td>
									<td>
										<div class="text-sm font-mono {provider.successRate > 0.95 ? 'text-success' : provider.successRate > 0.8 ? 'text-warning' : 'text-error'}">
											{formatPercentage(provider.successRate)}
										</div>
									</td>
									<td>
										<div class="text-sm font-mono">
											{formatDuration(provider.averageConnectionAge)}
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