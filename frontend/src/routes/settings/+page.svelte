<script lang="ts">
import { goto } from "$app/navigation";
import GeneralSection from "$lib/components/settings/GeneralSection.svelte";
import Par2Section from "$lib/components/settings/Par2Section.svelte";
import PostingSection from "$lib/components/settings/PostingSection.svelte";
import ServerSection from "$lib/components/settings/ServerSection.svelte";
import SettingsHeader from "$lib/components/settings/SettingsHeader.svelte";
import WatcherSection from "$lib/components/settings/WatcherSection.svelte";
import { type AppStatus, appStatus, settingsSaveFunction } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import { parseDuration, waitForWailsRuntime } from "$lib/utils";
import * as App from "$lib/wailsjs/go/backend/App";
import { config } from "$lib/wailsjs/go/models";
import { onMount, onDestroy } from "svelte";
import { Button, Heading, P, Spinner, DarkMode } from "flowbite-svelte";
import {
	CheckCircleSolid,
	CogSolid,
	ExclamationCircleOutline,
	FloppyDiskSolid,
	RefreshOutline,
} from "flowbite-svelte-icons";

let configData: ConfigData | null = null;
let localConfig: ConfigData | null = null;
let needsConfiguration = false;
let criticalConfigError = false;
let loading = true;
let loadError = false;

onMount(async () => {
	// Wait for Wails runtime to be ready
	await waitForWailsRuntime();
	await loadConfig();

	// Register save function with the store
	settingsSaveFunction.set(handleSaveConfig);

	// Subscribe to app status
	const unsubscribe = appStatus.subscribe((status: AppStatus) => {
		needsConfiguration = status.needsConfiguration;
		criticalConfigError = status.criticalConfigError;
	});

	return unsubscribe;
});

async function loadConfig() {
	try {
		loading = true;
		loadError = false;
		const config = await App.GetConfig();
		configData = config;
		// Initialize localConfig with the loaded config
		localConfig = JSON.parse(JSON.stringify(config));

		// If no servers exist, add a default one to make it easier for the user
		if (!localConfig.servers || localConfig.servers.length === 0) {
			localConfig.servers = [
				{
					host: "",
					port: 119,
					username: "",
					password: "",
					ssl: false,
					max_connections: 10,
					max_connection_idle_time_in_seconds: 300,
					max_connection_ttl_in_seconds: 3600,
					insecure_ssl: false,
				},
			];
		}
	} catch (error) {
		console.error("Failed to load config:", error);
		loadError = true;
		configData = null;
		localConfig = null;
	} finally {
		loading = false;
	}
}

async function handleSaveConfig() {
	try {
		// Validate that at least one server is configured
		if (!localConfig.servers || localConfig.servers.length === 0) {
			toastStore.error("Configuration Error", "No servers configured");
			return;
		}

		// Validate that all servers have required fields
		for (let i = 0; i < localConfig.servers.length; i++) {
			const server = localConfig.servers[i];
			if (!server.host || server.host.trim() === "") {
				toastStore.error(
					"Configuration Error",
					`Server ${i + 1}: Host is required`,
				);
				return;
			}
			if (!server.port || server.port <= 0 || server.port > 65535) {
				toastStore.error(
					"Configuration Error",
					`Server ${i + 1}: Valid port number is required (1-65535)`,
				);
				return;
			}
		}

		// Copy the config to a new object to avoid modifying the original
		const configToSave = JSON.parse(JSON.stringify(localConfig));

		// Convert server integer fields
		configToSave.servers = configToSave.servers.map((server: ServerConfig) => ({
			...server,
			port: Number.parseInt(server.port) || 119,
			max_connections: Number.parseInt(server.max_connections) || 10,
			max_connection_idle_time_in_seconds:
				Number.parseInt(server.max_connection_idle_time_in_seconds) || 300,
			max_connection_ttl_in_seconds:
				Number.parseInt(server.max_connection_ttl_in_seconds) || 3600,
		}));

		// Convert posting fields
		configToSave.posting.max_retries =
			Number.parseInt(configToSave.posting.max_retries) || 3;
		configToSave.posting.article_size_in_bytes =
			Number.parseInt(configToSave.posting.article_size_in_bytes) || 750000;

		// Convert duration fields to nanoseconds
		configToSave.posting.retry_delay ??= "5s";

		// Convert par2 integer fields
		configToSave.par2.volume_size =
			Number.parseInt(configToSave.par2.volume_size) || 153600000;
		configToSave.par2.max_input_slices =
			Number.parseInt(configToSave.par2.max_input_slices) || 4000;

		// Convert watcher fields
		if (configToSave.watcher) {
			configToSave.watcher.size_threshold =
				Number.parseInt(configToSave.watcher.size_threshold) || 104857600;
			configToSave.watcher.min_file_size =
				Number.parseInt(configToSave.watcher.min_file_size) || 1048576;

			configToSave.watcher.check_interval ??= "5m";
		}

		// Ensure output_dir is set
		if (!configToSave.output_dir || configToSave.output_dir.trim() === "") {
			configToSave.output_dir = "./output";
		}

		await App.SaveConfig(configToSave);
		configData = configToSave;
		// Update localConfig to the saved config to maintain reactivity
		localConfig = JSON.parse(JSON.stringify(configToSave));

		// Update app status
		const status = await App.GetAppStatus();
		appStatus.set(status);

		toastStore.success(
			"Configuration saved",
			"Your configuration has been saved successfully!",
		);

		// Redirect to dashboard if configuration was needed
		if (needsConfiguration) {
			goto("/");
		}
	} catch (error) {
		console.error("Failed to save config:", error);
		toastStore.error("Configuration save failed", String(error));
	}
}

