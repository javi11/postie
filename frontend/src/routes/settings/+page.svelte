<script lang="ts">
import { goto } from "$app/navigation";
import apiClient from "$lib/api/client";
import ConnectionPoolSection from "$lib/components/settings/ConnectionPoolSection.svelte";
import GeneralSection from "$lib/components/settings/GeneralSection.svelte";
import NzbCompressionSection from "$lib/components/settings/NzbCompressionSection.svelte";
import Par2Section from "$lib/components/settings/Par2Section.svelte";
import PostCheckSection from "$lib/components/settings/PostCheckSection.svelte";
import PostingSection from "$lib/components/settings/PostingSection.svelte";
import PostUploadScriptSection from "$lib/components/settings/PostUploadScriptSection.svelte";
import QueueSection from "$lib/components/settings/QueueSection.svelte";
import ServerSection from "$lib/components/settings/ServerSection.svelte";
import WatcherSection from "$lib/components/settings/WatcherSection.svelte";
import { t } from "$lib/i18n";
import { advancedMode, appStatus, settingsSaveFunction } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { parseDuration } from "$lib/utils";
import { backend, config as configType } from "$lib/wailsjs/go/models";
import { AlertCircle, CheckCircle, RefreshCw, Settings } from "lucide-svelte";
import { onDestroy, onMount } from "svelte";

let configData: configType.ConfigData | null = null;
let localConfig: configType.ConfigData | null = null;
let needsConfiguration = false;
let criticalConfigError = false;
let criticalConfigErrorMessage = "";
let loading = false;
let loadError = false;

