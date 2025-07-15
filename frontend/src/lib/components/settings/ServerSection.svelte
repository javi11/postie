<script lang="ts">
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { config as configType } from "$lib/wailsjs/go/models";
import {
	Check,
	CirclePlus,
	Loader2,
	Save,
	Server,
	ShieldCheck,
	Trash2,
} from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

let saving = $state(false);
// Track validation state for each server
let validationStates = $state<Record<number, { status: string; error: string }>>({});
// Track original server state to detect modifications
let originalServers = $state<configType.ServerConfig[]>([]);
// Track which servers have been modified
let modifiedServers = $state<Set<number>>(new Set());

// Local reactive state for server configurations
let serverConfigs = $state<Array<{
	enabled: boolean;
	host: string;
	port: number;
	username: string;
	password: string;
	max_connections: number;
	ssl: boolean;
	insecure_ssl: boolean;
	max_connection_idle_time_in_seconds: number;
	max_connection_ttl_in_seconds: number;
}>>([]);

// Ensure serverConfigs is always in sync with config.servers length
$effect(() => {
	if (config.servers && serverConfigs.length !== config.servers.length) {
		serverConfigs = config.servers.map(server => ({
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
		}));
	}
});

// Initialize original servers state when component loads
$effect(() => {
	if (config.servers && originalServers.length === 0) {
		originalServers = JSON.parse(JSON.stringify(config.servers));
		modifiedServers = new Set<number>();
		
		// Mark previously saved servers as valid if they have required fields
		config.servers.forEach((server, index) => {
			if (server.host && server.port) {
				validationStates = {
					...validationStates,
					[index]: { status: "valid", error: "" },
				};
			}
		});
	}
});

// Sync local state back to config
$effect(() => {
	if (serverConfigs.length > 0) {
		serverConfigs.forEach((localServer, index) => {
			if (config.servers[index]) {
				config.servers[index].enabled = localServer.enabled;
				config.servers[index].host = localServer.host;
				config.servers[index].port = localServer.port;
				config.servers[index].username = localServer.username;
				config.servers[index].password = localServer.password;
				config.servers[index].max_connections = localServer.max_connections;
				config.servers[index].ssl = localServer.ssl;
				config.servers[index].insecure_ssl = localServer.insecure_ssl;
				config.servers[index].max_connection_idle_time_in_seconds = localServer.max_connection_idle_time_in_seconds;
				config.servers[index].max_connection_ttl_in_seconds = localServer.max_connection_ttl_in_seconds;
			}
		});
	}
});

// Duration preset definitions
const idleTimePresets = [
	{ label: "2m", value: 2, unit: "m" },
	{ label: "5m", value: 5, unit: "m" },
	{ label: "15m", value: 15, unit: "m" },
];

const ttlPresets = [
	{ label: "30m", value: 30, unit: "m" },
	{ label: "1h", value: 1, unit: "h" },
	{ label: "6h", value: 6, unit: "h" },
];

function addServer() {
	const newServer: configType.ServerConfig = new configType.ServerConfig();

	config.servers = [...config.servers, newServer];
	
	// serverConfigs will be automatically updated by the $effect
	
	// Mark new server as modified
	modifiedServers.add(config.servers.length - 1);
}

function removeServer(index: number) {
	config.servers = config.servers.filter((_, i) => i !== index);
	// serverConfigs will be automatically updated by the $effect
}

// Convert seconds to duration strings and back
function secondsToDuration(seconds: number): string {
	if (seconds >= 3600 && seconds % 3600 === 0) {
		return `${seconds / 3600}h`;
	}
	if (seconds >= 60 && seconds % 60 === 0) {
		return `${seconds / 60}m`;
	}
	return `${seconds}s`;
}

function durationToSeconds(duration: string): number {
	const match = duration.match(/^(\d+)([smh])$/);
	if (match) {
		const value = Number.parseInt(match[1]);
		const unit = match[2];
		switch (unit) {
			case "h":
				return value * 3600;
			case "m":
				return value * 60;
			case "s":
				return value;
			default:
				return value;
		}
	}
	return 300; // fallback
}

// Reactive duration strings for each server
let serverDurations = $derived(serverConfigs.map((server) => ({
	idle: secondsToDuration(server.max_connection_idle_time_in_seconds || 300),
	ttl: secondsToDuration(server.max_connection_ttl_in_seconds || 3600),
})));

function updateIdleTime(serverIndex: number, duration: string) {
	if (serverConfigs[serverIndex]) {
		serverConfigs[serverIndex].max_connection_idle_time_in_seconds = durationToSeconds(duration);
	}
}

function updateTTL(serverIndex: number, duration: string) {
	if (serverConfigs[serverIndex]) {
		serverConfigs[serverIndex].max_connection_ttl_in_seconds = durationToSeconds(duration);
	}
}

