<script lang="ts">
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import * as App from "$lib/wailsjs/go/backend/App";
import {
	Button,
	Card,
	Heading,
	Input,
	Label,
	P,
	Select,
	Textarea,
	Timepicker,
	Toggle,
} from "flowbite-svelte";
import {
	CirclePlusSolid,
	EyeSolid,
	FloppyDiskSolid,
	FolderOpenSolid,
	FolderSolid,
	TrashBinSolid,
} from "flowbite-svelte-icons";
import { onMount } from "svelte";

export let config: ConfigData;

let watchDirectory = "";
let saving = false;

// Initialize watcher config if it doesn't exist
if (!config.watcher) {
	config.watcher = {
		enabled: false,
		watch_directory: "",
		size_threshold: 104857600, // 100MB
		schedule: {
			start_time: "00:00",
			end_time: "23:59",
		},
		ignore_patterns: ["*.tmp", "*.part", "*.!ut"],
		min_file_size: 1048576, // 1MB
		check_interval: 300000000000, // 5m in nanoseconds
		delete_original_file: false,
	};
}

// Preset definitions
const checkIntervalPresets = [
	{ label: "30s", value: 30, unit: "s" },
	{ label: "2m", value: 2, unit: "m" },
	{ label: "5m", value: 5, unit: "m" },
	{ label: "10m", value: 10, unit: "m" },
];

const sizeThresholdPresets = [
	{ label: "50MB", value: 50, unit: "MB" },
	{ label: "100MB", value: 100, unit: "MB" },
	{ label: "500MB", value: 500, unit: "MB" },
	{ label: "1GB", value: 1, unit: "GB" },
];

const minFileSizePresets = [
	{ label: "1MB", value: 1, unit: "MB" },
	{ label: "10MB", value: 10, unit: "MB" },
	{ label: "50MB", value: 50, unit: "MB" },
	{ label: "100MB", value: 100, unit: "MB" },
];

// Convert nanoseconds to duration string for DurationInput
function nanosToSeconds(nanos: number): number {
	return Math.round(nanos / 1000000000);
}

function secondsToNanos(seconds: number): number {
	return seconds * 1000000000;
}

// Convert check interval for display
let checkIntervalSeconds: number;
$: checkIntervalSeconds = nanosToSeconds(
	config.watcher.check_interval || 300000000000,
);

// Convert duration string back to nanoseconds
function updateCheckInterval(durationString: string) {
	const match = durationString.match(/^(\d+)([smh])$/);
	if (match) {
		const value = Number.parseInt(match[1]);
		const unit = match[2];
		let seconds = value;
		if (unit === "m") seconds = value * 60;
		if (unit === "h") seconds = value * 3600;
		config.watcher.check_interval = secondsToNanos(seconds);
	}
}

onMount(async () => {
	try {
		watchDirectory = await App.GetWatchDirectory();
		// Sync with config if it's not already set
		if (!config.watcher.watch_directory && watchDirectory) {
			config.watcher.watch_directory = watchDirectory;
		} else if (config.watcher.watch_directory) {
			watchDirectory = config.watcher.watch_directory;
		}
	} catch (error) {
		console.error("Failed to get watch directory:", error);
	}
});

async function selectWatchDirectory() {
	try {
		const dir = await App.SelectWatchDirectory();
		if (dir) {
			watchDirectory = dir;
			config.watcher.watch_directory = dir;
		}
	} catch (error) {
		console.error("Failed to select directory:", error);
	}
}

function addIgnorePattern() {
	if (!config.watcher.ignore_patterns) {
		config.watcher.ignore_patterns = [];
	}
	config.watcher.ignore_patterns = [...config.watcher.ignore_patterns, ""];
}

function removeIgnorePattern(index: number) {
	config.watcher.ignore_patterns = config.watcher.ignore_patterns.filter(
		(_, i) => i !== index,
	);
}

