<script lang="ts">
import apiClient from "$lib/api/client";
import PercentageInput from "$lib/components/inputs/PercentageInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { CirclePlus, Info, Save, ShieldCheck, Trash2 } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

let saving = $state(false);

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

// Reactive local state
let enabled = $state(config.par2?.enabled ?? false);
let par2Path = $state(config.par2?.par2_path || "");
let tempDir = $state(config.par2?.temp_dir || "");
let redundancy = $state(config.par2?.redundancy || "10%");
let volumeSize = $state(config.par2?.volume_size || 209715200);
let maxInputSlices = $state(config.par2?.max_input_slices || 4000);
let extraPar2Options = $state<string[]>(config.par2?.extra_par2_options || []);
let maintainPar2Files = $state(config.par2?.maintain_par2_files ?? false);
let volumeSizeValue = $state<number>(0);
let volumeSizeUnit = $state("MB");
let redundancyValue = $state<number>(10);
let isAdvanced = $derived($advancedMode);

// Derived state
let canSave = $derived(
	(!enabled || (enabled && par2Path.trim())) &&
	!saving
);

// Sync local state back to config
$effect(() => {
	config.par2.enabled = enabled;
});

$effect(() => {
	config.par2.par2_path = par2Path;
});

$effect(() => {
	config.par2.temp_dir = tempDir;
});

$effect(() => {
	config.par2.redundancy = redundancy;
});

$effect(() => {
	config.par2.volume_size = volumeSize;
});

$effect(() => {
	config.par2.max_input_slices = maxInputSlices;
});

$effect(() => {
	config.par2.extra_par2_options = extraPar2Options;
});

$effect(() => {
	config.par2.maintain_par2_files = maintainPar2Files;
});

// Parse existing values and update local state
$effect(() => {
	const size = volumeSize || 209715200; // 200MB default
	if (size >= 1073741824 && size % 1073741824 === 0) {
		volumeSizeValue = size / 1073741824;
		volumeSizeUnit = "GB";
	} else {
		volumeSizeValue = Math.round(size / 1048576);
		volumeSizeUnit = "MB";
	}
});

// Parse redundancy percentage
$effect(() => {
	const redundancyStr = redundancy || "10%";
	if (typeof redundancyStr === "string") {
		redundancyValue = Number.parseInt(redundancyStr.replace("%", "")) || 10;
	} else {
		redundancyValue = 10;
	}
});

// Update local values when display values change
$effect(() => {
	if (volumeSizeValue !== undefined && volumeSizeUnit) {
		volumeSize = unitToBytes(volumeSizeValue, volumeSizeUnit);
	}
});

$effect(() => {
	if (redundancyValue !== undefined && redundancyValue > 0) {
		redundancy = `${redundancyValue}%`;
	}
});

function addExtraOption() {
	extraPar2Options = [...extraPar2Options, ""];
}

function removeExtraOption(index: number) {
	extraPar2Options = extraPar2Options.filter((_, i) => i !== index);
}

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

async function savePar2Settings() {
	if (!canSave) return;

	try {
		saving = true;

		// Validation
		if (enabled && !par2Path.trim()) {
			throw new Error("PAR2 path is required when PAR2 is enabled");
		}

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Only update the par2 fields with proper type conversion
		currentConfig.par2 = {
			...currentConfig.par2,
			enabled: enabled,
			par2_path: par2Path.trim(),
			temp_dir: tempDir.trim(),
			redundancy: redundancy,
			volume_size: volumeSize || 153600000,
			max_input_slices: maxInputSlices || 4000,
			extra_par2_options: extraPar2Options,
			maintain_par2_files: maintainPar2Files,
		};

		await apiClient.saveConfig(currentConfig);
	} catch (error) {
		console.error("Failed to save PAR2 settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}

// Display values for status cards
let redundancyDisplay = $derived(redundancy || "10%");
let volumeSizeDisplay = $derived(volumeSize
	? volumeSize >= 1073741824
		? `${Math.round(volumeSize / 1073741824)} GB`
		: `${Math.round(volumeSize / 1048576)} MB`
	: "200 MB");
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
              <label for="par2-path" class="label">
                <span class="label-text">{$t('settings.par2.par2_path')}</span>
              </label>
              <input
                id="par2-path"
                class="input input-bordered w-full"
                bind:value={par2Path}
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

            <SizeInput
              value={volumeSize}
              onchange={(value) => volumeSize = value}
              label={$t('settings.par2.volume_size')}
              description={$t('settings.par2.volume_size_description')}
              presets={volumeSizePresets}
              minValue={1}
              maxValue={2000}
              showBytes={true}
              id="volume-size"
            />

{#if isAdvanced}
            <div>
              <label for="max-slices" class="label">
                <span class="label-text">{$t('settings.par2.max_input_slices')}</span>
              </label>
              <input
                id="max-slices"
                type="number"
                class="input input-bordered w-full"
                bind:value={maxInputSlices}
                min="100"
                max="10000"
              />
              <p class="text-sm text-base-content/70 mt-1">
                {$t('settings.par2.max_input_slices_description')}
              </p>
            </div>
{/if}
          </div>

{#if isAdvanced}
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

            {#if extraPar2Options && extraPar2Options.length > 0}
              <div class="space-y-3">
                {#each extraPar2Options as option, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <input
                        class="input input-bordered w-full"
                        bind:value={extraPar2Options[index]}
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
                  {maxInputSlices.toLocaleString()}
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
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.par2.save_button')}
      </button>
    </div>
  </div>
</div>
