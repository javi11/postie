<script lang="ts">
import { page } from "$app/stores";
import apiClient from "$lib/api/client";
import LanguageSwitcher from "$lib/components/LanguageSwitcher.svelte";
import ThemeSwitcher from "$lib/components/ThemeSwitcher.svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Cog, FolderOpen, Save } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

// Reactive local state
let outputDir = $state(config.output_dir || "./output");
let maintainOriginalExtension = $state(config.maintain_original_extension ?? true);
let saving = $state(false);

// Derived state
let canSave = $derived(outputDir.trim() && !saving);

// Sync local state back to config
$effect(() => {
	config.output_dir = outputDir;
});

$effect(() => {
	config.maintain_original_extension = maintainOriginalExtension;
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

async function saveGeneralSettings() {
	if (!canSave) return;
	
	try {
		saving = true;

		// Validation
		if (!outputDir.trim()) {
			throw new Error("Output directory is required");
		}

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Update only general settings fields
		currentConfig.output_dir = outputDir.trim();
		currentConfig.maintain_original_extension = maintainOriginalExtension;

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.general.saved_success"),
			$t("settings.general.saved_success_description")
		);
	} catch (error) {
		console.error("Failed to save general settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
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

				<div class="form-control">
					<label class="label" for="maintain-extension">
						<span class="label-text">{$t('settings.general.maintain_original_extension')}</span>
					</label>
					<div class="flex items-center gap-2">
						<input
							id="maintain-extension"
							type="checkbox"
							class="checkbox"
							bind:checked={maintainOriginalExtension}
						/>
						<span class="text-sm">
							{maintainOriginalExtension ? $t('settings.general.maintain_extension_enabled') : $t('settings.general.maintain_extension_disabled')}
						</span>
					</div>
					<div class="label">
						<span class="label-text-alt">
							{$t('settings.general.maintain_original_extension_description')}
						</span>
					</div>
				</div>
			</div>

			<!-- Save Button -->
			<div class="card-actions pt-4 border-t border-base-300">
				<button
					type="button"
					class="btn btn-success"
					onclick={saveGeneralSettings}
					disabled={!canSave}
				>
					<Save class="w-4 h-4" />
					{saving ? $t('common.common.saving') : $t('settings.general.save_button')}
				</button>
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
