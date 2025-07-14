<script lang="ts">
import { t } from "$lib/i18n";
import { AlertCircle, CheckCircle, Cog, Save } from "lucide-svelte";

interface Props {
	needsConfiguration?: boolean;
	criticalConfigError?: boolean;
	onsave?: () => void;
}

let {
	needsConfiguration = false,
	criticalConfigError = false,
	onsave,
}: Props = $props();

function handleSave() {
	onsave?.();
}
</script>

<header class="card bg-base-100 shadow-xl border border-base-300">
  <div class="card-body">
    <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
      <div class="flex-1">
        <div class="flex items-center gap-3 mb-2">
          <Cog class="w-6 h-6 text-base-content/70" />
          <h1 class="text-2xl font-bold">
            {$t('settings.header.title')}
          </h1>
          {#if criticalConfigError}
            <div class="badge badge-error gap-2">
              <AlertCircle class="w-4 h-4" />
              {$t('settings.header.status.configuration_error')}
            </div>
          {:else if needsConfiguration}
            <div class="badge badge-warning gap-2">
              <AlertCircle class="w-4 h-4" />
              {$t('settings.header.status.configuration_required')}
            </div>
          {:else}
            <div class="badge badge-success gap-2">
              <CheckCircle class="w-4 h-4" />
              {$t('settings.header.status.configured')}
            </div>
          {/if}
        </div>

        <p class="text-base-content/70">
          {$t('settings.header.description')}
        </p>

        {#if criticalConfigError}
          <div class="alert alert-error mt-4">
            <AlertCircle class="w-4 h-4" />
            <span>
              <strong>{$t('settings.header.alerts.configuration_error')}</strong> {$t('settings.header.alerts.configuration_error_description')}
            </span>
          </div>
        {:else if needsConfiguration}
          <div class="alert alert-warning mt-4">
            <AlertCircle class="w-4 h-4" />
            <span>
              <strong>{$t('settings.header.alerts.setup_required')}</strong> {$t('settings.header.alerts.setup_required_description')}
            </span>
          </div>
        {/if}
      </div>

      <div class="flex flex-col sm:flex-row gap-3">
        <button
          class="btn btn-primary"
          onclick={handleSave}
        >
          <Save class="w-4 h-4" />
          {$t('settings.header.save_configuration')}
        </button>
      </div>
    </div>
  </div>
</header>
