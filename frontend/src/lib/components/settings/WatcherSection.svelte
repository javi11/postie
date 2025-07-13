<script lang="ts">
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import {
	CirclePlus,
	Clock,
	Eye,
	Folder,
	FolderOpen,
	Save,
	Trash2,
} from "lucide-svelte";
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

// Convert duration string to Go duration format (nanoseconds)
function durationStringToNanos(durationStr: string): number {
	const match = durationStr.match(/^(\d+)([smh])$/);
	if (match) {
		const value = Number.parseInt(match[1]);
		const unit = match[2];

		let seconds = value;
		if (unit === "m") seconds = value * 60;
		if (unit === "h") seconds = value * 3600;

		return seconds * 1000000000; // Convert to nanoseconds
	}
	return 300000000000; // Default 5 minutes
}

// Convert nanoseconds to duration string for DurationInput
function nanosToSeconds(nanos: number): number {
	return Math.round(nanos / 1000000000);
}

function getCheckIntervalDuration(): string {
	const totalSeconds = nanosToSeconds(
		config.watcher.check_interval || 300000000000,
	);

	if (totalSeconds >= 3600) {
		return `${Math.round(totalSeconds / 3600)}h`;
	}
	if (totalSeconds >= 60) {
		return `${Math.round(totalSeconds / 60)}m`;
	}

	return `${Math.round(totalSeconds)}s`;
}

function updateCheckIntervalFromDuration(durationStr: string) {
	config.watcher.check_interval = durationStringToNanos(durationStr);
}

// Reactive duration value for DurationInput
$: checkIntervalDuration = getCheckIntervalDuration();

onMount(async () => {
	try {
		// Check if we're in Wails environment
		if (apiClient.environment === "wails") {
			const { App } = await import("$lib/wailsjs/go/backend/App");
			watchDirectory = await App.GetWatchDirectory();
			// Sync with config if it's not already set
			if (!config.watcher.watch_directory && watchDirectory) {
				config.watcher.watch_directory = watchDirectory;
			} else if (config.watcher.watch_directory) {
				watchDirectory = config.watcher.watch_directory;
			}
		} else {
			// In web mode, use the config value directly
			watchDirectory = config.watcher.watch_directory || "";
		}
	} catch (error) {
		console.error("Failed to get watch directory:", error);
	}
});

