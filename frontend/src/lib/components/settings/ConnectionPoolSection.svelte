<script lang="ts">
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import type { ConfigData } from "$lib/types";
import { t } from "$lib/i18n";
import { Card, Checkbox, Heading, Input, Label, P } from "flowbite-svelte";
import { LinkSolid } from "flowbite-svelte-icons";

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

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <LinkSolid class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        {$t('settings.connection_pool.title')}
      </Heading>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <Label for="min-connections" class="mb-2">{$t('settings.connection_pool.min_connections')}</Label>
        <Input
          id="min-connections"
          type="number"
          bind:value={config.connection_pool.min_connections}
          min="1"
          max="50"
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          {$t('settings.connection_pool.min_connections_description')}
        </P>
      </div>

      <DurationInput
        bind:value={config.connection_pool.health_check_interval}
        label={$t('settings.connection_pool.health_check_interval')}
        description={$t('settings.connection_pool.health_check_interval_description')}
        presets={healthCheckPresets}
        id="health-check-interval"
      />
    </div>

    <div class="space-y-3">
      <div class="flex items-center gap-3">
        <Checkbox
          bind:checked={
            config.connection_pool.skip_providers_verification_on_creation
          }
        />
        <Label class="text-sm font-medium"
          >{$t('settings.connection_pool.skip_providers_verification_on_creation')}</Label
        >
      </div>
      <P class="text-sm text-gray-600 dark:text-gray-400 ml-6">
        {$t('settings.connection_pool.skip_providers_verification_on_creation_description')}
      </P>
    </div>

    <div
      class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
    >
      <P class="text-sm text-blue-800 dark:text-blue-200">
        <strong>{$t('settings.connection_pool.info_title')}</strong> {$t('settings.connection_pool.info_description')}
      </P>
    </div>
  </div>
</Card>
