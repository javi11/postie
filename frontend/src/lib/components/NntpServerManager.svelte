<script lang="ts">
  import apiClient from "$lib/api/client";
  import { t } from "$lib/i18n";
  import { toastStore } from "$lib/stores/toast";
  import { backend, config as configType } from "$lib/wailsjs/go/models";
  import { Check, CirclePlus, Loader2, Trash2, Server } from "lucide-svelte";

  interface ValidationState {
    status: "pending" | "validating" | "valid" | "invalid" | "incomplete";
    error: string;
  }

  interface Props {
    servers: configType.ServerConfig[];
    onupdate?: (servers: configType.ServerConfig[]) => void;
    onvalidationchange?: (data: { hasValidServers: boolean }) => void;
    showAdvancedFields?: boolean;
    variant?: "setup" | "settings";
  }

  let {
    servers = $bindable([]),
    onupdate,
    onvalidationchange,
    showAdvancedFields = false,
    variant = "setup",
  }: Props = $props();

  // Track validation state for each server
  let validationStates = $state<Record<number, ValidationState>>({});

  // Local reactive state for server properties
  let localServers = $state<configType.ServerConfig[]>([]);

  // Initialize local state from props only once
  $effect(() => {
    if (localServers.length === 0 && servers.length > 0) {
      localServers = servers.map((server) => ({ ...server }));
    }
  });

  // Update servers prop when local state changes
  function updateServers(): void {
    servers = localServers.map((server) => ({ ...server }));
    onupdate?.(servers);
  }

  function addServer(): void {
    const newServer = new configType.ServerConfig({
      max_connections: 10,
      inflight: 10,
      max_connection_idle_time_in_seconds: 300,
      max_connection_ttl_in_seconds: 3600,
      insecure_ssl: false,
    });

    localServers = [...localServers, newServer];
    const newIndex = localServers.length - 1;

    validationStates = {
      ...validationStates,
      [newIndex]: { status: "pending", error: "" },
    };

    updateServers();
  }

  function removeServer(index: number): void {
    localServers = localServers.filter((_, i) => i !== index);

    // Rebuild validation states with new indices
    const newValidationStates: Record<number, ValidationState> = {};
    let newIndex = 0;
    for (let i = 0; i < localServers.length + 1; i++) {
      if (i === index || !validationStates[i]) {
        continue;
      }
      newValidationStates[newIndex] = validationStates[i];
      newIndex++;
    }
    validationStates = newValidationStates;
    updateServers();
  }

  function onServerFieldChange(index: number): void {
    // Clear validation state when server data changes
    const currentState = validationStates[index];
    if (currentState && currentState.status !== "pending") {
      validationStates = {
        ...validationStates,
        [index]: { status: "pending", error: "" },
      };
    }
    updateServers();
  }

  function onServerEnabledChange(index: number, enabled: boolean): void {
    // If trying to disable, check if it's the last enabled server
    if (!enabled) {
      const enabledCount = localServers.filter(s => s.enabled !== false).length;
      if (enabledCount <= 1) {
        toastStore.error($t("settings.server.validation.cannot_disable_last"));
        // Revert the change
        localServers[index].enabled = true;
        return;
      }
    }

    localServers[index].enabled = enabled;
    onServerFieldChange(index);
  }

  function onServerCheckOnlyChange(index: number, checkOnly: boolean): void {
    // If trying to enable check_only, verify at least one posting server will remain
    if (checkOnly) {
      const postingServersCount = localServers.filter((s, i) => {
        const isEnabled = s.enabled !== false;
        const willBeCheckOnly = i === index ? true : (s.check_only === true);
        return isEnabled && !willBeCheckOnly;
      }).length;

      if (postingServersCount < 1) {
        toastStore.error($t("settings.server.validation.cannot_set_all_check_only"));
        // Revert the change
        localServers[index].check_only = false;
        return;
      }
    }

    localServers[index].check_only = checkOnly;
    onServerFieldChange(index);
  }

  function getServerValidationState(index: number): ValidationState {
    const state = validationStates[index];
    if (!state) {
      return { status: "pending", error: "" };
    }
    return state;
  }

  function isServerComplete(index: number): boolean {
    const validationState = getServerValidationState(index);
    return validationState.status === "valid";
  }

  // Check if any servers are valid and emit validation state
  function checkValidationState(): boolean {
    const hasValid = localServers.some((_, index) => isServerComplete(index));
    onvalidationchange?.({ hasValidServers: hasValid });
    return hasValid;
  }

  // Reactive effect to automatically check validation state when validation states change
  $effect(() => {
    // This will run whenever validationStates changes
    checkValidationState();
  });

  // Bulk operations
  function enableAllServers(): void {
    localServers.forEach((server, index) => {
      if (!server.enabled) {
        localServers[index].enabled = true;
        onServerFieldChange(index);
      }
    });
  }

  function disableAllServers(): void {
    // Ensure at least one server remains enabled
    const enabledCount = localServers.filter(s => s.enabled !== false).length;
    if (enabledCount <= 1) {
      toastStore.error($t("settings.server.bulk_operations.cannot_disable_all"));
      return;
    }
    
    localServers.forEach((server, index) => {
      if (server.enabled !== false) {
        localServers[index].enabled = false;
        onServerFieldChange(index);
      }
    });
  }

  // Computed values for bulk operations
  let enabledServersCount = $derived(localServers.filter(s => s.enabled !== false).length);
  let totalServersCount = $derived(localServers.length);
  let hasDisabledServers = $derived(localServers.some(s => s.enabled === false));
  let allServersDisabled = $derived(enabledServersCount === 0);

  async function validateServer(index: number): Promise<void> {
    const server = localServers[index];
    if (!server) {
      return;
    }

    // Basic validation first
    if (!server.host || !server.port) {
      validationStates = {
        ...validationStates,
        [index]: { status: "incomplete", error: "Host and port are required" },
      };
      checkValidationState();
      return;
    }

    validationStates = {
      ...validationStates,
      [index]: { status: "validating", error: "" },
    };

    try {
      const result = await apiClient.validateNNTPServer({
        host: server.host,
        port: server.port,
        username: server.username || "",
        password: server.password || "",
        ssl: server.ssl || false,
        maxConnections: server.max_connections || 10,
      });

      if (result.valid) {
        validationStates = {
          ...validationStates,
          [index]: { status: "valid", error: "" },
        };
        toastStore.success($t("settings.server.valid"));
        checkValidationState();
        return;
      }

      console.log("Setting server", index, "as invalid:", result.error);
      validationStates = {
        ...validationStates,
        [index]: { status: "invalid", error: result.error },
      };
      toastStore.error($t("settings.server.invalid"), String(result.error));
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : String(error);
      validationStates = {
        ...validationStates,
        [index]: {
          status: "invalid",
          error: `Validation failed: ${errorMessage}`,
        },
      };
      toastStore.error($t("settings.server.invalid"), String(errorMessage));
      console.error("Server validation error:", errorMessage);
    }

    checkValidationState();
  }

  // Add default server if none exist (only for setup variant)
  $effect(() => {
    if (
      variant === "setup" &&
      localServers.length === 0 &&
      servers.length === 0
    ) {
      addServer();
    }
  });
