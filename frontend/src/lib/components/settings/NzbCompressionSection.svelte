<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Archive, Save } from "lucide-svelte";

interface Props {
	config: configType.ConfigData;
}

const { config }: Props = $props();

// Ensure nzb_compression exists with defaults
if (!config.nzb_compression) {
	config.nzb_compression = {
		enabled: false,
		type: "none",
		level: 0,
	};
}

// Reactive local state
let enabled = $state(config.nzb_compression.enabled ?? false);
let compressionType = $state(config.nzb_compression.type || "none");
let compressionLevel = $state(config.nzb_compression.level || 0);
let saving = $state(false);

// Dynamic compression types based on translations
let compressionTypes = $derived([
	{
		value: "zstd",
		name: $t("settings.nzb_compression.compression_types.zstd"),
	},
	{
		value: "brotli",
		name: $t("settings.nzb_compression.compression_types.brotli"),
	},
	{
		value: "zip",
		name: $t("settings.nzb_compression.compression_types.zip"),
	},
]);

// Get compression level limits based on type
let compressionLimits = $derived(getCompressionLimits(compressionType));
let defaultLevel = $derived(getDefaultLevel(compressionType));

// Derived state
let canSave = $derived(
	compressionType && 
	(compressionType === "none" || 
	 (compressionLevel >= compressionLimits.min && compressionLevel <= compressionLimits.max)) && 
	!saving
);

// Sync local state back to config
$effect(() => {
	config.nzb_compression.enabled = enabled;
});

$effect(() => {
	config.nzb_compression.type = compressionType;
});

$effect(() => {
	config.nzb_compression.level = compressionLevel;
});

// Auto-adjust level when compression type changes
$effect(() => {
	if (compressionType === "none") {
		compressionLevel = 0;
	} else {
		const limits = getCompressionLimits(compressionType);
		if (compressionLevel < limits.min || compressionLevel > limits.max) {
			compressionLevel = getDefaultLevel(compressionType);
		}
	}
});

function getCompressionLimits(type: string) {
	switch (type) {
		case "zstd":
			return { min: 1, max: 22 };
		case "brotli":
			return { min: 0, max: 11 };
		case "zip":
			return { min: 0, max: 9 };
		default:
			return { min: 0, max: 0 };
	}
}

function getDefaultLevel(type: string) {
	switch (type) {
		case "zstd":
			return 3;
		case "brotli":
			return 4;
		case "zip":
			return 6;
		default:
			return 0;
	}
}

async function saveNzbCompressionSettings() {
	if (!canSave) return;
	
	try {
		saving = true;

		// Validation
		if (compressionType !== "none") {
			const limits = getCompressionLimits(compressionType);
			if (compressionLevel < limits.min || compressionLevel > limits.max) {
				throw new Error(`Compression level must be between ${limits.min} and ${limits.max}`);
			}
		}

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Update only nzb compression section
		currentConfig.nzb_compression = {
			...currentConfig.nzb_compression,
			enabled: enabled,
			type: compressionType,
			level: compressionLevel,
		};

		await apiClient.saveConfig(currentConfig);
	} catch (error) {
		console.error("Failed to save NZB compression settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}

// Auto-set default level when compression type changes
$effect(() => {
  if (
    config.nzb_compression.type !== "none" &&
    config.nzb_compression.level === 0
  ) {
    config.nzb_compression.level = defaultLevel;
  }
});

// Reset level when compression is disabled
$effect(() => {
  if (config.nzb_compression.type === "none") {
    config.nzb_compression.level = 0;
  }
});
</script>

<div class="card bg-base-100 shadow-xl">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <Archive class="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
      <h2 class="card-title text-lg">
        {$t('settings.nzb_compression.title')}
      </h2>
    </div>

    <div class="form-control">
      <label class="label cursor-pointer justify-start gap-3">
        <input type="checkbox" class="checkbox" bind:checked={enabled} />
        <span class="label-text">{$t('settings.nzb_compression.enable')}</span>
      </label>
      <div class="label">
        <span class="label-text-alt ml-8">
          {$t('settings.nzb_compression.enable_description')}
        </span>
      </div>
    </div>

    {#if enabled}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="form-control">
          <label class="label" for="compression-type">
            <span class="label-text">{$t('settings.nzb_compression.compression_type')}</span>
          </label>
          <select
            id="compression-type"
            class="select select-bordered"
            bind:value={compressionType}
          >
            {#each compressionTypes as type}
              <option value={type.value}>{type.name}</option>
            {/each}
          </select>
          <div class="label">
            <span class="label-text-alt">
              {$t('settings.nzb_compression.compression_type_description')}
            </span>
          </div>
        </div>

        {#if compressionType !== "none"}
          <div class="form-control">
            <label class="label" for="compression-level">
              <span class="label-text">{$t('settings.nzb_compression.compression_level')}</span>
            </label>
            <input
              id="compression-level"
              type="number"
              class="input input-bordered"
              bind:value={compressionLevel}
              min={compressionLimits.min}
              max={compressionLimits.max}
            />
            <div class="label">
              <span class="label-text-alt">
                {$t('settings.nzb_compression.compression_level_description', { values:{ min: compressionLimits.min, max: compressionLimits.max } })}
              </span>
            </div>
          </div>
        {/if}
      </div>

      {#if compressionType === "zstd"}
        <div class="alert alert-info">
          <span class="text-sm">
            <strong>{$t('settings.nzb_compression.info.zstd_title')}</strong> {$t('settings.nzb_compression.info.zstd_description')}
          </span>
        </div>
      {:else if compressionType === "brotli"}
        <div class="alert alert-success">
          <span class="text-sm">
            <strong>{$t('settings.nzb_compression.info.brotli_title')}</strong> {$t('settings.nzb_compression.info.brotli_description')}
          </span>
        </div>
      {:else if compressionType === "zip"}
        <div class="alert alert-warning">
          <span class="text-sm">
            <strong>{$t('settings.nzb_compression.info.zip_title')}</strong> {$t('settings.nzb_compression.info.zip_description')}
          </span>
        </div>
      {/if}
    {:else}
      <div class="alert">
        <span class="text-sm">
          {$t('settings.nzb_compression.info.disabled_description')}
        </span>
      </div>
    {/if}

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        type="button"
        class="btn btn-success"
        onclick={saveNzbCompressionSettings}
        disabled={!canSave}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.nzb_compression.save_button')}
      </button>
    </div>
  </div>
</div>
