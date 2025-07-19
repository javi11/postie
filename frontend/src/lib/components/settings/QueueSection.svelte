<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Quote, Save, Trash2 } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

// Reactive local state
let databaseType = $state(config.queue?.database_type || "sqlite");
let databasePath = $state(config.queue?.database_path || "./postie_queue.db");
let maxConcurrentUploads = $state(config.queue?.max_concurrent_uploads || 3);
let saving = $state(false);
let showClearModal = $state(false);
let clearing = $state(false);

// Derived state
let canSave = $derived(
	databaseType && 
	databasePath.trim() && 
	maxConcurrentUploads > 0 && 
	maxConcurrentUploads <= 20 && 
	!saving
);

// Database types available
const databaseTypes = [
	{ value: "sqlite", name: $t("settings.queue.database_types.sqlite") },
] as const;

// Sync local state back to config
$effect(() => {
	if (!config.queue) {
		config.queue = {
			database_type: "sqlite",
			database_path: "./postie_queue.db",
			max_concurrent_uploads: 3,
		};
	}
	config.queue.database_type = databaseType;
});

$effect(() => {
	config.queue.database_path = databasePath;
});

$effect(() => {
	config.queue.max_concurrent_uploads = maxConcurrentUploads;
});

async function saveQueueSettings() {
	if (!canSave) return;
	
	try {
		saving = true;

		// Validation
		if (!databasePath.trim()) {
			throw new Error("Database path is required");
		}
		
		if (maxConcurrentUploads < 1 || maxConcurrentUploads > 20) {
			throw new Error("Max concurrent uploads must be between 1 and 20");
		}

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Update only queue section
		currentConfig.queue = {
			...currentConfig.queue,
			database_type: databaseType,
			database_path: databasePath.trim(),
			max_concurrent_uploads: maxConcurrentUploads,
		};

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.queue.saved_success"),
			$t("settings.queue.saved_success_description")
		);
	} catch (error) {
		console.error("Failed to save queue settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}

async function clearQueue() {
	if (!clearing) {
		try {
			clearing = true;
			await apiClient.clearQueue();
			toastStore.success(
				$t("common.messages.queue_cleared"),
				$t("common.messages.queue_cleared_description")
			);
		} catch (error) {
			console.error("Failed to clear queue:", error);
			toastStore.error(
				$t("common.messages.failed_to_clear_queue"),
				String(error)
			);
		} finally {
			clearing = false;
			showClearModal = false;
		}
	}
}
</script>

<div class="card bg-base-100 shadow-xl">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <Quote class="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
      <h2 class="card-title text-lg">
        {$t('settings.queue.title')}
      </h2>
    </div>

    <div class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="form-control">
          <label class="label" for="database-type">
            <span class="label-text">{$t('settings.queue.database_type')}</span>
          </label>
          <select
            id="database-type"
            class="select select-bordered"
            bind:value={databaseType}
          >
            {#each databaseTypes as type}
              <option value={type.value}>{type.name}</option>
            {/each}
          </select>
          <div class="label">
            <span class="label-text-alt">
              {$t('settings.queue.database_type_description')}
            </span>
          </div>
        </div>

        <div class="form-control">
          <label class="label" for="database-path">
            <span class="label-text">{$t('settings.queue.database_path')}</span>
          </label>
          <input
            id="database-path"
            class="input input-bordered"
            bind:value={databasePath}
            placeholder={databaseType === "sqlite"
              ? $t('settings.queue.database_path_placeholder_sqlite')
              : $t('settings.queue.database_path_placeholder_network')}
          />
          <div class="label">
            <span class="label-text-alt">
              {databaseType === "sqlite"
                ? $t('settings.queue.database_path_description_sqlite')
                : $t('settings.queue.database_path_description_network')}
            </span>
          </div>
        </div>

        <div class="form-control">
          <label class="label" for="max-concurrent">
            <span class="label-text">{$t('settings.queue.max_concurrent_uploads')}</span>
          </label>
          <input
            id="max-concurrent"
            type="number"
            class="input input-bordered"
            bind:value={maxConcurrentUploads}
            min="1"
            max="20"
          />
          <div class="label">
            <span class="label-text-alt">
              {$t('settings.queue.max_concurrent_uploads_description')}
            </span>
          </div>
        </div>
      </div>
    </div>

    <div class="alert alert-info">
      <span class="text-sm">
        {@html $t('settings.queue.info')}
      </span>
    </div>

    <!-- Action Buttons -->
    <div class="pt-4 border-t border-base-300 flex justify-between items-center">
      <button
        type="button"
        class="btn btn-error btn-outline"
        onclick={() => showClearModal = true}
        disabled={clearing}
      >
        <Trash2 class="w-4 h-4" />
        {$t('dashboard.header.clear_completed')}
      </button>
      
      <button
        type="button"
        class="btn btn-success"
        onclick={saveQueueSettings}
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.queue.save_button')}
      </button>
    </div>
  </div>
</div>

<!-- Clear Queue Confirmation Modal -->
{#if showClearModal}
  <div class="modal modal-open">
    <div class="modal-box">
      <h3 class="font-bold text-lg mb-4">
        {$t('dashboard.header.clear_completed')}
      </h3>
      <p class="py-4">
        Are you sure you want to clear all completed items from the queue? This action cannot be undone.
      </p>
      <div class="modal-action">
        <button
          type="button"
          class="btn btn-ghost"
          onclick={() => showClearModal = false}
          disabled={clearing}
        >
          {$t('common.common.cancel')}
        </button>
        <button
          type="button"
          class="btn btn-error"
          onclick={clearQueue}
          disabled={clearing}
        >
          <Trash2 class="w-4 h-4" />
          {clearing ? $t('common.common.clearing') : $t('common.common.clear')}
        </button>
      </div>
    </div>
  </div>
{/if}
