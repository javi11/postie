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
import { QuoteOutline } from "flowbite-svelte-icons";

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

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <QuoteOutline class="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        {$t('settings.queue.title')}
      </Heading>
    </div>

    <div class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <Label for="database-type" class="mb-2">{$t('settings.queue.database_type')}</Label>
          <Select
            id="database-type"
            items={databaseTypes}
            bind:value={config.queue.database_type}
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {$t('settings.queue.database_type_description')}
          </P>
        </div>

        <div>
          <Label for="database-path" class="mb-2">
            {$t('settings.queue.database_path')}
          </Label>
          <Input
            id="database-path"
            bind:value={config.queue.database_path}
            placeholder={config.queue.database_type === "sqlite"
              ? $t('settings.queue.database_path_placeholder_sqlite')
              : $t('settings.queue.database_path_placeholder_network')}
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {config.queue.database_type === "sqlite"
              ? $t('settings.queue.database_path_description_sqlite')
              : $t('settings.queue.database_path_description_network')}
          </P>
        </div>

        <div>
          <Label for="max-concurrent" class="mb-2">{$t('settings.queue.max_concurrent_uploads')}</Label>
          <Input
            id="max-concurrent"
            type="number"
            bind:value={config.queue.max_concurrent_uploads}
            min="1"
            max="20"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {$t('settings.queue.max_concurrent_uploads_description')}
          </P>
        </div>
      </div>
    </div>

    <div
      class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
    >
      <P class="text-sm text-blue-800 dark:text-blue-200">
        {@html $t('settings.queue.info')}
      </P>
    </div>
  </div>
</Card>
