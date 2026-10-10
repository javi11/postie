<script lang="ts">
import { onMount } from "svelte";
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

let isDesktop = $state(false);
let configPath = $state("");
let relocatingConfig = $state(false);

onMount(async () => {
 try {
  await apiClient.initialize();
  isDesktop = apiClient.environment === "wails";
  if (isDesktop) {
   const { GetConfigPath } = await import("$lib/wailsjs/go/backend/App");
   configPath = await GetConfigPath();
  }
 } catch (error) {
  toastStore.error($t("settings.general.config_location_error"), String(error));
 }
});

async function selectConfigDirectory() {
 relocatingConfig = true;
 try {
  const { SelectConfigDirectory } = await import("$lib/wailsjs/go/backend/App");
  const path = await SelectConfigDirectory();
  if (path && path !== configPath) {
   configPath = path;
   toastStore.success($t("settings.general.config_location_success"));
  }
 } catch (error) {
  toastStore.error($t("settings.general.config_location_error"), String(error));
 } finally {
  relocatingConfig = false;
 }
}

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

                {#if isDesktop}
                    <div class="form-control">
                        <label class="label" for="config-path">
                            <span class="label-text">{$t('settings.general.config_file')}</span>
                        </label>
                        <div class="flex items-center gap-2">
                            <input id="config-path" class="input input-bordered flex-1 min-w-0" value={configPath} readonly />
                            <button type="button" class="btn btn-outline btn-sm" onclick={selectConfigDirectory} disabled={relocatingConfig || !configPath}>
                                {#if relocatingConfig}<span class="loading loading-spinner loading-xs"></span>{:else}<FolderOpen class="w-4 h-4" />{/if}
                                {$t('settings.general.change_config_location')}
                            </button>
                        </div>
                        <p class="text-sm text-base-content/70 mt-2">{$t('settings.general.config_location_description')}</p>
                    </div>
                {/if}

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
