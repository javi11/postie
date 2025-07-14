<script lang="ts">
import apiClient from "$lib/api/client";
import PercentageInput from "$lib/components/inputs/PercentageInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { CirclePlus, Info, Save, ShieldCheck, Trash2 } from "lucide-svelte";

export let config: configType.ConfigData;

let saving = false;

// Ensure extra_par2_options exists
if (!config.par2.extra_par2_options) {
	config.par2.extra_par2_options = [];
}

const sizeUnitOptions = [
	{ value: "MB", name: "MB" },
	{ value: "GB", name: "GB" },
];

// Preset definitions
const redundancyPresets = [
	{ label: "5%", value: 5 },
	{ label: "10%", value: 10 },
	{ label: "15%", value: 15 },
	{ label: "20%", value: 20 },
];

const volumeSizePresets = [
	{ label: "100MB", value: 100, unit: "MB" },
	{ label: "200MB", value: 200, unit: "MB" },
	{ label: "500MB", value: 500, unit: "MB" },
	{ label: "1GB", value: 1, unit: "GB" },
];

// Helper function to format bytes to different units for display
function bytesToUnit(bytes: number, unit: string): number {
	switch (unit) {
		case "GB":
			return Math.round((bytes / 1024 / 1024 / 1024) * 100) / 100;
		case "MB":
			return Math.round(bytes / 1024 / 1024);
		default:
			return bytes;
	}
}

// Helper function to convert units back to bytes
function unitToBytes(value: number, unit: string): number {
	switch (unit) {
		case "GB":
			return value * 1024 * 1024 * 1024;
		case "MB":
			return value * 1024 * 1024;
		default:
			return value;
	}
}

// Reactive variables for easier editing
let volumeSizeValue: number;
let volumeSizeUnit = "MB";
let redundancyValue: number;

// Parse existing values
$: {
	const size = config.par2.volume_size || 209715200; // 200MB default
	if (size >= 1073741824 && size % 1073741824 === 0) {
		volumeSizeValue = size / 1073741824;
		volumeSizeUnit = "GB";
	} else {
		volumeSizeValue = Math.round(size / 1048576);
		volumeSizeUnit = "MB";
	}
}

// Parse redundancy percentage
$: {
	const redundancyStr = config.par2.redundancy || "10%";
	if (typeof redundancyStr === "string") {
		redundancyValue = Number.parseInt(redundancyStr.replace("%", "")) || 10;
	} else {
		redundancyValue = 10;
	}
}

// Update config when display values change
$: {
	if (volumeSizeValue !== undefined && volumeSizeUnit) {
		config.par2.volume_size = unitToBytes(volumeSizeValue, volumeSizeUnit);
	}
}

$: {
	if (redundancyValue !== undefined && redundancyValue > 0) {
		config.par2.redundancy = `${redundancyValue}%`;
	}
}

function addExtraOption() {
	config.par2.extra_par2_options = [...config.par2.extra_par2_options, ""];
}

function removeExtraOption(index: number) {
	config.par2.extra_par2_options = config.par2.extra_par2_options.filter(
		(_, i) => i !== index,
	);
}

async function selectTempDirectory() {
	try {
		const selectedDir = await apiClient.selectTempDirectory();
		if (selectedDir) {
			config.par2.temp_dir = selectedDir;
		}
	} catch (error) {
		console.error("Failed to select temp directory:", error);
		toastStore.error($t("common.messages.error"), "Failed to select directory");
	}
}

