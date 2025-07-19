<script lang="ts">
import { page } from "$app/state";
import { AlertCircle, ArrowLeft } from "lucide-svelte";
</script>

<svelte:head>
  <title>Error {page.status} - Postie</title>
</svelte:head>

<div class="flex flex-col items-center justify-center min-h-[60vh] text-center space-y-6">
  <div class="p-6 bg-error/20 rounded-full">
    <AlertCircle class="w-16 h-16 text-error" />
  </div>

  <div class="space-y-2">
    <h1 class="text-4xl font-bold">
      {page.status}
    </h1>
    <p class="text-xl text-base-content/70">
      {page.error?.message || "Something went wrong"}
    </p>
  </div>

  <div class="space-y-4">
    <p class="text-base-content/70 max-w-md">
      We apologize for the inconvenience. Please try refreshing the page or go
      back to the dashboard.
    </p>

    <div class="flex gap-4 justify-center">
      <a href="/" class="btn btn-primary">
        <ArrowLeft class="w-4 h-4" />
        Back to Dashboard
      </a>
      <button class="btn btn-outline" onclick={() => window.location.reload()}>
        Refresh Page
      </button>
    </div>
  </div>
</div>
