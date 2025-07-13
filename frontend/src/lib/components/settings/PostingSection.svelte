<script lang="ts">
import apiClient from "$lib/api/client";
import ByteSizeInput from "$lib/components/inputs/ByteSizeInput.svelte";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import ThrottleRateInput from "$lib/components/inputs/ThrottleRateInput.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import type { ConfigData } from "$lib/types";
import { CloudUpload, Plus, Save, Trash2 } from "lucide-svelte";

export let config: ConfigData;

let saving = false;

// Ensure posting config has all required fields with defaults
if (!config.posting.wait_for_par2) {
	config.posting.wait_for_par2 = true;
}
if (!config.posting.throttle_rate) {
	config.posting.throttle_rate = 0; // unlimited
}
if (!config.posting.max_workers) {
	config.posting.max_workers = 0;
}
if (!config.posting.message_id_format) {
	config.posting.message_id_format = "random";
}
if (!config.posting.post_headers) {
	config.posting.post_headers = {
		add_ngx_header: false,
		default_from: "",
		custom_headers: [],
	};
}
if (!config.posting.par2_obfuscation_policy) {
	config.posting.par2_obfuscation_policy = "full";
}
if (!config.posting.group_policy) {
	config.posting.group_policy = "each_file";
}

// Create reactive arrays for dropdowns
$: obfuscationOptions = [
	{ value: "none", name: $t("settings.posting.obfuscation.none") },
	{ value: "partial", name: $t("settings.posting.obfuscation.partial") },
	{ value: "full", name: $t("settings.posting.obfuscation.full") },
];

$: messageIdOptions = [
	{ value: "random", name: $t("settings.posting.message_id.random") },
	{ value: "ngx", name: $t("settings.posting.message_id.ngx") },
];

$: groupPolicyOptions = [
	{ value: "all", name: $t("settings.posting.group_policy_options.all") },
	{
		value: "each_file",
		name: $t("settings.posting.group_policy_options.each_file"),
	},
];

// Preset definitions
const retryDelayPresets = [
	{ label: "5s", value: 5, unit: "s" },
	{ label: "30s", value: 30, unit: "s" },
	{ label: "1m", value: 1, unit: "m" },
	{ label: "5m", value: 5, unit: "m" },
];

const articleSizePresets = [
	{ label: "500KB", value: 500000 },
	{ label: "750KB", value: 750000 },
	{ label: "1MB", value: 1000000 },
];

const throttleRatePresets = [
	{ label: "Unlimited", value: 0 },
	{ label: "5 MB/s", value: 5 },
	{ label: "10 MB/s", value: 10 },
	{ label: "25 MB/s", value: 25 },
	{ label: "50 MB/s", value: 50 },
];

function addGroup() {
	if (!config.posting.groups) {
		config.posting.groups = [];
	}
	config.posting.groups = [...config.posting.groups, ""];
}

function removeGroup(index: number) {
	config.posting.groups = config.posting.groups.filter((_, i) => i !== index);
}

function addCustomHeader() {
	if (!config.posting.post_headers.custom_headers) {
		config.posting.post_headers.custom_headers = [];
	}
	config.posting.post_headers.custom_headers = [
		...config.posting.post_headers.custom_headers,
		{ name: "", value: "" },
	];
}

function removeCustomHeader(index: number) {
	config.posting.post_headers.custom_headers =
		config.posting.post_headers.custom_headers.filter((_, i) => i !== index);
}

// Ensure we have at least one group
$: if (config.posting.groups.length === 0) {
	config.posting.groups = ["alt.binaries.test"];
}

// Reactive variable for display (MB/s)
let throttleRateMB: number;

// Convert throttle rate for display (bytes to MB/s)
$: throttleRateMB = Math.round(config.posting.throttle_rate / 1048576);

// Update throttle rate when MB value changes
$: if (
	throttleRateMB !== undefined &&
	!Number.isNaN(throttleRateMB) &&
	throttleRateMB * 1048576 !== config.posting.throttle_rate
) {
	config.posting.throttle_rate = throttleRateMB * 1048576;
}

async function savePostingSettings() {
	try {
		saving = true;

		// Get the current config from the server to avoid conflicts
		const currentConfig = await apiClient.getConfig();

		// Only update the posting fields with proper type conversion
		currentConfig.posting = {
			...config.posting,
			max_retries: Number.parseInt(config.posting.max_retries) || 3,
			article_size_in_bytes:
				Number.parseInt(config.posting.article_size_in_bytes) || 750000,
			retry_delay: config.posting.retry_delay || "5s",
			throttle_rate: config.posting.throttle_rate || 0,
		};

		await apiClient.saveConfig(currentConfig);

		toastStore.success(
			$t("settings.posting.saved_success"),
			$t("settings.posting.saved_success_description"),
		);
	} catch (error) {
		console.error("Failed to save posting settings:", error);
		toastStore.error($t("common.messages.error_saving"), String(error));
	} finally {
		saving = false;
	}
}
</script>