async function selectWatchDirectory() {
	try {
		// Check if we're in Wails environment
		if (apiClient.environment === "wails") {
			const { App } = await import("$lib/wailsjs/go/backend/App");
			const dir = await App.SelectWatchDirectory();
			if (dir) {
				watchDirectory = dir;
				config.watcher.watch_directory = dir;
			}
		}
		// In web mode, users can just type the path directly in the input field
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
		const currentConfig = await apiClient.getConfig();

		// Only update the watcher fields with proper type conversion
		if (config.watcher) {
			currentConfig.watcher = {
				...config.watcher,
				watch_directory: config.watcher.watch_directory || "",
				size_threshold:
					Number.parseInt(config.watcher.size_threshold) || 104857600,
				min_file_size: Number.parseInt(config.watcher.min_file_size) || 1048576,
				check_interval: config.watcher.check_interval || 300000000000,
				delete_original_file: config.watcher.delete_original_file || false,
			};
		}

		await apiClient.saveConfig(currentConfig);

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

<div class="card bg-base-100 shadow-sm">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <Eye class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <h2 class="text-lg font-semibold text-base-content">
        {$t('settings.watcher.title')}
      </h2>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <input type="checkbox" class="toggle toggle-primary" bind:checked={config.watcher.enabled} id="watcher-enabled" />
        <div class="flex items-center gap-2">
          <Folder class="w-4 h-4 text-purple-600 dark:text-purple-400" />
          <label class="label-text text-sm font-medium" for="watcher-enabled">{$t('settings.watcher.enable')}</label>
        </div>
      </div>

      <div class="alert alert-info">
        <p class="text-sm">
          <strong>{$t('settings.watcher.title')}:</strong> {$t('settings.watcher.description')}
        </p>
      </div>

      {#if config.watcher.enabled}
        <div class="pl-4 border-l-2 border-purple-200 dark:border-purple-800 space-y-6">
          <div class="space-y-4">
            <div>
              <h3 class="text-md font-medium text-base-content mb-2">
                {$t('settings.watcher.directories')}
              </h3>
              <p class="text-sm text-base-content/70 mb-4">
                {$t('settings.watcher.directories_description')}
              </p>

              <div>
                <label class="label" for="watch-directory">
                  <span class="label-text">{$t('settings.watcher.watch_directory')}</span>
                </label>
                <div class="flex items-center gap-2">
                    {#if apiClient.environment === 'wails'}
                      <input
                        id="watch-directory"
                        class="input input-bordered flex-1"
                        value={watchDirectory}
                        readonly
                        placeholder={$t('common.inputs.select_directory')}
                      />
                      <button
                        class="btn btn-sm btn-outline"
                        onclick={selectWatchDirectory}
                      >
                        <FolderOpen class="w-4 h-4" />
                        {$t('common.inputs.browse')}
                      </button>
                    {:else}
                      <input
                        id="watch-directory"
                        class="input input-bordered flex-1"
                        bind:value={config.watcher.watch_directory}
                        placeholder="/path/to/watch/directory"
                        oninput={() => {
                          // Keep watchDirectory in sync for consistency
                          watchDirectory = config.watcher.watch_directory;
                        }}
                      />
                    {/if}
                </div>
                <p class="text-sm text-base-content/70 mt-1">
                  {$t('settings.watcher.watch_directory_description')}
                    {#if apiClient.environment === 'web'}
                      <br /><span class="text-primary text-xs">Enter the container path directly (e.g., /app/watch)</span>
                    {/if}
                </p>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <DurationInput
              value={checkIntervalDuration}
              label={$t('settings.watcher.check_interval')}
              description={$t('settings.watcher.check_interval_description')}
              presets={checkIntervalPresets}
              minValue={1}
              maxValue={3600}
              id="check-interval"
              onchange={(newValue) => updateCheckIntervalFromDuration(newValue)}
            />

            <SizeInput
              value={config.watcher.size_threshold}
              onchange={(value) => config.watcher.size_threshold = value}
              label={$t('settings.watcher.size_threshold')}
              description={$t('settings.watcher.size_threshold_description')}
              presets={sizeThresholdPresets}
              minValue={1}
              id="size-threshold"
            />

            <SizeInput
              value={config.watcher.min_file_size}
              onchange={(value) => config.watcher.min_file_size = value}
              label={$t('settings.watcher.min_file_size')}
              description={$t('settings.watcher.min_file_size_description')}
              presets={minFileSizePresets}
              minValue={0}
              id="min-file-size"
            />
          </div>

      {#if $advancedMode}
          <div class="space-y-4">
            <div>
              <h3 class="text-md font-medium text-base-content mb-2">
                {$t('settings.watcher.behavior')}
              </h3>
              <p class="text-sm text-base-content/70 mb-4">
                {$t('settings.watcher.behavior_description')}
              </p>

              <div class="space-y-4">
                <div>
                  <div class="form-control">
                    <label class="label cursor-pointer justify-start gap-3">
                      <input type="checkbox" class="toggle toggle-primary" bind:checked={config.watcher.delete_original_file} />
                      <span class="label-text">{$t('settings.watcher.delete_original_file')}</span>
                    </label>
                  </div>
                  <p class="text-sm text-base-content/70">
                    {$t('settings.watcher.delete_original_file_description')}
                  </p>
                </div>
              </div>
            </div>
          </div>
      {/if}

          <div class="space-y-4">
            <div>
              <h3 class="text-md font-medium text-base-content mb-2">
                {$t('settings.watcher.posting_schedule')}
              </h3>
              <p class="text-sm text-base-content/70 mb-4">
                {$t('settings.watcher.posting_schedule_description')}
              </p>

              <div class="space-y-4">
                <div>
                  <div class="form-control">
                    <div class="mb-2">
                      <span class="text-sm font-medium text-base-content">{$t('settings.watcher.time_range')}</span>
                    </div>
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
                  </div>
                  <p class="text-sm text-base-content/70 mt-2">
                    {$t('settings.watcher.time_range_description')}
                  </p>
                </div>
              </div>
            </div>

          {#if $advancedMode}
            <div class="space-y-4">
              <div class="flex items-center justify-between">
                <div>
                  <h3 class="text-md font-medium text-base-content">
                    {$t('settings.watcher.ignore_patterns')}
                  </h3>
                  <p class="text-sm text-base-content/70">
                    {$t('settings.watcher.ignore_patterns_description')}
                  </p>
                </div>
                <button
                  class="btn btn-sm btn-outline"
                  onclick={addIgnorePattern}
                >
                  <CirclePlus class="w-4 h-4" />
                  {$t('settings.watcher.add_pattern')}
                </button>
              </div>

              <div class="space-y-3">
                {#each config.watcher.ignore_patterns as pattern, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <input
                        class="input input-bordered w-full"
                        bind:value={config.watcher.ignore_patterns[index]}
                        placeholder={$t('settings.watcher.pattern_placeholder')}
                      />
                    </div>
                    <button
                      class="btn btn-sm btn-error btn-outline"
                      onclick={() => removeIgnorePattern(index)}
                    >
                      <Trash2 class="w-3 h-3" />
                      {$t('settings.watcher.remove')}
                    </button>
                  </div>
                {/each}
              </div>

              <div class="alert">
                <p class="text-sm">
                  {@html $t('settings.watcher.common_patterns')}
                </p>
              </div>
            </div>
          {/if}
          </div>
        </div>
      {/if}
    </div>

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        class="btn btn-success"
        onclick={saveWatcherSettings}
        disabled={saving}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('settings.watcher.saving') : $t('settings.watcher.save_button')}
      </button>
    </div>
  </div>
</div>
