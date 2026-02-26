<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { onMount, onDestroy } from "svelte";
import { Eye, Clock, FolderOpen, Calendar, AlertCircle, RefreshCw } from "lucide-svelte";
import { watcher } from "$lib/wailsjs/go/models";

let watcherStatuses = $state<watcher.WatcherStatusInfo[]>([]);
let loading = $state(true);
let error = $state<string>("");
let scanning = $state(false);
let statusCheckInterval: NodeJS.Timeout | null = null;

// Whether any watcher is enabled
let anyEnabled = $derived(watcherStatuses.some(s => s.enabled));

// Format next run time for display
function formatNextRun(nextRunISO: string): string {
  if (!nextRunISO) return "";

  const nextRun = new Date(nextRunISO);
  const now = new Date();
  const diffMs = nextRun.getTime() - now.getTime();

  if (diffMs <= 0) return $t("dashboard.watcher.next_run_soon");

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

function formatAbsoluteTime(nextRunISO: string): string {
  if (!nextRunISO) return "";
  return new Date(nextRunISO).toLocaleString();
}

function getWatcherDisplayName(status: watcher.WatcherStatusInfo, index: number): string {
  if (status.name) return status.name;
  return $t("dashboard.watcher.watcher_number", { values: { n: index + 1 } });
}

async function checkWatcherStatus() {
  try {
    loading = true;
    error = "";
    watcherStatuses = await apiClient.getWatcherStatus();
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

onMount(async () => {
  await checkWatcherStatus();
  statusCheckInterval = setInterval(checkWatcherStatus, 30000);
});

onDestroy(() => {
  if (statusCheckInterval) {
    clearInterval(statusCheckInterval);
    statusCheckInterval = null;
  }
});
</script>

{#if anyEnabled}
  <div class="card bg-base-100/60 backdrop-blur-sm border border-base-300/60 shadow-lg">
    <div class="card-body p-4">
      <div class="flex items-center gap-3 mb-3">
        <Eye class="w-5 h-5 text-primary" />
        <h3 class="text-lg font-semibold text-base-content">
          {$t("dashboard.watcher.title")}
        </h3>

        {#if loading}
          <div class="loading loading-spinner loading-xs"></div>
        {:else if error}
          <AlertCircle class="w-4 h-4 text-error"/>
        {/if}

        <div class="ml-auto">
          <button
            type="button"
            class="btn btn-xs btn-outline"
            onclick={handleScanNow}
            disabled={scanning || watcherStatuses.every(s => !s.initialized)}
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
        <div class="space-y-3">
          {#each watcherStatuses.filter(s => s.enabled) as status, i (status.watch_directory || i)}
            <div class="p-3 bg-base-200/50 rounded-lg space-y-2 text-sm">
              <!-- Watcher name + status dot -->
              <div class="flex items-center gap-2">
                {#if status.initialized}
                  <div class="w-2 h-2 bg-success rounded-full animate-pulse flex-shrink-0"
                       title={$t("dashboard.watcher.status_active")}></div>
                {:else}
                  <div class="w-2 h-2 bg-warning rounded-full flex-shrink-0"
                       title={$t("dashboard.watcher.status_inactive")}></div>
                {/if}
                <span class="font-medium text-base-content text-xs">
                  {getWatcherDisplayName(status, watcherStatuses.filter(s => s.enabled).indexOf(status))}
                </span>
              </div>

              <!-- Watch Directory -->
              <div class="flex items-center gap-2 text-base-content/80">
                <FolderOpen class="w-4 h-4 text-base-content/60 flex-shrink-0" />
                <span class="truncate" title={status.watch_directory}>
                  {status.watch_directory || $t("dashboard.watcher.no_directory")}
                </span>
              </div>

              <!-- Next Run Time -->
              {#if status.next_run && status.initialized}
                <div class="flex items-center gap-2 text-base-content/80">
                  <Clock class="w-4 h-4 text-base-content/60 flex-shrink-0" />
                  <span title={formatAbsoluteTime(status.next_run)}>
                    {$t("dashboard.watcher.next_scan")}: {formatNextRun(status.next_run)}
                  </span>
                </div>
              {/if}

              <!-- Schedule Info -->
              {#if status.schedule}
                <div class="flex items-center gap-2 text-base-content/60 text-xs">
                  <Calendar class="w-3 h-3 flex-shrink-0" />
                  <span>
                    {$t("dashboard.watcher.schedule")}: {status.schedule.start_time} - {status.schedule.end_time}
                  </span>
                  {#if !status.is_within_schedule}
                    <span class="text-warning">({$t("dashboard.watcher.outside_schedule")})</span>
                  {/if}
                </div>
              {/if}

              <!-- Status Messages -->
              {#if !status.initialized}
                <div class="text-xs text-warning">
                  {$t("dashboard.watcher.not_initialized")}
                </div>
              {:else if !status.is_within_schedule}
                <div class="text-xs text-info">
                  {$t("dashboard.watcher.waiting_for_schedule")}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}
