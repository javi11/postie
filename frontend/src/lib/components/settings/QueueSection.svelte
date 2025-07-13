<script lang="ts">
import { t } from "$lib/i18n";
import type { ConfigData } from "$lib/types";
import { Quote } from "lucide-svelte";

export let config: ConfigData;

// Ensure queue exists with defaults
if (!config.queue) {
	config.queue = {
		database_type: "sqlite",
		database_path: "./postie_queue.db",
		max_concurrent_uploads: 3,
	};
}

// Create reactive array for database types dropdown
$: databaseTypes = [
	{ value: "sqlite", name: $t("settings.queue.database_types.sqlite") },
];
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
            bind:value={config.queue.database_type}
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
            bind:value={config.queue.database_path}
            placeholder={config.queue.database_type === "sqlite"
              ? $t('settings.queue.database_path_placeholder_sqlite')
              : $t('settings.queue.database_path_placeholder_network')}
          />
          <div class="label">
            <span class="label-text-alt">
              {config.queue.database_type === "sqlite"
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
            bind:value={config.queue.max_concurrent_uploads}
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
  </div>
</div>
