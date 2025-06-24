<script lang="ts">
import { goto } from "$app/navigation";
import GeneralSection from "$lib/components/settings/GeneralSection.svelte";
import NzbCompressionSection from "$lib/components/settings/NzbCompressionSection.svelte";
import Par2Section from "$lib/components/settings/Par2Section.svelte";
import PostCheckSection from "$lib/components/settings/PostCheckSection.svelte";
import PostUploadScriptSection from "$lib/components/settings/PostUploadScriptSection.svelte";
import PostingSection from "$lib/components/settings/PostingSection.svelte";
import ServerSection from "$lib/components/settings/ServerSection.svelte";
import SettingsHeader from "$lib/components/settings/SettingsHeader.svelte";
import WatcherSection from "$lib/components/settings/WatcherSection.svelte";
import { t } from "$lib/i18n";
import {
	type AppStatus,
	appStatus,
	settingsSaveFunction,
} from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import { parseDuration, waitForWailsRuntime } from "$lib/utils";
import * as App from "$lib/wailsjs/go/backend/App";
import { config } from "$lib/wailsjs/go/models";
import {
	Button,
	DarkMode,
	Heading,
	P,
	Spinner,
	TabItem,
	Tabs,
} from "flowbite-svelte";
import {
	CheckCircleSolid,
	CloudArrowUpSolid,
	CogSolid,
	ExclamationCircleOutline,
	EyeSolid,
	FileSolid,
	FloppyDiskSolid,
	RefreshOutline,
} from "flowbite-svelte-icons";
import { onDestroy, onMount } from "svelte";

