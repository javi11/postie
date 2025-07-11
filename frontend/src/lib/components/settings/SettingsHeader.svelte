<script lang="ts">
import { t } from "$lib/i18n";
import type { ConfigData } from "$lib/types";
import { Button, ButtonGroup, Heading, P } from "flowbite-svelte";
import {
	CheckCircleSolid,
	CogSolid,
	ExclamationCircleOutline,
	FloppyDiskSolid,
	FolderOpenSolid,
} from "flowbite-svelte-icons";
import { createEventDispatcher } from "svelte";

export const needsConfiguration = false;
export const criticalConfigError = false;

const dispatch = createEventDispatcher<{
	save: undefined;
	selectFile: undefined;
}>();

function handleSave() {
	dispatch("save");
}

function handleSelectFile() {
	dispatch("selectFile");
}
</script>

<header
  class="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700"
>
  <div
    class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4"
  >
    <div class="flex-1">
      <div class="flex items-center gap-3 mb-2">
        <CogSolid class="w-6 h-6 text-gray-600 dark:text-gray-400" />
        <Heading
          tag="h1"
          class="text-2xl font-bold text-gray-900 dark:text-white"
        >
          {$t('settings.header.title')}
        </Heading>
        {#if criticalConfigError}
          <div
            class="flex items-center gap-2 px-3 py-1 bg-red-100 dark:bg-red-900/30 rounded-full"
          >
            <ExclamationCircleOutline
              class="w-4 h-4 text-red-600 dark:text-red-400"
            />
            <span class="text-sm font-medium text-red-800 dark:text-red-200"
              >{$t('settings.header.status.configuration_error')}</span
            >
          </div>
        {:else if needsConfiguration}
          <div
            class="flex items-center gap-2 px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 rounded-full"
          >
            <ExclamationCircleOutline
              class="w-4 h-4 text-yellow-600 dark:text-yellow-400"
            />
            <span
              class="text-sm font-medium text-yellow-800 dark:text-yellow-200"
              >{$t('settings.header.status.configuration_required')}</span
            >
          </div>
        {:else}
          <div
            class="flex items-center gap-2 px-3 py-1 bg-green-100 dark:bg-green-900/30 rounded-full"
          >
            <CheckCircleSolid
              class="w-4 h-4 text-green-600 dark:text-green-400"
            />
            <span class="text-sm font-medium text-green-800 dark:text-green-200"
              >{$t('settings.header.status.configured')}</span
            >
          </div>
        {/if}
      </div>

      <P class="text-gray-600 dark:text-gray-400">
        {$t('settings.header.description')}
      </P>

      {#if criticalConfigError}
        <div
          class="mt-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg"
        >
          <P class="text-red-800 dark:text-red-200">
            <strong>{$t('settings.header.alerts.configuration_error')}</strong> {$t('settings.header.alerts.configuration_error_description')}
          </P>
        </div>
      {:else if needsConfiguration}
        <div
          class="mt-4 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg"
        >
          <P class="text-yellow-800 dark:text-yellow-200">
            <strong>{$t('settings.header.alerts.setup_required')}</strong> {$t('settings.header.alerts.setup_required_description')}
          </P>
        </div>
      {/if}
    </div>

    <div class="flex flex-col sm:flex-row gap-3">
      <Button
        color="primary"
        onclick={handleSave}
        class="cursor-pointer flex items-center gap-2"
      >
        <FloppyDiskSolid class="w-4 h-4" />
        {$t('settings.header.save_configuration')}
      </Button>
    </div>
  </div>
</header>
