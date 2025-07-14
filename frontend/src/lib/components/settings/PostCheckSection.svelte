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

export let config: configType.ConfigData;

let saving = false;

// Ensure post_check exists with defaults
if (!config.post_check) {
	config.post_check = {
		enabled: true,
		delay: "10s",
		max_reposts: 1,
	};
}

async function savePostCheckSettings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Only update the post_check fields with proper type conversion
		currentConfig.post_check = {
			enabled: config.post_check.enabled || false,
			delay: config.post_check.delay || "10s",
			max_reposts: config.post_check.max_reposts || 1,
		};

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.post_check.saved_success"),
			$t("settings.post_check.saved_success_description"),
		);
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
        <input type="checkbox" class="checkbox" bind:checked={config.post_check.enabled} />
        <span class="label-text">{$t('settings.post_check.enable')}</span>
      </label>
      <div class="label">
        <span class="label-text-alt ml-8">
          {$t('settings.post_check.enable_description')}
        </span>
      </div>
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

        <div class="form-control">
          <label class="label" for="max-reposts">
            <span class="label-text">{$t('settings.post_check.max_reposts')}</span>
          </label>
          <input
            id="max-reposts"
            type="number"
            class="input input-bordered"
            bind:value={config.post_check.max_reposts}
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
        disabled={saving}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('settings.post_check.saving') : $t('settings.post_check.save_button')}
      </button>
    </div>
  </div>
</div>
