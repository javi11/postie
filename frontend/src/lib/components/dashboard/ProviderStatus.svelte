<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { backend } from "$lib/wailsjs/go/models";
import { CheckCircle, Clock, AlertCircle, Server, WifiOff } from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let poolMetrics = $state<backend.NntpPoolMetrics | null>(null);
let initialLoad = $state(true);
let error = $state("");
let refreshInterval: NodeJS.Timeout | null = null;

// Refresh every 5 seconds
const REFRESH_INTERVAL = 5000;

async function fetchProviderStatus() {
	try {
		error = "";
		poolMetrics = await apiClient.getNntpPoolMetrics();
	} catch (err) {
		console.error("Failed to fetch provider status:", err);
		error = String(err);
		poolMetrics = null;
	} finally {
		initialLoad = false;
	}
}

function getProviderStatusIcon(provider: backend.NntpProviderMetrics) {
	switch (provider.state?.toLowerCase()) {
		case "connected":
		case "active":
			return CheckCircle;
		case "connecting":
		case "reconnecting":
			return Clock;
		case "failed":
		case "error":
			return AlertCircle;
		case "disconnected":
		case "offline":
		default:
			return WifiOff;
	}
}

function getProviderStatusClass(provider: backend.NntpProviderMetrics) {
	switch (provider.state?.toLowerCase()) {
		case "connected":
		case "active":
			return "text-success";
		case "connecting":
		case "reconnecting":
			return "text-warning";
		case "failed":
		case "error":
			return "text-error";
		case "disconnected":
		case "offline":
		default:
			return "text-base-content/50";
	}
}

function getProviderStatusText(provider: backend.NntpProviderMetrics) {
	const state = provider.state?.toLowerCase() || "unknown";
	switch (state) {
		case "connected":
		case "active":
			return $t("dashboard.provider.status.connected");
		case "connecting":
			return $t("dashboard.provider.status.connecting");
		case "reconnecting":
			return $t("dashboard.provider.status.reconnecting");
		case "failed":
		case "error":
			return $t("dashboard.provider.status.failed");
		case "disconnected":
		case "offline":
			return $t("dashboard.provider.status.disconnected");
		default:
			return $t("dashboard.provider.status.unknown");
	}
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return "0 B";
	const k = 1024;
	const sizes = ["B", "KB", "MB", "GB", "TB"];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

onMount(() => {
	fetchProviderStatus();

	// Set up auto-refresh
	refreshInterval = setInterval(fetchProviderStatus, REFRESH_INTERVAL);
});

onDestroy(() => {
	if (refreshInterval) {
		clearInterval(refreshInterval);
	}
});
</script>

<div class="card bg-base-100 shadow-sm">
	<div class="card-body">
		<h2 class="card-title text-base-content flex items-center gap-2">
			<Server class="w-5 h-5" />
			{$t("dashboard.provider.title")}
		</h2>

		{#if initialLoad && !poolMetrics}
			<div class="flex justify-center py-4">
				<span class="loading loading-spinner loading-md"></span>
			</div>
		{:else if error}
			<div class="alert alert-error">
				<AlertCircle class="w-4 h-4" />
				<span>{$t("dashboard.provider.error")}: {error}</span>
			</div>
		{:else if poolMetrics?.providers && poolMetrics.providers.length > 0}
			<div class="space-y-3">
				{#each poolMetrics.providers as provider (provider.host)}
					{@const StatusIcon = getProviderStatusIcon(provider)}
					<div class="border border-base-300 rounded-lg p-4">
						<div class="flex items-center justify-between mb-3">
							<div class="flex items-center gap-3">
								<StatusIcon class="w-5 h-5 {getProviderStatusClass(provider)}" />
								<div>
									<h3 class="font-semibold text-base-content">
										{provider.host}
									</h3>
								</div>
							</div>
							<div class="text-right">
								<div class="text-sm font-medium {getProviderStatusClass(provider)}">
									{getProviderStatusText(provider)}
								</div>
								<div class="text-xs text-base-content/60">
									{$t("dashboard.provider.connections")}: {provider.activeConnections}/{provider.maxConnections}
								</div>
							</div>
						</div>

						{#if provider.state?.toLowerCase() === "connected" || provider.state?.toLowerCase() === "active"}
							<div class="grid grid-cols-3 gap-4 text-sm">
								<div>
									<span class="text-base-content/70">{$t("dashboard.provider.uploaded")}:</span>
									<span class="font-medium ml-1">{formatBytes(provider.totalBytesUploaded)}</span>
								</div>
								<div>
									<span class="text-base-content/70">{$t("dashboard.provider.articles_posted")}:</span>
									<span class="font-medium ml-1">{provider.totalArticlesPosted.toLocaleString()}</span>
								</div>
								<div>
									<span class="text-base-content/70">{$t("dashboard.provider.errors")}:</span>
									<span class="font-medium ml-1 {provider.totalErrors > 0 ? 'text-error' : ''}">{provider.totalErrors.toLocaleString()}</span>
								</div>
							</div>
						{/if}
					</div>
				{/each}
			</div>

			<!-- Pool Summary -->
			<div class="mt-4 p-3 bg-base-200 rounded-lg">
				<div class="flex justify-between items-center text-sm">
					<span class="text-base-content/70">{$t("dashboard.provider.total_uploaded")}:</span>
					<span class="font-medium">{formatBytes(poolMetrics.totalBytesUploaded)}</span>
				</div>
				<div class="flex justify-between items-center text-sm mt-1">
					<span class="text-base-content/70">{$t("dashboard.provider.total_articles")}:</span>
					<span class="font-medium">{poolMetrics.totalArticlesPosted.toLocaleString()}</span>
				</div>
				<div class="flex justify-between items-center text-sm mt-1">
					<span class="text-base-content/70">{$t("dashboard.provider.total_errors")}:</span>
					<span class="font-medium">{poolMetrics.totalErrors.toLocaleString()}</span>
				</div>
			</div>
		{:else}
			<div class="text-center py-8 text-base-content/60">
				<WifiOff class="w-12 h-12 mx-auto mb-2 opacity-50" />
				<p>{$t("dashboard.provider.no_providers")}</p>
			</div>
		{/if}
	</div>
</div>
