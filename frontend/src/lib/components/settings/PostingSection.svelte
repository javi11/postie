<script lang="ts">
// Imports
import apiClient from "$lib/api/client";
import DurationInput from "$lib/components/inputs/DurationInput.svelte";
import ThrottleRateInput from "$lib/components/inputs/ThrottleRateInput.svelte";
import { t } from "$lib/i18n";
import { advancedMode } from "$lib/stores/app";
import { toastStore } from "$lib/stores/toast";
import { config as configType } from "$lib/wailsjs/go/models";
import { CloudUpload, Plus, Save, Trash2 } from "lucide-svelte";
  import SizeInput from "$lib/components/inputs/SizeInput.svelte";

// Types
interface ComponentProps {
	config: configType.ConfigData;
}

// Props
const { config }: ComponentProps = $props();

// State
let saving = $state(false);
// Reactive local state for form inputs (following best practices)
let maxRetries = $state(config.posting.max_retries || 3);
let retryDelay = $state(config.posting.retry_delay || "5s");
let articleSizeInBytes = $state(config.posting.article_size_in_bytes || 750000);
let waitForPar2 = $state(config.posting.wait_for_par2 ?? true);
let groups = $state(
	config.posting.groups ? [...config.posting.groups] : ["alt.binaries.test"],
);
let customHeaders = $state(
	config.posting.post_headers?.custom_headers
		? [...config.posting.post_headers.custom_headers]
		: [],
);
let obfuscationPolicy = $state(config.posting.obfuscation_policy || "full");
let par2ObfuscationPolicy = $state(
	config.posting.par2_obfuscation_policy || "full",
);
let messageIdFormat = $state(config.posting.message_id_format || "random");
let groupPolicy = $state(config.posting.group_policy || "each_file");
let defaultFrom = $state(config.posting.post_headers?.default_from || "");
let addNgxHeader = $state(config.posting.post_headers?.add_nxg_header ?? false);
let throttleRateMB = $state(
	Math.round((config.posting.throttle_rate || 0) / 1048576),
);

// Initialize defaults (following initialization pattern)
function initializeDefaults() {
	if (!config.posting.post_headers) {
		config.posting.post_headers = new configType.PostHeaders();
	}

	// Set other defaults
	config.posting.throttle_rate = config.posting.throttle_rate || 0;
	config.posting.message_id_format =
		config.posting.message_id_format || "random";
	config.posting.par2_obfuscation_policy =
		config.posting.par2_obfuscation_policy || "full";
	config.posting.group_policy = config.posting.group_policy || "each_file";
}

// Initialize on component load
initializeDefaults();

// Effects - Sync local state back to config (consolidated pattern)
$effect(() => {
	// Sync all posting values in one effect for better performance
	config.posting.max_retries = maxRetries;
	config.posting.retry_delay = retryDelay;
	config.posting.article_size_in_bytes = articleSizeInBytes;
	config.posting.wait_for_par2 = waitForPar2;
	config.posting.groups = [...groups];
	config.posting.obfuscation_policy = obfuscationPolicy;
	config.posting.par2_obfuscation_policy = par2ObfuscationPolicy;
	config.posting.message_id_format = messageIdFormat;
	config.posting.group_policy = groupPolicy;
	config.posting.throttle_rate = throttleRateMB * 1048576;
});

$effect(() => {
	// Sync post headers separately to handle initialization
	if (!config.posting.post_headers) {
		config.posting.post_headers = new configType.PostHeaders();
	}
	config.posting.post_headers.default_from = defaultFrom;
	config.posting.post_headers.add_nxg_header = addNgxHeader;
	config.posting.post_headers.custom_headers = [...customHeaders];
});

// Ensure at least one group exists
$effect(() => {
	if (groups.length === 0) {
		groups = ["alt.binaries.test"];
	}
});

