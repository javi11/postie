<script lang="ts">
import { goto } from "$app/navigation";
import { page } from "$app/state";
import apiClient from "$lib/api/client";
import logo from "$lib/assets/images/logo.png";
import ToastContainer from "$lib/components/ToastContainer.svelte";
import { t } from "$lib/i18n";
import { appStatus } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { Par2DownloadStatus } from "$lib/types";
import { ChartPie, FileText, Settings, Activity } from "lucide-svelte";
import { onMount } from "svelte";
import "../style.css";

let needsConfiguration = false;
let criticalConfigError = false;

onMount(async () => {
	// Initialize theme from localStorage or system preference
	const savedTheme = localStorage.getItem("theme");
	const systemPrefersDark = window.matchMedia(
		"(prefers-color-scheme: dark)",
	).matches;
	const defaultTheme = savedTheme || (systemPrefersDark ? "dark" : "light");
	document.documentElement.setAttribute("data-theme", defaultTheme);

	// Initialize API client (detects environment and sets up appropriate client)
	await apiClient.initialize();

	// Listen for config updates
	await apiClient.on("config-updated", async () => {
		await loadAppStatus();
	});

	// Listen for par2 download events
	await apiClient.on("par2-download-status", (data) => {
		const d = data as Par2DownloadStatus;
		if (d.status === "downloading") {
			toastStore.info($t("common.common.loading"), d.message);
		} else if (d.status === "completed") {
			toastStore.success($t("common.common.success"), d.message);
		} else if (d.status === "error") {
			toastStore.error($t("common.common.error"), d.message);
		}
	});

	// Listen for menu navigation events (desktop only)
	if (apiClient.environment === "wails") {
		await apiClient.on("navigate-to-settings", () => {
			goto("/settings");
		});

		await apiClient.on("navigate-to-dashboard", () => {
			goto("/");
		});

		// Listen for edit menu events (desktop only)
		await apiClient.on("menu-cut", () => {
			document.execCommand("cut");
		});

		await apiClient.on("menu-copy", () => {
			document.execCommand("copy");
		});

		await apiClient.on("menu-paste", () => {
			document.execCommand("paste");
		});

		await apiClient.on("menu-undo", () => {
			document.execCommand("undo");
		});

		await apiClient.on("menu-redo", () => {
			document.execCommand("redo");
		});

		await apiClient.on("menu-select-all", () => {
			document.execCommand("selectAll");
		});
	}

	// Load initial app status
	await loadAppStatus();
});

async function loadAppStatus() {
	try {
		const status = await apiClient.getAppStatus();
		appStatus.set(status);
		needsConfiguration = status.needsConfiguration || false;
		criticalConfigError = status.criticalConfigError || false;

		// Force redirect to setup wizard if this is first start and not already on setup page
		if (status.isFirstStart && page.route.id !== "/setup") {
			goto("/setup");
			return;
		}

		// Auto-redirect to settings if there's a critical configuration error
		if (
			criticalConfigError &&
			page.route.id !== "/settings" &&
			page.route.id !== "/setup"
		) {
			toastStore.error(
				$t("common.common.error"),
				$t("common.messages.error_saving"),
			);
			goto("/settings");
		}
	} catch (error) {
		console.error("Failed to load app status:", error);
		// If we can't load app status, redirect to setup to be safe
		if (page.route.id !== "/setup") {
			goto("/setup");
		}
	}
}

function handler(error: unknown, _reset: () => void) {
	const err = error as Error;
	console.error("Error in layout:", err);
	toastStore.error($t("common.common.error"), err.message);
}
</script>

<div class="min-h-screen bg-base-200">
	<svelte:boundary onerror={handler}>
		<!-- Show navbar only if not on setup page -->
		{#if page.route.id !== "/setup"}
			<!-- Header/Navigation -->
			<div class="navbar bg-base-100/95 backdrop-blur-md shadow-lg border-b border-base-300/50 sticky top-0 z-50">
				<div class="navbar-start">
					<!-- Logo and Brand -->
					<div class="flex items-center gap-2 px-2 md:gap-3 md:px-4">
						<div class="avatar">
							<div class="w-8 h-8 md:w-12 md:h-10">
								<img src={logo} alt="Postie UI" class="w-full h-full object-contain" loading="lazy" />
							</div>
						</div>
						<div class="hidden sm:block">
							<div class="flex items-center gap-2">
								<h1 class="text-lg md:text-xl font-bold bg-clip-text">
									Postie
								</h1>
							</div>
							<p class="text-xs text-base-content/60">
								Upload Manager
							</p>
						</div>
					</div>
				</div>

				<div class="navbar-center">
					<!-- Navigation -->
					<div class="flex items-center gap-1">
						<button
							class="btn btn-xs sm:btn-sm {page.route.id === "/" ? "btn-primary shadow-lg" : "btn-ghost hover:bg-base-200"} transition-all duration-200"
							onclick={() => goto("/")}
							disabled={needsConfiguration || criticalConfigError}
							aria-current={page.route.id === "/" ? "page" : undefined}
							title={$t('common.nav.dashboard')}
						>
							<ChartPie class="w-4 h-4" />
							<span class="hidden sm:inline font-medium">{$t('common.nav.dashboard')}</span>
						</button>
						<button
							class="btn btn-xs sm:btn-sm {page.route.id === "/settings" ? "btn-secondary shadow-lg" : "btn-ghost hover:bg-base-200"} transition-all duration-200"
							onclick={() => goto("/settings")}
							aria-current={page.route.id === "/settings" ? "page" : undefined}
							title={$t('common.nav.settings')}
						>
							<Settings class="w-4 h-4" />
							<span class="hidden sm:inline font-medium">{$t('common.nav.settings')}</span>
						</button>
						<button
							class="btn btn-xs sm:btn-sm {page.route.id === "/metrics" ? "btn-info shadow-lg" : "btn-ghost hover:bg-base-200"} transition-all duration-200"
							onclick={() => goto("/metrics")}
							disabled={needsConfiguration || criticalConfigError}
							aria-current={page.route.id === "/metrics" ? "page" : undefined}
							title={$t('common.nav.metrics')}
						>
							<Activity class="w-4 h-4" />
							<span class="hidden sm:inline font-medium">{$t('common.nav.metrics')}</span>
						</button>
						<button
							class="btn btn-xs sm:btn-sm {page.route.id === "/logs" ? "btn-accent shadow-lg" : "btn-ghost hover:bg-base-200"} transition-all duration-200"
							onclick={() => goto("/logs")}
							aria-current={page.route.id === "/logs" ? "page" : undefined}
							title={$t('common.nav.logs')}
						>
							<FileText class="w-4 h-4" />
							<span class="hidden sm:inline font-medium">{$t('common.nav.logs')}</span>
						</button>
					</div>
				</div>

				<div class="navbar-end">
					<!-- Empty for now, can be used for user menu or other actions -->
				</div>
			</div>

			<!-- Page Content -->
			<main class="container mx-auto px-4 py-8 max-w-7xl animate-fade-in">
				<slot />
			</main>
		{:else}
			<!-- Setup page takes full screen -->
			<slot />
		{/if}
	</svelte:boundary>

	<!-- Toast notifications -->
	<ToastContainer />
</div>