async function saveWatcherSettings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await App.GetConfig();

		// Only update the watcher fields with proper type conversion
		if (config.watcher) {
			currentConfig.watcher = {
				...config.watcher,
				watch_directory: config.watcher.watch_directory || "",
				size_threshold:
					Number.parseInt(config.watcher.size_threshold) || 104857600,
				min_file_size: Number.parseInt(config.watcher.min_file_size) || 1048576,
				check_interval: config.watcher.check_interval || "5m",
				delete_original_file: config.watcher.delete_original_file || false,
			};
		}

		await App.SaveConfig(currentConfig);

		toastStore.success(
			$t("settings.watcher.saved_success"),
			$t("settings.watcher.saved_success_description"),
		);
	} catch (error) {
		console.error("Failed to save watcher settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <EyeSolid class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        {$t('settings.watcher.title')}
      </Heading>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Toggle bind:checked={config.watcher.enabled} />
        <div class="flex items-center gap-2">
          <FolderSolid class="w-4 h-4 text-purple-600 dark:text-purple-400" />
          <Label class="text-sm font-medium">{$t('settings.watcher.enable')}</Label>
        </div>
      </div>

      <div
        class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
      >
        <P class="text-sm text-blue-800 dark:text-blue-200">
          <strong>{$t('settings.watcher.title')}:</strong> {$t('settings.watcher.description')}
        </P>
      </div>

      {#if config.watcher.enabled}
        <div
          class="pl-4 border-l-2 border-purple-200 dark:border-purple-800 space-y-6"
        >
          <div class="space-y-4">
            <div>
              <Heading
                tag="h3"
                class="text-md font-medium text-gray-900 dark:text-white mb-2"
              >
                {$t('settings.watcher.directories')}
              </Heading>
              <P class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {$t('settings.watcher.directories_description')}
              </P>

              <div>
                <Label class="mb-2">{$t('settings.watcher.watch_directory')}</Label>
                <div class="flex items-center gap-2">
                  <Input
                    value={watchDirectory}
                    readonly
                    placeholder={$t('common.inputs.select_directory')}
                    class="flex-1"
                  />
                  <Button
                    size="sm"
                    onclick={selectWatchDirectory}
                    class="cursor-pointer flex items-center gap-2"
                  >
                    <FolderOpenSolid class="w-4 h-4" />
                    {$t('common.inputs.browse')}
                  </Button>
                </div>
                <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  {$t('settings.watcher.watch_directory_description')}
                </P>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Custom DurationInput for check interval since it needs nanosecond conversion -->
            <div>
              <Label for="check-interval" class="mb-2">{$t('settings.watcher.check_interval')}</Label>
              <div class="flex gap-2">
                <div class="flex-1">
                  <Input
                    id="check-interval"
                    type="number"
                    value={Math.round(checkIntervalSeconds >= 3600 ? checkIntervalSeconds / 3600 : checkIntervalSeconds >= 60 ? checkIntervalSeconds / 60 : checkIntervalSeconds)}
                    min="1"
                    max="3600"
                    oninput={(e) => {
                      const val = parseInt(e.target.value) || 5;
                      const seconds = checkIntervalSeconds >= 3600 ? val * 3600 : checkIntervalSeconds >= 60 ? val * 60 : val;
                      config.watcher.check_interval = secondsToNanos(seconds);
                    }}
                  />
                </div>
                <div class="w-24">
                  <select
                    class="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500"
                    value={checkIntervalSeconds >= 3600 ? 'h' : checkIntervalSeconds >= 60 ? 'm' : 's'}
                    onchange={(e) => {
                      const currentVal = Math.round(checkIntervalSeconds >= 3600 ? checkIntervalSeconds / 3600 : checkIntervalSeconds >= 60 ? checkIntervalSeconds / 60 : checkIntervalSeconds);
                      const unit = e.target.value;
                      let seconds = currentVal;
                      if (unit === 'm') seconds = currentVal * 60;
                      if (unit === 'h') seconds = currentVal * 3600;
                      config.watcher.check_interval = secondsToNanos(seconds);
                    }}
                  >
                    <option value="s">{$t('common.inputs.time_units.seconds')}</option>
                    <option value="m">{$t('common.inputs.time_units.minutes')}</option>
                    <option value="h">{$t('common.inputs.time_units.hours')}</option>
                  </select>
                </div>
              </div>
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                {$t('settings.watcher.check_interval_description')}
              </P>
              <div class="mt-2 flex flex-wrap gap-2">
                <button
                  type="button"
                  class="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                  onclick={() => { config.watcher.check_interval = secondsToNanos(30); }}
                >
                  30s
                </button>
                <button
                  type="button"
                  class="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                  onclick={() => { config.watcher.check_interval = secondsToNanos(120); }}
                >
                  2m
                </button>
                <button
                  type="button"
                  class="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                  onclick={() => { config.watcher.check_interval = secondsToNanos(300); }}
                >
                  5m
                </button>
                <button
                  type="button"
                  class="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                  onclick={() => { config.watcher.check_interval = secondsToNanos(600); }}
                >
                  10m
                </button>
              </div>
            </div>

            <SizeInput
              bind:value={config.watcher.size_threshold}
              label={$t('settings.watcher.size_threshold')}
              description={$t('settings.watcher.size_threshold_description')}
              presets={sizeThresholdPresets}
              minValue={1}
              maxValue={10000}
              id="size-threshold"
            />

            <SizeInput
              bind:value={config.watcher.min_file_size}
              label={$t('settings.watcher.min_file_size')}
              description={$t('settings.watcher.min_file_size_description')}
              presets={minFileSizePresets}
              minValue={0}
              maxValue={1000}
              id="min-file-size"
            />
          </div>

          <div class="space-y-4">
            <div>
              <Heading
                tag="h3"
                class="text-md font-medium text-gray-900 dark:text-white mb-2"
              >
                {$t('settings.watcher.behavior')}
              </Heading>
              <P class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {$t('settings.watcher.behavior_description')}
              </P>

              <div class="space-y-4">
                <div>
                  <Toggle
                    bind:checked={config.watcher.delete_original_file}
                    class="mb-2"
                  >
                    {$t('settings.watcher.delete_original_file')}
                  </Toggle>
                  <P class="text-sm text-gray-600 dark:text-gray-400">
                    {$t('settings.watcher.delete_original_file_description')}
                  </P>
                </div>
              </div>
            </div>
          </div>

          <div class="space-y-4">
            <div>
              <Heading
                tag="h3"
                class="text-md font-medium text-gray-900 dark:text-white mb-2"
              >
                {$t('settings.watcher.posting_schedule')}
              </Heading>
              <P class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {$t('settings.watcher.posting_schedule_description')}
              </P>

              <div class="space-y-4">
                <div>
                  <Label class="mb-2">{$t('settings.watcher.time_range')}</Label>
                  <Timepicker
                    type="range"
                    value={config.watcher.schedule.start_time}
                    endValue={config.watcher.schedule.end_time}
                    onselect={(data) => {
                      if (data.time) config.watcher.schedule.start_time = data.time;
                      if (data.endTime) config.watcher.schedule.end_time = data.endTime;
                    }}
                    divClass="shadow-none"
                  />
                  <P class="text-sm text-gray-600 dark:text-gray-400 mt-2">
                    {$t('settings.watcher.time_range_description')}
                  </P>
                </div>
              </div>
            </div>

            <div class="space-y-4">
              <div class="flex items-center justify-between">
                <div>
                  <Heading
                    tag="h3"
                    class="text-md font-medium text-gray-900 dark:text-white"
                  >
                    {$t('settings.watcher.ignore_patterns')}
                  </Heading>
                  <P class="text-sm text-gray-600 dark:text-gray-400">
                    {$t('settings.watcher.ignore_patterns_description')}
                  </P>
                </div>
                <Button
                  size="sm"
                  onclick={addIgnorePattern}
                  class="cursor-pointer flex items-center gap-2"
                >
                  <CirclePlusSolid class="w-4 h-4" />
                  {$t('settings.watcher.add_pattern')}
                </Button>
              </div>

              <div class="space-y-3">
                {#each config.watcher.ignore_patterns as pattern, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <Input
                        bind:value={config.watcher.ignore_patterns[index]}
                        placeholder={$t('settings.watcher.pattern_placeholder')}
                      />
                    </div>
                    <Button
                      size="sm"
                      color="red"
                      variant="outline"
                      onclick={() => removeIgnorePattern(index)}
                      class="cursor-pointer flex items-center gap-1"
                    >
                      <TrashBinSolid class="w-3 h-3" />
                      {$t('settings.watcher.remove')}
                    </Button>
                  </div>
                {/each}
              </div>

              <div
                class="p-3 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded"
              >
                <P class="text-sm text-gray-700 dark:text-gray-300">
                  {@html $t('settings.watcher.common_patterns')}
                </P>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Save Button -->
    <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
      <Button
        color="green"
        onclick={saveWatcherSettings}
        disabled={saving}
        class="cursor-pointer flex items-center gap-2"
      >
        <FloppyDiskSolid class="w-4 h-4" />
        {saving ? $t('settings.watcher.saving') : $t('settings.watcher.save_button')}
      </Button>
    </div>
  </div>
</Card>
