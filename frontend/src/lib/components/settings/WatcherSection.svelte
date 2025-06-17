<script lang="ts">
import type { ConfigData } from "$lib/types";
import * as App from "$lib/wailsjs/go/backend/App";
import {
	Button,
	Card,
	Checkbox,
	Heading,
	Input,
	Label,
	P,
	Textarea,
	Select,
	Timepicker,
} from "flowbite-svelte";
import {
	CirclePlusSolid,
	EyeSolid,
	FolderOpenSolid,
	FolderSolid,
	TrashBinSolid,
} from "flowbite-svelte-icons";
import { onMount } from "svelte";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";

export let config: ConfigData;

let watchDirectory = "";

// Initialize watcher config if it doesn't exist
if (!config.watcher) {
	config.watcher = {
		enabled: false,
		size_threshold: 104857600, // 100MB
		schedule: {
			start_time: "00:00",
			end_time: "23:59",
		},
		ignore_patterns: ["*.tmp", "*.part", "*.!ut"],
		min_file_size: 1048576, // 1MB
		check_interval: 300000000000, // 5m in nanoseconds
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
$: checkIntervalSeconds = nanosToSeconds(config.watcher.check_interval || 300000000000);

// Convert duration string back to nanoseconds
function updateCheckInterval(durationString: string) {
	const match = durationString.match(/^(\d+)([smh])$/);
	if (match) {
		const value = parseInt(match[1]);
		const unit = match[2];
		let seconds = value;
		if (unit === 'm') seconds = value * 60;
		if (unit === 'h') seconds = value * 3600;
		config.watcher.check_interval = secondsToNanos(seconds);
	}
}

onMount(async () => {
	try {
		watchDirectory = await App.GetWatchDirectory();
	} catch (error) {
		console.error("Failed to get watch directory:", error);
	}
});

async function selectWatchDirectory() {
	try {
		const dir = await App.SelectWatchDirectory();
		if (dir) {
			watchDirectory = dir;
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
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <EyeSolid class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        File Watcher
      </Heading>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Checkbox bind:checked={config.watcher.enabled} />
        <div class="flex items-center gap-2">
          <FolderSolid class="w-4 h-4 text-purple-600 dark:text-purple-400" />
          <Label class="text-sm font-medium">Enable File Watcher</Label>
        </div>
      </div>

      <div
        class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
      >
        <P class="text-sm text-blue-800 dark:text-blue-200">
          <strong>File Watcher:</strong> Automatically monitors a directory for new
          files and uploads them when they meet the configured criteria. Files are
          queued for processing and moved to the global output directory after successful
          upload.
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
                Directories
              </Heading>
              <P class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Configure where the watcher monitors files for automatic uploads
              </P>

              <div>
                <Label class="mb-2">Watch Directory</Label>
                <div class="flex items-center gap-2">
                  <Input
                    value={watchDirectory}
                    readonly
                    placeholder="Select watch directory..."
                    class="flex-1"
                  />
                  <Button
                    size="sm"
                    onclick={selectWatchDirectory}
                    class="cursor-pointer flex items-center gap-2"
                  >
                    <FolderOpenSolid class="w-4 h-4" />
                    Browse
                  </Button>
                </div>
                <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Directory where new files will be monitored for automatic
                  upload. Processed files will be moved to the global output
                  directory.
                </P>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Custom DurationInput for check interval since it needs nanosecond conversion -->
            <div>
              <Label for="check-interval" class="mb-2">Check Interval</Label>
              <div class="flex gap-2">
                <div class="flex-1">
                  <Input
                    id="check-interval"
                    type="number"
                    value={Math.round(checkIntervalSeconds >= 3600 ? checkIntervalSeconds / 3600 : checkIntervalSeconds >= 60 ? checkIntervalSeconds / 60 : checkIntervalSeconds)}
                    min="1"
                    max="3600"
                    on:input={(e) => {
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
                    <option value="s">Seconds</option>
                    <option value="m">Minutes</option>
                    <option value="h">Hours</option>
                  </select>
                </div>
              </div>
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                How often to scan for new files
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
              label="Size Threshold"
              description="Minimum accumulated size before batch processing"
              presets={sizeThresholdPresets}
              minValue={1}
              maxValue={10000}
              id="size-threshold"
            />

            <SizeInput
              bind:value={config.watcher.min_file_size}
              label="Min File Size"
              description="Minimum size of individual files to process"
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
                Posting Schedule
              </Heading>
              <P class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Define when the watcher is allowed to post files (24-hour
                format)
              </P>

              <div class="space-y-4">
                <div>
                  <Label class="mb-2">Time Range</Label>
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
                    Define when the watcher is allowed to post files (24-hour format)
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
                    Ignore Patterns
                  </Heading>
                  <P class="text-sm text-gray-600 dark:text-gray-400">
                    File patterns to ignore (uses glob syntax)
                  </P>
                </div>
                <Button
                  size="sm"
                  onclick={addIgnorePattern}
                  class="cursor-pointer flex items-center gap-2"
                >
                  <CirclePlusSolid class="w-4 h-4" />
                  Add Pattern
                </Button>
              </div>

              <div class="space-y-3">
                {#each config.watcher.ignore_patterns as pattern, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <Input
                        bind:value={config.watcher.ignore_patterns[index]}
                        placeholder="*.tmp"
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
                      Remove
                    </Button>
                  </div>
                {/each}
              </div>

              <div
                class="p-3 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded"
              >
                <P class="text-sm text-gray-700 dark:text-gray-300">
                  <strong>Common patterns:</strong> *.tmp (temporary files), *.part
                  (partial downloads), *.!ut (uTorrent), *.crdownload (Chrome downloads),
                  *.download (Firefox)
                </P>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</Card>
