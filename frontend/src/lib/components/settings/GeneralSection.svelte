<script lang="ts">
import { page } from "$app/stores";
import { loadTranslations, locale, t } from "$lib/i18n";
import { availableLocales, setStoredLocale } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import * as App from "$lib/wailsjs/go/backend/App";
import {
	Button,
	Card,
	DarkMode,
	Heading,
	Input,
	Label,
	P,
	Select,
} from "flowbite-svelte";
import {
	CogSolid,
	FloppyDiskSolid,
	FolderOpenSolid,
} from "flowbite-svelte-icons";
import { onMount } from "svelte";

export let config: ConfigData;

let outputDirectory = "";
let saving = false;
let selectedLanguage = $locale;

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

	selectedLanguage = newLocale;
}

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

async function saveGeneralSettings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await App.GetConfig();

		// Only update the general settings fields
		currentConfig.output_dir = config.output_dir || "./output";
		currentConfig.maintain_original_extension =
			config.maintain_original_extension ?? true;

		await App.SaveConfig(currentConfig);

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

// Update display when config changes
$: if (config.output_dir) {
	outputDirectory = config.output_dir;
}

// Keep selectedLanguage in sync with locale store
$: selectedLanguage = $locale;
</script>

<div class="space-y-6">
	<Card class="max-w-full shadow-sm p-5">
		<div class="space-y-6">
			<div class="flex items-center gap-3">
				<CogSolid class="w-5 h-5 text-gray-600 dark:text-gray-400" />
				<Heading tag="h2" class="text-lg font-semibold text-gray-900 dark:text-white">
					{$t('settings.general.title')}
				</Heading>
			</div>

			<div class="space-y-4">
				<div>
					<Label for="output-dir" class="mb-2">{$t('settings.general.output_directory')}</Label>
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
							{$t('settings.general.browse')}
						</Button>
					</div>
					<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
						{$t('settings.general.output_directory_description')}
					</P>
				</div>

				<div
					class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
				>
					<P class="text-sm text-blue-800 dark:text-blue-200">
						<strong>{$t('settings.general.info_title')}</strong>
						{$t('settings.general.info_description')}
					</P>
				</div>

				<div>
					<Label for="maintain-extension" class="mb-2">
						{$t('settings.general.maintain_original_extension')}
					</Label>
					<div class="flex items-center gap-2">
						<input
							id="maintain-extension"
							type="checkbox"
							bind:checked={config.maintain_original_extension}
							class="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
						/>
						<span class="text-sm text-gray-700 dark:text-gray-300">
							{config.maintain_original_extension ? $t('settings.general.maintain_extension_enabled') : $t('settings.general.maintain_extension_disabled')}
						</span>
					</div>
					<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
						{$t('settings.general.maintain_original_extension_description')}
					</P>
				</div>

				{#if outputDirectory && outputDirectory !== config.output_dir}
					<div
						class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded"
					>
						<P class="text-sm text-amber-800 dark:text-amber-200">
							<strong>{$t('settings.general.current_active_directory')}</strong>
							{outputDirectory}<br />
							<strong>{$t('settings.general.new_directory_after_save')}</strong>
							{config.output_dir}
						</P>
					</div>
				{/if}
			</div>

			<!-- Save Button -->
			<div class="pt-4 border-t border-gray-200 dark:border-gray-700">
				<Button
					color="green"
					onclick={saveGeneralSettings}
					disabled={saving}
					class="cursor-pointer flex items-center gap-2"
				>
					<FloppyDiskSolid class="w-4 h-4" />
					{saving ? $t('settings.general.saving') : $t('settings.general.save_button')}
				</Button>
			</div>
		</div>
	</Card>

	<Card class="max-w-full shadow-sm p-5">
		<div class="space-y-6">
			<div class="flex items-center gap-3">
				<CogSolid class="w-5 h-5 text-gray-600 dark:text-gray-400" />
				<Heading tag="h2" class="text-lg font-semibold text-gray-900 dark:text-white">
					{$t('settings.general.ui_preferences')}
				</Heading>
			</div>
			<div class="space-y-4">
				<P class="text-sm text-gray-600 dark:text-gray-400 -mt-4">
					{$t('settings.general.ui_preferences_description')}
				</P>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<!-- Language Selection -->
					<div>
						<Label for="language-select" class="mb-2">
							{$t('settings.general.language')}
						</Label>
						<Select
							id="language-select"
							bind:value={selectedLanguage}
							onchange={changeLanguage}
							items={languageOptions}
							class="w-full"
						/>
						<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
							{$t('settings.general.language_description')}
						</P>
					</div>

					<!-- Theme Selection -->
					<div>
						<Label class="mb-2">
							{$t('settings.general.theme')}
						</Label>
						<div class="flex items-center gap-2 mt-2">
							<span class="text-sm text-gray-700 dark:text-gray-300">
								{$t('settings.general.theme_toggle')}
							</span>
							<DarkMode
								class="cursor-pointer text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 rounded-lg text-sm p-2.5 transition-all"
							/>
						</div>
						<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
							{$t('settings.general.theme_description')}
						</P>
					</div>
				</div>
			</div>
		</div>
	</Card>
</div>
