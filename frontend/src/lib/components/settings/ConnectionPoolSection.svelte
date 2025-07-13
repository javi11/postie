<script lang="ts">
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import { t } from "$lib/i18n";
import type { ConfigData } from "$lib/types";
import { Link } from "lucide-svelte";

export let config: ConfigData;

// Ensure connection_pool exists with defaults
if (!config.connection_pool) {
	config.connection_pool = {
		min_connections: 5,
		health_check_interval: "1m",
		skip_providers_verification_on_creation: false,
	};
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
          bind:value={config.connection_pool.min_connections}
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
        bind:value={config.connection_pool.health_check_interval}
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
          bind:checked={
            config.connection_pool.skip_providers_verification_on_creation
          }
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
  </div>
</div>
