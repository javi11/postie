<script lang="ts">
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import { t } from "$lib/i18n";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Link } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

// Ensure connection_pool exists with defaults
if (!config.connection_pool) {
	config.connection_pool = {
		min_connections: 5,
		health_check_interval: "1m",
	};
}

// Reactive local state
let minConnections = $state(config.connection_pool.min_connections || 5);
let healthCheckInterval = $state(config.connection_pool.health_check_interval || "1m");

// Sync local state back to config
$effect(() => {
	config.connection_pool.min_connections = minConnections;
});

$effect(() => {
	config.connection_pool.health_check_interval = healthCheckInterval;
});


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


    <div class="alert alert-info">
      <span class="text-sm">
        <strong>{$t('settings.connection_pool.info_title')}</strong> {$t('settings.connection_pool.info_description')}
      </span>
    </div>

  </div>
</div>
