<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { FileCode, Save, Terminal } from "lucide-svelte";
import DurationInput from "../inputs/DurationInput.svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

// Initialize config defaults
if (config && !config.post_upload_script) {
	config.post_upload_script = {
		enabled: false,
		command: "",
		timeout: "30s",
	};
}

// Reactive local state
let enabled = $state(config.post_upload_script?.enabled ?? false);
let command = $state(config.post_upload_script?.command || "");
let timeout = $state(config.post_upload_script?.timeout || "30s");
let saving = $state(false);

// Derived state
let canSave = $derived(
	(!enabled || (enabled && command.trim())) && 
	timeout.trim() && 
	!saving
);

// Sync local state back to config
$effect(() => {
	config.post_upload_script.enabled = enabled;
});

$effect(() => {
	config.post_upload_script.command = command;
});

$effect(() => {
	config.post_upload_script.timeout = timeout;
});

async function savePostUploadScriptSettings() {
	if (!config?.post_upload_script) {
		return;
	}

	try {
		saving = true;

		const currentConfig = await apiClient.getConfig();

		// Validation
		if (enabled && !command.trim()) {
			throw new Error("Command is required when script is enabled");
		}
		
		if (!timeout.trim()) {
			throw new Error("Timeout is required");
		}

		currentConfig.post_upload_script = {
			enabled: enabled,
			command: command.trim(),
			timeout: timeout.trim(),
		};

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.post_upload_script.saved_success"),
			$t("settings.post_upload_script.saved_success_description"),
		);
	} catch (error) {
		console.error("Failed to save post upload script settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}
</script>

{#if config && config.post_upload_script}
<div class="card bg-base-100 shadow-xl">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <Terminal class="w-5 h-5 text-base-content/60" />
      <h2 class="card-title text-lg">
        {$t('settings.post_upload_script.title')}
      </h2>
    </div>

    <div class="space-y-4">
      <div class="form-control">
        <label class="label cursor-pointer justify-start gap-3">
          <input type="checkbox" class="toggle" bind:checked={enabled} />
          <span class="label-text">{$t('settings.post_upload_script.enable')}</span>
        </label>
        <div class="label">
          <span class="label-text-alt ml-8">
            {$t('settings.post_upload_script.enable_description')}
          </span>
        </div>
      </div>

      {#if enabled}
        <div class="space-y-4 pl-4 border-l-2 border-blue-200 dark:border-blue-700">
          <div class="form-control">
            <label class="label" for="script-command">
              <span class="label-text">{$t('settings.post_upload_script.command')}</span>
            </label>
            <input
              id="script-command"
              class="input input-bordered font-mono"
              bind:value={command}
              placeholder={$t('settings.post_upload_script.command_placeholder')}
            />
            <div class="label">
              <span class="label-text-alt">
                {@html $t('settings.post_upload_script.command_description')}
              </span>
            </div>
          </div>

          <div class="form-control">
            <label class="label" for="script-timeout">
              <span class="label-text">{$t('settings.post_upload_script.timeout')}</span>
            </label>
            <DurationInput
              id="script-timeout"
              bind:value={timeout}
            />
            <div class="label">
              <span class="label-text-alt">
                {$t('settings.post_upload_script.timeout_description')}
              </span>
            </div>
          </div>
        </div>
      {/if}

      <div class="alert alert-info">
        <div class="flex items-start gap-3">
          <FileCode class="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" />
          <div class="space-y-2">
            <p class="text-sm font-medium">
              {$t('settings.post_upload_script.examples.title')}
            </p>
            <p class="text-sm">
              {$t('settings.post_upload_script.examples.description')}
            </p>
            <div class="space-y-2">
              <div class="bg-base-200 p-3 rounded text-xs font-mono space-y-2">
                <div>
                  <div class="badge badge-success mb-1">{$t('settings.post_upload_script.examples.webhook')}</div>
                  <div class="break-all text-base-content/70">{$t('settings.post_upload_script.examples.webhook_example')}</div>
                </div>
                <div>
                  <div class="badge badge-info mb-1">{$t('settings.post_upload_script.examples.copy_file')}</div>
                  <div class="break-all text-base-content/70">{$t('settings.post_upload_script.examples.copy_file_example')}</div>
                </div>
                <div>
                  <div class="badge badge-secondary mb-1">{$t('settings.post_upload_script.examples.custom_script')}</div>
                  <div class="break-all text-base-content/70">{$t('settings.post_upload_script.examples.custom_script_example')}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="card-actions pt-4 border-t border-base-300">
      <button
        class="btn btn-success"
        onclick={savePostUploadScriptSettings}
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.post_upload_script.save_button')}
      </button>
    </div>
  </div>
</div>
{:else}
<div class="card bg-base-100 shadow-xl">
  <div class="card-body">
    <div class="flex items-center justify-center py-8">
      <p class="text-base-content/50">{$t('settings.post_upload_script.loading')}</p>
    </div>
  </div>
</div>
{/if} 