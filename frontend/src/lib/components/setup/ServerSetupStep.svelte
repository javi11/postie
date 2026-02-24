<script lang="ts">
import NntpServerManager from "$lib/components/NntpServerManager.svelte";
import { backend } from "$lib/wailsjs/go/models";

interface Props {
	servers?: backend.ServerData[];
	onupdate?: (data: { servers: backend.ServerData[] }) => void;
	onvalidationchange: (data: { hasValidServers: boolean }) => void;
}

let { servers = $bindable([]), onupdate, onvalidationchange }: Props = $props();

function handleServerUpdate(updatedServers: any[]) {
	// Convert from our generic server format to backend.ServerData
	servers = updatedServers.map(server => {
		const serverData = new backend.ServerData();
		serverData.host = server.host;
		serverData.port = server.port;
		serverData.username = server.username || "";
		serverData.password = server.password || "";
		serverData.maxConnections = server.max_connections || 10;
		serverData.ssl = server.ssl || false;
		return serverData;
	});
	
	onupdate?.({ servers });
}

// Convert servers to the format expected by NntpServerManager
let managedServers = $derived(servers.map(server => ({
	host: server.host,
	port: server.port,
	username: server.username,
	password: server.password,
	max_connections: server.maxConnections || 10,
	ssl: server.ssl,
	max_connection_idle_time_in_seconds: 300,
	max_connection_ttl_in_seconds: 3600,
	insecure_ssl: false,
	inflight: 10,
})));
</script>

<NntpServerManager
	servers={managedServers}
	onupdate={handleServerUpdate}
	onvalidationchange={onvalidationchange}
	variant="setup"
/>