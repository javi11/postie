<script lang="ts">
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import apiClient from "$lib/api/client";
import { X, Upload, Loader, ArrowLeft, Home } from "lucide-svelte";
import { onMount } from "svelte";

interface FileSystemItem {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  modTime: string;
}

interface FileManagerItem {
  id: string;
  size: number;
  date: Date;
  type: "folder" | "file";
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

let { isOpen, onClose }: Props = $props();

let fileManagerData = $state<FileManagerItem[]>([]);
let selectedFiles = $state<Set<string>>(new Set());
let importing = $state(false);
let loading = $state(false);
let pathInput = $state("");
let pathHistory = $state<string[]>([]);

let selectedFileCount = $derived(selectedFiles.size);
let hasSelectedFiles = $derived(selectedFiles.size > 0);

let currentPath = $state("/");

async function loadDirectory(path: string = "/"): Promise<FileManagerItem[]> {
  try {
    const response = await apiClient.browseFilesystem(path);
    // Convert our API response to SVAR FileManager format
    const items: FileManagerItem[] = response.items.map((item: FileSystemItem) => ({
      id: item.path,
      size: item.size,
      date: new Date(item.modTime),
      type: item.isDir ? "folder" : "file"
    }));

    return items;
  } catch (error) {
    console.error("Failed to load directory:", error);
    toastStore.error($t("common.messages.error"), String(error));
    return [];
  }
}

async function navigateToPath(path: string) {
  if (path !== currentPath) {
    pathHistory.push(currentPath);
  }
  
  loading = true;
  try {
    const items = await loadDirectory(path);
    fileManagerData = items;
    currentPath = path;
    pathInput = path;
    
    // Clear selection when navigating
    selectedFiles.clear();
    selectedFiles = new Set(selectedFiles);
  } finally {
    loading = false;
  }
}

async function loadInitialData() {
  loading = true;
  try {
    fileManagerData = await loadDirectory("/");
    pathInput = "/";
  } finally {
    loading = false;
  }
}

function goBack() {
  if (pathHistory.length > 0) {
    const previousPath = pathHistory.pop()!;
    navigateToPath(previousPath);
  }
}

function goHome() {
  pathHistory = [];
  navigateToPath("/");
}

function handlePathInputKeydown(event: KeyboardEvent) {
  if (event.key === "Enter") {
    navigateToPath(pathInput);
  }
}

async function importSelectedFiles() {
  if (selectedFiles.size === 0) {
    toastStore.error($t("common.messages.error"), $t("dashboard.file_explorer.no_files_selected"));
    return;
  }

  importing = true;
  try {
    const filePaths = Array.from(selectedFiles);
    await apiClient.importFiles(filePaths);

    toastStore.success(
      $t("dashboard.file_explorer.import_success"),
      $t("dashboard.file_explorer.import_success_description", { values: { count: filePaths.length } })
    );
    
    // Clear selection and close modal
    selectedFiles.clear();
    selectedFiles = new Set(selectedFiles);
    onClose();
  } catch (error) {
    console.error("Failed to import files:", error);
    toastStore.error($t("common.messages.error"), String(error));
  } finally {
    importing = false;
  }
}

function selectAll() {
  for (const item of fileManagerData) {
    selectedFiles.add(item.id);
  }
  selectedFiles = new Set(selectedFiles);
}

function clearSelection() {
  selectedFiles.clear();
  selectedFiles = new Set(selectedFiles);
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") {
    onClose();
  }
}

function handleBackdropKeydown(event: KeyboardEvent) {
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    onClose();
  }
}

// Load initial data when modal opens
$effect(() => {
  if (isOpen) {
    loadInitialData();
  }
});

onMount(() => {
  document.addEventListener("keydown", handleKeydown);
  return () => {
    document.removeEventListener("keydown", handleKeydown);
  };
});
</script>

