<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { FileText, Save } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

// Reactive local state
let maintainOriginalExtension = $state(config.maintain_original_extension ?? true);
let saving = $state(false);

// Derived state
let canSave = $derived(!saving);

// Sync local state back to config
$effect(() => {
	config.maintain_original_extension = maintainOriginalExtension;
});

async function saveFileNamingSettings() {
	if (!canSave) return;
	
	try {
		saving = true;

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Update only file naming settings
		currentConfig.maintain_original_extension = maintainOriginalExtension;

		await apiClient.saveConfig(currentConfig);
	} catch (error) {
		console.error("Failed to save file naming settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}
</script>

<div class="card bg-base-100 shadow-xl">
	<div class="card-body space-y-6">
		<div class="flex items-center gap-3">
			<FileText class="w-5 h-5 text-purple-600 dark:text-purple-400" />
			<h2 class="card-title text-lg">
				{$t('settings.file_naming.title')}
			</h2>
		</div>

		<div class="form-control">
			<label class="label cursor-pointer justify-start gap-3">
				<input type="checkbox" class="checkbox" bind:checked={maintainOriginalExtension} />
				<span class="label-text">{$t('settings.general.maintain_original_extension')}</span>
			</label>
			<div class="label">
				<span class="label-text-alt ml-8 whitespace-normal break-words text-wrap">
					{$t('settings.general.maintain_original_extension_description')}
				</span>
			</div>
		</div>

		<div class="alert alert-info">
			<span class="text-sm">
				<strong>{$t('settings.file_naming.info_title')}</strong>
				{maintainOriginalExtension ? $t('settings.general.maintain_extension_enabled') : $t('settings.general.maintain_extension_disabled')}
			</span>
		</div>

		<!-- Save Button -->
		<div class="pt-4 border-t border-base-300">
			<button
				type="button"
				class="btn btn-success"
				onclick={saveFileNamingSettings}
				disabled={!canSave}
			>
				<Save class="w-4 h-4" />
				{saving ? $t('common.common.saving') : $t('settings.file_naming.save_button')}
			</button>
		</div>
	</div>
</div>