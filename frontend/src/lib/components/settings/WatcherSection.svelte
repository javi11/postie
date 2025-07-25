<script lang="ts">
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import {
	CirclePlus,
	Eye,
	Folder,
	FolderOpen,
	Save,
	Trash2,
} from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

// Reactive local state
let watchDirectory = $state("");
let enabled = $state(config.watcher?.enabled ?? false);
let checkInterval = $state(config.watcher?.check_interval || "5m");
let sizeThreshold = $state(config.watcher?.size_threshold || 104857600); // 100MB
let minFileSize = $state(config.watcher?.min_file_size || 1048576); // 1MB
let deleteOriginalFile = $state(config.watcher?.delete_original_file ?? false);
let startTime = $state(config.watcher?.schedule?.start_time || "00:00");
let endTime = $state(config.watcher?.schedule?.end_time || "23:59");
let saving = $state(false);
let initialized = $state(false);

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

// Derived state
let isAdvanced = $derived($advancedMode);
let canSave = $derived(watchDirectory.trim() && !saving);

// Sync all local state back to config
$effect(() => {
	if (initialized) {
		config.watcher.watch_directory = watchDirectory;
		config.watcher.enabled = enabled;
		config.watcher.check_interval = checkInterval;
		config.watcher.size_threshold = sizeThreshold;
		config.watcher.min_file_size = minFileSize;
		config.watcher.delete_original_file = deleteOriginalFile;
		if (!config.watcher.schedule) {
			config.watcher.schedule = { start_time: "00:00", end_time: "23:59" };
		}
		config.watcher.schedule.start_time = startTime;
		config.watcher.schedule.end_time = endTime;
	}
});

// Initialize watch directory
$effect(() => {
	async function initializeWatchDirectory() {
		if (initialized) return;
		
		try {
			await apiClient.initialize();
			
			if (apiClient.environment === "wails") {
				const { GetWatchDirectory } = await import("$lib/wailsjs/go/backend/App");
				const dir = await GetWatchDirectory();
				
				if (!config.watcher.watch_directory && dir) {
					config.watcher.watch_directory = dir;
					watchDirectory = dir;
				} else if (config.watcher.watch_directory) {
					watchDirectory = config.watcher.watch_directory;
				}
			} else {
				watchDirectory = config.watcher.watch_directory || "";
			}
			
			// Initialize all local state from config
			enabled = config.watcher?.enabled ?? false;
			checkInterval = config.watcher?.check_interval || "5m";
			sizeThreshold = config.watcher?.size_threshold || 104857600;
			minFileSize = config.watcher?.min_file_size || 1048576;
			deleteOriginalFile = config.watcher?.delete_original_file ?? false;
			startTime = config.watcher?.schedule?.start_time || "00:00";
			endTime = config.watcher?.schedule?.end_time || "23:59";
			
			initialized = true;
		} catch (error) {
			console.error("Failed to get watch directory:", error);
			toastStore.error($t("common.messages.error_loading"), String(error));
		}
	}
	
	initializeWatchDirectory();
});

async function selectWatchDirectory() {
	try {
		await apiClient.initialize();
		
		if (apiClient.environment !== "wails") {
			toastStore.warning($t("common.messages.wails_only_feature"));
			return;
		}
		
		const { SelectWatchDirectory } = await import("$lib/wailsjs/go/backend/App");
		const dir = await SelectWatchDirectory();
		
		if (dir) {
			watchDirectory = dir;
			config.watcher.watch_directory = dir;
		}
	} catch (error) {
		console.error("Failed to select directory:", error);
		toastStore.error($t("common.messages.error_selecting_directory"), String(error));
	}
}

function addIgnorePattern() {
	if (!config.watcher.ignore_patterns) {
		config.watcher.ignore_patterns = [];
	}
	config.watcher.ignore_patterns = [...config.watcher.ignore_patterns, ""];
}

function removeIgnorePattern(index: number) {
	if (!config.watcher.ignore_patterns) return;
	
	config.watcher.ignore_patterns = config.watcher.ignore_patterns.filter(
		(_, i) => i !== index,
	);
}

function handlePatternInput(index: number, value: string) {
	if (!config.watcher.ignore_patterns) return;
	
	config.watcher.ignore_patterns[index] = value;
}

