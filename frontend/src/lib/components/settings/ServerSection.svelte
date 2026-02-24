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
	Upload,
	ShieldCheck,
} from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

let saving = $state(false);
let isAdvanced = $derived($advancedMode);

// Helper to build a ServerConfig from a plain object
function toServerConfig(server: any, role: "upload" | "verify"): configType.ServerConfig {
	const s = new configType.ServerConfig();
	s.enabled = server.enabled ?? true;
	s.host = server.host || "";
	s.port = server.port || 119;
	s.username = server.username || "";
	s.password = server.password || "";
	s.max_connections = server.max_connections || 10;
	s.inflight = server.inflight || 10;
	s.ssl = server.ssl ?? false;
	s.insecure_ssl = server.insecure_ssl ?? false;
	s.max_connection_idle_time_in_seconds = server.max_connection_idle_time_in_seconds || 300;
	s.max_connection_ttl_in_seconds = server.max_connection_ttl_in_seconds || 3600;
	s.role = role;
	s.proxy_url = server.proxy_url || "";
	return s;
}

// Derive filtered lists to pass to each NntpServerManager
let uploadManagedServers = $derived(
	config.servers
		.filter(s => (s.role || "upload") !== "verify")
		.map(s => ({
			enabled: s.enabled ?? true,
			host: s.host || "",
			port: s.port || 119,
			username: s.username || "",
			password: s.password || "",
			max_connections: s.max_connections || 10,
			ssl: s.ssl ?? false,
			insecure_ssl: s.insecure_ssl ?? false,
			max_connection_idle_time_in_seconds: s.max_connection_idle_time_in_seconds || 300,
			max_connection_ttl_in_seconds: s.max_connection_ttl_in_seconds || 3600,
			role: "upload" as const,
			inflight: s.inflight || 10,
			proxy_url: s.proxy_url || "",
		}))
);

let verifyManagedServers = $derived(
	config.servers
		.filter(s => s.role === "verify")
		.map(s => ({
			enabled: s.enabled ?? true,
			host: s.host || "",
			port: s.port || 119,
			username: s.username || "",
			password: s.password || "",
			max_connections: s.max_connections || 10,
			ssl: s.ssl ?? false,
			insecure_ssl: s.insecure_ssl ?? false,
			max_connection_idle_time_in_seconds: s.max_connection_idle_time_in_seconds || 300,
			max_connection_ttl_in_seconds: s.max_connection_ttl_in_seconds || 3600,
			role: "verify" as const,
			inflight: s.inflight || 10,
			proxy_url: s.proxy_url || "",
		}))
);

function handleUploadUpdate(updatedServers: any[]) {
	config.servers = [
		...updatedServers.map(s => toServerConfig(s, "upload")),
		...config.servers.filter(s => s.role === "verify").map(s => toServerConfig(s, "verify")),
	];
}

function handleVerifyUpdate(updatedServers: any[]) {
	config.servers = [
		...config.servers.filter(s => (s.role || "upload") !== "verify").map(s => toServerConfig(s, "upload")),
		...updatedServers.map(s => toServerConfig(s, "verify")),
	];
}

async function saveServerSettings() {
	try {
		saving = true;

		// Validate upload servers have required fields
		const uploadServers = config.servers.filter(s => (s.role || "upload") !== "verify");
		for (let i = 0; i < uploadServers.length; i++) {
			const server = uploadServers[i];
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

		// Update servers with proper type conversion
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

<div class="space-y-6">
  <!-- Upload Pool -->
  <div class="card bg-base-100 shadow-sm">
    <div class="card-body space-y-4">
      <div class="flex items-center gap-3">
        <Upload class="w-5 h-5 text-primary" />
        <div>
          <h2 class="text-lg font-semibold text-base-content">
            {$t("settings.server.upload_pool_title")}
          </h2>
          <p class="text-sm text-base-content/60">
            {$t("settings.server.upload_pool_description")}
          </p>
        </div>
      </div>

      <NntpServerManager
        servers={uploadManagedServers}
        onupdate={handleUploadUpdate}
        showAdvancedFields={isAdvanced}
        variant="settings"
        restrictedRole="upload"
      />
    </div>
  </div>

  <!-- Verify Pool -->
  <div class="card bg-base-100 shadow-sm">
    <div class="card-body space-y-4">
      <div class="flex items-center gap-3">
        <ShieldCheck class="w-5 h-5 text-secondary" />
        <div>
          <h2 class="text-lg font-semibold text-base-content">
            {$t("settings.server.verify_pool_title")}
            <span class="badge badge-ghost badge-sm ml-2">
              {$t("settings.server.verify_pool_optional")}
            </span>
          </h2>
          <p class="text-sm text-base-content/60">
            {$t("settings.server.verify_pool_description")}
          </p>
        </div>
      </div>

      <NntpServerManager
        servers={verifyManagedServers}
        onupdate={handleVerifyUpdate}
        showAdvancedFields={isAdvanced}
        variant="settings"
        restrictedRole="verify"
      />
    </div>
  </div>

  <!-- Save Button -->
  <div class="pt-2">
    <button
      type="button"
      class="btn btn-success"
      onclick={saveServerSettings}
      disabled={saving}
    >
      <Save class="w-4 h-4" />
      {saving ? $t("common.common.saving") : $t("settings.server.save_button")}
    </button>

    <p class="text-sm text-base-content/70 mt-3">
      {@html $t("settings.server.tip")}
    </p>
  </div>
</div>
