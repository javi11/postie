<script lang="ts">
import { page } from "$app/stores";
import apiClient from "$lib/api/client";
import { loadTranslations, locale, t } from "$lib/i18n";
import { availableLocales, setStoredLocale } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import { Cog, FolderOpen, Save } from "lucide-svelte";
import { onMount } from "svelte";

export let config: ConfigData;

let saving = false;

// Initialize config defaults if they don't exist
if (!config.output_dir) {
	config.output_dir = "./output";
}

if (config.maintain_original_extension === undefined) {
	config.maintain_original_extension = true;
}

// Prepare language options for select
const languageOptions = availableLocales.map((lang) => ({
	value: lang.code,
	name: `${lang.flag} ${lang.name}`,
}));

async function changeLanguage(event: Event) {
	const target = event.target as HTMLSelectElement;
	const newLocale = target.value;

	// Store the selected locale
	setStoredLocale(newLocale);

	// The sveltekit-i18n library should automatically handle loading the new translations
	// when the locale store is updated.
	locale.set(newLocale);
}

async function selectOutputDirectory() {
	try {
		// Check if we're in Wails environment
		if (apiClient.environment === "wails") {
			const { App } = await import("$lib/wailsjs/go/backend/App");
			const dir = await App.SelectOutputDirectory();
			if (dir) {
				config.output_dir = dir;
			}
		}
		// In web mode, users can just type the path directly in the input field
	} catch (error) {
		console.error("Failed to select output directory:", error);
	}
}

async function saveGeneralSettings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Only update the general settings fields
		currentConfig.output_dir = config.output_dir || "./output";
		currentConfig.maintain_original_extension =
			config.maintain_original_extension ?? true;

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.general.saved_success"),
			$t("settings.general.saved_success_description"),
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
				<Cog class="w-5 h-5 text-gray-600 dark:text-gray-400" />
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
							value={config.output_dir}
							onchange={(e) => {
								config.output_dir = (e.target as HTMLInputElement).value;
							}}
							placeholder="./output"
						/>
						{#if apiClient.environment === 'wails'}
							<button
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
							bind:checked={config.maintain_original_extension}
						/>
						<span class="text-sm">
							{config.maintain_original_extension ? $t('settings.general.maintain_extension_enabled') : $t('settings.general.maintain_extension_disabled')}
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
					class="btn btn-success"
					onclick={saveGeneralSettings}
					disabled={saving}
				>
					<Save class="w-4 h-4" />
					{saving ? $t('settings.general.saving') : $t('settings.general.save_button')}
				</button>
			</div>
		</div>
	</div>

	<div class="card bg-base-100 shadow-xl">
		<div class="card-body space-y-6">
			<div class="flex items-center gap-3">
				<Cog class="w-5 h-5 text-gray-600 dark:text-gray-400" />
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
						<select
							id="language-select"
							class="select select-bordered w-full"
							value={$locale}
							onchange={changeLanguage}
						>
							{#each languageOptions as option}
								<option value={option.value}>{option.name}</option>
							{/each}
						</select>
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
						<div class="flex items-center gap-2 mt-2">
							<span class="text-sm">
								{$t('settings.general.theme_toggle')}
							</span>
							<input type="checkbox" class="toggle" checked />
						</div>
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
