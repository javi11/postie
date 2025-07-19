<script lang="ts">
import apiClient from "$lib/api/client";
import NntpServerManager from "$lib/components/NntpServerManager.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { config as configType } from "$lib/wailsjs/go/models";
import {
	Save,
	Server,
} from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

let saving = $state(false);
let isAdvanced = $derived($advancedMode);

// Convert config servers to the format expected by NntpServerManager
let managedServers = $derived(config.servers.map(server => ({
	enabled: server.enabled ?? true,
	host: server.host || "",
	port: server.port || 119,
	username: server.username || "",
	password: server.password || "",
	max_connections: server.max_connections || 10,
	ssl: server.ssl ?? false,
	insecure_ssl: server.insecure_ssl ?? false,
	max_connection_idle_time_in_seconds: server.max_connection_idle_time_in_seconds || 300,
	max_connection_ttl_in_seconds: server.max_connection_ttl_in_seconds || 3600,
})));

function handleServerUpdate(updatedServers: any[]) {
	// Convert from our generic server format to config.ServerConfig
	config.servers = updatedServers.map(server => {
		const serverConfig = new configType.ServerConfig();
		serverConfig.enabled = server.enabled ?? true;
		serverConfig.host = server.host || "";
		serverConfig.port = server.port || 119;
		serverConfig.username = server.username || "";
		serverConfig.password = server.password || "";
		serverConfig.max_connections = server.max_connections || 10;
		serverConfig.ssl = server.ssl ?? false;
		serverConfig.insecure_ssl = server.insecure_ssl ?? false;
		serverConfig.max_connection_idle_time_in_seconds = server.max_connection_idle_time_in_seconds || 300;
		serverConfig.max_connection_ttl_in_seconds = server.max_connection_ttl_in_seconds || 3600;
		return serverConfig;
	});
}


async function saveServerSettings() {
	try {
		saving = true;

		// Validate that all servers have required fields
		for (let i = 0; i < config.servers.length; i++) {
			const server = config.servers[i];
			if (!server.host || server.host.trim() === "") {
				toastStore.error(
					"Configuration Error",
					$t("settings.server.validation_errors.host_required", {
							values: { number: i + 1 },
					}),
				);
				return;
			}
			if (!server.port || server.port <= 0 || server.port > 65535) {
				toastStore.error(
					"Configuration Error",
					$t("settings.server.validation_errors.port_invalid", {
							values: { number: i + 1 },
					}),
				);
				return;
			}
		}

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Only update the server fields with proper type conversion
		currentConfig.servers = config.servers.map(
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

		console.log("Saving server settings:", currentConfig);

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.server.saved_success"),
			$t("settings.server.saved_success_description"),
		);
	} catch (error) {
		console.error("Failed to save server settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}
</script>

<div class="card bg-base-100 shadow-sm">
  <div class="card-body space-y-6">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <Server class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        <h2 class="text-lg font-semibold text-base-content">
          {$t('settings.server.title')}
        </h2>
      </div>
    </div>

    <NntpServerManager
      servers={managedServers}
      onupdate={handleServerUpdate}
      showAdvancedFields={isAdvanced}
      variant="settings"
    />

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        type="button"
        class="btn btn-success mb-4"
        onclick={saveServerSettings}
        disabled={saving}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.server.save_button')}
      </button>
      
      <p class="text-sm text-base-content/70">
        {@html $t('settings.server.tip')}
      </p>
    </div>
  </div>
</div>
