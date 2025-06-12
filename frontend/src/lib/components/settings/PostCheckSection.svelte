<script lang="ts">
  import { Card, Heading, Input, Label, Checkbox, P } from "flowbite-svelte";
  import { CheckCircleSolid } from "flowbite-svelte-icons";
  import type { ConfigData } from "$lib/types";

  export let config: ConfigData;

  // Ensure post_check exists with defaults
  if (!config.post_check) {
    config.post_check = {
      enabled: true,
      delay: "10s",
      max_reposts: 1,
    };
  }
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <CheckCircleSolid class="w-5 h-5 text-orange-600 dark:text-orange-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        Post Check Configuration
      </Heading>
    </div>

    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <Checkbox bind:checked={config.post_check.enabled} />
        <Label class="text-sm font-medium">Enable Post Check</Label>
      </div>
      <P class="text-sm text-gray-600 dark:text-gray-400 ml-6">
        Verify that articles are successfully uploaded and propagated after
        posting
      </P>
    </div>

    {#if config.post_check.enabled}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <Label for="check-delay" class="mb-2">Check Delay</Label>
          <Input
            id="check-delay"
            bind:value={config.post_check.delay}
            placeholder="10s"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Delay before checking if articles are available (e.g., 10s, 30s, 1m)
          </P>
        </div>

        <div>
          <Label for="max-reposts" class="mb-2">Max Re-posts</Label>
          <Input
            id="max-reposts"
            type="number"
            bind:value={config.post_check.max_reposts}
            min="0"
            max="10"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Maximum number of times to retry posting if article check fails
          </P>
        </div>
      </div>
    {/if}

    <div
      class="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded"
    >
      <P class="text-sm text-yellow-800 dark:text-yellow-200">
        <strong>Post Check:</strong> Verifies article availability after upload to
        ensure successful propagation. May increase upload time but improves reliability.
      </P>
    </div>
  </div>
</Card>
