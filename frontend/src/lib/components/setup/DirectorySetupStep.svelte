<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { FolderOpen } from "lucide-svelte";

interface Props {
	outputDirectory?: string;
	watchDirectory?: string;
	onupdate?: (data: {
		outputDirectory: string;
		watchDirectory: string;
	}) => void;
}

let { outputDirectory = "", watchDirectory = "", onupdate }: Props = $props();

function updateDirectories() {
	onupdate?.({ outputDirectory, watchDirectory });
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

// Set default output directory if empty
if (!outputDirectory) {
	outputDirectory = "./output";
	updateDirectories();
}
</script>

<div class="space-y-6">
	<div>
		<h3 class="text-lg font-semibold text-base-content mb-2">
			{$t("setup.directories.title")}
		</h3>
		<p class="text-base-content/70 mb-4">
			{$t("setup.directories.description")}
		</p>
	</div>

	<div class="space-y-6">
		<!-- Output Directory -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<div class="mb-4">
					<h4 class="card-title">
						{$t("setup.directories.outputDirectory")}
					</h4>
					<p class="text-sm text-base-content/70 mb-4">
						{$t("setup.directories.outputDescription")}
					</p>
				</div>
				<div class="form-control">
					<label class="label" for="output-dir">
						<span class="label-text">{$t("setup.directories.outputPath")} *</span>
					</label>
					<div class="flex gap-2">
						<input
							id="output-dir"
							class="input input-bordered flex-1"
							bind:value={outputDirectory}
							placeholder="./output"
							required
							oninput={updateDirectories}
						/>
						{#if apiClient.environment === "wails"}
							<button
								class="btn btn-outline"
								onclick={selectOutputDirectory}
							>
								<FolderOpen class="w-4 h-4" />
								{$t("setup.directories.browse")}
							</button>
						{/if}
					</div>
				</div>
			</div>
		</div>
	</div>

	<div class="text-sm text-base-content/50">
		<p>* {$t("setup.directories.requiredFields")}</p>
	</div>
</div>