// Derived state - reactive arrays for dropdowns
const obfuscationOptions = $derived([
	{ value: "none", name: $t("settings.posting.obfuscation.none") },
	{ value: "partial", name: $t("settings.posting.obfuscation.partial") },
	{ value: "full", name: $t("settings.posting.obfuscation.full") },
]);

const messageIdOptions = $derived([
	{ value: "random", name: $t("settings.posting.message_id.random") },
	{ value: "nxg", name: $t("settings.posting.message_id.nxg") },
]);

const groupPolicyOptions = $derived([
	{ value: "all", name: $t("settings.posting.group_policy_options.all") },
	{
		value: "each_file",
		name: $t("settings.posting.group_policy_options.each_file"),
	},
]);

// Preset definitions (static data)
const retryDelayPresets = [
	{ label: "5s", value: 5, unit: "s" },
	{ label: "30s", value: 30, unit: "s" },
	{ label: "1m", value: 1, unit: "m" },
	{ label: "5m", value: 5, unit: "m" },
];

const articleSizePresets = [
	{ label: "500KB", value: 500, unit: "KB" },
	{ label: "750KB", value: 750, unit: "KB" },
	{ label: "1MB", value: 1, unit: "MB" },
];

const throttleRatePresets = [
	{ label: "Unlimited", value: 0 },
	{ label: "5 MB/s", value: 5 },
	{ label: "10 MB/s", value: 10 },
	{ label: "25 MB/s", value: 25 },
	{ label: "50 MB/s", value: 50 },
];

// Functions
function addGroup() {
	groups = [...groups, ""];
}

function removeGroup(index: number) {
	groups = groups.filter((_, i) => i !== index);
}

function addCustomHeader() {
	customHeaders = [...customHeaders, { name: "", value: "" }];
}

function removeCustomHeader(index: number) {
	customHeaders = customHeaders.filter((_, i) => i !== index);
}

// ThrottleRate update function (removed as it's now handled in $effect)
function updateThrottleRate() {
	// This function is kept for ThrottleRateInput component compatibility
	// Actual sync happens in $effect above
}

