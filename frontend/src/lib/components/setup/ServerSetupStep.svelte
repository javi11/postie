<script lang="ts">
import NntpServerManager from "$lib/components/NntpServerManager.svelte";
import { t } from "$lib/i18n";
import { backend } from "$lib/wailsjs/go/models";
import { Upload, ShieldCheck } from "lucide-svelte";

interface Props {
	servers?: backend.ServerData[];
	onupdate?: (data: { servers: backend.ServerData[] }) => void;
	onvalidationchange: (data: { hasValidServers: boolean }) => void;
}

let { servers = $bindable([]), onupdate, onvalidationchange }: Props = $props();

// Inject a default upload server synchronously so NntpServerManager
// initializes with it immediately on first render
if (servers.length === 0) {
	const sd = new backend.ServerData();
	sd.host = "";
	sd.port = 119;
	sd.ssl = false;
	sd.username = "";
	sd.password = "";
	sd.maxConnections = 10;
	sd.role = "upload";
	servers = [sd];
	onupdate?.({ servers });
}

function toNntpFormat(s: backend.ServerData) {
	return {
		host: s.host,
		port: s.port,
		ssl: s.ssl || false,
		username: s.username || "",
		password: s.password || "",
		max_connections: s.maxConnections || 10,
		inflight: 10,
		insecure_ssl: false,
		max_connection_idle_time_in_seconds: 300,
		max_connection_ttl_in_seconds: 3600,
		role: s.role || "upload",
	};
}

function toServerData(s: any, role: string): backend.ServerData {
	const sd = new backend.ServerData();
	sd.host = s.host || "";
	sd.port = s.port || 119;
	sd.ssl = s.ssl || false;
	sd.username = s.username || "";
	sd.password = s.password || "";
	sd.maxConnections = s.max_connections || 10;
	sd.role = role;
	return sd;
}

let uploadManagedServers = $derived(
	servers.filter(s => (s.role || "upload") !== "verify").map(s => toNntpFormat(s))
);

let verifyManagedServers = $derived(
	servers.filter(s => s.role === "verify").map(s => toNntpFormat(s))
);

function handleUploadUpdate(updated: any[]) {
	servers = [
		...updated.map(s => toServerData(s, "upload")),
		...servers.filter(s => s.role === "verify").map(s => toServerData(s, "verify")),
	];
	onupdate?.({ servers });
}

function handleVerifyUpdate(updated: any[]) {
	servers = [
		...servers.filter(s => (s.role || "upload") !== "verify").map(s => toServerData(s, "upload")),
		...updated.map(s => toServerData(s, "verify")),
	];
	onupdate?.({ servers });
}

// Only upload pool validation gates the "Next" button
function handleUploadValidation(data: { hasValidServers: boolean }) {
	onvalidationchange(data);
}
</script>

<div class="space-y-6">
  <!-- Upload Pool -->
  <div class="card bg-base-100 shadow-sm">
    <div class="card-body space-y-4">
      <div class="flex items-center gap-3">
        <Upload class="w-5 h-5 text-primary" />
        <div>
          <h2 class="text-lg font-semibold text-base-content">
            {$t("settings.server.upload_pool_title")}
          </h2>
          <p class="text-sm text-base-content/60">
            {$t("settings.server.upload_pool_description")}
          </p>
        </div>
      </div>

      <NntpServerManager
        servers={uploadManagedServers}
        onupdate={handleUploadUpdate}
        onvalidationchange={handleUploadValidation}
        variant="settings"
        restrictedRole="upload"
      />
    </div>
  </div>

  <!-- Verify Pool -->
  <div class="card bg-base-100 shadow-sm">
    <div class="card-body space-y-4">
      <div class="flex items-center gap-3">
        <ShieldCheck class="w-5 h-5 text-secondary" />
        <div>
          <h2 class="text-lg font-semibold text-base-content">
            {$t("settings.server.verify_pool_title")}
            <span class="badge badge-ghost badge-sm ml-2">
              {$t("settings.server.verify_pool_optional")}
            </span>
          </h2>
          <p class="text-sm text-base-content/60">
            {$t("settings.server.verify_pool_description")}
          </p>
        </div>
      </div>

      <NntpServerManager
        servers={verifyManagedServers}
        onupdate={handleVerifyUpdate}
        variant="settings"
        restrictedRole="verify"
      />
    </div>
  </div>
</div>
