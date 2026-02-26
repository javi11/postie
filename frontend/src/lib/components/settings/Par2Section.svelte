<script lang="ts">
import apiClient from "$lib/api/client";
import PercentageInput from "$lib/components/inputs/PercentageInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Info, ShieldCheck } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

// Preset definitions
const redundancyPresets = [
	{ label: "5%", value: 5 },
	{ label: "10%", value: 10 },
	{ label: "15%", value: 15 },
	{ label: "20%", value: 20 },
];

// Reactive local state
let enabled = $state(config.par2?.enabled ?? false);
let tempDir = $state(config.par2?.temp_dir || "");
let redundancy = $state(config.par2?.redundancy || "10%");
let maintainPar2Files = $state(config.par2?.maintain_par2_files ?? false);

// Sync local state back to config
$effect(() => {
	config.par2.enabled = enabled;
});

$effect(() => {
	config.par2.temp_dir = tempDir;
});

$effect(() => {
	config.par2.redundancy = redundancy;
});

$effect(() => {
	config.par2.maintain_par2_files = maintainPar2Files;
});

async function selectTempDirectory() {
	try {
		const selectedDir = await apiClient.selectTempDirectory();
		if (selectedDir) {
			tempDir = selectedDir;
		}
	} catch (error) {
		console.error("Failed to select temp directory:", error);
		toastStore.error($t("common.messages.error"), "Failed to select directory");
	}
}

// Display values for status cards
let redundancyDisplay = $derived(redundancy || "10%");
</script>

<div class="card bg-base-100 shadow-sm">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <ShieldCheck class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <h2 class="text-lg font-semibold text-base-content">
        {$t('settings.par2.title')}
      </h2>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <input name="par2enable" type="checkbox" class="checkbox" bind:checked={enabled} />
        <div>
          <label for="par2enable" class="text-base font-medium text-base-content">{$t('settings.par2.enable')}</label>
          <p class="text-sm text-base-content/70">
            {$t('settings.par2.enable_description')}
          </p>
        </div>
      </div>

      {#if enabled}
        <div
          class="ml-6 space-y-6 p-4 bg-base-200 rounded-lg border border-base-300"
        >
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label for="temp-dir" class="label">
                <span class="label-text">{$t('settings.par2.temp_dir')}</span>
              </label>
              <div class="flex gap-2">
                <input
                  id="temp-dir"
                  class="input input-bordered flex-1"
                  bind:value={tempDir}
                  placeholder={$t('settings.par2.temp_dir_placeholder')}
                />
                {#if apiClient.environment === 'wails'}
                  <button
                    class="btn btn-outline"
                    onclick={selectTempDirectory}
                  >
                    {$t('settings.general.browse')}
                  </button>
                {/if}
              </div>
              <p class="text-sm text-base-content/70 mt-1">
                {$t('settings.par2.temp_dir_description')}
              </p>
            </div>

            <!-- Maintain PAR2 Files -->
            <div class="form-control">
              <label for="maintain-par2-files" class="cursor-pointer label">
                <span class="label-text">{$t('settings.par2.maintain_par2_files')}</span>
                <input
                  id="maintain-par2-files"
                  type="checkbox"
                  class="checkbox"
                  bind:checked={maintainPar2Files}
                />
              </label>
              <p class="text-sm text-base-content/70 mt-1">
                {$t('settings.par2.maintain_par2_files_description')}
              </p>
            </div>

            <PercentageInput
              bind:value={redundancy}
              label={$t('settings.par2.redundancy')}
              description={$t('settings.par2.redundancy_description')}
              presets={redundancyPresets}
              minValue={1}
              maxValue={50}
              id="redundancy"
            />
          </div>

          <div class="space-y-4">
            <div
              class="alert alert-info"
            >
              <div class="flex items-start gap-3">
                <Info
                  class="w-5 h-5 mt-0.5"
                />
                <div>
                  <p
                    class="text-sm font-medium mb-2"
                  >
                    {$t('settings.par2.info.title')}
                  </p>
                  <ul
                    class="text-sm space-y-1 list-disc list-inside"
                  >
                    <li>{$t('settings.par2.info.features.redundancy_percentage_determines_how_much_data_can_be_recovered')}</li>
                    <li>{$t('settings.par2.info.features.higher_redundancy_better_recovery_but_larger_par2_files')}</li>
                  </ul>
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-4 text-center">
              <div class="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-green-800 dark:text-green-200"
                >
                  {redundancyDisplay}
                </div>
                <div class="text-sm text-green-600 dark:text-green-400">
                  {$t('settings.par2.status.redundancy')}
                </div>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <div
          class="ml-6 p-4 alert alert-warning"
        >
          <p class="text-sm">
            {@html $t('settings.par2.disabled_message')}
          </p>
        </div>
      {/if}
    </div>

  </div>
</div>
