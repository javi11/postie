<script lang="ts">
import type { ConfigData } from "$lib/types";
import {
	Button,
	Card,
	Checkbox,
	Heading,
	Input,
	Label,
	P,
	Select,
} from "flowbite-svelte";
import {
	CirclePlusSolid,
	CloudArrowUpSolid,
	TrashBinSolid,
} from "flowbite-svelte-icons";

export let config: ConfigData;

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

const obfuscationOptions = [
	{ value: "none", name: "None - No obfuscation" },
	{ value: "partial", name: "Partial - Basic obfuscation" },
	{ value: "full", name: "Full - Complete obfuscation" },
];

const messageIdOptions = [
	{ value: "random", name: "Random - Random message IDs" },
	{ value: "ngx", name: "NGX - NGX format message IDs" },
];

const groupPolicyOptions = [
	{ value: "all", name: "All - Post to all groups" },
	{ value: "each_file", name: "Each File - Random group per file" },
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
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <CloudArrowUpSolid class="w-5 h-5 text-green-600 dark:text-green-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        Posting Configuration
      </Heading>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <Label for="max-retries" class="mb-2">Max Retries</Label>
        <Input
          id="max-retries"
          type="number"
          bind:value={config.posting.max_retries}
          min="0"
          max="10"
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Number of times to retry failed uploads
        </P>
      </div>

      <div>
        <Label for="retry-delay" class="mb-2">Retry Delay</Label>
        <Input
          id="retry-delay"
          bind:value={config.posting.retry_delay}
          placeholder="5s"
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Delay between retries (e.g., 5s, 30s, 1m)
        </P>
      </div>

      <div>
        <Label for="article-size" class="mb-2">Article Size (bytes)</Label>
        <Input
          id="article-size"
          type="number"
          bind:value={config.posting.article_size_in_bytes}
          min="1000"
          max="10000000"
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Size of each article chunk (default: 750000)
        </P>
      </div>

      <div>
        <Label for="throttle-rate" class="mb-2">Throttle Rate (MB/s)</Label>
        <Input
          id="throttle-rate"
          type="number"
          bind:value={throttleRateMB}
          min="0"
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Upload speed limit in MB per second (0 = unlimited)
        </P>
      </div>

      <div>
        <Label for="obfuscation" class="mb-2">Obfuscation Policy</Label>
        <Select
          id="obfuscation"
          items={obfuscationOptions}
          bind:value={config.posting.obfuscation_policy}
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Filename obfuscation level
        </P>
      </div>

      <div>
        <Label for="par2-obfuscation" class="mb-2"
          >PAR2 Obfuscation Policy</Label
        >
        <Select
          id="par2-obfuscation"
          items={obfuscationOptions}
          bind:value={config.posting.par2_obfuscation_policy}
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          PAR2 file obfuscation level
        </P>
      </div>

      <div>
        <Label for="message-id-format" class="mb-2">Message ID Format</Label>
        <Select
          id="message-id-format"
          items={messageIdOptions}
          bind:value={config.posting.message_id_format}
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          Format for article message IDs
        </P>
      </div>

      <div>
        <Label for="group-policy" class="mb-2">Group Policy</Label>
        <Select
          id="group-policy"
          items={groupPolicyOptions}
          bind:value={config.posting.group_policy}
        />
        <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
          How to distribute files across newsgroups
        </P>
      </div>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Checkbox bind:checked={config.posting.wait_for_par2} />
        <Label class="text-sm font-medium">Wait for PAR2</Label>
      </div>
      <P class="text-sm text-gray-600 dark:text-gray-400 ml-6">
        Wait for PAR2 files to be created before starting upload
      </P>
    </div>

    <!-- Post Headers Section -->
    <div class="space-y-4">
      <Heading
        tag="h3"
        class="text-md font-medium text-gray-900 dark:text-white"
      >
        Post Headers
      </Heading>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <Label for="default-from" class="mb-2">Default From</Label>
          <Input
            id="default-from"
            bind:value={config.posting.post_headers.default_from}
            placeholder="poster@example.com"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Default From header (leave empty for random)
          </P>
        </div>

        <div class="flex items-center gap-3 mt-6">
          <Checkbox bind:checked={config.posting.post_headers.add_ngx_header} />
          <Label class="text-sm font-medium">Add NGX Header</Label>
        </div>
      </div>

      <!-- Custom Headers -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <Heading
              tag="h4"
              class="text-sm font-medium text-gray-900 dark:text-white"
            >
              Custom Headers
            </Heading>
            <P class="text-sm text-gray-600 dark:text-gray-400">
              Additional headers to include in posts
            </P>
          </div>
          <Button
            size="sm"
            onclick={addCustomHeader}
            class="cursor-pointer flex items-center gap-2"
          >
            <CirclePlusSolid class="w-4 h-4" />
            Add Header
          </Button>
        </div>

        {#if config.posting.post_headers.custom_headers && config.posting.post_headers.custom_headers.length > 0}
          <div class="space-y-3">
            {#each config.posting.post_headers.custom_headers as header, index (index)}
              <div class="flex items-center gap-3">
                <div class="flex-1">
                  <Input bind:value={header.name} placeholder="Header-Name" />
                </div>
                <div class="flex-1">
                  <Input bind:value={header.value} placeholder="Header Value" />
                </div>
                <Button
                  size="sm"
                  color="red"
                  variant="outline"
                  onclick={() => removeCustomHeader(index)}
                  class="cursor-pointer flex items-center gap-1"
                >
                  <TrashBinSolid class="w-3 h-3" />
                  Remove
                </Button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <!-- Newsgroups Section -->
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <Heading
            tag="h3"
            class="text-md font-medium text-gray-900 dark:text-white"
          >
            Newsgroups
          </Heading>
          <P class="text-sm text-gray-600 dark:text-gray-400">
            Groups where files will be posted
          </P>
        </div>
        <Button
          size="sm"
          onclick={addGroup}
          class="cursor-pointer flex items-center gap-2"
        >
          <CirclePlusSolid class="w-4 h-4" />
          Add Group
        </Button>
      </div>

      <div class="space-y-3">
        {#each config.posting.groups as group, index (index)}
          <div class="flex items-center gap-3">
            <div class="flex-1">
              <Input
                bind:value={config.posting.groups[index]}
                placeholder="alt.binaries.example"
                required
              />
            </div>
            {#if config.posting.groups.length > 1}
              <Button
                size="sm"
                color="red"
                variant="outline"
                onclick={() => removeGroup(index)}
                class="cursor-pointer flex items-center gap-1"
              >
                <TrashBinSolid class="w-3 h-3" />
                Remove
              </Button>
            {/if}
          </div>
        {/each}
      </div>

      <div
        class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
      >
        <P class="text-sm text-blue-800 dark:text-blue-200">
          <strong>Common newsgroups:</strong> alt.binaries.test (for testing), alt.binaries.misc,
          alt.binaries.multimedia. Choose groups appropriate for your content type.
        </P>
      </div>
    </div>
  </div>
</Card>