function getServerValidationState(index: number) {
	const state = validationStates[index];
	if (!state) return { status: "pending", error: "" };
	return state;
}

function isServerModified(index: number): boolean {
	if (index >= originalServers.length) return true; // New server
	const current = config.servers[index];
	const original = originalServers[index];

	// Compare key fields that would affect server validation
	return (
		current.host !== original.host ||
		current.port !== original.port ||
		current.username !== original.username ||
		current.password !== original.password ||
		current.ssl !== original.ssl ||
		current.insecure_ssl !== original.insecure_ssl
	);
}

function onServerFieldChange(index: number) {
	// Mark server as modified
	modifiedServers.add(index);

	// Clear validation state only if server was modified
	if (isServerModified(index)) {
		validationStates = {
			...validationStates,
			[index]: { status: "pending", error: "" },
		};
	}
}

async function validateServer(index: number) {
	const server = serverConfigs[index];

	// Basic validation first
	if (!server.host || !server.port) {
		validationStates = {
			...validationStates,
			[index]: { status: "incomplete", error: "Host and port are required" },
		};
		return;
	}

	validationStates = {
		...validationStates,
		[index]: { status: "validating", error: "" },
	};

	try {
		const result = await apiClient.validateNNTPServer({
			host: server.host,
			port: server.port,
			username: server.username,
			password: server.password,
			ssl: server.ssl,
			maxConnections: server.max_connections,
		});

		if (result.valid) {
			console.log("Setting server", index, "as valid");
			validationStates = {
				...validationStates,
				[index]: { status: "valid", error: "" },
			};
			toastStore.success($t("setup.servers.valid"));
		} else {
			console.log("Setting server", index, "as invalid:", result.error);
			validationStates = {
				...validationStates,
				[index]: { status: "invalid", error: result.error },
			};
			toastStore.error($t("setup.servers.invalid"), String(result.error));
		}
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		validationStates = {
			...validationStates,
			[index]: {
				status: "invalid",
				error: `Validation failed: ${errorMessage}`,
			},
		};
		toastStore.error($t("setup.servers.invalid"), String(errorMessage));
		console.error("Server validation error:", error);
	}
}

function areAllServersValid(): boolean {
	return config.servers.every((_, index) => {
		const validationState = getServerValidationState(index);
		// Consider unmodified servers with required fields as valid
		if (
			!isServerModified(index) &&
			config.servers[index].host &&
			config.servers[index].port
		) {
			return true;
		}
		return validationState.status === "valid";
	});
}

async function saveServerSettings() {
	try {
		saving = true;

		// Check if all servers have been validated successfully
		if (!areAllServersValid()) {
			const invalidServers = config.servers
				.map((_, index) => {
					const validationState = getServerValidationState(index);
					const isModified = isServerModified(index);
					const hasRequiredFields =
						config.servers[index].host && config.servers[index].port;

					if (isModified && validationState.status !== "valid") {
						return index + 1;
					}
					if (!hasRequiredFields) {
						return index + 1;
					}
					return null;
				})
				.filter(Boolean);

			toastStore.error(
				"Validation Required",
				`Please test server connections for server(s): ${invalidServers.join(", ")}. All modified servers must pass validation before saving.`,
			);
			return;
		}

		// Validate that all servers have required fields
		for (let i = 0; i < serverConfigs.length; i++) {
			const server = serverConfigs[i];
			if (!server.host || server.host.trim() === "") {
				toastStore.error(
					"Configuration Error",
					$t("settings.server.validation_errors.host_required", {
							number: i + 1,
					}),
				);
				return;
			}
			if (!server.port || server.port <= 0 || server.port > 65535) {
				toastStore.error(
					"Configuration Error",
					$t("settings.server.validation_errors.port_invalid", {
							number: i + 1,
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

		await apiClient.saveConfig(currentConfig);

		// Reset tracking after successful save
		originalServers = JSON.parse(JSON.stringify(config.servers));
		modifiedServers.clear();

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
      <button
        type="button"
        class="btn btn-sm btn-outline"
        onclick={addServer}
      >
        <CirclePlus class="w-4 h-4" />
        {$t('settings.server.add_server')}
      </button>
    </div>

    {#if config.servers.length === 0}
      <div
        class="text-center py-8 border-2 border-dashed border-base-300 rounded-lg"
      >
        <Server class="w-12 h-12 text-base-content/40 mx-auto mb-4" />
        <p class="text-base-content/70 mb-4">
          {$t('settings.server.no_servers_description')}
        </p>
        <button
          type="button"
          class="btn btn-outline"
          onclick={addServer}
        >
          <CirclePlus class="w-4 h-4" />
          {$t('settings.server.add_first_server')}
        </button>
      </div>
    {:else}
      <div class="space-y-6">
      </div>
    {/if}

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
