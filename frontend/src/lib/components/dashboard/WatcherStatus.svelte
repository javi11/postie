<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { onMount, onDestroy } from "svelte";
import { Eye, Clock, FolderOpen, Calendar, AlertCircle, RefreshCw } from "lucide-svelte";
import { watcher } from "$lib/wailsjs/go/models";

let watcherStatus = $state<watcher.WatcherStatusInfo>(new watcher.WatcherStatusInfo());
let loading = $state(true);
let error = $state<string>("");
let scanning = $state(false);
let statusCheckInterval: NodeJS.Timeout | null = null;

// Format next run time for display
function formatNextRun(nextRunISO: string): string {
  if (!nextRunISO) return "";
  
  const nextRun = new Date(nextRunISO);
  const now = new Date();
  const diffMs = nextRun.getTime() - now.getTime();
  
  // If it's in the past or very soon, show "Soon"
  if (diffMs <= 0) return $t("dashboard.watcher.next_run_soon");
  
  // Format relative time
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  
  if (diffDays > 0) {
    return $t("dashboard.watcher.next_run_days", { values: { count: diffDays } });
  } else if (diffHours > 0) {
    return $t("dashboard.watcher.next_run_hours", { values: { count: diffHours } });
  } else if (diffMinutes > 0) {
    return $t("dashboard.watcher.next_run_minutes", { values: { count: diffMinutes } });
  } else {
    return $t("dashboard.watcher.next_run_soon");
  }
}

// Format absolute time for tooltip
function formatAbsoluteTime(nextRunISO: string): string {
  if (!nextRunISO) return "";
  
  const nextRun = new Date(nextRunISO);
  return nextRun.toLocaleString();
}

// Check watcher status
async function checkWatcherStatus() {
  try {
    loading = true;
    error = "";
    watcherStatus = await apiClient.getWatcherStatus();
  } catch (err) {
    error = String(err);
    console.error("Failed to get watcher status:", err);
  } finally {
    loading = false;
  }
}

async function handleScanNow() {
  try {
    scanning = true;
    await apiClient.triggerScan();
    toastStore.success(
      $t("dashboard.watcher.scan_triggered"),
      $t("dashboard.watcher.scan_triggered_description")
    );
  } catch (err) {
    console.error("Failed to trigger scan:", err);
    toastStore.error("Error", String(err));
  } finally {
    scanning = false;
  }
}

// Setup periodic status checks
onMount(async () => {
  await checkWatcherStatus();
  // Check status every 30 seconds
  statusCheckInterval = setInterval(checkWatcherStatus, 30000);
});

onDestroy(() => {
  if (statusCheckInterval) {
    clearInterval(statusCheckInterval);
    statusCheckInterval = null;
  }
});
</script>

<!-- Only show if watcher is enabled -->
{#if watcherStatus.enabled}
  <div class="card bg-base-100/60 backdrop-blur-sm border border-base-300/60 shadow-lg">
    <div class="card-body p-4">
      <div class="flex items-center gap-3 mb-3">
        <Eye class="w-5 h-5 text-primary" />
        <h3 class="text-lg font-semibold text-base-content">
          {$t("dashboard.watcher.title")}
        </h3>

        <!-- Status indicator -->
        {#if loading}
          <div class="loading loading-spinner loading-xs"></div>
        {:else if error}
          <AlertCircle class="w-4 h-4 text-error"/>
        {:else if watcherStatus.initialized}
          <div class="w-2 h-2 bg-success rounded-full animate-pulse"
               title={$t("dashboard.watcher.status_active")}></div>
        {:else}
          <div class="w-2 h-2 bg-warning rounded-full"
               title={$t("dashboard.watcher.status_inactive")}></div>
        {/if}

        <div class="ml-auto">
          <button
            type="button"
            class="btn btn-xs btn-outline"
            onclick={handleScanNow}
            disabled={scanning || !watcherStatus.initialized}
            title={$t("dashboard.watcher.scan_now")}
          >
            <RefreshCw class="w-3 h-3 {scanning ? 'animate-spin' : ''}" />
            {$t("dashboard.watcher.scan_now")}
          </button>
        </div>
      </div>

      {#if error}
        <div class="text-sm text-error">
          {$t("dashboard.watcher.error_loading")}
        </div>
      {:else if !loading}
        <div class="space-y-2 text-sm">
          <!-- Watch Directory -->
          <div class="flex items-center gap-2 text-base-content/80">
            <FolderOpen class="w-4 h-4 text-base-content/60" />
            <span class="truncate" title={watcherStatus.watch_directory}>
              {watcherStatus.watch_directory || $t("dashboard.watcher.no_directory")}
            </span>
          </div>

          <!-- Next Run Time -->
          {#if watcherStatus.next_run && watcherStatus.initialized}
            <div class="flex items-center gap-2 text-base-content/80">
              <Clock class="w-4 h-4 text-base-content/60" />
              <span title={formatAbsoluteTime(watcherStatus.next_run)}>
                {$t("dashboard.watcher.next_scan")}: {formatNextRun(watcherStatus.next_run)}
              </span>
            </div>
          {/if}

          <!-- Schedule Info -->
          {#if watcherStatus.schedule}
            <div class="flex items-center gap-2 text-base-content/60 text-xs">
              <Calendar class="w-3 h-3" />
              <span>
                {$t("dashboard.watcher.schedule")}: {watcherStatus.schedule.start_time} - {watcherStatus.schedule.end_time}
              </span>
              {#if !watcherStatus.is_within_schedule}
                <span class="text-warning">({$t("dashboard.watcher.outside_schedule")})</span>
              {/if}
            </div>
          {/if}

          <!-- Status Messages -->
          {#if !watcherStatus.initialized}
            <div class="text-xs text-warning">
              {$t("dashboard.watcher.not_initialized")}
            </div>
          {:else if !watcherStatus.is_within_schedule}
            <div class="text-xs text-info">
              {$t("dashboard.watcher.waiting_for_schedule")}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}