onDestroy(() => {
	// Clean up when the component is destroyed
	settingsSaveFunction.set(null);
});
</script>

<svelte:head>
  <title>Settings - Postie</title>
  <meta name="description" content="Configure your upload settings" />
</svelte:head>

<div class="space-y-6">
  <!-- Main header section (not sticky) -->
  <div class="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
    <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
      <div class="flex-1">
        <div class="flex items-center gap-3 mb-2">
          <CogSolid class="w-6 h-6 text-gray-600 dark:text-gray-400" />
          <Heading tag="h1" class="text-2xl font-bold text-gray-900 dark:text-white">
            Settings
          </Heading>
          {#if criticalConfigError}
            <div class="flex items-center gap-2 px-3 py-1 bg-red-100 dark:bg-red-900/30 rounded-full">
              <ExclamationCircleOutline class="w-4 h-4 text-red-600 dark:text-red-400" />
              <span class="text-sm font-medium text-red-800 dark:text-red-200">Configuration Error</span>
            </div>
          {:else if needsConfiguration}
            <div class="flex items-center gap-2 px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 rounded-full">
              <ExclamationCircleOutline class="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
              <span class="text-sm font-medium text-yellow-800 dark:text-yellow-200">Configuration Required</span>
            </div>
          {:else}
            <div class="flex items-center gap-2 px-3 py-1 bg-green-100 dark:bg-green-900/30 rounded-full">
              <CheckCircleSolid class="w-4 h-4 text-green-600 dark:text-green-400" />
              <span class="text-sm font-medium text-green-800 dark:text-green-200">Configured</span>
            </div>
          {/if}
        </div>

        <P class="text-gray-600 dark:text-gray-400">
          Configure your upload servers, posting settings, and PAR2 options.
        </P>

        <!-- Dark Mode Toggle -->
        <div class="flex items-center gap-3 mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Theme:</span>
          <DarkMode
            class="cursor-pointer text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 rounded-lg text-sm p-2.5 transition-all"
          />
        </div>

        {#if criticalConfigError}
          <div class="mt-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <P class="text-red-800 dark:text-red-200">
              <strong>Configuration Error:</strong> There was an error with your server
              configuration (e.g., invalid hostname like "Locahost", connection failure).
              Please check and fix your server settings below, then click "Save Configuration".
            </P>
          </div>
        {:else if needsConfiguration}
          <div class="mt-4 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
            <P class="text-yellow-800 dark:text-yellow-200">
              <strong>Setup Required:</strong> Please configure at least one server
              to start uploading files. All settings are saved automatically when you
              click "Save Configuration".
            </P>
          </div>
        {/if}
      </div>
    </div>
  </div>

  {#if loading}
    <div class="flex items-center justify-center py-12">
      <div class="text-center">
        <Spinner class="mb-4 w-8 h-8 mx-auto" />
        <P class="text-gray-600 dark:text-gray-400">Loading configuration...</P>
      </div>
    </div>
  {:else if loadError}
    <div class="flex items-center justify-center py-12">
      <div class="text-center max-w-md">
        <ExclamationCircleOutline class="mb-4 w-12 h-12 mx-auto text-red-500 dark:text-red-400" />
        <Heading tag="h3" class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          Failed to Load Configuration
        </Heading>
        <P class="mb-4 text-gray-600 dark:text-gray-400">
          There was an error loading the configuration from the server. 
          Please check your connection and try again.
        </P>
        <Button color="primary" on:click={loadConfig}>
          <RefreshOutline class="w-4 h-4 mr-2" />
          Retry
        </Button>
      </div>
    </div>
  {:else if localConfig}
    <div class="grid gap-3 md:grid-cols-1 lg:grid-cols-2">
      <div class="space-y-6">
        <GeneralSection bind:config={localConfig} />
        <ServerSection bind:config={localConfig} />
        <PostingSection bind:config={localConfig} />
      </div>

      <div class="space-y-6">
        <Par2Section bind:config={localConfig} />
        <WatcherSection bind:config={localConfig} />
      </div>
    </div>
  {:else}
    <div class="flex items-center justify-center py-12">
      <div class="text-center">
        <P class="text-gray-600 dark:text-gray-400">No configuration available.</P>
      </div>
    </div>
  {/if}
</div>
