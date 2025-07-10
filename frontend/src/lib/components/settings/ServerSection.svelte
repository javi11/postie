<script lang="ts">
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData, ServerConfig } from "$lib/types";
import apiClient from "$lib/api/client";
import {
	Button,
	ButtonGroup,
	Card,
	Checkbox,
	Heading,
	Input,
	Label,
	P,
	Select,
} from "flowbite-svelte";
import {
	CirclePlusSolid,
	FloppyDiskSolid,
	ServerSolid,
	ShieldCheckSolid,
	TrashBinSolid,
} from "flowbite-svelte-icons";

export let config: ConfigData;

let saving = false;

const timeUnitOptions = [
	{ value: "s", name: "Seconds" },
	{ value: "m", name: "Minutes" },
	{ value: "h", name: "Hours" },
];

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

// Helper functions for time conversion
function secondsToTimeUnit(seconds: number): { value: number; unit: string } {
	if (seconds >= 3600 && seconds % 3600 === 0) {
		return { value: seconds / 3600, unit: "h" };
	}
	if (seconds >= 60 && seconds % 60 === 0) {
		return { value: seconds / 60, unit: "m" };
	}
	return { value: seconds, unit: "s" };
}

function timeUnitToSeconds(value: number, unit: string): number {
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

function addServer() {
	const newServer: ServerConfig = {
		host: "",
		port: 119,
		username: "",
		password: "",
		ssl: false,
		max_connections: 10,
		max_connection_idle_time_in_seconds: 300,
		max_connection_ttl_in_seconds: 3600,
		insecure_ssl: false,
		enabled: true,
	};

	config.servers = [...config.servers, newServer];
}

function removeServer(index: number) {
	config.servers = config.servers.filter((_, i) => i !== index);
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
$: serverDurations = config.servers.map((server) => ({
	idle: secondsToDuration(server.max_connection_idle_time_in_seconds || 300),
	ttl: secondsToDuration(server.max_connection_ttl_in_seconds || 3600),
}));

function updateIdleTime(serverIndex: number, duration: string) {
	config.servers[serverIndex].max_connection_idle_time_in_seconds =
		durationToSeconds(duration);
}

function updateTTL(serverIndex: number, duration: string) {
	config.servers[serverIndex].max_connection_ttl_in_seconds =
		durationToSeconds(duration);
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
		currentConfig.servers = config.servers.map((server: ServerConfig) => ({
			...server,
			port: Number.parseInt(server.port) || 119,
			max_connections: Number.parseInt(server.max_connections) || 10,
			max_connection_idle_time_in_seconds:
				Number.parseInt(server.max_connection_idle_time_in_seconds) || 300,
			max_connection_ttl_in_seconds:
				Number.parseInt(server.max_connection_ttl_in_seconds) || 3600,
		}));

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

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <ServerSolid class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        <Heading
          tag="h2"
          class="text-lg font-semibold text-gray-900 dark:text-white"
        >
          {$t('settings.server.title')}
        </Heading>
      </div>
      <Button
        size="sm"
        onclick={addServer}
        class="cursor-pointer flex items-center gap-2"
      >
        <CirclePlusSolid class="w-4 h-4" />
        {$t('settings.server.add_server')}
      </Button>
    </div>

    {#if config.servers.length === 0}
      <div
        class="text-center py-8 border-2 border-dashed border-gray-200 dark:border-gray-700 rounded-lg"
      >
        <ServerSolid class="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <P class="text-gray-600 dark:text-gray-400 mb-4">
          {$t('settings.server.no_servers_description')}
        </P>
        <Button
          onclick={addServer}
          class="cursor-pointer flex items-center gap-2 mx-auto"
        >
          <CirclePlusSolid class="w-4 h-4" />
          {$t('settings.server.add_first_server')}
        </Button>
      </div>
    {:else}
      <div class="space-y-6">
        {#each config.servers as server, index (index)}
          <div
            class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800/50"
          >
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-3">
                <Heading
                  tag="h3"
                  class="text-md font-medium text-gray-900 dark:text-white"
                >
                  {$t('settings.server.server_number', { number: index + 1 })}
                </Heading>
                <div class="flex items-center gap-2">
                  <Checkbox bind:checked={server.enabled} />
                  <Label class="text-sm font-medium">{$t('settings.server.enabled')}</Label>
                </div>
              </div>
              <Button
                size="xs"
                color="red"
                variant="outline"
                onclick={() => removeServer(index)}
                class="cursor-pointer flex items-center gap-1"
              >
                <TrashBinSolid class="w-3 h-3" />
                {$t('settings.server.remove')}
              </Button>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Label for="host-{index}" class="mb-2">{$t('settings.server.host_required')}</Label>
                <Input
                  id="host-{index}"
                  bind:value={server.host}
                  placeholder={$t('settings.server.host_placeholder')}
                  required
                />
              </div>

              <div>
                <Label for="port-{index}" class="mb-2">{$t('settings.server.port')}</Label>
                <Input
                  id="port-{index}"
                  type="number"
                  bind:value={server.port}
                  min="1"
                  max="65535"
                />
              </div>

              <div>
                <Label for="username-{index}" class="mb-2">{$t('settings.server.username')}</Label>
                <Input
                  id="username-{index}"
                  bind:value={server.username}
                  placeholder={$t('settings.server.username_placeholder')}
                  autocomplete="username"
                />
              </div>

              <div>
                <Label for="password-{index}" class="mb-2">{$t('settings.server.password')}</Label>
                <Input
                  id="password-{index}"
                  type="password"
                  bind:value={server.password}
                  placeholder={$t('settings.server.password_placeholder')}
                  autocomplete="current-password"
                />
              </div>

              <div>
                <Label for="max-connections-{index}" class="mb-2">
                  {$t('settings.server.max_connections')}
                </Label>
                <Input
                  id="max-connections-{index}"
                  type="number"
                  bind:value={server.max_connections}
                  min="1"
                  max="50"
                />
              </div>

              <DurationInput
                value={serverDurations[index]?.idle || "5m"}
                label={$t('settings.server.connection_idle_timeout')}
                description={$t('settings.server.connection_idle_timeout_description')}
                presets={idleTimePresets}
                id="idle-time-{index}"
                onchange={(e) => updateIdleTime(index, e.detail)}
              />

              <DurationInput
                value={serverDurations[index]?.ttl || "1h"}
                label={$t('settings.server.connection_ttl')}
                description={$t('settings.server.connection_ttl_description')}
                presets={ttlPresets}
                id="ttl-{index}"
                onchange={(e) => updateTTL(index, e.detail)}
              />
            </div>

            <div class="mt-4 space-y-3">
              <div class="flex items-center gap-3">
                <Checkbox bind:checked={server.ssl} />
                <div class="flex items-center gap-2">
                  <ShieldCheckSolid
                    class="w-4 h-4 text-green-600 dark:text-green-400"
                  />
                  <Label class="text-sm font-medium">{$t('settings.server.use_ssl_tls')}</Label>
                </div>
              </div>

              {#if server.ssl}
                <div class="ml-6">
                  <div class="flex items-center gap-3">
                    <Checkbox bind:checked={server.insecure_ssl} />
                    <Label class="text-sm">
                      {$t('settings.server.allow_insecure_ssl')}
                    </Label>
                  </div>
                </div>
              {/if}
            </div>

            {#if !server.host || !server.port}
              <div
                class="mt-3 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded"
              >
                <P class="text-sm text-yellow-800 dark:text-yellow-200">
                  {$t('settings.server.validation_warning')}
                </P>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <!-- Save Button -->
    <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
      <Button
        color="green"
        onclick={saveServerSettings}
        disabled={saving}
        class="cursor-pointer flex items-center gap-2 mb-4"
      >
        <FloppyDiskSolid class="w-4 h-4" />
        {saving ? $t('settings.server.saving') : $t('settings.server.save_button')}
      </Button>
      
      <P class="text-sm text-gray-600 dark:text-gray-400">
        {@html $t('settings.server.tip')}
      </P>
    </div>
  </div>
</Card>
