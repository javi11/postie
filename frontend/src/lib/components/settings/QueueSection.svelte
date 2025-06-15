<script lang="ts">
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
import { QueueListSolid } from "flowbite-svelte-icons";

export let config: ConfigData;

// Ensure queue exists with defaults
if (!config.queue) {
	config.queue = {
		database_type: "sqlite",
		database_path: "./postie_queue.db",
		batch_size: 10,
		max_retries: 3,
		retry_delay: "5m",
		max_queue_size: 1000,
		cleanup_after: "24h",
		priority_processing: false,
		max_concurrent_uploads: 3,
	};
}

const databaseTypes = [
	{ value: "sqlite", name: "SQLite - File-based database" },
	{ value: "postgres", name: "PostgreSQL - Network database" },
	{ value: "mysql", name: "MySQL - Network database" },
];
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center gap-3">
      <QueueListSolid class="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
      <Heading
        tag="h2"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        Queue Configuration
      </Heading>
    </div>

    <div class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <Label for="database-type" class="mb-2">Database Type</Label>
          <Select
            id="database-type"
            items={databaseTypes}
            bind:value={config.queue.database_type}
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Database system to use for the upload queue
          </P>
        </div>

        <div>
          <Label for="database-path" class="mb-2"
            >Database Path/Connection</Label
          >
          <Input
            id="database-path"
            bind:value={config.queue.database_path}
            placeholder={config.queue.database_type === "sqlite"
              ? "./postie_queue.db"
              : "host=localhost user=postie dbname=postie"}
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {config.queue.database_type === "sqlite"
              ? "File path for SQLite database"
              : "Connection string for database"}
          </P>
        </div>

        <div>
          <Label for="batch-size" class="mb-2">Batch Size</Label>
          <Input
            id="batch-size"
            type="number"
            bind:value={config.queue.batch_size}
            min="1"
            max="100"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Number of items to process in each batch
          </P>
        </div>

        <div>
          <Label for="max-concurrent" class="mb-2">Max Concurrent Uploads</Label
          >
          <Input
            id="max-concurrent"
            type="number"
            bind:value={config.queue.max_concurrent_uploads}
            min="1"
            max="20"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Maximum number of simultaneous uploads from queue
          </P>
        </div>

        <div>
          <Label for="queue-max-retries" class="mb-2">Max Retries</Label>
          <Input
            id="queue-max-retries"
            type="number"
            bind:value={config.queue.max_retries}
            min="0"
            max="10"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Maximum retry attempts for failed uploads
          </P>
        </div>

        <div>
          <Label for="queue-retry-delay" class="mb-2">Retry Delay</Label>
          <Input
            id="queue-retry-delay"
            bind:value={config.queue.retry_delay}
            placeholder="5m"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Delay between retry attempts (e.g., 30s, 5m, 1h)
          </P>
        </div>

        <div>
          <Label for="max-queue-size" class="mb-2">Max Queue Size</Label>
          <Input
            id="max-queue-size"
            type="number"
            bind:value={config.queue.max_queue_size}
            min="0"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Maximum items in queue (0 = unlimited)
          </P>
        </div>

        <div>
          <Label for="cleanup-after" class="mb-2">Cleanup After</Label>
          <Input
            id="cleanup-after"
            bind:value={config.queue.cleanup_after}
            placeholder="24h"
          />
          <P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Auto-cleanup completed items after duration (0 = keep forever)
          </P>
        </div>
      </div>

      <div class="space-y-3">
        <div class="flex items-center gap-3">
          <Checkbox bind:checked={config.queue.priority_processing} />
          <Label class="text-sm font-medium">Priority Processing</Label>
        </div>
        <P class="text-sm text-gray-600 dark:text-gray-400 ml-6">
          Process larger files first to optimize upload efficiency
        </P>
      </div>
    </div>

    <div
      class="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded"
    >
      <P class="text-sm text-blue-800 dark:text-blue-200">
        <strong>Upload Queue:</strong> Manages file uploads with persistence, retry
        logic, and concurrent processing. Use SQLite for simple setups or PostgreSQL/MySQL
        for high-volume operations.
      </P>
    </div>
  </div>
</Card>
