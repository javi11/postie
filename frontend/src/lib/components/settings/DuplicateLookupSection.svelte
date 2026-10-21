<script lang="ts">
import { t } from "$lib/i18n";
import type { config as configType } from "$lib/wailsjs/go/models";
let { config = $bindable() }: { config: configType.ConfigData } = $props();
if (!config.duplicate_lookup) config.duplicate_lookup = { enabled: false, url: "", api_key: "", timeout: "60s" };
let enabled = $state(config.duplicate_lookup.enabled ?? false);
let url = $state(config.duplicate_lookup.url ?? "");
let apiKey = $state(config.duplicate_lookup.api_key ?? "");
let timeout = $state(config.duplicate_lookup.timeout || "60s");
$effect(() => { config.duplicate_lookup.enabled = enabled; config.duplicate_lookup.url = url; config.duplicate_lookup.api_key = apiKey; config.duplicate_lookup.timeout = timeout; });
</script>
<div class="card bg-base-100 shadow-xl">
  <div class="card-body space-y-4">
    <h2 class="card-title text-lg">{$t('settings.duplicate_lookup.title')}</h2>
    <p class="text-sm text-base-content/70">{$t('settings.duplicate_lookup.description')}</p>
    <label class="label cursor-pointer justify-start gap-3">
      <input type="checkbox" class="checkbox" bind:checked={enabled} />
      <span class="label-text">{$t('settings.duplicate_lookup.enable')}</span>
    </label>
    {#if enabled}
      <label class="form-control" for="duplicate-lookup-url">
        <span class="label-text">{$t('settings.duplicate_lookup.url')}</span>
        <input id="duplicate-lookup-url" type="url" class="input input-bordered w-full" bind:value={url} placeholder="https://lookup.example/api/duplicate-lookup" />
      </label>
      <label class="form-control" for="duplicate-lookup-key">
        <span class="label-text">{$t('settings.duplicate_lookup.api_key')}</span>
        <input id="duplicate-lookup-key" type="password" autocomplete="off" class="input input-bordered w-full" bind:value={apiKey} />
      </label>
      <label class="form-control" for="duplicate-lookup-timeout">
        <span class="label-text">{$t('settings.duplicate_lookup.timeout')}</span>
        <input id="duplicate-lookup-timeout" type="text" class="input input-bordered w-full" bind:value={timeout} placeholder="60s" />
      </label>
    {/if}
  </div>
</div>
