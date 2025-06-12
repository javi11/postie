<script lang="ts">
  import {
    Card,
    Heading,
    Input,
    Label,
    Checkbox,
    Button,
    P,
    Textarea,
  } from "flowbite-svelte";
  import {
    EyeSolid,
    CirclePlusSolid,
    TrashBinSolid,
    FolderSolid,
    FolderOpenSolid,
  } from "flowbite-svelte-icons";
  import type { ConfigData } from "$lib/types";
  import * as App from "$lib/wailsjs/go/main/App";
  import { onMount } from "svelte";

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

  // Convert nanoseconds to human readable format for display
  function nanosToHumanReadable(nanos: number): string {
    const seconds = nanos / 1000000000;
    if (seconds < 60) return `${seconds}s`;
    const minutes = seconds / 60;
    if (minutes < 60) return `${Math.round(minutes)}m`;
    const hours = minutes / 60;
    return `${Math.round(hours)}h`;
  }

  // Convert human readable format to nanoseconds
  function humanReadableToNanos(str: string): number {
    const num = parseFloat(str);
    if (str.includes("h")) return num * 60 * 60 * 1000000000;
    if (str.includes("m")) return num * 60 * 1000000000;
    if (str.includes("s")) return num * 1000000000;
    return num * 1000000000; // Default to seconds
  }

  // Reactive variable for check interval display
  $: checkIntervalDisplay =
    typeof config.watcher.check_interval === "number"
      ? nanosToHumanReadable(config.watcher.check_interval)
      : config.watcher.check_interval || "5m";

  function updateCheckInterval(event: Event) {
    const target = event.target as HTMLInputElement;
    const value = target.value;
    config.watcher.check_interval = humanReadableToNanos(value);
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
      (_, i) => i !== index
    );
  }

  // Convert bytes to MB for display
  function bytesToMB(bytes: number): number {
    return Math.round(bytes / 1048576);
  }

  // Convert MB to bytes for storage
  function mbToBytes(mb: number): number {
    return mb * 1048576;
  }

  // Reactive variables for display
  $: sizeThresholdMB = bytesToMB(config.watcher.size_threshold || 100);
  $: minFileSizeMB = bytesToMB(config.watcher.min_file_size || 1);

  // Update config when display values change
  function updateSizeThreshold(event: Event) {
    const target = event.target as HTMLInputElement;
    config.watcher.size_threshold = mbToBytes(parseInt(target.value) || 100);
  }

  function updateMinFileSize(event: Event) {
    const target = event.target as HTMLInputElement;
    config.watcher.min_file_size = mbToBytes(parseInt(target.value) || 1);
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
            <div>
              <Label for="check-interval" class="mb-2">Check Interval</Label>
              <Input
                id="check-interval"
                value={checkIntervalDisplay}
                on:input={updateCheckInterval}
                placeholder="5m"
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                How often to scan for new files (e.g., 30s, 5m, 1h)
              </P>
            </div>

            <div>
              <Label for="size-threshold" class="mb-2"
                >Size Threshold (MB)</Label
              >
              <Input
                id="size-threshold"
                type="number"
                value={sizeThresholdMB}
                on:input={updateSizeThreshold}
                min="1"
                max="10000"
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Minimum accumulated size before batch processing
              </P>
            </div>

            <div>
              <Label for="min-file-size" class="mb-2">Min File Size (MB)</Label>
              <Input
                id="min-file-size"
                type="number"
                value={minFileSizeMB}
                on:input={updateMinFileSize}
                min="0"
                max="1000"
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Minimum size of individual files to process
              </P>
            </div>
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

              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label for="start-time" class="mb-2">Start Time</Label>
                  <Input
                    id="start-time"
                    type="time"
                    bind:value={config.watcher.schedule.start_time}
                  />
                </div>

                <div>
                  <Label for="end-time" class="mb-2">End Time</Label>
                  <Input
                    id="end-time"
                    type="time"
                    bind:value={config.watcher.schedule.end_time}
                  />
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
