<script lang="ts">
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import * as App from "$lib/wailsjs/go/backend/App";
import {
	Badge,
	Button,
	Card,
	Heading,
	Input,
	Label,
	P,
	Toggle,
} from "flowbite-svelte";
import {
	CommandOutline,
	FileCodeSolid,
	FloppyDiskSolid,
} from "flowbite-svelte-icons";
import DurationInput from "../inputs/DurationInput.svelte";

export let config: ConfigData;

let saving = false;

// Reactive statement to initialize config defaults
$: {
	if (config && !config.post_upload_script) {
		config.post_upload_script = {
			enabled: false,
			command: "",
			timeout: "30s",
		};
	}
}

async function savePostUploadScriptSettings() {
	if (!config?.post_upload_script) {
		return;
	}

	try {
		saving = true;

		const currentConfig = await App.GetConfig();

		currentConfig.post_upload_script = {
			enabled: config.post_upload_script.enabled,
			command: config.post_upload_script.command || "",
			timeout: config.post_upload_script.timeout || "30s",
		};

		await App.SaveConfig(currentConfig);

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
<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <CommandOutline class="w-5 h-5 text-gray-600 dark:text-gray-400" />
      <Heading tag="h2" class="text-lg font-semibold text-gray-900 dark:text-white">
        {$t('settings.post_upload_script.title')}
      </Heading>
    </div>

    <div class="space-y-4">
      <div>
        <Label class="flex items-center gap-3 cursor-pointer">
          <Toggle bind:checked={config.post_upload_script.enabled} />
          <span>{$t('settings.post_upload_script.enable')}</span>
        </Label>
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          {$t('settings.post_upload_script.enable_description')}
        </P>
      </div>

      {#if config.post_upload_script.enabled}
        <div class="space-y-4 pl-4 border-l-2 border-blue-200 dark:border-blue-700">
          <div>
            <Label for="script-command" class="mb-2">{$t('settings.post_upload_script.command')}</Label>
            <Input
              id="script-command"
              bind:value={config.post_upload_script.command}
              placeholder={$t('settings.post_upload_script.command_placeholder')}
              class="font-mono"
            />
            <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {@html $t('settings.post_upload_script.command_description')}
            </P>
          </div>

          <div>
            <Label for="script-timeout" class="mb-2">{$t('settings.post_upload_script.timeout')}</Label>
            <DurationInput
              id="script-timeout"
              bind:value={config.post_upload_script.timeout}
            />
            <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {$t('settings.post_upload_script.timeout_description')}
            </P>
          </div>
        </div>
      {/if}

      <div class="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded">
        <div class="flex items-start gap-3">
          <FileCodeSolid class="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5" />
          <div class="space-y-2">
            <P class="text-sm font-medium text-blue-800 dark:text-blue-200">
              {$t('settings.post_upload_script.examples.title')}
            </P>
            <P class="text-sm text-blue-700 dark:text-blue-300">
              {$t('settings.post_upload_script.examples.description')}
            </P>
            <div class="space-y-2">
              <div class="bg-white dark:bg-gray-800 p-3 rounded text-xs font-mono space-y-2 text-gray-600 dark:text-gray-400">
                <div>
                  <Badge color="green" class="mb-1">{$t('settings.post_upload_script.examples.webhook')}</Badge>
                  <div class="break-all">{$t('settings.post_upload_script.examples.webhook_example')}</div>
                </div>
                <div>
                  <Badge color="blue" class="mb-1">{$t('settings.post_upload_script.examples.copy_file')}</Badge>
                  <div class="break-all">{$t('settings.post_upload_script.examples.copy_file_example')}</div>
                </div>
                <div>
                  <Badge color="purple" class="mb-1">{$t('settings.post_upload_script.examples.custom_script')}</Badge>
                  <div class="break-all">{$t('settings.post_upload_script.examples.custom_script_example')}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
      <Button
        color="green"
        onclick={savePostUploadScriptSettings}
        disabled={saving}
        class="cursor-pointer flex items-center gap-2"
      >
        <FloppyDiskSolid class="w-4 h-4" />
        {saving ? $t('settings.post_upload_script.saving') : $t('settings.post_upload_script.save_button')}
      </Button>
    </div>
  </div>
</Card>
{:else}
<Card class="max-w-full shadow-sm p-5">
  <div class="flex items-center justify-center py-8">
    <P class="text-gray-500 dark:text-gray-400">{$t('settings.post_upload_script.loading')}</P>
  </div>
</Card>
{/if} 