async function savePar2Settings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Only update the par2 fields with proper type conversion
		currentConfig.par2 = {
			...config.par2,
			volume_size: config.par2.volume_size || 153600000,
			max_input_slices: config.par2.max_input_slices || 4000,
		};

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.par2.saved_success"),
			$t("settings.par2.saved_success_description"),
		);
	} catch (error) {
		console.error("Failed to save PAR2 settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}

// Display values for status cards
$: redundancyDisplay = config.par2.redundancy || "10%";
$: volumeSizeDisplay = config.par2.volume_size
	? config.par2.volume_size >= 1073741824
		? `${Math.round(config.par2.volume_size / 1073741824)} GB`
		: `${Math.round(config.par2.volume_size / 1048576)} MB`
	: "200 MB";
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
        <input name="par2enable" type="checkbox" class="checkbox" bind:checked={config.par2.enabled} />
        <div>
          <label for="par2enable" class="text-base font-medium text-base-content">{$t('settings.par2.enable')}</label>
          <p class="text-sm text-base-content/70">
            {$t('settings.par2.enable_description')}
          </p>
        </div>
      </div>

      {#if config.par2.enabled}
        <div
          class="ml-6 space-y-6 p-4 bg-base-200 rounded-lg border border-base-300"
        >
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label for="par2-path" class="label">
                <span class="label-text">{$t('settings.par2.par2_path')}</span>
              </label>
              <input
                id="par2-path"
                class="input input-bordered w-full"
                bind:value={config.par2.par2_path}
                placeholder="./parpar"
              />
              <p class="text-sm text-base-content/70 mt-1">
                {$t('settings.par2.par2_path_description')}
              </p>
            </div>

            <div>
              <label for="temp-dir" class="label">
                <span class="label-text">{$t('settings.par2.temp_dir')}</span>
              </label>
              <div class="flex gap-2">
                <input
                  id="temp-dir"
                  class="input input-bordered flex-1"
                  bind:value={config.par2.temp_dir}
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

            <PercentageInput
              bind:value={config.par2.redundancy}
              label={$t('settings.par2.redundancy')}
              description={$t('settings.par2.redundancy_description')}
              presets={redundancyPresets}
              minValue={1}
              maxValue={50}
              id="redundancy"
            />

            <SizeInput
              value={config.par2.volume_size}
              onchange={(value) => config.par2.volume_size = value}
              label={$t('settings.par2.volume_size')}
              description={$t('settings.par2.volume_size_description')}
              presets={volumeSizePresets}
              minValue={1}
              maxValue={2000}
              showBytes={true}
              id="volume-size"
            />

{#if $advancedMode}
            <div>
              <label for="max-slices" class="label">
                <span class="label-text">{$t('settings.par2.max_input_slices')}</span>
              </label>
              <input
                id="max-slices"
                type="number"
                class="input input-bordered w-full"
                bind:value={config.par2.max_input_slices}
                min="100"
                max="10000"
              />
              <p class="text-sm text-base-content/70 mt-1">
                {$t('settings.par2.max_input_slices_description')}
              </p>
            </div>
{/if}
          </div>

{#if $advancedMode}
          <!-- Extra PAR2 Options Section -->
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <h4 class="text-sm font-medium text-base-content">
                  {$t('settings.par2.extra_options.title')}
                </h4>
                <p class="text-sm text-base-content/70">
                  {$t('settings.par2.extra_options.description')}
                </p>
              </div>
              <button
                class="btn btn-sm btn-outline"
                onclick={addExtraOption}
              >
                <CirclePlus class="w-4 h-4" />
                {$t('settings.par2.extra_options.add_option')}
              </button>
            </div>

            {#if config.par2.extra_par2_options && config.par2.extra_par2_options.length > 0}
              <div class="space-y-3">
                {#each config.par2.extra_par2_options as option, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <input
                        class="input input-bordered w-full"
                        bind:value={config.par2.extra_par2_options[index]}
                        placeholder={$t('settings.par2.extra_options.placeholder')}
                      />
                    </div>
                    <button
                      class="btn btn-sm btn-error btn-outline"
                      onclick={() => removeExtraOption(index)}
                    >
                      <Trash2 class="w-3 h-3" />
                      {$t('settings.par2.extra_options.remove')}
                    </button>
                  </div>
                {/each}
              </div>
            {:else}
              <div
                class="p-3 bg-base-200 border border-base-300 rounded"
              >
                <p class="text-sm text-base-content/70">
                  {$t('settings.par2.extra_options.no_options')}
                </p>
              </div>
            {/if}
          </div>
{/if}

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
                    <li>{$t('settings.par2.info.features.volume_size_controls_how_par2_data_is_split_across_files')}</li>
                    <li>{$t('settings.par2.info.features.extra_options_allow_fine_tuning_of_par2_generation')}</li>
                  </ul>
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-center">
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

              <div class="p-3 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-purple-800 dark:text-purple-200"
                >
                  {volumeSizeDisplay}
                </div>
                <div class="text-sm text-purple-600 dark:text-purple-400">
                  {$t('settings.par2.status.volume_size')}
                </div>
              </div>

              <div class="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-blue-800 dark:text-blue-200"
                >
                  {config.par2.max_input_slices.toLocaleString()}
                </div>
                <div class="text-sm text-blue-600 dark:text-blue-400">
                  {$t('settings.par2.status.max_slices')}
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

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        class="btn btn-success"
        onclick={savePar2Settings}
        disabled={saving}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('settings.par2.saving') : $t('settings.par2.save_button')}
      </button>
    </div>
  </div>
</div>
