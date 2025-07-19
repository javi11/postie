<script lang="ts">
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Link, Save } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

// Ensure connection_pool exists with defaults
if (!config.connection_pool) {
	config.connection_pool = {
		min_connections: 5,
		health_check_interval: "1m",
		skip_providers_verification_on_creation: false,
	};
}

// Reactive local state
let minConnections = $state(config.connection_pool.min_connections || 5);
let healthCheckInterval = $state(config.connection_pool.health_check_interval || "1m");
let skipProvidersVerification = $state(config.connection_pool.skip_providers_verification_on_creation ?? false);
let saving = $state(false);

// Derived state
let canSave = $derived(
	minConnections > 0 && 
	minConnections <= 50 && 
	healthCheckInterval.trim() && 
	!saving
);

// Sync local state back to config
$effect(() => {
	config.connection_pool.min_connections = minConnections;
});

$effect(() => {
	config.connection_pool.health_check_interval = healthCheckInterval;
});

$effect(() => {
	config.connection_pool.skip_providers_verification_on_creation = skipProvidersVerification;
});

async function saveConnectionPoolSettings() {
	if (!canSave) return;
	
	try {
		saving = true;

		// Validation
		if (minConnections < 1 || minConnections > 50) {
			throw new Error("Min connections must be between 1 and 50");
		}
		
		if (!healthCheckInterval.trim()) {
			throw new Error("Health check interval is required");
		}

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Update only connection pool section
		currentConfig.connection_pool = {
			...currentConfig.connection_pool,
			min_connections: minConnections,
			health_check_interval: healthCheckInterval.trim(),
			skip_providers_verification_on_creation: skipProvidersVerification,
		};

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.connection_pool.saved_success"),
			$t("settings.connection_pool.saved_success_description")
		);
	} catch (error) {
		console.error("Failed to save connection pool settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}

const healthCheckPresets = [
	{ label: "30s", value: 30, unit: "s" },
	{ label: "1m", value: 1, unit: "m" },
	{ label: "5m", value: 5, unit: "m" },
	{ label: "15m", value: 15, unit: "m" },
];
</script>

<div class="card bg-base-100 shadow-xl">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <Link class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <h2 class="card-title text-lg">
        {$t('settings.connection_pool.title')}
      </h2>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div class="form-control">
        <label class="label" for="min-connections">
          <span class="label-text">{$t('settings.connection_pool.min_connections')}</span>
        </label>
        <input
          id="min-connections"
          type="number"
          class="input input-bordered"
          bind:value={minConnections}
          min="1"
          max="50"
        />
        <div class="label">
          <span class="label-text-alt">
            {$t('settings.connection_pool.min_connections_description')}
          </span>
        </div>
      </div>

      <DurationInput
        bind:value={healthCheckInterval}
        label={$t('settings.connection_pool.health_check_interval')}
        description={$t('settings.connection_pool.health_check_interval_description')}
        presets={healthCheckPresets}
        id="health-check-interval"
      />
    </div>

    <div class="form-control">
      <label class="label cursor-pointer justify-start gap-3">
        <input
          type="checkbox"
          class="checkbox"
          bind:checked={skipProvidersVerification}
        />
        <span class="label-text">{$t('settings.connection_pool.skip_providers_verification_on_creation')}</span>
      </label>
      <div class="label">
        <span class="label-text-alt ml-8">
          {$t('settings.connection_pool.skip_providers_verification_on_creation_description')}
        </span>
      </div>
    </div>

    <div class="alert alert-info">
      <span class="text-sm">
        <strong>{$t('settings.connection_pool.info_title')}</strong> {$t('settings.connection_pool.info_description')}
      </span>
    </div>

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        type="button"
        class="btn btn-success"
        onclick={saveConnectionPoolSettings}
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.connection_pool.save_button')}
      </button>
    </div>
  </div>
</div>
