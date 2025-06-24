<script lang="ts">
import { t } from "$lib/i18n";
import type { ConfigData } from "$lib/types";
import { Card, Checkbox, Heading, Input, Label, P } from "flowbite-svelte";
import { CheckCircleSolid } from "flowbite-svelte-icons";
import DurationInput from "../inputs/DurationInput.svelte";

const presets = [
	{ label: "5s", value: 5, unit: "s" },
	{ label: "10s", value: 10, unit: "s" },
	{ label: "30s", value: 30, unit: "s" },
	{ label: "1m", value: 1, unit: "m" },
];

export let config: ConfigData;

// Ensure post_check exists with defaults
if (!config.post_check) {
	config.post_check = {
		enabled: true,
		delay: "10s",
		max_reposts: 1,
	};
}
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <CheckCircleSolid class="w-5 h-5 text-orange-600 dark:text-orange-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        {$t('settings.post_check.title')}
      </Heading>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Checkbox bind:checked={config.post_check.enabled} />
        <Label class="text-sm font-medium">{$t('settings.post_check.enable')}</Label>
      </div>
      <P class="text-sm text-gray-600 dark:text-gray-400 ml-6">
        {$t('settings.post_check.enable_description')}
      </P>
    </div>

    {#if config.post_check.enabled}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <DurationInput
            id="check-delay"
            bind:value={config.post_check.delay}
            label={$t('settings.post_check.check_delay')}
            description={$t('settings.post_check.check_delay_description')}
            placeholder="10"
            minValue={1}
            maxValue={300}
            presets={presets}
          />
        </div>

        <div>
          <Label for="max-reposts" class="mb-2">{$t('settings.post_check.max_reposts')}</Label>
          <Input
            id="max-reposts"
            type="number"
            bind:value={config.post_check.max_reposts}
            min="0"
            max="10"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {$t('settings.post_check.max_reposts_description')}
          </P>
        </div>
      </div>
    {/if}

    <div
      class="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded"
    >
      <P class="text-sm text-yellow-800 dark:text-yellow-200">
        <strong>{$t('settings.post_check.info_title')}</strong> {$t('settings.post_check.info_description')}
      </P>
    </div>
  </div>
</Card>
