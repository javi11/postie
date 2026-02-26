<script lang="ts">
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { config as configType } from "$lib/wailsjs/go/models";
import {
	ChevronDown,
	ChevronUp,
	CirclePlus,
	Eye,
	Folder,
	FolderOpen,
	Trash2,
} from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

// Ensure watchers array is initialized
$effect(() => {
	if (!config.watchers || config.watchers.length === 0) {
		config.watchers = [createDefaultWatcher()];
	}
});

// Track expanded state per watcher card
let expanded = $state<boolean[]>([]);
$effect(() => {
	// Grow expanded array to match watchers length, default new ones to true
	const watchers = config.watchers || [];
	while (expanded.length < watchers.length) {
		expanded.push(true);
	}
});

// Preset definitions
const checkIntervalPresets = [
  { label: "30s", value: 30, unit: "s" },
  { label: "2m", value: 2, unit: "m" },
  { label: "5m", value: 5, unit: "m" },
  { label: "10m", value: 10, unit: "m" },
];

const minFileAgePresets = [
  { label: "30s", value: 30, unit: "s" },
  { label: "1m", value: 1, unit: "m" },
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
  { label: "0B", value: 0, unit: "B" },
  { label: "1MB", value: 1, unit: "MB" },
  { label: "10MB", value: 10, unit: "MB" },
  { label: "50MB", value: 50, unit: "MB" },
  { label: "100MB", value: 100, unit: "MB" },
];

function createDefaultWatcher(): configType.WatcherConfig {
	const w = new configType.WatcherConfig();
	w.name = "";
	w.enabled = false;
	w.watch_directory = "";
	w.check_interval = "5m";
	w.min_file_age = "60s";
	w.size_threshold = 104857600;
	w.min_file_size = 1048576;
	w.delete_original_file = false;
	w.single_nzb_per_folder = false;
	w.follow_symlinks = false;
	w.schedule = { start_time: "00:00", end_time: "23:59" };
	w.ignore_patterns = [];
	return w;
}

function addWatcher() {
	config.watchers = [...(config.watchers || []), createDefaultWatcher()];
	expanded = [...expanded, true];
}

function removeWatcher(index: number) {
	if ((config.watchers || []).length <= 1) return;
	config.watchers = (config.watchers || []).filter((_, i) => i !== index);
	expanded = expanded.filter((_, i) => i !== index);
}

function toggleExpanded(index: number) {
	expanded[index] = !expanded[index];
}

async function selectWatchDirectory(index: number) {
	try {
		await apiClient.initialize();

		if (apiClient.environment !== "wails") {
			toastStore.warning($t("common.messages.wails_only_feature"));
			return;
		}

		const { SelectWatchDirectory } = await import("$lib/wailsjs/go/backend/App");
		const dir = await SelectWatchDirectory();

		if (dir) {
			config.watchers[index].watch_directory = dir;
		}
	} catch (error) {
		console.error("Failed to select directory:", error);
		toastStore.error($t("common.messages.error_selecting_directory"), String(error));
	}
}

function addIgnorePattern(index: number) {
	const w = config.watchers[index];
	w.ignore_patterns = [...(w.ignore_patterns || []), ""];
	config.watchers[index] = w;
}

function removeIgnorePattern(watcherIndex: number, patternIndex: number) {
	const w = config.watchers[watcherIndex];
	w.ignore_patterns = (w.ignore_patterns || []).filter((_, i) => i !== patternIndex);
	config.watchers[watcherIndex] = w;
}

function handlePatternInput(watcherIndex: number, patternIndex: number, value: string) {
	const w = config.watchers[watcherIndex];
	w.ignore_patterns = (w.ignore_patterns || []).map((p, i) => i === patternIndex ? value : p);
	config.watchers[watcherIndex] = w;
}

function getWatcherDisplayName(w: configType.WatcherConfig, index: number): string {
	if (w.name) return w.name;
	return $t("settings.watcher.watcher_number", { values: { n: index + 1 } });
}

</script>

