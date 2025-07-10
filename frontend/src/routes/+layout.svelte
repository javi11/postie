<script lang="ts">
import { goto } from "$app/navigation";
import { page } from "$app/stores";
import logo from "$lib/assets/images/logo.png";
import ToastContainer from "$lib/components/ToastContainer.svelte";
import { t } from "$lib/i18n";
import { appStatus, settingsSaveFunction } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import apiClient from "$lib/api/client";
import {
	Button,
	DarkMode,
	NavBrand,
	NavHamburger,
	NavLi,
	NavUl,
	Navbar,
} from "flowbite-svelte";
import {
	ChartPieSolid,
	CogSolid,
	FileDocOutline,
	FloppyDiskSolid,
} from "flowbite-svelte-icons";
import { onMount } from "svelte";
import "../style.css";

let needsConfiguration = false;
let criticalConfigError = false;

async function handleSaveSettings() {
	const saveFunction = $settingsSaveFunction;
	if (saveFunction) {
		await saveFunction();
	}
}

onMount(async () => {
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

		// Auto-redirect to settings if there's a critical configuration error
		if (criticalConfigError && $page.route.id !== "/settings") {
			toastStore.error(
				$t("common.common.error"),
				$t("common.messages.error_saving"),
			);
			goto("/settings");
		}
	} catch (error) {
		console.error("Failed to load app status:", error);
		// If we can't load app status, assume we need configuration
		needsConfiguration = true;
		criticalConfigError = false;
	}
}
</script>

<div
	class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 overflow-hidden"
>
	<DarkMode class="hidden" />

	<!-- Header/Navigation -->
	<header
		class="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm border-b border-gray-200/60 dark:border-gray-700/60 sticky top-0 z-50"
	>
		<div class="max-w-7xl mx-auto px-6 py-4">
			<div class="flex items-center justify-between">
				<!-- Logo and Brand -->
				<div class="flex items-center gap-3">
					<img src={logo} alt="Postie UI" class="w-8 h-8" loading="lazy" />
					<div>
						<div class="flex items-center gap-2">
							<h1 class="text-xl font-bold text-gray-900 dark:text-white">
								Postie
							</h1>
						</div>
						<p class="text-xs text-gray-500 dark:text-gray-400">
							Upload Manager
						</p>
					</div>
				</div>

				<!-- Navigation -->
				<nav class="flex items-center gap-2">
					<Button
						color={$page.route.id === "/" ? "primary" : "alternative"}
						onclick={() => goto("/")}
						class="cursor-pointer flex items-center gap-2 px-4 py-2 text-sm font-medium transition-all"
						disabled={needsConfiguration || criticalConfigError}
						aria-current={$page.route.id === "/" ? "page" : undefined}
					>
						<ChartPieSolid class="w-4 h-4" />
						{$t('common.nav.dashboard')}
					</Button>
					<Button
						color={$page.route.id === "/settings" ? "secondary" : "gray"}
						onclick={() => goto("/settings")}
						class="cursor-pointer flex items-center text-sm font-medium transition-all"
						aria-current={$page.route.id === "/settings" ? "page" : undefined}
					>
						<CogSolid class="w-4 h-4" />
						<span class="hidden md:inline ml-2">{$t('common.nav.settings')}</span>
					</Button>
					<Button
						color={$page.route.id === "/logs" ? "secondary" : "gray"}
						onclick={() => goto("/logs")}
						class="cursor-pointer flex items-center text-sm font-medium transition-all"
						aria-current={$page.route.id === "/logs" ? "page" : undefined}
					>
						<FileDocOutline class="w-4 h-4" />
						<span class="hidden md:inline ml-2">{$t('common.nav.logs')}</span>
					</Button>
				</nav>
			</div>
		</div>
	</header>

	<!-- Page Content -->
	<main class="max-w-7xl mx-auto px-6 py-8">
		<slot />
	</main>

	<!-- Toast notifications -->
	<ToastContainer />
</div>