<div class="card bg-base-100 shadow-sm">
  <div class="card-body space-y-6">
    <div class="flex items-center gap-3">
      <CloudUpload class="w-5 h-5 text-green-600 dark:text-green-400" />
      <h2 class="text-lg font-semibold text-base-content">
        {$t('settings.posting.title')}
      </h2>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <label for="max-retries" class="label">
          <span class="label-text">{$t('settings.posting.max_retries')}</span>
        </label>
        <input
          id="max-retries"
          type="number"
          class="input input-bordered w-full"
          bind:value={config.posting.max_retries}
          min="0"
          max="10"
        />
        <p class="text-sm text-base-content/70 mt-1">
          {$t('settings.posting.max_retries_description')}
        </p>
      </div>

      <DurationInput
        bind:value={config.posting.retry_delay}
        label={$t('settings.posting.retry_delay')}
        description={$t('settings.posting.retry_delay_description')}
        presets={retryDelayPresets}
        id="retry-delay"
      />

      <ByteSizeInput
        bind:value={config.posting.article_size_in_bytes}
        label={$t('settings.posting.article_size')}
        description={$t('settings.posting.article_size_description')}
        presets={articleSizePresets}
        minValue={1000}
        maxValue={10000000}
        id="article-size"
      />

{#if $advancedMode}
      <ThrottleRateInput
        bind:value={throttleRateMB}
        label={$t('settings.posting.throttle_rate')}
        description={$t('settings.posting.throttle_rate_description')}
        presets={throttleRatePresets}
        id="throttle-rate"
      />

      <div>
        <label for="obfuscation" class="label">
          <span class="label-text">{$t('settings.posting.obfuscation_policy')}</span>
        </label>
        <select
          id="obfuscation"
          class="select select-bordered w-full"
          bind:value={config.posting.obfuscation_policy}
        >
          {#each obfuscationOptions as option}
            <option value={option.value}>{option.name}</option>
          {/each}
        </select>
        <div class="mt-2 p-3 bg-base-200 rounded text-xs">
          {#if config.posting.obfuscation_policy === "none"}
            <p class="text-base-content/70">
              {@html $t('settings.posting.obfuscation.none_description')}
            </p>
          {:else if config.posting.obfuscation_policy === "partial"}
            <p class="text-base-content/70">
              {@html $t('settings.posting.obfuscation.partial_description')}
            </p>
          {:else if config.posting.obfuscation_policy === "full"}
            <p class="text-base-content/70">
              {@html $t('settings.posting.obfuscation.full_description')}
            </p>
          {/if}
        </div>
      </div>

      <div>
        <label for="par2-obfuscation" class="label">
          <span class="label-text">{$t('settings.posting.par2_obfuscation_policy')}</span>
        </label>
        <select
          id="par2-obfuscation"
          class="select select-bordered w-full"
          bind:value={config.posting.par2_obfuscation_policy}
        >
          {#each obfuscationOptions as option}
            <option value={option.value}>{option.name}</option>
          {/each}
        </select>
        <div class="mt-2 p-3 bg-base-200 rounded text-xs">
          {#if config.posting.par2_obfuscation_policy === "none"}
            <p class="text-base-content/70">
              {@html $t('settings.posting.par2_obfuscation.none_description')}
            </p>
          {:else if config.posting.par2_obfuscation_policy === "partial"}
            <p class="text-base-content/70">
              {@html $t('settings.posting.par2_obfuscation.partial_description')}
            </p>
          {:else if config.posting.par2_obfuscation_policy === "full"}
            <p class="text-base-content/70">
              {@html $t('settings.posting.par2_obfuscation.full_description')}
            </p>
          {/if}
        </div>
      </div>

      <div>
        <label for="message-id-format" class="label">
          <span class="label-text">{$t('settings.posting.message_id_format')}</span>
        </label>
        <select
          id="message-id-format"
          class="select select-bordered w-full"
          bind:value={config.posting.message_id_format}
        >
          {#each messageIdOptions as option}
            <option value={option.value}>{option.name}</option>
          {/each}
        </select>
        <p class="text-sm text-base-content/70 mt-1">
          {$t('settings.posting.message_id_format_description')}
        </p>
      </div>

      <div>
        <label for="group-policy" class="label">
          <span class="label-text">{$t('settings.posting.group_policy')}</span>
        </label>
        <select
          id="group-policy"
          class="select select-bordered w-full"
          bind:value={config.posting.group_policy}
        >
          {#each groupPolicyOptions as option}
            <option value={option.value}>{option.name}</option>
          {/each}
        </select>
        <p class="text-sm text-base-content/70 mt-1">
          {$t('settings.posting.group_policy_description')}
        </p>
      </div>
{/if}
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <input type="checkbox" class="checkbox" bind:checked={config.posting.wait_for_par2} />
        <label class="text-sm font-medium text-base-content">{$t('settings.posting.wait_for_par2')}</label>
      </div>
      <p class="text-sm text-base-content/70 ml-6">
        {$t('settings.posting.wait_for_par2_description')}
      </p>
    </div>

{#if $advancedMode}
    <!-- Post Headers Section -->
    <div class="space-y-4">
      <h3 class="text-md font-medium">
        {$t('settings.posting.headers.title')}
      </h3>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label for="default-from" class="label mb-2">
            <span class="label-text">{$t('settings.posting.headers.default_from')}</span>
          </label>
          <input
            id="default-from"
            class="input input-bordered w-full"
            bind:value={config.posting.post_headers.default_from}
            placeholder="poster@example.com"
          />
          <p class="text-sm text-base-content/70 mt-1">
            {$t('settings.posting.headers.default_from_description')}
          </p>
        </div>

        <div class="flex items-center gap-3 mt-6">
          <input type="checkbox" class="checkbox checkbox-primary" bind:checked={config.posting.post_headers.add_ngx_header} />
          <label class="label-text text-sm font-medium">{$t('settings.posting.headers.add_ngx_header')}</label>
        </div>
      </div>

      <!-- Custom Headers -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h4 class="text-sm font-medium">
              {$t('settings.posting.headers.custom_headers.title')}
            </h4>
            <p class="text-sm text-base-content/70">
              {$t('settings.posting.headers.custom_headers.description')}
            </p>
          </div>
          <button
            class="btn btn-sm btn-primary flex items-center gap-2"
            onclick={addCustomHeader}
          >
            <Plus class="w-4 h-4" />
            {$t('settings.posting.headers.custom_headers.add_header')}
          </button>
        </div>

        {#if config.posting.post_headers.custom_headers && config.posting.post_headers.custom_headers.length > 0}
          <div class="space-y-3">
            {#each config.posting.post_headers.custom_headers as header, index (index)}
              <div class="flex items-center gap-3">
                <div class="flex-1">
                  <input class="input input-bordered w-full" bind:value={header.name} placeholder={$t('settings.posting.headers.custom_headers.header_name_placeholder')} />
                </div>
                <div class="flex-1">
                  <input class="input input-bordered w-full" bind:value={header.value} placeholder={$t('settings.posting.headers.custom_headers.header_value_placeholder')} />
                </div>
                <button
                  class="btn btn-sm btn-error btn-outline flex items-center gap-1"
                  onclick={() => removeCustomHeader(index)}
                >
                  <Trash2 class="w-3 h-3" />
                  {$t('settings.posting.headers.custom_headers.remove')}
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
{/if}

    <!-- Newsgroups Section -->
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-md font-medium">
            {$t('settings.posting.newsgroups.title')}
          </h3>
          <p class="text-sm text-base-content/70">
            {$t('settings.posting.newsgroups.description')}
          </p>
        </div>
        <button
          class="btn btn-sm btn-primary flex items-center gap-2"
          onclick={addGroup}
        >
          <Plus class="w-4 h-4" />
          {$t('settings.posting.newsgroups.add_group')}
        </button>
      </div>

      <div class="space-y-3">
        {#each config.posting.groups as group, index (index)}
          <div class="flex items-center gap-3">
            <div class="flex-1">
              <input
                class="input input-bordered w-full"
                bind:value={config.posting.groups[index]}
                placeholder={$t('settings.posting.newsgroups.placeholder')}
                required
              />
            </div>
            {#if config.posting.groups.length > 1}
              <button
                class="btn btn-sm btn-error btn-outline flex items-center gap-1"
                onclick={() => removeGroup(index)}
              >
                <Trash2 class="w-3 h-3" />
                {$t('settings.posting.newsgroups.remove')}
              </button>
            {/if}
          </div>
        {/each}
      </div>

      <div
        class="alert alert-info"
      >
        <p class="text-sm">
          {@html $t('settings.posting.newsgroups.info')}
        </p>
      </div>
    </div>

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        class="btn btn-success"
        onclick={savePostingSettings}
        disabled={saving}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('settings.posting.saving') : $t('settings.posting.save_button')}
      </button>
    </div>
  </div>
</div>
