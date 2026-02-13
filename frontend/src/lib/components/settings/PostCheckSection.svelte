<script lang="ts">
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { CheckCircle, Save } from "lucide-svelte";

const presets = [
	{ label: "5s", value: 5, unit: "s" },
	{ label: "10s", value: 10, unit: "s" },
	{ label: "30s", value: 30, unit: "s" },
	{ label: "1m", value: 1, unit: "m" },
];

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

// Reactive local state
let enabled = $state(config.post_check?.enabled ?? true);
let delay = $state(config.post_check?.delay || "10s");
let maxReposts = $state(config.post_check?.max_reposts || 1);
let saving = $state(false);

// Derived state
let canSave = $derived(
	delay.trim() && 
	maxReposts >= 0 && 
	maxReposts <= 10 && 
	!saving
);

// Ensure post_check exists with defaults
if (!config.post_check) {
	config.post_check = {
		enabled: true,
		delay: "10s",
		max_reposts: 1,
	};
}

// Sync local state back to config
$effect(() => {
	config.post_check.enabled = enabled;
});

$effect(() => {
	config.post_check.delay = delay;
});

$effect(() => {
	config.post_check.max_reposts = maxReposts;
});

async function savePostCheckSettings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Validation
		if (!delay.trim()) {
			throw new Error("Delay is required");
		}
		
		if (maxReposts < 0 || maxReposts > 10) {
			throw new Error("Max reposts must be between 0 and 10");
		}

		// Only update the post_check fields with proper type conversion
		currentConfig.post_check = {
			enabled: enabled,
			delay: delay.trim(),
			max_reposts: maxReposts,
		};

		await apiClient.saveConfig(currentConfig);
	} catch (error) {
		console.error("Failed to save post check settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}
</script>

<div class="card bg-base-100 shadow-xl">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <CheckCircle class="w-5 h-5 text-orange-600 dark:text-orange-400" />
      <h2 class="card-title text-lg">
        {$t('settings.post_check.title')}
      </h2>
    </div>

    <div class="form-control">
      <label class="label cursor-pointer justify-start gap-3">
        <input type="checkbox" class="checkbox" bind:checked={enabled} />
        <span class="label-text">{$t('settings.post_check.enable')}</span>
      </label>
      <div class="label">
        <span class="label-text-alt ml-8">
          {$t('settings.post_check.enable_description')}
        </span>
      </div>
    </div>

    {#if enabled}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <DurationInput
            id="check-delay"
            bind:value={delay}
            label={$t('settings.post_check.check_delay')}
            description={$t('settings.post_check.check_delay_description')}
            placeholder="10"
            minValue={1}
            maxValue={300}
            presets={presets}
          />
        </div>

        <div class="form-control">
          <label class="label" for="max-reposts">
            <span class="label-text">{$t('settings.post_check.max_reposts')}</span>
          </label>
          <input
            id="max-reposts"
            type="number"
            class="input input-bordered"
            bind:value={maxReposts}
            min="0"
            max="10"
          />
          <div class="label">
            <span class="label-text-alt">
              {$t('settings.post_check.max_reposts_description')}
            </span>
          </div>
        </div>
      </div>
    {/if}

    <div class="alert alert-warning">
      <span class="text-sm">
        <strong>{$t('settings.post_check.info_title')}</strong> {$t('settings.post_check.info_description')}
      </span>
    </div>

    <!-- Save Button -->
    <div class="card-actions pt-4 border-t border-base-300">
      <button
        class="btn btn-success"
        onclick={savePostCheckSettings}
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.post_check.save_button')}
      </button>
    </div>
  </div>
</div>