<div class="card bg-base-100 shadow-sm">
  <div class="card-body space-y-6">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <Eye class="w-5 h-5 text-primary" />
        <h2 class="text-lg font-semibold text-base-content">
          {$t('settings.watcher.title')}
        </h2>
      </div>
      <button
        type="button"
        class="btn btn-sm btn-outline"
        onclick={addWatcher}
      >
        <CirclePlus class="w-4 h-4" />
        {$t('settings.watcher.add_directory')}
      </button>
    </div>

    <div class="alert alert-info">
      <p class="text-sm">
        <strong>{$t('settings.watcher.title')}:</strong> {$t('settings.watcher.description')}
      </p>
    </div>

    <div class="space-y-4">
      {#each (config.watchers || []) as watcher, index (index)}
        <div class="card bg-base-200/50 border border-base-300/60">
          <!-- Card Header -->
          <div class="px-4 py-3 flex items-center gap-3 cursor-pointer"
               role="button"
               tabindex="0"
               onclick={() => toggleExpanded(index)}
               onkeydown={(e) => e.key === 'Enter' && toggleExpanded(index)}>
            <input
              type="checkbox"
              class="toggle toggle-primary toggle-sm"
              bind:checked={watcher.enabled}
              onclick={(e) => e.stopPropagation()}
            />
            <div class="flex-1 min-w-0">
              {#if watcher.name}
                <span class="font-medium text-base-content">{watcher.name}</span>
              {:else}
                <span class="text-base-content/60 italic text-sm">
                  {$t('settings.watcher.watcher_number', { values: { n: index + 1 } })}
                </span>
              {/if}
              {#if watcher.watch_directory}
                <p class="text-xs text-base-content/50 truncate">{watcher.watch_directory}</p>
              {/if}
            </div>
            <div class="flex items-center gap-2">
              {#if (config.watchers || []).length > 1}
                <button
                  type="button"
                  class="btn btn-xs btn-error btn-outline"
                  onclick={(e) => { e.stopPropagation(); removeWatcher(index); }}
                  title={$t('settings.watcher.remove_directory')}
                >
                  <Trash2 class="w-3 h-3" />
                </button>
              {/if}
              {#if expanded[index]}
                <ChevronUp class="w-4 h-4 text-base-content/60" />
              {:else}
                <ChevronDown class="w-4 h-4 text-base-content/60" />
              {/if}
            </div>
          </div>

          <!-- Card Body (collapsible) -->
          {#if expanded[index]}
            <div class="px-4 pb-4 space-y-6 border-t border-base-300/40">
              <!-- Name -->
              <div class="pt-4">
                <label class="label" for="watcher-name-{index}">
                  <span class="label-text">{$t('settings.watcher.watcher_name')}</span>
                </label>
                <input
                  id="watcher-name-{index}"
                  type="text"
                  class="input input-bordered w-full"
                  bind:value={watcher.name}
                  placeholder={$t('settings.watcher.watcher_number', { values: { n: index + 1 } })}
                />
                <p class="text-sm text-base-content/70 mt-1">
                  {$t('settings.watcher.watcher_name_description')}
                </p>
              </div>

              <!-- Watch Directory -->
              <div>
                <label class="label" for="watch-directory-{index}">
                  <span class="label-text">{$t('settings.watcher.watch_directory')}</span>
                </label>
                <div class="flex items-center gap-2">
                  {#if apiClient.environment === 'wails'}
                    <input
                      id="watch-directory-{index}"
                      class="input input-bordered flex-1"
                      value={watcher.watch_directory}
                      readonly
                      placeholder={$t('common.inputs.select_directory')}
                    />
                    <button
                      type="button"
                      class="btn btn-sm btn-outline"
                      onclick={() => selectWatchDirectory(index)}
                    >
                      <FolderOpen class="w-4 h-4" />
                      {$t('common.inputs.browse')}
                    </button>
                  {:else}
                    <input
                      id="watch-directory-{index}"
                      class="input input-bordered flex-1"
                      bind:value={watcher.watch_directory}
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

              <!-- Timing Settings -->
              <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <DurationInput
                  bind:value={watcher.check_interval}
                  label={$t('settings.watcher.check_interval')}
                  description={$t('settings.watcher.check_interval_description')}
                  presets={checkIntervalPresets}
                  id="check-interval-{index}"
                />

                <DurationInput
                  bind:value={watcher.min_file_age}
                  label={$t('settings.watcher.min_file_age')}
                  description={$t('settings.watcher.min_file_age_description')}
                  presets={minFileAgePresets}
                  id="min-file-age-{index}"
                />

                <SizeInput
                  bind:value={watcher.size_threshold}
                  label={$t('settings.watcher.size_threshold')}
                  description={$t('settings.watcher.size_threshold_description')}
                  presets={sizeThresholdPresets}
                  minValue={1}
                  id="size-threshold-{index}"
                />

                <SizeInput
                  bind:value={watcher.min_file_size}
                  label={$t('settings.watcher.min_file_size')}
                  description={$t('settings.watcher.min_file_size_description')}
                  presets={minFileSizePresets}
                  minValue={0}
                  id="min-file-size-{index}"
                />
              </div>

              <!-- Behavior -->
              <div class="space-y-4">
                <h3 class="text-md font-medium text-base-content">
                  {$t('settings.watcher.behavior')}
                </h3>

                <div>
                  <div class="form-control">
                    <label class="label cursor-pointer justify-start gap-3">
                      <input type="checkbox" class="toggle toggle-primary" bind:checked={watcher.delete_original_file} />
                      <span class="label-text">{$t('settings.watcher.delete_original_file')}</span>
                    </label>
                  </div>
                  <p class="text-sm text-base-content/70">
                    {$t('settings.watcher.delete_original_file_description')}
                  </p>
                </div>

                <div>
                  <div class="form-control">
                    <label class="label cursor-pointer justify-start gap-3">
                      <input type="checkbox" class="toggle toggle-primary" bind:checked={watcher.single_nzb_per_folder} />
                      <span class="label-text">{$t('settings.watcher.single_nzb_per_folder')}</span>
                    </label>
                  </div>
                  <p class="text-sm text-base-content/70">
                    {$t('settings.watcher.single_nzb_per_folder_description')}
                  </p>
                </div>
              </div>

              <!-- Schedule -->
              <div class="space-y-4">
                <h3 class="text-md font-medium text-base-content">
                  {$t('settings.watcher.posting_schedule')}
                </h3>
                <p class="text-sm text-base-content/70">
                  {$t('settings.watcher.posting_schedule_description')}
                </p>

                <div class="grid grid-cols-2 gap-4">
                  <div>
                    <label class="label" for="start-time-{index}">
                      <span class="label-text text-sm">{$t('settings.watcher.start_time')}</span>
                    </label>
                    <input
                      id="start-time-{index}"
                      type="time"
                      class="input input-bordered w-full"
                      bind:value={watcher.schedule.start_time}
                    />
                  </div>
                  <div>
                    <label class="label" for="end-time-{index}">
                      <span class="label-text text-sm">{$t('settings.watcher.end_time')}</span>
                    </label>
                    <input
                      id="end-time-{index}"
                      type="time"
                      class="input input-bordered w-full"
                      bind:value={watcher.schedule.end_time}
                    />
                  </div>
                </div>
              </div>

              <!-- Ignore Patterns -->
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
                    onclick={() => addIgnorePattern(index)}
                  >
                    <CirclePlus class="w-4 h-4" />
                    {$t('settings.watcher.add_pattern')}
                  </button>
                </div>

                <div class="space-y-3">
                  {#each (watcher.ignore_patterns || []) as pattern, pi (pi)}
                    <div class="flex items-center gap-3">
                      <input
                        class="input input-bordered flex-1"
                        value={pattern}
                        placeholder={$t('settings.watcher.pattern_placeholder')}
                        oninput={(e) => handlePatternInput(index, pi, e.currentTarget.value)}
                      />
                      <button
                        type="button"
                        class="btn btn-sm btn-error btn-outline"
                        onclick={() => removeIgnorePattern(index, pi)}
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
            </div>
          {/if}
        </div>
      {/each}
    </div>

  </div>
</div>
