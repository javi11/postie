<script lang="ts">
import type { ConfigData } from "$lib/types";
import * as App from "$lib/wailsjs/go/backend/App";
import { Button, Card, Heading, Input, Label, P } from "flowbite-svelte";
import { CogSolid, FolderOpenSolid } from "flowbite-svelte-icons";
import { onMount } from "svelte";

export let config: ConfigData;

let outputDirectory = "";

// Initialize config defaults if they don't exist
if (!config.output_dir) {
	config.output_dir = "./output";
}

onMount(async () => {
	try {
		outputDirectory = await App.GetOutputDirectory();
	} catch (error) {
		console.error("Failed to get output directory:", error);
		outputDirectory = config.output_dir || "./output";
	}
});

async function selectOutputDirectory() {
	try {
		const dir = await App.SelectOutputDirectory();
		if (dir) {
			config.output_dir = dir;
			outputDirectory = dir;
		}
	} catch (error) {
		console.error("Failed to select output directory:", error);
	}
}

// Update display when config changes
$: if (config.output_dir) {
	outputDirectory = config.output_dir;
}
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <CogSolid class="w-5 h-5 text-gray-600 dark:text-gray-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        General Settings
      </Heading>
    </div>

    <div class="space-y-4">
      <div>
        <Label for="output-dir" class="mb-2">Output Directory</Label>
        <div class="flex items-center gap-2">
          <Input
            id="output-dir"
            bind:value={config.output_dir}
            placeholder="./output"
            class="flex-1"
          />
          <Button
            size="sm"
            onclick={selectOutputDirectory}
            class="cursor-pointer flex items-center gap-2"
          >
            <FolderOpenSolid class="w-4 h-4" />
            Browse
          </Button>
        </div>
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Directory where processed files and NZB files will be stored for both
          manual uploads and watcher
        </P>
      </div>

      <div
        class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
      >
        <P class="text-sm text-blue-800 dark:text-blue-200">
          <strong>Output Directory:</strong> This is a global setting that applies
          to both manual file uploads and automatic file watcher uploads. All processed
          files and generated NZB files will be saved to this location.
        </P>
      </div>

      {#if outputDirectory && outputDirectory !== config.output_dir}
        <div
          class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded"
        >
          <P class="text-sm text-amber-800 dark:text-amber-200">
            <strong>Current active directory:</strong>
            {outputDirectory}<br />
            <strong>New directory after save:</strong>
            {config.output_dir}
          </P>
        </div>
      {/if}
    </div>
  </div>
</Card>
