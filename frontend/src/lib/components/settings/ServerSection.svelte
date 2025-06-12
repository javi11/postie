<script lang="ts">
  import {
    Card,
    Heading,
    Input,
    Label,
    Checkbox,
    Button,
    ButtonGroup,
    P,
  } from "flowbite-svelte";
  import {
    ServerSolid,
    CirclePlusSolid,
    TrashBinSolid,
    ShieldCheckSolid,
  } from "flowbite-svelte-icons";
  import type { ConfigData, ServerConfig } from "$lib/types";

  export let config: ConfigData;

  function addServer() {
    const newServer: ServerConfig = {
      host: "",
      port: 119,
      username: "",
      password: "",
      ssl: false,
      max_connections: 10,
      max_connection_idle_time_in_seconds: 300,
      max_connection_ttl_in_seconds: 3600,
      insecure_ssl: false,
    };

    config.servers = [...config.servers, newServer];
  }

  function removeServer(index: number) {
    config.servers = config.servers.filter((_, i) => i !== index);
  }
</script>

<Card class="max-w-full shadow-sm p-5">
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <ServerSolid class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        <Heading
          tag="h2"
          class="text-lg font-semibold text-gray-900 dark:text-white"
        >
          NNTP Servers
        </Heading>
      </div>
      <Button
        size="sm"
        onclick={addServer}
        class="cursor-pointer flex items-center gap-2"
      >
        <CirclePlusSolid class="w-4 h-4" />
        Add Server
      </Button>
    </div>

    {#if config.servers.length === 0}
      <div
        class="text-center py-8 border-2 border-dashed border-gray-200 dark:border-gray-700 rounded-lg"
      >
        <ServerSolid class="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <P class="text-gray-600 dark:text-gray-400 mb-4">
          No servers configured. Add at least one NNTP server to start
          uploading.
        </P>
        <Button
          onclick={addServer}
          class="cursor-pointer flex items-center gap-2 mx-auto"
        >
          <CirclePlusSolid class="w-4 h-4" />
          Add Your First Server
        </Button>
      </div>
    {:else}
      <div class="space-y-6">
        {#each config.servers as server, index (index)}
          <div
            class="p-4 border border-gray-200 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800/50"
          >
            <div class="flex items-center justify-between mb-4">
              <Heading
                tag="h3"
                class="text-md font-medium text-gray-900 dark:text-white"
              >
                Server {index + 1}
              </Heading>
              <Button
                size="xs"
                color="red"
                variant="outline"
                onclick={() => removeServer(index)}
                class="cursor-pointer flex items-center gap-1"
              >
                <TrashBinSolid class="w-3 h-3" />
                Remove
              </Button>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Label for="host-{index}" class="mb-2">Host *</Label>
                <Input
                  id="host-{index}"
                  bind:value={server.host}
                  placeholder="news.example.com"
                  required
                />
              </div>

              <div>
                <Label for="port-{index}" class="mb-2">Port</Label>
                <Input
                  id="port-{index}"
                  type="number"
                  bind:value={server.port}
                  min="1"
                  max="65535"
                />
              </div>

              <div>
                <Label for="username-{index}" class="mb-2">Username</Label>
                <Input
                  id="username-{index}"
                  bind:value={server.username}
                  placeholder="your-username"
                  autocomplete="username"
                />
              </div>

              <div>
                <Label for="password-{index}" class="mb-2">Password</Label>
                <Input
                  id="password-{index}"
                  type="password"
                  bind:value={server.password}
                  placeholder="your-password"
                  autocomplete="current-password"
                />
              </div>

              <div>
                <Label for="max-connections-{index}" class="mb-2"
                  >Max Connections</Label
                >
                <Input
                  id="max-connections-{index}"
                  type="number"
                  bind:value={server.max_connections}
                  min="1"
                  max="50"
                />
              </div>

              <div>
                <Label for="idle-time-{index}" class="mb-2"
                  >Idle Time (seconds)</Label
                >
                <Input
                  id="idle-time-{index}"
                  type="number"
                  bind:value={server.max_connection_idle_time_in_seconds}
                  min="0"
                />
              </div>

              <div>
                <Label for="ttl-{index}" class="mb-2"
                  >Connection TTL (seconds)</Label
                >
                <Input
                  id="ttl-{index}"
                  type="number"
                  bind:value={server.max_connection_ttl_in_seconds}
                  min="0"
                />
              </div>
            </div>

            <div class="mt-4 space-y-3">
              <div class="flex items-center gap-3">
                <Checkbox bind:checked={server.ssl} />
                <div class="flex items-center gap-2">
                  <ShieldCheckSolid
                    class="w-4 h-4 text-green-600 dark:text-green-400"
                  />
                  <Label class="text-sm font-medium">Use SSL/TLS</Label>
                </div>
              </div>

              {#if server.ssl}
                <div class="ml-6">
                  <div class="flex items-center gap-3">
                    <Checkbox bind:checked={server.insecure_ssl} />
                    <Label class="text-sm"
                      >Allow insecure SSL (skip certificate validation)</Label
                    >
                  </div>
                </div>
              {/if}
            </div>

            {#if !server.host || !server.port}
              <div
                class="mt-3 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded"
              >
                <P class="text-sm text-yellow-800 dark:text-yellow-200">
                  ⚠️ Host and port are required for this server to work.
                </P>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
      <P class="text-sm text-gray-600 dark:text-gray-400">
        <strong>Tip:</strong> Configure multiple servers for better redundancy and
        upload speeds. The application will automatically distribute uploads across
        all configured servers.
      </P>
    </div>
  </div>
</Card>
