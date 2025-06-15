<script lang="ts">
import { goto } from "$app/navigation";
import GeneralSection from "$lib/components/settings/GeneralSection.svelte";
import Par2Section from "$lib/components/settings/Par2Section.svelte";
import PostingSection from "$lib/components/settings/PostingSection.svelte";
import ServerSection from "$lib/components/settings/ServerSection.svelte";
import SettingsHeader from "$lib/components/settings/SettingsHeader.svelte";
import WatcherSection from "$lib/components/settings/WatcherSection.svelte";
import { appStatus } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import { parseDuration } from "$lib/utils";
import * as App from "$lib/wailsjs/go/backend/App";
import { config } from "$lib/wailsjs/go/models";
import { onMount } from "svelte";

let configData: ConfigData = {
	servers: [],
	posting: {
		max_retries: 3,
		retry_delay: "5s",
		article_size_in_bytes: 768000,
		groups: ["alt.binaries.test"],
		obfuscation_policy: "none",
	},
	par2: {
		enabled: true,
		par2_path: "./parpar",
		redundancy: "10%",
		volume_size: 200000000,
		max_input_slices: 4000,
	},
	watcher: {
		enabled: false,
		size_threshold: 104857600, // 100MB
		schedule: {
			start_time: "00:00",
			end_time: "23:59",
		},
		ignore_patterns: ["*.tmp", "*.part", "*.!ut"],
		min_file_size: 1048576, // 1MB
		check_interval: 300000000000, // 5m in nanoseconds
	},
	output_dir: "./output",
};

let localConfig: ConfigData;
let needsConfiguration = false;
let criticalConfigError = false;

onMount(async () => {
	await loadConfig();

	// Subscribe to app status
	const unsubscribe = appStatus.subscribe((status: AppStatus) => {
		needsConfiguration = status.needsConfiguration;
		criticalConfigError = status.criticalConfigError;
	});

	return unsubscribe;
});

async function loadConfig() {
	try {
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

		// Convert string values to numbers for integer fields
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
		configToSave.posting.retry_delay =
			typeof configToSave.posting.retry_delay === "string"
				? parseDuration(configToSave.posting.retry_delay) || 5000000000 // default 5s
				: configToSave.posting.retry_delay || 5000000000;

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

			// Convert check_interval to nanoseconds if it's a string
			if (typeof configToSave.watcher.check_interval === "string") {
				configToSave.watcher.check_interval =
					parseDuration(configToSave.watcher.check_interval) || 300000000000; // default 5m
			} else {
				configToSave.watcher.check_interval =
					configToSave.watcher.check_interval || 300000000000;
			}
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

async function handleSelectConfigFile() {
	try {
		const file = await App.SelectConfigFile();
		if (file) {
			const config = await App.GetConfig();
			configData = config;
			// Update localConfig to the loaded config
			localConfig = JSON.parse(JSON.stringify(config));

			// Update app status
			const status = await App.GetAppStatus();
			appStatus.set(status);

			toastStore.success("Configuration loaded", `Config loaded from: ${file}`);
		}
	} catch (error) {
		console.error("Failed to select config file:", error);
		toastStore.error("Failed to load config file", String(error));
	}
}
</script>

<svelte:head>
  <title>Settings - Postie</title>
  <meta name="description" content="Configure your upload settings" />
</svelte:head>

<div class="space-y-6">
  <SettingsHeader
    {needsConfiguration}
    {criticalConfigError}
    on:save={handleSaveConfig}
    on:selectFile={handleSelectConfigFile}
  />

  {#if localConfig}
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
  {/if}
</div>
