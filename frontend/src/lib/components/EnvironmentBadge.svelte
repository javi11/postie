<script lang="ts">
import apiClient from "$lib/api/client";
import { Badge } from "flowbite-svelte";
import { onMount } from "svelte";

let environment: "wails" | "web" | "unknown" = "unknown";
let isReady = false;

onMount(async () => {
	await apiClient.initialize();
	environment = apiClient.environment;
	isReady = apiClient.isReady;
});

function getEnvironmentColor() {
	switch (environment) {
		case "wails":
			return "blue";
		case "web":
			return "green";
		default:
			return "gray";
	}
}

function getEnvironmentText() {
	switch (environment) {
		case "wails":
			return "🖥️ Desktop";
		case "web":
			return "🌐 Web";
		default:
			return "❓ Unknown";
	}
}
</script>

{#if isReady}
	<Badge color={getEnvironmentColor()} class="text-xs">
		{getEnvironmentText()}
	</Badge>
{:else}
	<Badge color="gray" class="text-xs">
		⏳ Loading...
	</Badge>
{/if} 