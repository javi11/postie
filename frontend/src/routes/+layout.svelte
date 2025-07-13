<script lang="ts">
import { goto } from "$app/navigation";
import { page } from "$app/stores";
import apiClient from "$lib/api/client";
import logo from "$lib/assets/images/logo.png";
import ToastContainer from "$lib/components/ToastContainer.svelte";
import { t } from "$lib/i18n";
import { appStatus, settingsSaveFunction } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import {
	FileText,
	Menu,
	Moon,
	PieChart,
	Save,
	Settings,
	Sun,
} from "lucide-svelte";
import { onMount } from "svelte";
import "../style.css";

let needsConfiguration = false;
let criticalConfigError = false;
let isDarkMode = false;

// Dark mode functionality
function toggleDarkMode() {
	isDarkMode = !isDarkMode;
	if (isDarkMode) {
		document.documentElement.setAttribute("data-theme", "dark");
		document.documentElement.classList.add("dark");
	} else {
		document.documentElement.setAttribute("data-theme", "light");
		document.documentElement.classList.remove("dark");
	}
	localStorage.setItem("theme", isDarkMode ? "dark" : "light");
}

async function handleSaveSettings() {
	const saveFunction = $settingsSaveFunction;
	if (saveFunction) {
		await saveFunction();
	}
}

onMount(async () => {
	// Initialize dark mode from localStorage
	const savedTheme = localStorage.getItem("theme");
	isDarkMode =
		savedTheme === "dark" ||
		(!savedTheme && window.matchMedia("(prefers-color-scheme: dark)").matches);
	if (isDarkMode) {
		document.documentElement.setAttribute("data-theme", "dark");
		document.documentElement.classList.add("dark");
	} else {
		document.documentElement.setAttribute("data-theme", "light");
		document.documentElement.classList.remove("dark");
	}

	// Initialize API client (detects environment and sets up appropriate client)
	await apiClient.initialize();

	// Listen for config updates
	await apiClient.on("config-updated", async () => {
		await loadAppStatus();
	});

	// Listen for par2 download events
	await apiClient.on("par2-download-status", (data) => {
		if (data.status === "downloading") {
			toastStore.info($t("common.common.loading"), data.message);
		} else if (data.status === "completed") {
			toastStore.success($t("common.common.success"), data.message);
		} else if (data.status === "error") {
			toastStore.error($t("common.common.error"), data.message);
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
		if (status.isFirstStart && $page.route.id !== "/setup") {
			goto("/setup");
			return;
		}

		// Auto-redirect to settings if there's a critical configuration error
		if (
			criticalConfigError &&
			$page.route.id !== "/settings" &&
			$page.route.id !== "/setup"
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
		if ($page.route.id !== "/setup") {
			goto("/setup");
		}
	}
}
</script>

<div
	class="min-h-screen bg-gradient-to-br from-base-200 to-base-300 overflow-hidden"
>
	<!-- Show navbar only if not on setup page -->
	{#if $page.route.id !== "/setup"}
		<!-- Header/Navigation -->
		<header
			class="navbar bg-base-100/80 backdrop-blur-sm border-b border-base-300/60 sticky top-0 z-50"
		>
			<div class="navbar-start">
				<!-- Logo and Brand -->
				<div class="flex items-center gap-3 px-4">
					<img src={logo} alt="Postie UI" class="w-8 h-8" loading="lazy" />
					<div>
						<div class="flex items-center gap-2">
							<h1 class="text-xl font-bold">
								Postie
							</h1>
						</div>
						<p class="text-xs opacity-60">
							Upload Manager
						</p>
					</div>
				</div>
			</div>

			<div class="navbar-center">
				<!-- Navigation -->
				<div class="flex items-center gap-2">
					<button
						class="btn btn-sm {$page.route.id === "/" ? "btn-primary" : "btn-ghost"}"
						onclick={() => goto("/")}
						disabled={needsConfiguration || criticalConfigError}
						aria-current={$page.route.id === "/" ? "page" : undefined}
					>
						<PieChart class="w-4 h-4" />
						{$t('common.nav.dashboard')}
					</button>
					<button
						class="btn btn-sm {$page.route.id === "/settings" ? "btn-secondary" : "btn-ghost"}"
						onclick={() => goto("/settings")}
						aria-current={$page.route.id === "/settings" ? "page" : undefined}
					>
						<Settings class="w-4 h-4" />
						<span class="hidden md:inline">{$t('common.nav.settings')}</span>
					</button>
					<button
						class="btn btn-sm {$page.route.id === "/logs" ? "btn-secondary" : "btn-ghost"}"
						onclick={() => goto("/logs")}
						aria-current={$page.route.id === "/logs" ? "page" : undefined}
					>
						<FileText class="w-4 h-4" />
						<span class="hidden md:inline">{$t('common.nav.logs')}</span>
					</button>
				</div>
			</div>

			<div class="navbar-end">
				<!-- Dark mode toggle -->
				<button
					class="btn btn-sm btn-ghost btn-circle"
					onclick={toggleDarkMode}
					aria-label="Toggle dark mode"
				>
					{#if isDarkMode}
						<Sun class="w-5 h-5" />
					{:else}
						<Moon class="w-5 h-5" />
					{/if}
				</button>
			</div>
		</header>

		<!-- Page Content -->
		<main class="max-w-7xl mx-auto px-6 py-8">
			<slot />
		</main>
	{:else}
		<!-- Setup page takes full screen -->
		<slot />
	{/if}

	<!-- Toast notifications -->
	<ToastContainer />
</div>