async function saveWatcherSettings() {
	if (!canSave) return;
	
	try {
		saving = true;

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		if (!config.watcher) {
			throw new Error("Watcher configuration is missing");
		}

    currentConfig.watcher.enabled = config.watcher.enabled;
    currentConfig.watcher.watch_directory = watchDirectory || currentConfig.watcher.watch_directory;
    currentConfig.watcher.size_threshold = config.watcher.size_threshold ?? currentConfig.watcher.size_threshold;
    currentConfig.watcher.min_file_size = config.watcher.min_file_size ?? currentConfig.watcher.min_file_size;
    currentConfig.watcher.check_interval = config.watcher.check_interval || currentConfig.watcher.check_interval;
    currentConfig.watcher.delete_original_file = config.watcher.delete_original_file ?? currentConfig.watcher.delete_original_file;
    
    // Update schedule if it exists
    if (config.watcher.schedule) {
      currentConfig.watcher.schedule.start_time = config.watcher.schedule.start_time;
      currentConfig.watcher.schedule.end_time = config.watcher.schedule.end_time;
    }
    
    // Update ignore patterns if they exist
    if (config.watcher.ignore_patterns) {
      currentConfig.watcher.ignore_patterns = config.watcher.ignore_patterns;
    }

		// Preserve convertValues method if it exists
		if (currentConfig.watcher.convertValues) {
			currentConfig.watcher.convertValues = currentConfig.watcher.convertValues;
		}

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.watcher.saved_success"),
			$t("settings.watcher.saved_success_description")
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
      <Eye class="w-5 h-5 text-primary" />
      <h2 class="text-lg font-semibold text-base-content">
        {$t('settings.watcher.title')}
      </h2>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <input type="checkbox" class="toggle toggle-primary" bind:checked={enabled} id="watcher-enabled" />
        <div class="flex items-center gap-2">
          <Folder class="w-4 h-4 text-primary" />
          <label class="label-text text-sm font-medium" for="watcher-enabled">{$t('settings.watcher.enable')}</label>
        </div>
      </div>

      <div class="alert alert-info">
        <p class="text-sm">
          <strong>{$t('settings.watcher.title')}:</strong> {$t('settings.watcher.description')}
        </p>
      </div>

      {#if enabled}
        <div class="animate-fade-in pl-4 border-l-2 border-primary/20 space-y-6">
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
                        type="button"
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
                        bind:value={watchDirectory}
                        placeholder="/path/to/watch/directory"
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
              bind:value={checkInterval}
              label={$t('settings.watcher.check_interval')}
              description={$t('settings.watcher.check_interval_description')}
              presets={checkIntervalPresets}
              id="check-interval"
            />

            <SizeInput
              bind:value={sizeThreshold}
              label={$t('settings.watcher.size_threshold')}
              description={$t('settings.watcher.size_threshold_description')}
              presets={sizeThresholdPresets}
              minValue={1}
              id="size-threshold"
            />

            <SizeInput
              bind:value={minFileSize}
              label={$t('settings.watcher.min_file_size')}
              description={$t('settings.watcher.min_file_size_description')}
              presets={minFileSizePresets}
              minValue={0}
              id="min-file-size"
            />
          </div>

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
                      <input type="checkbox" class="toggle toggle-primary" bind:checked={deleteOriginalFile} />
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
                    <div class="grid grid-cols-2 gap-4">
                      <div>
                        <label class="label" for="start-time">
                          <span class="label-text text-sm">{$t('settings.watcher.start_time')}</span>
                        </label>
                        <input
                          id="start-time"
                          type="time"
                          class="input input-bordered w-full"
                          bind:value={startTime}
                        />
                      </div>
                      <div>
                        <label class="label" for="end-time">
                          <span class="label-text text-sm">{$t('settings.watcher.end_time')}</span>
                        </label>
                        <input
                          id="end-time"
                          type="time"
                          class="input input-bordered w-full"
                          bind:value={endTime}
                        />
                      </div>
                    </div>
                  </div>
                  <p class="text-sm text-base-content/70 mt-2">
                    {$t('settings.watcher.time_range_description')}
                  </p>
                </div>
              </div>
            </div>

          {#if isAdvanced}
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
                  type="button"
                  class="btn btn-sm btn-outline"
                  onclick={addIgnorePattern}
                >
                  <CirclePlus class="w-4 h-4" />
                  {$t('settings.watcher.add_pattern')}
                </button>
              </div>

              <div class="space-y-3">
                {#each config.watcher.ignore_patterns || [] as pattern, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <input
                        class="input input-bordered w-full"
                        value={pattern}
                        placeholder={$t('settings.watcher.pattern_placeholder')}
                        oninput={(e) => handlePatternInput(index, e.currentTarget.value)}
                      />
                    </div>
                    <button
                      type="button"
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
        type="button"
        class="btn btn-success"
        onclick={saveWatcherSettings}
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.watcher.save_button')}
      </button>
    </div>
  </div>
</div>