onMount(() => {
	loadConfig();

	// Register save function with the store
	settingsSaveFunction.set(handleSaveConfig);

	// Subscribe to app status
	const unsubscribe = appStatus.subscribe((status: backend.AppStatus) => {
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
		await apiClient.initialize();
		const config = await apiClient.getConfig();
		configData = config;
		// Initialize localConfig with the loaded config
		localConfig = JSON.parse(JSON.stringify(config));

		// If no servers exist, add a default one to make it easier for the user
		if (
			localConfig &&
			(!localConfig.servers || localConfig.servers.length === 0)
		) {
			localConfig.servers = [new configType.ServerConfig()];
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
		if (!localConfig) {
			toastStore.error(
				$t("common.messages.configuration_error"),
				$t("common.messages.no_configuration_loaded"),
			);
			return;
		}

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
					$t("common.messages.server_host_required", {
						number: i + 1,
					}),
				);
				return;
			}
			if (!server.port || server.port <= 0 || server.port > 65535) {
				toastStore.error(
					$t("common.messages.configuration_error"),
					$t("common.messages.server_port_required", {
						number: i + 1,
					}),
				);
				return;
			}
		}

		// Copy the config to a new object to avoid modifying the original
		const configToSave = JSON.parse(JSON.stringify(localConfig));

		// Convert server integer fields
		configToSave.servers = configToSave.servers.map(
			(server: configType.ServerConfig) => ({
				...server,
				port: server.port || 119,
				max_connections: server.max_connections || 10,
				max_connection_idle_time_in_seconds:
					server.max_connection_idle_time_in_seconds || 300,
				max_connection_ttl_in_seconds:
					server.max_connection_ttl_in_seconds || 3600,
			}),
		);

		// Convert posting fields
		configToSave.posting.max_retries =
			Number.parseInt(configToSave.posting.max_retries) || 3;
		configToSave.posting.article_size_in_bytes =
			Number.parseInt(configToSave.posting.article_size_in_bytes) || 750000;

		// Convert duration fields to nanoseconds
		configToSave.posting.retry_delay = parseDuration(
			configToSave.posting.retry_delay || "5s",
		);

		// Convert post_check duration fields
		if (configToSave.post_check) {
			configToSave.post_check.delay = parseDuration(
				configToSave.post_check.delay || "10s",
			);
		}

		// Convert queue duration fields
		if (configToSave.queue) {
			configToSave.queue.retry_delay = parseDuration(
				configToSave.queue.retry_delay || "5m",
			);
			configToSave.queue.cleanup_after = parseDuration(
				configToSave.queue.cleanup_after || "24h",
			);
		}

		// Convert post_upload_script timeout
		if (configToSave.post_upload_script) {
			configToSave.post_upload_script.timeout = parseDuration(
				configToSave.post_upload_script.timeout || "30s",
			);
		}

		// Convert connection pool duration fields
		if (configToSave.connection_pool) {
			configToSave.connection_pool.health_check_interval = parseDuration(
				configToSave.connection_pool.health_check_interval || "1m",
			);
		}

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

			configToSave.watcher.check_interval = parseDuration(
				configToSave.watcher.check_interval || "5m",
			);
		}

		// Ensure output_dir is set
		if (!configToSave.output_dir || configToSave.output_dir.trim() === "") {
			configToSave.output_dir = "./output";
		}

		await apiClient.saveConfig(configToSave);
		configData = configToSave;
		// Update localConfig to the saved config to maintain reactivity
		localConfig = JSON.parse(JSON.stringify(configToSave));

		// Update app status
		const status = await apiClient.getAppStatus();
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
  <div class="bg-base-100 p-6 rounded-lg shadow-sm border border-base-300">
    <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
      <div class="flex-1">
        <div class="flex items-center gap-3 mb-2">
          <Settings class="w-6 h-6 text-base-content/60" />
          <h1 class="text-2xl font-bold">
            {$t('settings.title')}
          </h1>
          {#if criticalConfigError}
            <div class="flex items-center gap-2 px-3 py-1 bg-red-100 dark:bg-red-900/30 rounded-full">
              <AlertCircle class="w-4 h-4 text-red-600 dark:text-red-400" />
              <span class="text-sm font-medium text-red-800 dark:text-red-200">{$t('settings.header.status.configuration_error')}</span>
            </div>
          {:else if needsConfiguration}
            <div class="flex items-center gap-2 px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 rounded-full">
              <AlertCircle class="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
              <span class="text-sm font-medium text-yellow-800 dark:text-yellow-200">{$t('settings.header.status.configuration_required')}</span>
            </div>
          {:else}
            <div class="flex items-center gap-2 px-3 py-1 bg-green-100 dark:bg-green-900/30 rounded-full">
              <CheckCircle class="w-4 h-4 text-green-600 dark:text-green-400" />
              <span class="text-sm font-medium text-green-800 dark:text-green-200">{$t('settings.header.status.configured')}</span>
            </div>
          {/if}
        </div>

        <p class="text-base-content/70">
          {$t('settings.header.description')}
        </p>

        {#if criticalConfigError}
          <div class="mt-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p class="text-error">
              <strong>{$t('settings.header.status.configuration_error')}:</strong>
              {criticalConfigErrorMessage}
            </p>
          </div>
        {:else if needsConfiguration}
          <div class="mt-4 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
            <p class="text-warning">
              <strong>{$t('settings.header.alerts.setup_required')}</strong> {$t('settings.header.alerts.setup_required_description')}
            </p>
          </div>
        {/if}
      </div>
      
      <div class="flex flex-col sm:flex-row gap-3 sm:items-center">
        <div class="flex items-center gap-3">
          <input type="checkbox" class="toggle toggle-primary" bind:checked={$advancedMode} />
          <div>
            <p class="text-sm font-medium">
              {$t('settings.header.advanced_mode')}
            </p>
            <p class="text-xs text-base-content/70">
              {$t('settings.header.advanced_mode_description')}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>

  {#if loading === true}
    <div class="flex items-center justify-center py-12">
      <div class="text-center">
        <div class="loading loading-spinner w-8 h-8 mb-4 mx-auto"></div>
        <p class="text-base-content/70">{$t('common.common.loading')}</p>
      </div>
    </div>
  {:else if loadError}
    <div class="flex items-center justify-center py-12">
      <div class="text-center max-w-md">
        <AlertCircle class="mb-4 w-12 h-12 mx-auto text-red-500 dark:text-red-400" />
        <h3 class="mb-2 text-lg font-semibold">
          {$t('settings.header.status.failed_to_load_configuration')}
        </h3>
        <p class="mb-4 text-base-content/70">
          {$t('settings.header.status.failed_to_load_configuration_description')}
        </p>
        <button class="btn btn-primary" onclick={loadConfig}>
          <RefreshCw class="w-4 h-4 mr-2" />
          {$t('settings.retry')}
        </button>
      </div>
    </div>
      {:else if localConfig}
      <div class="bg-base-100 rounded-lg shadow-sm border border-base-300">
        <div role="tablist" class="tabs tabs-bordered">
          <input type="radio" name="settings_tabs" role="tab" class="tab" aria-label="Core Configuration" checked />
          <div role="tabpanel" class="tab-content p-6">
            <div class="space-y-6">
              <GeneralSection config={localConfig} />
              <ServerSection config={localConfig} />
            </div>
          </div>

          <input type="radio" name="settings_tabs" role="tab" class="tab" aria-label="Upload Settings" />
          <div role="tabpanel" class="tab-content p-6">
            <div class="space-y-6">
              <PostingSection config={localConfig} />
              <PostCheckSection config={localConfig} />
              {#if $advancedMode}
                <QueueSection config={localConfig} />
                <ConnectionPoolSection config={localConfig} />
              {/if}
            </div>
          </div>

          <input type="radio" name="settings_tabs" role="tab" class="tab" aria-label="File Processing" />
          <div role="tabpanel" class="tab-content p-6">
            <div class="space-y-6">
              <Par2Section config={localConfig} />
              <NzbCompressionSection config={localConfig} />
            </div>
          </div>

          <input type="radio" name="settings_tabs" role="tab" class="tab" aria-label="Automation" />
          <div role="tabpanel" class="tab-content p-6">
            <div class="space-y-6">
              <WatcherSection config={localConfig} />
              <PostUploadScriptSection config={localConfig} />
            </div>
          </div>
        </div>
      </div>
  {:else}
    <div class="flex items-center justify-center py-12">
      <div class="text-center">
        <p class="text-base-content/60">{$t('settings.no_configuration_available')}</p>
      </div>
    </div>
  {/if}
</div>