{#if isOpen}
  <!-- Modal backdrop -->
  <div 
    class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
    onclick={onClose}
    onkeydown={handleBackdropKeydown}
    role="presentation"
    tabindex="-1"
  ></div>
  
  <!-- Modal content -->
  <div 
    class="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none"
    role="dialog"
    aria-modal="true"
    aria-labelledby="file-explorer-title"
  >
    <div 
      class="bg-base-100 rounded-lg shadow-2xl border border-base-300 w-full max-w-6xl h-[80vh] flex flex-col pointer-events-auto"
    >
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b border-base-300">
        <h2 id="file-explorer-title" class="text-xl font-semibold text-base-content">
          {$t("dashboard.file_explorer.title")}
        </h2>
        <button
          class="btn btn-ghost btn-sm"
          onclick={onClose}
          aria-label="Close"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Navigation Toolbar -->
      <div class="p-3 bg-base-50 border-b border-base-300 space-y-3">
        <!-- Navigation Controls -->
        <div class="flex items-center gap-2">
          <button
            class="btn btn-ghost btn-sm"
            onclick={goBack}
            disabled={pathHistory.length === 0}
            title={$t('dashboard.file_explorer.go_back_tooltip')}
          >
            <ArrowLeft class="w-4 h-4" />
          </button>

          <button
            class="btn btn-ghost btn-sm"
            onclick={goHome}
            title={$t('dashboard.file_explorer.go_home_tooltip')}
          >
            <Home class="w-4 h-4" />
          </button>

          <!-- Path Input -->
          <div class="flex-1">
            <input
              type="text"
              class="input input-bordered input-sm w-full"
              bind:value={pathInput}
              onkeydown={handlePathInputKeydown}
              placeholder={$t('dashboard.file_explorer.path_placeholder')}
            />
          </div>

          <button
            class="btn btn-primary btn-sm"
            onclick={() => navigateToPath(pathInput)}
            disabled={loading}
          >
            {$t('dashboard.file_explorer.go_button')}
          </button>
        </div>

        <!-- Selection Controls -->
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            {#if hasSelectedFiles}
              <span class="text-sm text-base-content/70">
                {$t('dashboard.file_explorer.items_selected', { values: { count: selectedFileCount } })}
              </span>
              <button
                class="btn btn-ghost btn-xs"
                onclick={clearSelection}
              >
                {$t('dashboard.file_explorer.clear_selection')}
              </button>
            {/if}

            <button
              class="btn btn-ghost btn-xs"
              onclick={selectAll}
              disabled={fileManagerData.length === 0}
            >
              {$t('dashboard.file_explorer.select_all')}
            </button>
          </div>

          <div class="text-sm text-base-content/70">
            {$t('dashboard.file_explorer.current_path', { values: { path: currentPath } })}
          </div>
        </div>
      </div>

      <!-- File Browser -->
      <div class="flex-1 overflow-hidden relative">
        {#if loading}
          <div class="absolute inset-0 bg-base-100/80 flex items-center justify-center z-10">
            <div class="flex items-center gap-2">
              <Loader class="w-6 h-6 animate-spin text-primary" />
              <span class="text-base-content/70">{$t('dashboard.file_explorer.loading')}</span>
            </div>
          </div>
        {/if}

        <div class="h-full overflow-y-auto">
          {#if fileManagerData.length === 0 && !loading}
            <div class="flex items-center justify-center h-full text-base-content/50">
              <div class="text-center">
                <div class="text-6xl mb-4">📁</div>
                <p>{$t('dashboard.file_explorer.empty_directory')}</p>
              </div>
            </div>
          {:else}
            <div class="divide-y divide-base-300">
              {#each fileManagerData as item}
                <div class="flex items-center gap-3 p-3 hover:bg-base-100 transition-colors">
                  <div class="text-2xl flex-shrink-0">
                    {item.type === "folder" ? "📁" : "📄"}
                  </div>
                  
                  <button
                    onclick={() => {
                      // Toggle selection for both files and folders
                      if (selectedFiles.has(item.id)) {
                        selectedFiles.delete(item.id);
                      } else {
                        selectedFiles.add(item.id);
                      }
                      selectedFiles = new Set(selectedFiles);
                    }}
                    ondblclick={() => {
                      // Double-click navigates into folders
                      if (item.type === "folder") {
                        navigateToPath(item.id);
                      }
                    }}
                    class={`flex-1 text-left min-w-0 ${
                      selectedFiles.has(item.id)
                        ? "text-primary font-medium"
                        : "text-base-content hover:text-primary"
                    }`}
                  >
                    <div class="font-medium truncate">{item.id.split('/').pop()}</div>
                    <div class="text-sm text-base-content/70 flex items-center gap-2 mt-1">
                      <span>
                        {item.type === "file"
                          ? $t('dashboard.file_explorer.file_size_kb', { values: { size: (item.size / 1024).toFixed(1) } })
                          : $t('dashboard.file_explorer.folder_type')
                        }
                      </span>
                      <span>•</span>
                      <span>{item.date.toLocaleDateString()}</span>
                    </div>
                  </button>
                  
                  <div class="flex-shrink-0">
                    <input
                      type="checkbox"
                      class="checkbox checkbox-primary"
                      checked={selectedFiles.has(item.id)}
                      onchange={() => {
                        if (selectedFiles.has(item.id)) {
                          selectedFiles.delete(item.id);
                        } else {
                          selectedFiles.add(item.id);
                        }
                        selectedFiles = new Set(selectedFiles);
                      }}
                    />
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between p-4 border-t border-base-300 bg-base-50">
        <div class="text-sm text-base-content/70">
          {#if hasSelectedFiles}
            {$t('dashboard.file_explorer.items_selected', { values: { count: selectedFileCount } })}
          {:else}
            {$t('dashboard.file_explorer.item_count_footer', { values: { count: fileManagerData.length } })}
          {/if}
        </div>

        <div class="flex gap-2">
          <button
            class="btn btn-ghost"
            onclick={onClose}
          >
            {$t('dashboard.file_explorer.cancel_button')}
          </button>

          <button
            class="btn btn-primary"
            onclick={importSelectedFiles}
            disabled={!hasSelectedFiles || importing}
          >
            {#if importing}
              <Loader class="w-4 h-4 animate-spin" />
              {$t('dashboard.file_explorer.adding_to_queue')}
            {:else}
              <Upload class="w-4 h-4" />
              {$t('dashboard.file_explorer.add_to_queue_button', { values: { count: selectedFileCount } })}
            {/if}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  /* Override SVAR FileManager styles to fit our design */
  :global(.wx-filemanager) {
    height: 100% !important;
    border: none !important;
  }
  
  :global(.wx-filemanager .wx-panel) {
    border: none !important;
  }
  
  :global(.wx-filemanager .wx-toolbar) {
    display: none !important; /* Hide the built-in toolbar */
  }
</style>