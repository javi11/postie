<script lang="ts">
import { goto } from "$app/navigation";
import { page } from "$app/stores";
import logo from "$lib/assets/images/logo.png";
import ToastContainer from "$lib/components/ToastContainer.svelte";
import { appStatus, settingsSaveFunction } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { waitForWailsRuntime } from "$lib/utils";
import * as App from "$lib/wailsjs/go/backend/App";
import { EventsOn } from "$lib/wailsjs/runtime/runtime";
import {
	Button,
	DarkMode,
	NavBrand,
	NavHamburger,
	NavLi,
	NavUl,
	Navbar,
} from "flowbite-svelte";
import { ChartPieSolid, CogSolid, FloppyDiskSolid } from "flowbite-svelte-icons";
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
	// Wait for Wails runtime to be ready
	await waitForWailsRuntime();

	// Listen for config updates
	EventsOn("config-updated", async () => {
		await loadAppStatus();
	});

	// Listen for par2 download events
	EventsOn("par2-download-status", (data) => {
		if (data.status === "downloading") {
			toastStore.info("Downloading Dependencies", data.message);
		} else if (data.status === "completed") {
			toastStore.success("Dependencies Ready", data.message);
		} else if (data.status === "error") {
			toastStore.error("Download Failed", data.message);
		}
	});

	// Load initial app status
	await loadAppStatus();
});

async function loadAppStatus() {
	try {
		const status = await App.GetAppStatus();
		appStatus.set(status);
		needsConfiguration = status.needsConfiguration || false;
		criticalConfigError = status.criticalConfigError || false;

		// Auto-redirect to settings if there's a critical configuration error
		if (criticalConfigError && $page.route.id !== "/settings") {
			toastStore.error(
				"Configuration Error",
				"There was an error with your server configuration. Please check your settings.",
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
  class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800"
>
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
            <h1 class="text-xl font-bold text-gray-900 dark:text-white">
              Postie
            </h1>
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
            Dashboard
          </Button>
          <Button
            color={$page.route.id === "/settings" ? "primary" : "alternative"}
            onclick={() => goto("/settings")}
            class="cursor-pointer flex items-center gap-2 px-4 py-2 text-sm font-medium transition-all"
            aria-current={$page.route.id === "/settings" ? "page" : undefined}
          >
            <CogSolid class="w-4 h-4" />
            Settings
          </Button>
          
          {#if $page.route.id === "/settings" && $settingsSaveFunction}
            <Button
              color="green"
              onclick={handleSaveSettings}
              class="cursor-pointer flex items-center gap-2 px-4 py-2 text-sm font-medium transition-all ml-2"
            >
              <FloppyDiskSolid class="w-4 h-4" />
              Save Configuration
            </Button>
          {/if}
          
          <div class="ml-4 pl-4 border-l border-gray-200 dark:border-gray-700">
            <DarkMode
              class="cursor-pointer text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 rounded-lg text-sm p-2.5 transition-all"
            />
          </div>
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