</script>

<div class="space-y-6">
  {#if variant === "setup"}
    <div>
      <h3 class="text-lg font-semibold text-base-content mb-2">
        {$t("settings.server.title")}
      </h3>
      <p class="text-base-content/70 mb-4">
        {$t("settings.server.description")}
      </p>
    </div>
  {/if}

  <!-- Status and Bulk Operations -->
  {#if localServers.length > 0 && variant === "settings"}
    <div class="flex items-center justify-between p-4 bg-base-200 rounded-lg mb-4">
      <div class="flex items-center gap-4">
        <div class="stat">
          <div class="stat-title text-sm">Providers Status</div>
          <div class="stat-value text-lg">
            {enabledServersCount} / {totalServersCount}
          </div>
          <div class="stat-desc text-xs">
            {$t("settings.server.status.providers_active")}
          </div>
        </div>
      </div>
      
      <div class="flex gap-2">
        {#if hasDisabledServers}
          <button 
            class="btn btn-sm btn-success"
            onclick={enableAllServers}
          >
            {$t("settings.server.bulk_operations.enable_all")}
          </button>
        {/if}
        {#if enabledServersCount > 1}
          <button 
            class="btn btn-sm btn-warning"
            onclick={disableAllServers}
          >
            {$t("settings.server.bulk_operations.disable_all")}
          </button>
        {/if}
      </div>
    </div>
  {/if}

  {#if localServers.length === 0}
    <div
      class="text-center py-8 border-2 border-dashed border-base-300 rounded-lg"
    >
      <Server class="w-12 h-12 text-base-content/40 mx-auto mb-4" />
      <p class="text-base-content/70 mb-4">
        {$t("settings.server.no_servers_description")}
      </p>
      <button type="button" class="btn btn-outline" onclick={addServer}>
        <CirclePlus class="w-4 h-4" />
        {$t("settings.server.add_first_server")}
      </button>
    </div>
  {:else}
    <div class="space-y-4">
      {#each localServers as server, index}
        {@const validationState = getServerValidationState(index)}
        {@const isEnabled = server.enabled !== false}
        <div class="card bg-base-100 shadow-lg {!isEnabled ? 'opacity-60' : ''}">
          <div class="card-body p-4">
            <div class="flex justify-between items-start mb-4">
              <div class="flex items-center gap-2">
                <h4 class="font-medium text-base-content">
                  {$t("settings.server.server_number", {
                    values: { number: index + 1 },
                  })}
                </h4>
                {#if !isEnabled}
                  <div class="badge badge-warning">
                    {$t("settings.server.disabled")}
                  </div>
                {/if}
                {#if validationState.status === "validating"}
                  <div class="badge badge-primary gap-1">
                    <Loader2 class="w-3 h-3 animate-spin" />
                    {$t("settings.server.validating")}
                  </div>
                {:else if validationState.status === "valid"}
                  <div class="badge badge-success gap-1">
                    <Check class="w-3 h-3" />
                    {$t("settings.server.valid")}
                  </div>
                {:else if validationState.status === "invalid"}
                  <div class="badge badge-error">
                    {$t("settings.server.invalid")}
                  </div>
                {:else if validationState.status === "incomplete"}
                  <div class="badge badge-error">
                    {$t("settings.server.incomplete")}
                  </div>
                {/if}
              </div>
              <div class="flex items-center gap-2">
                {#if validationState.status !== "validating"}
                  <button
                    class="btn btn-primary btn-outline btn-sm"
                    onclick={() => validateServer(index)}
                  >
                    {$t("settings.server.test_connection")}
                  </button>
                {/if}
                {#if localServers.length > 1}
                  <button
                    class="btn btn-error btn-outline btn-sm"
                    onclick={() => removeServer(index)}
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                {/if}
              </div>
            </div>

            {#if validationState.error}
              <div class="alert alert-error mb-4">
                <p class="text-sm">
                  {validationState.error}
                </p>
              </div>
            {/if}

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              {#if variant === "settings" && server.enabled !== undefined}
                <div class="md:col-span-2">
                  <div class="flex items-center gap-3 p-3 bg-base-200 rounded-lg">
                    <input
                      id="server-enabled-{index}"
                      type="checkbox"
                      class="checkbox checkbox-primary"
                      checked={localServers[index].enabled !== false}
                      onchange={(e) => onServerEnabledChange(index, (e.target as HTMLInputElement).checked)}
                    />
                    <label
                      for="server-enabled-{index}"
                      class="label-text cursor-pointer flex-1"
                    >
                      <span class="font-medium">{$t("settings.server.enabled")}</span>
                      <p class="text-sm opacity-70">
                        {isEnabled
                          ? $t("settings.server.enabled_description")
                          : $t("settings.server.disabled_description")
                        }
                      </p>
                    </label>
                  </div>
                </div>
              {/if}

              <!-- Check Only toggle - shown in both setup and settings -->
              {#if variant === "settings"}
                {@const isCheckOnly = localServers[index].check_only === true}
                <div class="md:col-span-2">
                  <div class="flex items-center gap-3 p-3 bg-base-200 rounded-lg {isCheckOnly ? 'border border-info' : ''}">
                    <input
                      id="server-check-only-{index}"
                      type="checkbox"
                      class="checkbox checkbox-info"
                      checked={isCheckOnly}
                      onchange={(e) => onServerCheckOnlyChange(index, (e.target as HTMLInputElement).checked)}
                    />
                    <label
                      for="server-check-only-{index}"
                      class="label-text cursor-pointer flex-1"
                    >
                      <span class="font-medium">{$t("settings.server.check_only")}</span>
                      <p class="text-sm opacity-70">
                        {isCheckOnly
                          ? $t("settings.server.check_only_enabled_description")
                          : $t("settings.server.check_only_description")
                        }
                      </p>
                    </label>
                    {#if isCheckOnly}
                      <div class="badge badge-info badge-sm">
                        {$t("settings.server.check_only_badge")}
                      </div>
                    {/if}
                  </div>
                </div>
              {:else if variant === "setup"}
                {@const isCheckOnly = localServers[index].check_only === true}
                <div class="md:col-span-2">
                  <div class="flex items-center gap-3 p-3 bg-base-200 rounded-lg {isCheckOnly ? 'border border-info' : ''}">
                    <input
                      id="server-check-only-{index}"
                      type="checkbox"
                      class="checkbox checkbox-info"
                      checked={isCheckOnly}
                      onchange={(e) => onServerCheckOnlyChange(index, (e.target as HTMLInputElement).checked)}
                    />
                    <label
                      for="server-check-only-{index}"
                      class="label-text cursor-pointer flex-1"
                    >
                      <span class="font-medium">{$t("settings.server.check_only")}</span>
                      <p class="text-sm opacity-70">
                        {$t("settings.server.check_only_description")}
                      </p>
                    </label>
                    {#if isCheckOnly}
                      <div class="badge badge-info badge-sm">
                        {$t("settings.server.check_only_badge")}
                      </div>
                    {/if}
                  </div>
                </div>
              {/if}

              <div>
                <label for="host-{index}" class="label">
                  <span class="label-text">
                    {$t("settings.server.host")} *
                  </span>
                </label>
                <input
                  id="host-{index}"
                  class="input input-bordered w-full"
                  bind:value={localServers[index].host}
                  placeholder="news.example.com"
                  required
                  oninput={() => onServerFieldChange(index)}
                />
              </div>

              <div>
                <label for="port-{index}" class="label">
                  <span class="label-text">
                    {$t("settings.server.port")} *
                  </span>
                </label>
                <input
                  id="port-{index}"
                  class="input input-bordered w-full"
                  type="number"
                  bind:value={localServers[index].port}
                  min="1"
                  max="65535"
                  required
                  oninput={() => onServerFieldChange(index)}
                />
              </div>

              <div class="flex items-center space-x-4 pt-2">
                <div class="flex items-center">
                  <input
                    type="checkbox"
                    class="checkbox mr-2"
                    bind:checked={localServers[index].ssl}
                    onchange={() => onServerFieldChange(index)}
                  />
                  <label for="ssl-{index}" class="label-text cursor-pointer">
                    {$t("settings.server.use_ssl_tls")}
                  </label>
                </div>

                {#if variant === "settings" && showAdvancedFields}
                  <div class="flex items-center">
                    <input
                      type="checkbox"
                      class="checkbox mr-2"
                      bind:checked={localServers[index].insecure_ssl}
                      onchange={() => onServerFieldChange(index)}
                    />
                    <label
                      for="insecure-ssl-{index}"
                      class="label-text cursor-pointer"
                    >
                      {$t("settings.server.allow_insecure_ssl")}
                    </label>
                  </div>
                {/if}
              </div>

              <div>
                <label for="username-{index}" class="label">
                  <span class="label-text">
                    {$t("settings.server.username")}
                  </span>
                </label>
                <input
                  id="username-{index}"
                  class="input input-bordered w-full"
                  bind:value={localServers[index].username}
                  oninput={() => onServerFieldChange(index)}
                />
              </div>

              <div>
                <label for="password-{index}" class="label">
                  <span class="label-text">
                    {$t("settings.server.password")}
                  </span>
                </label>
                <input
                  id="password-{index}"
                  class="input input-bordered w-full"
                  type="password"
                  bind:value={localServers[index].password}
                  oninput={() => onServerFieldChange(index)}
                />
              </div>

              <div>
                <label for="maxConnections-{index}" class="label">
                  <span class="label-text">
                    {$t("settings.server.max_connections")}
                  </span>
                </label>
                <input
                  id="maxConnections-{index}"
                  class="input input-bordered w-full"
                  type="number"
                  bind:value={localServers[index].max_connections}
                  min="1"
                  max="50"
                  oninput={() => onServerFieldChange(index)}
                />
              </div>

              {#if variant === "settings" && showAdvancedFields}
                <div>
                  <label for="inflight-{index}" class="label">
                    <span class="label-text">
                      {$t("settings.server.inflight")}
                    </span>
                  </label>
                  <input
                    id="inflight-{index}"
                    class="input input-bordered w-full"
                    type="number"
                    bind:value={localServers[index].inflight}
                    min="0"
                    max="100"
                    placeholder="0"
                    oninput={() => onServerFieldChange(index)}
                  />
                  <p class="text-sm text-base-content/70 mt-1">
                    {$t("settings.server.inflight_description")}
                  </p>
                </div>
              {/if}

              {#if variant === "settings" && showAdvancedFields}
                <div>
                  <label for="idle-time-{index}" class="label">
                    <span class="label-text"
                      >{$t("settings.server.connection_idle_timeout")}</span
                    >
                  </label>
                  <input
                    id="idle-time-{index}"
                    type="number"
                    class="input input-bordered w-full"
                    bind:value={
                      localServers[index].max_connection_idle_time_in_seconds
                    }
                    min="1"
                    max="3600"
                    placeholder="300"
                    onchange={() => onServerFieldChange(index)}
                  />
                  <p class="text-sm text-base-content/70 mt-1">
                    {$t("settings.server.connection_idle_timeout_description")}
                  </p>
                </div>

                <div>
                  <label for="ttl-{index}" class="label">
                    <span class="label-text"
                      >{$t("settings.server.connection_ttl")}</span
                    >
                  </label>
                  <input
                    id="ttl-{index}"
                    type="number"
                    class="input input-bordered w-full"
                    bind:value={
                      localServers[index].max_connection_ttl_in_seconds
                    }
                    min="1"
                    max="86400"
                    placeholder="3600"
                    onchange={() => onServerFieldChange(index)}
                  />
                  <p class="text-sm text-base-content/70 mt-1">
                    {$t("settings.server.connection_ttl_description")}
                  </p>
                </div>
              {/if}

              <!-- Proxy URL Field -->
              <div>
                <label for="proxy-url-{index}" class="label">
                  <span class="label-text">
                    {$t("settings.server.proxy.url_label")}
                  </span>
                </label>
                <input
                  id="proxy-url-{index}"
                  type="text"
                  class="input input-bordered w-full"
                  bind:value={localServers[index].proxy_url}
                  placeholder="socks5://user:pass@proxy.example.com:1080"
                  oninput={() => onServerFieldChange(index)}
                />
                <p class="text-sm text-base-content/70 mt-1">
                  {$t("settings.server.proxy.url_description")}
                </p>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>

    <button class="btn btn-outline w-full" onclick={addServer}>
      <CirclePlus class="w-4 h-4 mr-2" />
      {$t("settings.server.add_server")}
    </button>
  {/if}
</div>
