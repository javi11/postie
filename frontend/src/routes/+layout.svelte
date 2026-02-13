<script lang="ts">
import { goto } from "$app/navigation";
import { page } from "$app/state";
import apiClient from "$lib/api/client";
import logo from "$lib/assets/images/logo.png";
import ToastContainer from "$lib/components/ToastContainer.svelte";
import { t } from "$lib/i18n";
import { appStatus } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { ChartPie, FileText, Settings, Activity, Palette, Globe } from "lucide-svelte";
import { availableThemes, currentTheme, type ThemeValue } from "$lib/stores/theme";
import { availableLocales, locale, setStoredLocale } from "$lib/i18n";
import { onMount, onDestroy } from "svelte";
import "../style.css";

let needsConfiguration = $state(false);
let criticalConfigError = $state(false);
let connectionStatus: "connected" | "reconnecting" | "disconnected" = $state("connected");
let connectionCheckInterval: ReturnType<typeof setInterval> | undefined;

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

	// Periodic connection health check
	connectionCheckInterval = setInterval(async () => {
		try {
			await apiClient.getAppStatus();
			if (connectionStatus !== "connected") connectionStatus = "connected";
		} catch {
			connectionStatus = connectionStatus === "connected" ? "reconnecting" : "disconnected";
		}
	}, 10000);
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

onDestroy(() => {
	if (connectionCheckInterval) clearInterval(connectionCheckInterval);
});

function handleKeydown(event: KeyboardEvent) {
	// Skip if user is typing in an input
	const target = event.target as HTMLElement;
	if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.tagName === "SELECT" || target.isContentEditable) return;

	const mod = event.metaKey || event.ctrlKey;
	if (!mod) return;

	const routes: Record<string, string> = {
		"1": "/",
		"2": "/settings",
		"3": "/metrics",
		"4": "/logs",
	};

	const route = routes[event.key];
	if (route) {
		event.preventDefault();
		goto(route);
	}
}

function handler(error: unknown, _reset: () => void) {
	const err = error as Error;
	console.error("Error in layout:", err);
	toastStore.error($t("common.common.error"), err.message);
}
</script>

<svelte:window onkeydown={handleKeydown} />

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

				<div class="navbar-end gap-1.5">
					<div class="tooltip tooltip-bottom" data-tip={connectionStatus === "connected" ? $t("common.common.status") + ": OK" : connectionStatus === "reconnecting" ? $t("common.common.loading") : $t("common.common.error")}>
						<div class="w-2.5 h-2.5 rounded-full {connectionStatus === 'connected' ? 'bg-success' : connectionStatus === 'reconnecting' ? 'bg-warning animate-pulse' : 'bg-error animate-pulse'}"></div>
					</div>

					<!-- Quick Theme Switcher -->
					<div class="dropdown dropdown-end">
						<div tabindex="0" role="button" class="btn btn-ghost btn-xs sm:btn-sm" title={$t("common.nav.theme")}>
							<Palette class="w-4 h-4" />
						</div>
						<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
						<ul tabindex="0" class="dropdown-content menu bg-base-200 rounded-box z-50 w-40 p-2 shadow-lg">
							{#each availableThemes as theme}
								<li>
									<button
										class:active={$currentTheme === theme.value}
										onclick={() => currentTheme.setTheme(theme.value)}
									>
										{theme.name}
									</button>
								</li>
							{/each}
						</ul>
					</div>

					<!-- Quick Language Switcher -->
					<div class="dropdown dropdown-end">
						<div tabindex="0" role="button" class="btn btn-ghost btn-xs sm:btn-sm" title={$t("common.nav.language")}>
							<Globe class="w-4 h-4" />
						</div>
						<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
						<ul tabindex="0" class="dropdown-content menu bg-base-200 rounded-box z-50 w-40 p-2 shadow-lg">
							{#each availableLocales as lang}
								<li>
									<button
										class:active={$locale === lang.code}
										onclick={() => { setStoredLocale(lang.code); locale.set(lang.code); }}
									>
										{lang.flag} {lang.name}
									</button>
								</li>
							{/each}
						</ul>
					</div>

					{#if $appStatus?.version}
						<span class="badge badge-ghost text-xs opacity-70 hidden sm:inline-flex">
							{$appStatus.version}
						</span>
					{/if}
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
