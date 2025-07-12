<script lang="ts">
import { createEventDispatcher } from "svelte";
import { t } from "$lib/i18n";
import { Button, Input, Label, Card } from "flowbite-svelte";
import { FolderOpenSolid } from "flowbite-svelte-icons";
import apiClient from "$lib/api/client";

const dispatch = createEventDispatcher();

export let outputDirectory = "";
export let watchDirectory = "";

function updateDirectories() {
	dispatch("update", { outputDirectory, watchDirectory });
}

async function selectOutputDirectory() {
	try {
		const dir = await apiClient.selectOutputDirectory();
		if (dir) {
			outputDirectory = dir;
			updateDirectories();
		}
	} catch (error) {
		console.error("Failed to select output directory:", error);
	}
}

async function selectWatchDirectory() {
	try {
		const dir = await apiClient.selectWatchDirectory();
		if (dir) {
			watchDirectory = dir;
			updateDirectories();
		}
	} catch (error) {
		console.error("Failed to select watch directory:", error);
	}
}

// Set default output directory if empty
if (!outputDirectory) {
	outputDirectory = "./output";
	updateDirectories();
}
</script>

<div class="space-y-6">
	<div>
		<h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
			{$t("setup.directories.title")}
		</h3>
		<p class="text-gray-600 dark:text-gray-400 mb-4">
			{$t("setup.directories.description")}
		</p>
	</div>

	<div class="space-y-6">
		<!-- Output Directory -->
		<Card class="p-4 max-w-full">
			<div class="mb-4">
				<h4 class="font-medium text-gray-900 dark:text-white mb-2">
					{$t("setup.directories.outputDirectory")}
				</h4>
				<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
					{$t("setup.directories.outputDescription")}
				</p>
			</div>
			<Label for="output-dir" class="mb-2">
				{$t("setup.directories.outputPath")} *
			</Label>
			<div class="flex gap-2">
				<Input
					id="output-dir"
					bind:value={outputDirectory}
					placeholder="./output"
					required
					oninput={updateDirectories}
					class="flex-1"
				/>
				{#if apiClient.environment === "wails"}
					<Button
						color="alternative"
						outline
						onclick={selectOutputDirectory}
					>
						<FolderOpenSolid class="w-4 h-4 mr-1" />
						{$t("setup.directories.browse")}
					</Button>
				{/if}
			</div>
		</Card>
	</div>

	<div class="text-sm text-gray-500 dark:text-gray-400">
		<p>* {$t("setup.directories.requiredFields")}</p>
	</div>
</div>