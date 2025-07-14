<script lang="ts">
import { t } from "$lib/i18n";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Archive } from "lucide-svelte";

export let config: configType.ConfigData;

// Ensure nzb_compression exists with defaults
if (!config.nzb_compression) {
	config.nzb_compression = {
		enabled: false,
		type: "none",
		level: 0,
	};
}

// Dynamic compression types based on translations
$: compressionTypes = [
	{
		value: "none",
		name: $t("settings.nzb_compression.compression_types.none"),
	},
	{
		value: "zstd",
		name: $t("settings.nzb_compression.compression_types.zstd"),
	},
	{
		value: "brotli",
		name: $t("settings.nzb_compression.compression_types.brotli"),
	},
];

// Get compression level limits based on type
$: compressionLimits = getCompressionLimits(config.nzb_compression.type);
$: defaultLevel = getDefaultLevel(config.nzb_compression.type);

function getCompressionLimits(type: string) {
	switch (type) {
		case "zstd":
			return { min: 1, max: 22 };
		case "brotli":
			return { min: 0, max: 11 };
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
		default:
			return 0;
	}
}

// Auto-set default level when compression type changes
$: if (
	config.nzb_compression.type !== "none" &&
	config.nzb_compression.level === 0
) {
	config.nzb_compression.level = defaultLevel;
}

// Reset level when compression is disabled
$: if (config.nzb_compression.type === "none") {
	config.nzb_compression.level = 0;
}
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
        <input type="checkbox" class="checkbox" bind:checked={config.nzb_compression.enabled} />
        <span class="label-text">{$t('settings.nzb_compression.enable')}</span>
      </label>
      <div class="label">
        <span class="label-text-alt ml-8">
          {$t('settings.nzb_compression.enable_description')}
        </span>
      </div>
    </div>

    {#if config.nzb_compression.enabled}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="form-control">
          <label class="label" for="compression-type">
            <span class="label-text">{$t('settings.nzb_compression.compression_type')}</span>
          </label>
          <select
            id="compression-type"
            class="select select-bordered"
            bind:value={config.nzb_compression.type}
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

        {#if config.nzb_compression.type !== "none"}
          <div class="form-control">
            <label class="label" for="compression-level">
              <span class="label-text">{$t('settings.nzb_compression.compression_level')}</span>
            </label>
            <input
              id="compression-level"
              type="number"
              class="input input-bordered"
              bind:value={config.nzb_compression.level}
              min={compressionLimits.min}
              max={compressionLimits.max}
            />
            <div class="label">
              <span class="label-text-alt">
                {$t('settings.nzb_compression.compression_level_description', { default:{ min: compressionLimits.min, max: compressionLimits.max } })}
              </span>
            </div>
          </div>
        {/if}
      </div>

      {#if config.nzb_compression.type === "zstd"}
        <div class="alert alert-info">
          <span class="text-sm">
            <strong>{$t('settings.nzb_compression.info.zstd_title')}</strong> {$t('settings.nzb_compression.info.zstd_description')}
          </span>
        </div>
      {:else if config.nzb_compression.type === "brotli"}
        <div class="alert alert-success">
          <span class="text-sm">
            <strong>{$t('settings.nzb_compression.info.brotli_title')}</strong> {$t('settings.nzb_compression.info.brotli_description')}
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
  </div>
</div>
