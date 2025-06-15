<script lang="ts">
import { page } from "$app/stores";
import { Button, Heading, P } from "flowbite-svelte";
import {
	ArrowLeftOutline,
	ExclamationCircleSolid,
} from "flowbite-svelte-icons";
</script>

<svelte:head>
  <title>Error {$page.status} - Postie</title>
</svelte:head>

<div
  class="flex flex-col items-center justify-center min-h-[60vh] text-center space-y-6"
>
  <div class="p-6 bg-red-50 dark:bg-red-900/20 rounded-full">
    <ExclamationCircleSolid class="w-16 h-16 text-red-500" />
  </div>

  <div class="space-y-2">
    <Heading tag="h1" class="text-4xl font-bold text-gray-900 dark:text-white">
      {$page.status}
    </Heading>
    <P class="text-xl text-gray-600 dark:text-gray-400">
      {$page.error?.message || "Something went wrong"}
    </P>
  </div>

  <div class="space-y-4">
    <P class="text-gray-600 dark:text-gray-400 max-w-md">
      We apologize for the inconvenience. Please try refreshing the page or go
      back to the dashboard.
    </P>

    <div class="flex gap-4 justify-center">
      <Button href="/" class="cursor-pointer flex items-center gap-2">
        <ArrowLeftOutline class="w-4 h-4" />
        Back to Dashboard
      </Button>
      <Button color="alternative" onclick={() => window.location.reload()}>
        Refresh Page
      </Button>
    </div>
  </div>
</div>
