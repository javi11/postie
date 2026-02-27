<script lang="ts">
import { page } from "$app/stores";
import apiClient from "$lib/api/client";
import LanguageSwitcher from "$lib/components/LanguageSwitcher.svelte";
import ThemeSwitcher from "$lib/components/ThemeSwitcher.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Cog, FolderOpen } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

// Reactive local state
let outputDir = $state(config.output_dir || "./output");

// Sync local state back to config
$effect(() => {
	config.output_dir = outputDir;
});

async function selectOutputDirectory() {
	try {
		await apiClient.initialize();
		
		if (apiClient.environment !== "wails") {
			toastStore.warning($t("common.messages.wails_only_feature"));
			return;
		}
		
		const { SelectOutputDirectory } = await import("$lib/wailsjs/go/backend/App");
		const dir = await SelectOutputDirectory();
		
		if (dir) {
			outputDir = dir;
		}
	} catch (error) {
		console.error("Failed to select output directory:", error);
		toastStore.error($t("common.messages.error_selecting_directory"), String(error));
	}
}

</script>

<div class="space-y-6">
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body space-y-6">
			<div class="flex items-center gap-3">
				<Cog class="w-5 h-5 text-base-content/60" />
				<h2 class="card-title text-lg">
					{$t('settings.general.title')}
				</h2>
			</div>

			<div class="space-y-4">
				<div class="form-control">
					<label class="label" for="output-dir">
						<span class="label-text">{$t('settings.general.output_directory')}</span>
					</label>
					<div class="flex items-center gap-2">
						<input
							id="output-dir"
							class="input input-bordered flex-1"
							bind:value={outputDir}
							placeholder="./output"
						/>
						{#if apiClient.environment === 'wails'}
							<button
								type="button"
								class="btn btn-outline btn-sm"
								onclick={selectOutputDirectory}
							>
								<FolderOpen class="w-4 h-4" />
								{$t('settings.general.browse')}
							</button>
						{/if}
					</div>
				</div>

				<div class="alert alert-info">
					<span class="text-sm">
						<strong>{$t('settings.general.info_title')}</strong>
						{$t('settings.general.info_description')}
					</span>
				</div>
			</div>

			</div>
	</div>

	<div class="card bg-base-100 shadow-xl">
		<div class="card-body space-y-6">
			<div class="flex items-center gap-3">
				<Cog class="w-5 h-5 text-base-content/60" />
				<h2 class="card-title text-lg">
					{$t('settings.general.ui_preferences')}
				</h2>
			</div>
			<div class="space-y-4">
				<p class="text-sm text-base-content/70 -mt-4">
					{$t('settings.general.ui_preferences_description')}
				</p>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<!-- Language Selection -->
					<div class="form-control">
						<label class="label" for="language-select">
							<span class="label-text">{$t('settings.general.language')}</span>
						</label>
						<LanguageSwitcher />
						<div class="label">
							<span class="label-text-alt">
								{$t('settings.general.language_description')}
							</span>
						</div>
					</div>

					<!-- Theme Selection -->
					<div class="form-control">
						<div class="label">
							<span class="label-text">{$t('settings.general.theme')}</span>
						</div>
						<ThemeSwitcher />
						<div class="label">
							<span class="label-text-alt">
								{$t('settings.general.theme_description')}
							</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
