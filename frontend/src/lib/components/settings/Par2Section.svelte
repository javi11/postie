<script lang="ts">
  import {
    Card,
    Heading,
    Input,
    Label,
    Checkbox,
    Button,
    P,
  } from "flowbite-svelte";
  import {
    ShieldCheckSolid,
    InfoCircleSolid,
    CirclePlusSolid,
    TrashBinSolid,
  } from "flowbite-svelte-icons";
  import type { ConfigData } from "$lib/types";

  export let config: ConfigData;

  // Ensure extra_par2_options exists
  if (!config.par2.extra_par2_options) {
    config.par2.extra_par2_options = [];
  }

  // Helper function to format bytes to MB for display
  function bytesToMB(bytes: number): number {
    return Math.round(bytes / 1024 / 1024);
  }

  // Helper function to convert MB back to bytes
  function mbToBytes(mb: number): number {
    return mb * 1024 * 1024;
  }

  // Reactive variables for easier editing
  $: volumeSizeMB = bytesToMB(config.par2.volume_size);

  function updateVolumeSize(event: Event) {
    const target = event.target as HTMLInputElement;
    const mb = parseInt(target.value) || 200;
    config.par2.volume_size = mbToBytes(mb);
  }

  function addExtraOption() {
    config.par2.extra_par2_options = [...config.par2.extra_par2_options, ""];
  }

  function removeExtraOption(index: number) {
    config.par2.extra_par2_options = config.par2.extra_par2_options.filter(
      (_, i) => i !== index
    );
  }
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <ShieldCheckSolid class="w-5 h-5 text-purple-600 dark:text-purple-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        PAR2 Recovery Files
      </Heading>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Checkbox bind:checked={config.par2.enabled} />
        <div>
          <Label class="text-base font-medium">Enable PAR2 generation</Label>
          <P class="text-sm text-gray-600 dark:text-gray-400">
            Generate recovery files for error correction and repair
          </P>
        </div>
      </div>

      {#if config.par2.enabled}
        <div
          class="ml-6 space-y-6 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700"
        >
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <Label for="par2-path" class="mb-2">PAR2 Executable Path</Label>
              <Input
                id="par2-path"
                bind:value={config.par2.par2_path}
                placeholder="./parpar"
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Path to PAR2 executable or parpar
              </P>
            </div>

            <div>
              <Label for="redundancy" class="mb-2">Redundancy</Label>
              <Input
                id="redundancy"
                bind:value={config.par2.redundancy}
                placeholder="10%"
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Recovery data percentage (e.g., 10%, 15%)
              </P>
            </div>

            <div>
              <Label for="volume-size" class="mb-2">Volume Size (MB)</Label>
              <Input
                id="volume-size"
                type="number"
                value={volumeSizeMB}
                min="1"
                max="1000"
                onchange={updateVolumeSize}
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Size of each PAR2 volume file ({config.par2.volume_size.toLocaleString()}
                bytes)
              </P>
            </div>

            <div>
              <Label for="max-slices" class="mb-2">Max Input Slices</Label>
              <Input
                id="max-slices"
                type="number"
                bind:value={config.par2.max_input_slices}
                min="100"
                max="10000"
              />
              <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Maximum number of input slices for processing
              </P>
            </div>
          </div>

          <!-- Extra PAR2 Options Section -->
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <Heading
                  tag="h4"
                  class="text-sm font-medium text-gray-900 dark:text-white"
                >
                  Extra PAR2 Options
                </Heading>
                <P class="text-sm text-gray-600 dark:text-gray-400">
                  Additional command-line arguments for PAR2 executable
                </P>
              </div>
              <Button
                size="sm"
                onclick={addExtraOption}
                class="cursor-pointer flex items-center gap-2"
              >
                <CirclePlusSolid class="w-4 h-4" />
                Add Option
              </Button>
            </div>

            {#if config.par2.extra_par2_options && config.par2.extra_par2_options.length > 0}
              <div class="space-y-3">
                {#each config.par2.extra_par2_options as option, index (index)}
                  <div class="flex items-center gap-3">
                    <div class="flex-1">
                      <Input
                        bind:value={config.par2.extra_par2_options[index]}
                        placeholder="--option=value"
                      />
                    </div>
                    <Button
                      size="sm"
                      color="red"
                      variant="outline"
                      onclick={() => removeExtraOption(index)}
                      class="cursor-pointer flex items-center gap-1"
                    >
                      <TrashBinSolid class="w-3 h-3" />
                      Remove
                    </Button>
                  </div>
                {/each}
              </div>
            {:else}
              <div
                class="p-3 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded"
              >
                <P class="text-sm text-gray-600 dark:text-gray-400">
                  No extra options configured. Add custom PAR2 arguments if
                  needed.
                </P>
              </div>
            {/if}
          </div>

          <div class="space-y-4">
            <div
              class="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg"
            >
              <div class="flex items-start gap-3">
                <InfoCircleSolid
                  class="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5"
                />
                <div>
                  <P
                    class="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2"
                  >
                    PAR2 Recovery Information
                  </P>
                  <ul
                    class="text-sm text-blue-700 dark:text-blue-300 space-y-1 list-disc list-inside"
                  >
                    <li>
                      PAR2 files allow recovery of damaged or missing files
                    </li>
                    <li>
                      Redundancy percentage determines how much data can be
                      recovered
                    </li>
                    <li>
                      Higher redundancy = better recovery but larger PAR2 files
                    </li>
                    <li>
                      Volume size controls how PAR2 data is split across files
                    </li>
                    <li>Extra options allow fine-tuning of PAR2 generation</li>
                  </ul>
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-center">
              <div class="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-green-800 dark:text-green-200"
                >
                  {config.par2.redundancy}
                </div>
                <div class="text-sm text-green-600 dark:text-green-400">
                  Redundancy
                </div>
              </div>

              <div class="p-3 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-purple-800 dark:text-purple-200"
                >
                  {volumeSizeMB} MB
                </div>
                <div class="text-sm text-purple-600 dark:text-purple-400">
                  Volume Size
                </div>
              </div>

              <div class="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-blue-800 dark:text-blue-200"
                >
                  {config.par2.max_input_slices.toLocaleString()}
                </div>
                <div class="text-sm text-blue-600 dark:text-blue-400">
                  Max Slices
                </div>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <div
          class="ml-6 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg"
        >
          <P class="text-sm text-yellow-800 dark:text-yellow-200">
            <strong>PAR2 disabled:</strong> Recovery files will not be generated.
            This may make it difficult to repair damaged uploads.
          </P>
        </div>
      {/if}
    </div>
  </div>
</Card>