async function savePostingSettings() {
	try {
		saving = true;

		// Get current config to avoid conflicts
		const currentConfig = await apiClient.getConfig();
		// Update posting section with type conversion
		currentConfig.posting = {
			...config.posting,
			max_retries: Number.parseInt(String(config.posting.max_retries)) || 3,
			article_size_in_bytes:
				Number.parseInt(String(config.posting.article_size_in_bytes)) || 750000,
			retry_delay: config.posting.retry_delay || "5s",
			throttle_rate: config.posting.throttle_rate || 0,
			convertValues: config.posting.convertValues,
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

<!-- Main Container -->
<div class="card bg-base-100 shadow-sm">
  <div class="card-body space-y-6">
    <!-- Section Header -->
    <div class="flex items-center gap-3 pb-2">
      <CloudUpload class="w-5 h-5 text-success" />
      <h2 class="text-lg font-semibold text-base-content">
        {$t('settings.posting.title')}
      </h2>
    </div>

    <!-- Basic Configuration Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <!-- Max Retries -->
      <div>
        <label for="max-retries" class="label">
          <span class="label-text">{$t('settings.posting.max_retries')}</span>
        </label>
        <input
          id="max-retries"
          type="number"
          class="input input-bordered w-full"
          bind:value={maxRetries}
          min="0"
          max="10"
        />
        <p class="text-sm text-base-content/70 mt-1">
          {$t('settings.posting.max_retries_description')}
        </p>
      </div>
      <!-- Retry Delay -->
      <DurationInput
        bind:value={retryDelay}
        label={$t('settings.posting.retry_delay')}
        description={$t('settings.posting.retry_delay_description')}
        presets={retryDelayPresets}
        id="retry-delay"
      />

      <!-- Article Size -->
      <SizeInput
        bind:value={articleSizeInBytes}
        label={$t('settings.posting.article_size')}
        description={$t('settings.posting.article_size_description')}
        presets={articleSizePresets}
        minValue={1000}
        maxValue={10000000}
        id="article-size"
      />

      <!-- Advanced Settings (conditionally shown) -->
      {#if $advancedMode}
        <!-- Throttle Rate -->
        <ThrottleRateInput
          bind:value={throttleRateMB}
          onchange={updateThrottleRate}
          label={$t('settings.posting.throttle_rate')}
          description={$t('settings.posting.throttle_rate_description')}
          presets={throttleRatePresets}
          id="throttle-rate"
        />

        
        <!-- Obfuscation Policy -->
        <div>
          <label for="obfuscation" class="label">
            <span class="label-text">{$t('settings.posting.obfuscation_policy')}</span>
          </label>
          <select
            id="obfuscation"
            class="select select-bordered w-full"
            bind:value={obfuscationPolicy}
          >
            {#each obfuscationOptions as option}
              <option value={option.value}>{option.name}</option>
            {/each}
          </select>
          <div class="mt-2 p-3 bg-base-200 rounded text-xs">
            {#if obfuscationPolicy === "none"}
              <p class="text-base-content/70">
                {@html $t('settings.posting.obfuscation.none_description')}
              </p>
            {:else if obfuscationPolicy === "partial"}
              <p class="text-base-content/70">
                {@html $t('settings.posting.obfuscation.partial_description')}
              </p>
            {:else if obfuscationPolicy === "full"}
              <p class="text-base-content/70">
                {@html $t('settings.posting.obfuscation.full_description')}
              </p>
            {/if}
          </div>
        </div>

        <!-- PAR2 Obfuscation Policy -->
        <div>
          <label for="par2-obfuscation" class="label">
            <span class="label-text">{$t('settings.posting.par2_obfuscation_policy')}</span>
          </label>
          <select
            id="par2-obfuscation"
            class="select select-bordered w-full"
            bind:value={par2ObfuscationPolicy}
          >
            {#each obfuscationOptions as option}
              <option value={option.value}>{option.name}</option>
            {/each}
          </select>
          <div class="mt-2 p-3 bg-base-200 rounded text-xs">
            {#if par2ObfuscationPolicy === "none"}
              <p class="text-base-content/70">
                {@html $t('settings.posting.par2_obfuscation.none_description')}
              </p>
            {:else if par2ObfuscationPolicy === "partial"}
              <p class="text-base-content/70">
                {@html $t('settings.posting.par2_obfuscation.partial_description')}
              </p>
            {:else if par2ObfuscationPolicy === "full"}
              <p class="text-base-content/70">
                {@html $t('settings.posting.par2_obfuscation.full_description')}
              </p>
            {/if}
          </div>
        </div>

        <!-- Message ID Format -->
        <div>
          <label for="message-id-format" class="label">
            <span class="label-text">{$t('settings.posting.message_id_format')}</span>
          </label>
          <select
            id="message-id-format"
            class="select select-bordered w-full"
            bind:value={messageIdFormat}
          >
            {#each messageIdOptions as option}
              <option value={option.value}>{option.name}</option>
            {/each}
          </select>
          <p class="text-sm text-base-content/70 mt-1">
            {$t('settings.posting.message_id_format_description')}
          </p>
        </div>

        <!-- Group Policy -->
        <div>
          <label for="group-policy" class="label">
            <span class="label-text">{$t('settings.posting.group_policy')}</span>
          </label>
          <select
            id="group-policy"
            class="select select-bordered w-full"
            bind:value={groupPolicy}
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

    <!-- PAR2 Wait Option -->
    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <input 
          name="waitforpar2" 
          type="checkbox" 
          class="checkbox checkbox-primary" 
          bind:checked={waitForPar2} 
        />
        <label for="waitforpar2" class="text-sm font-medium text-base-content">
          {$t('settings.posting.wait_for_par2')}
        </label>
      </div>
      <p class="text-sm text-base-content/70 ml-6">
        {$t('settings.posting.wait_for_par2_description')}
      </p>
    </div>

    <!-- Advanced Sections (Headers & Custom Headers) -->
    {#if $advancedMode}
      <!-- Post Headers Section -->
      <div class="space-y-4 pt-2 border-t border-base-300">
        <h3 class="text-md font-medium text-base-content">
          {$t('settings.posting.headers.title')}
        </h3>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <!-- Default From -->
          <div>
            <label for="default-from" class="label mb-2">
              <span class="label-text">{$t('settings.posting.headers.default_from')}</span>
            </label>
            <input
              id="default-from"
              class="input input-bordered w-full"
              bind:value={defaultFrom}
              placeholder="poster@example.com"
              type="email"
            />
            <p class="text-sm text-base-content/70 mt-1">
              {$t('settings.posting.headers.default_from_description')}
            </p>
          </div>

          <!-- NXG Header Toggle -->
          <div class="flex items-center gap-3">
            <input 
              name="addNgxHeader" 
              type="checkbox" 
              class="checkbox checkbox-primary" 
              bind:checked={addNgxHeader} 
            />
            <label for="addNgxHeader" class="label-text text-sm font-medium">
              {$t('settings.posting.headers.add_nxg_header')} <a href="https://github.com/javi11/nxg" class="text-secondary">?</a>
            </label>
          </div>
        </div>

        <!-- Custom Headers -->
        <div class="space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <h4 class="text-sm font-medium text-base-content">
                {$t('settings.posting.headers.custom_headers.title')}
              </h4>
              <p class="text-sm text-base-content/70">
                {$t('settings.posting.headers.custom_headers.description')}
              </p>
            </div>
            <button
              type="button"
              class="btn btn-sm btn-primary flex items-center gap-2"
              onclick={addCustomHeader}
            >
              <Plus class="w-4 h-4" />
              {$t('settings.posting.headers.custom_headers.add_header')}
            </button>
          </div>

          {#if customHeaders.length > 0}
            <div class="space-y-3">
              {#each customHeaders as header, index (index)}
                <div class="flex items-center gap-3">
                  <div class="flex-1">
                    <input 
                      class="input input-bordered w-full" 
                      bind:value={customHeaders[index].name} 
                      placeholder={$t('settings.posting.headers.custom_headers.header_name_placeholder')} 
                    />
                  </div>
                  <div class="flex-1">
                    <input 
                      class="input input-bordered w-full" 
                      bind:value={customHeaders[index].value} 
                      placeholder={$t('settings.posting.headers.custom_headers.header_value_placeholder')} 
                    />
                  </div>
                  <button
                    type="button"
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
    <div class="space-y-4 pt-2 border-t border-base-300">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-md font-medium text-base-content">
            {$t('settings.posting.newsgroups.title')}
          </h3>
          <p class="text-sm text-base-content/70">
            {$t('settings.posting.newsgroups.description')}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-sm btn-primary flex items-center gap-2"
          onclick={addGroup}
        >
          <Plus class="w-4 h-4" />
          {$t('settings.posting.newsgroups.add_group')}
        </button>
      </div>

      <div class="space-y-3">
        {#each groups as group, index (index)}
          <div class="flex items-center gap-3">
            <div class="flex-1">
              <input
                class="input input-bordered w-full"
                bind:value={groups[index]}
                placeholder={$t('settings.posting.newsgroups.placeholder')}
                required
              />
            </div>
            {#if groups.length > 1}
              <button
                type="button"
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

      <div class="alert alert-info">
        <p class="text-sm">
          {@html $t('settings.posting.newsgroups.info')}
        </p>
      </div>
    </div>

    <!-- Save Button -->
    <div class="pt-4 border-t border-base-300">
      <button
        type="button"
        class="btn btn-success flex items-center gap-2"
        onclick={savePostingSettings}
        disabled={saving}
      >
        <Save class="w-4 h-4" />
        {saving ? $t('common.common.saving') : $t('settings.posting.save_button')}
      </button>
    </div>
  </div>
</div>