let configData: ConfigData | null = null;
let localConfig: ConfigData | null = null;
let needsConfiguration = false;
let criticalConfigError = false;
let criticalConfigErrorMessage = "";
let loading = false;
let loadError = false;
onMount(async () => {
	await waitForWailsRuntime();
	await loadConfig();

	// Register save function with the store
	settingsSaveFunction.set(handleSaveConfig);

	// Subscribe to app status
	const unsubscribe = appStatus.subscribe((status: AppStatus) => {
		needsConfiguration = status.needsConfiguration;
		criticalConfigError = status.criticalConfigError;
		criticalConfigErrorMessage = status.error;
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
			toastStore.error(
				$t("common.messages.configuration_error"),
				$t("common.messages.no_servers_configured"),
			);
			return;
		}

		// Validate that all servers have required fields
		for (let i = 0; i < localConfig.servers.length; i++) {
			const server = localConfig.servers[i];
			if (!server.host || server.host.trim() === "") {
				toastStore.error(
					$t("common.messages.configuration_error"),
					$t("common.messages.server_host_required", { number: i + 1 }),
				);
				return;
			}
			if (!server.port || server.port <= 0 || server.port > 65535) {
				toastStore.error(
					$t("common.messages.configuration_error"),
					$t("common.messages.server_port_required", { number: i + 1 }),
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
			$t("common.messages.configuration_saved"),
			$t("common.messages.configuration_saved_description"),
		);

		// Redirect to dashboard if configuration was needed
		if (needsConfiguration) {
			goto("/");
		}
	} catch (error) {
		console.error("Failed to save config:", error);
		toastStore.error(
			$t("common.messages.configuration_save_failed"),
			String(error),
		);
	}
}

onDestroy(() => {
	// Clean up when the component is destroyed
	settingsSaveFunction.set(null);
});
</script>

<svelte:head>
  <title>{$t('settings.title')} - Postie</title>
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
            {$t('settings.title')}
          </Heading>
          {#if criticalConfigError}
            <div class="flex items-center gap-2 px-3 py-1 bg-red-100 dark:bg-red-900/30 rounded-full">
              <ExclamationCircleOutline class="w-4 h-4 text-red-600 dark:text-red-400" />
              <span class="text-sm font-medium text-red-800 dark:text-red-200">{$t('settings.header.status.configuration_error')}</span>
            </div>
          {:else if needsConfiguration}
            <div class="flex items-center gap-2 px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 rounded-full">
              <ExclamationCircleOutline class="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
              <span class="text-sm font-medium text-yellow-800 dark:text-yellow-200">{$t('settings.header.status.configuration_required')}</span>
            </div>
          {:else}
            <div class="flex items-center gap-2 px-3 py-1 bg-green-100 dark:bg-green-900/30 rounded-full">
              <CheckCircleSolid class="w-4 h-4 text-green-600 dark:text-green-400" />
              <span class="text-sm font-medium text-green-800 dark:text-green-200">{$t('settings.header.status.configured')}</span>
            </div>
          {/if}
        </div>

        <P class="text-gray-600 dark:text-gray-400">
          {$t('settings.header.description')}
        </P>

        {#if criticalConfigError}
          <div class="mt-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <P class="text-red-800 dark:text-red-200">
              <strong>{$t('settings.header.status.configuration_error')}:</strong>
              {criticalConfigErrorMessage}
            </P>
          </div>
        {:else if needsConfiguration}
          <div class="mt-4 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
            <P class="text-yellow-800 dark:text-yellow-200">
              <strong>{$t('settings.header.alerts.setup_required')}</strong> {$t('settings.header.alerts.setup_required_description')}
            </P>
          </div>
        {/if}
      </div>
    </div>
  </div>

  {#if loading === true}
    <div class="flex items-center justify-center py-12">
      <div class="text-center">
        <Spinner class="mb-4 w-8 h-8 mx-auto" />
        <P class="text-gray-600 dark:text-gray-400">{$t('common.common.loading')}</P>
      </div>
    </div>
  {:else if loadError}
    <div class="flex items-center justify-center py-12">
      <div class="text-center max-w-md">
        <ExclamationCircleOutline class="mb-4 w-12 h-12 mx-auto text-red-500 dark:text-red-400" />
        <Heading tag="h3" class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          {$t('settings.header.status.failed_to_load_configuration')}
        </Heading>
        <P class="mb-4 text-gray-600 dark:text-gray-400">
          {$t('settings.header.status.failed_to_load_configuration_description')}
        </P>
        <Button color="primary" onclick={loadConfig}>
          <RefreshOutline class="w-4 h-4 mr-2" />
          {$t('settings.retry')}
        </Button>
      </div>
    </div>
      {:else if localConfig}
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
        <Tabs style="underline" defaultClass="flex rounded-t-lg overflow-hidden bg-gray-50 dark:bg-gray-700" contentClass="p-6 bg-white dark:bg-gray-800 rounded-b-lg">
          <TabItem open title="{$t('settings.tabs.core_configuration')}">
            <svelte:fragment slot="title">
              <div class="flex items-center gap-2">
                <CogSolid class="w-4 h-4" />
                {$t('settings.tabs.core_configuration')}
              </div>
            </svelte:fragment>
            <div class="space-y-6">
              <GeneralSection bind:config={localConfig} />
              <ServerSection bind:config={localConfig} />
            </div>
          </TabItem>

          <TabItem title="{$t('settings.tabs.upload_settings')}">
            <svelte:fragment slot="title">
              <div class="flex items-center gap-2">
                <CloudArrowUpSolid class="w-4 h-4" />
                {$t('settings.tabs.upload_settings')}
              </div>
            </svelte:fragment>
            <div class="space-y-6">
              <PostingSection bind:config={localConfig} />
              <PostCheckSection bind:config={localConfig} />
            </div>
          </TabItem>

          <TabItem title="{$t('settings.tabs.file_processing')}">
            <svelte:fragment slot="title">
              <div class="flex items-center gap-2">
                <FileSolid class="w-4 h-4" />
                {$t('settings.tabs.file_processing')}
              </div>
            </svelte:fragment>
            <div class="space-y-6">
              <Par2Section bind:config={localConfig} />
              <NzbCompressionSection bind:config={localConfig} />
            </div>
          </TabItem>

          <TabItem title="{$t('settings.tabs.automation')}">
            <svelte:fragment slot="title">
              <div class="flex items-center gap-2">
                <EyeSolid class="w-4 h-4" />
                {$t('settings.tabs.automation')}
              </div>
            </svelte:fragment>
            <div class="space-y-6">
              <WatcherSection bind:config={localConfig} />
              <PostUploadScriptSection bind:config={localConfig} />
            </div>
          </TabItem>
        </Tabs>
      </div>
  {:else}
    <div class="flex items-center justify-center py-12">
      <div class="text-center">
        <P class="text-gray-600 dark:text-gray-400">{$t('settings.no_configuration_available')}</P>
      </div>
    </div>
  {/if}
</div>
