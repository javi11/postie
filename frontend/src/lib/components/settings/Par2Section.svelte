<script lang="ts">
import type { ConfigData } from "$lib/types";
import {
	Button,
	Card,
	Checkbox,
	Heading,
	Input,
	Label,
	P,
	Select,
} from "flowbite-svelte";
import {
	CirclePlusSolid,
	InfoCircleSolid,
	ShieldCheckSolid,
	TrashBinSolid,
} from "flowbite-svelte-icons";
import PercentageInput from "$lib/components/inputs/PercentageInput.svelte";
import SizeInput from "$lib/components/inputs/SizeInput.svelte";

export let config: ConfigData;

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
		case "GB": return Math.round((bytes / 1024 / 1024 / 1024) * 100) / 100;
		case "MB": return Math.round(bytes / 1024 / 1024);
		default: return bytes;
	}
}

// Helper function to convert units back to bytes
function unitToBytes(value: number, unit: string): number {
	switch (unit) {
		case "GB": return value * 1024 * 1024 * 1024;
		case "MB": return value * 1024 * 1024;
		default: return value;
	}
}

// Reactive variables for easier editing
let volumeSizeValue: number;
let volumeSizeUnit: string = "MB";
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
	if (typeof redundancyStr === 'string') {
		redundancyValue = parseInt(redundancyStr.replace('%', '')) || 10;
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

// Display values for status cards
$: redundancyDisplay = config.par2.redundancy || "10%";
$: volumeSizeDisplay = config.par2.volume_size 
	? config.par2.volume_size >= 1073741824 
		? `${Math.round(config.par2.volume_size / 1073741824)} GB`
		: `${Math.round(config.par2.volume_size / 1048576)} MB`
	: "200 MB";
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

            <PercentageInput
              bind:value={config.par2.redundancy}
              label="Redundancy"
              description="Recovery data percentage"
              presets={redundancyPresets}
              minValue={1}
              maxValue={50}
              id="redundancy"
            />

            <SizeInput
              bind:value={config.par2.volume_size}
              label="Volume Size"
              description="Size of each PAR2 volume file"
              presets={volumeSizePresets}
              minValue={1}
              maxValue={2000}
              showBytes={true}
              id="volume-size"
            />

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
                  {redundancyDisplay}
                </div>
                <div class="text-sm text-green-600 dark:text-green-400">
                  Redundancy
                </div>
              </div>

              <div class="p-3 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                <div
                  class="text-lg font-semibold text-purple-800 dark:text-purple-200"
                >
                  {volumeSizeDisplay}
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
