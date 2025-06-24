<script lang="ts">
import { t } from "$lib/i18n";
import type { ConfigData } from "$lib/types";
import {
	Card,
	Checkbox,
	Heading,
	Input,
	Label,
	P,
	Select,
} from "flowbite-svelte";
import { ArchiveSolid } from "flowbite-svelte-icons";

export let config: ConfigData;

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

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <ArchiveSolid class="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        {$t('settings.nzb_compression.title')}
      </Heading>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Checkbox bind:checked={config.nzb_compression.enabled} />
        <Label class="text-sm font-medium">{$t('settings.nzb_compression.enable')}</Label>
      </div>
      <P class="text-sm text-gray-600 dark:text-gray-400 ml-6">
        {$t('settings.nzb_compression.enable_description')}
      </P>
    </div>

    {#if config.nzb_compression.enabled}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <Label for="compression-type" class="mb-2">{$t('settings.nzb_compression.compression_type')}</Label>
          <Select
            id="compression-type"
            items={compressionTypes}
            bind:value={config.nzb_compression.type}
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {$t('settings.nzb_compression.compression_type_description')}
          </P>
        </div>

        {#if config.nzb_compression.type !== "none"}
          <div>
            <Label for="compression-level" class="mb-2">{$t('settings.nzb_compression.compression_level')}</Label
            >
            <Input
              id="compression-level"
              type="number"
              bind:value={config.nzb_compression.level}
              min={compressionLimits.min}
              max={compressionLimits.max}
            />
            <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {$t('settings.nzb_compression.compression_level_description', { min: compressionLimits.min, max: compressionLimits.max })}
            </P>
          </div>
        {/if}
      </div>

      {#if config.nzb_compression.type === "zstd"}
        <div
          class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
        >
          <P class="text-sm text-blue-800 dark:text-blue-200">
            <strong>{$t('settings.nzb_compression.info.zstd_title')}</strong> {$t('settings.nzb_compression.info.zstd_description')}
          </P>
        </div>
      {:else if config.nzb_compression.type === "brotli"}
        <div
          class="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded"
        >
          <P class="text-sm text-green-800 dark:text-green-200">
            <strong>{$t('settings.nzb_compression.info.brotli_title')}</strong> {$t('settings.nzb_compression.info.brotli_description')}
          </P>
        </div>
      {/if}
    {:else}
      <div
        class="p-3 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded"
      >
        <P class="text-sm text-gray-600 dark:text-gray-400">
          {$t('settings.nzb_compression.info.disabled_description')}
        </P>
      </div>
    {/if}
  </div>
</Card>
