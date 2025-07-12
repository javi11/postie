<script lang="ts">
import { createEventDispatcher } from "svelte";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { Button, Input, Label, Checkbox, Card, Badge, Spinner, toast } from "flowbite-svelte";
import { PlusOutline, TrashBinSolid, CheckOutline } from "flowbite-svelte-icons";
import apiClient from "$lib/api/client";

const dispatch = createEventDispatcher();

export let servers = [];

// Track validation state for each server
let validationStates = {};

function addServer() {
	const serverIndex = servers.length;
	servers = [...servers, {
		host: "",
		port: 563,
		username: "",
		password: "",
		ssl: true,
		maxConnections: 10,
		enabled: true
	}];
	validationStates = { ...validationStates, [serverIndex]: { status: "pending", error: "" } };
	updateServers();
}

function removeServer(index) {
	servers = servers.filter((_, i) => i !== index);
	// Rebuild validation states with new indices
	const newValidationStates = {};
	let newIndex = 0;
	for (let i = 0; i < servers.length + 1; i++) {
		if (i !== index && validationStates[i]) {
			newValidationStates[newIndex] = validationStates[i];
			newIndex++;
		}
	}
	validationStates = newValidationStates;
	updateServers();
}

function updateServers() {
	dispatch("update", { servers });
}

function onServerFieldChange(index) {
	// Clear validation state when server data changes
	if (validationStates[index] && validationStates[index].status !== "pending") {
		validationStates = { ...validationStates, [index]: { status: "pending", error: "" } };
	}
	updateServers();
}

function getServerValidationState(index) {
	const state = validationStates[index];
	if (!state) return { status: "pending", error: "" };
	return state;
}

function isServerComplete(server, index) {
	const validationState = getServerValidationState(index);
	return  validationState.status === "valid";
}

// Check if any servers are valid and emit validation state
function checkValidationState() {
	const hasValid = servers.some((server, index) => isServerComplete(server, index));
	dispatch("validationChange", { hasValidServers: hasValid });
	return hasValid;
}

async function validateServer(index) {
	const server = servers[index];
	
	// Basic validation first
	if (!server.host || !server.port) {
		validationStates = { ...validationStates, [index]: { status: "incomplete", error: "Host and port are required" } };
		return;
	}

	validationStates = { ...validationStates, [index]: { status: "validating", error: "" } };
	
	try {
		const result = await apiClient.validateNNTPServer({
			host: server.host,
			port: server.port,
			username: server.username,
			password: server.password,
			ssl: server.ssl,
			maxConnections: server.maxConnections
		});

		if (result.valid) {
			validationStates = { ...validationStates, [index]: { status: "valid", error: "" } };
			toastStore.success($t("setup.servers.valid"));
		} else {
			console.log("Setting server", index, "as invalid:", result.error);
			validationStates = { ...validationStates, [index]: { status: "invalid", error: result.error } };
			toastStore.error($t("setup.servers.invalid"), String(result.error));
		}
		
		// Emit validation state change
		checkValidationState();
	} catch (error) {
		validationStates = { ...validationStates, [index]: { status: "invalid", error: `Validation failed: ${error.message}` } };
		toastStore.error($t("setup.servers.invalid"), String(error));
		console.error("Server validation error:", error);
		
		// Emit validation state change
		checkValidationState();
	}
}

// Add default server if none exist
if (servers.length === 0) {
	addServer();
}
</script>

<div class="space-y-6">
	<div>
		<h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
			{$t("setup.servers.title")}
		</h3>
		<p class="text-gray-600 dark:text-gray-400 mb-4">
			{$t("setup.servers.description")}
		</p>
	</div>

	<div class="space-y-4">
		{#each servers as server, index}
			{@const validationState = getServerValidationState(index)}
			<Card class="p-4 max-w-full">
				<div class="flex justify-between items-start mb-4">
					<div class="flex items-center gap-2">
						<h4 class="font-medium text-gray-900 dark:text-white">
							{$t("setup.servers.server")} {index + 1}
						</h4>
						{#if validationState.status === "validating"}
							<Badge color="blue">
								<Spinner class="w-3 h-3 mr-1" />
								{$t("setup.servers.validating")}
							</Badge>
						{:else if validationState.status === "valid"}
							<Badge color="green">
								<CheckOutline class="w-3 h-3 mr-1" />
								{$t("setup.servers.valid")}
							</Badge>
						{:else if validationState.status === "invalid"}
							<Badge color="red">{$t("setup.servers.invalid")}</Badge>
						{:else}
							<Badge color="red">{$t("setup.servers.incomplete")}</Badge>
						{/if}
					</div>
					<div class="flex items-center gap-2">
						{#if validationState.status !== "validating"}
							<Button
								size="sm"
								color="primary"
								outline
								onclick={() => validateServer(index)}
							>
								{$t("setup.servers.testConnection")}
							</Button>
						{/if}
						{#if servers.length > 1}
							<Button
								size="sm"
								color="red"
								outline
								onclick={() => removeServer(index)}
							>
								<TrashBinSolid class="w-4 h-4" />
							</Button>
						{/if}
					</div>
				</div>
				
				{#if validationState.error}
					<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
						<p class="text-sm text-red-600 dark:text-red-400">
							{validationState.error}
						</p>
					</div>
				{/if}

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<Label for="host-{index}" class="mb-2">
							{$t("setup.servers.host")} *
						</Label>
						<Input
							id="host-{index}"
							bind:value={server.host}
							placeholder="news.example.com"
							required
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<Label for="port-{index}" class="mb-2">
							{$t("setup.servers.port")} *
						</Label>
						<Input
							id="port-{index}"
							type="number"
							bind:value={server.port}
							min="1"
							max="65535"
							required
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<Label for="username-{index}" class="mb-2">
							{$t("setup.servers.username")}
						</Label>
						<Input
							id="username-{index}"
							bind:value={server.username}
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<Label for="password-{index}" class="mb-2">
							{$t("setup.servers.password")}
						</Label>
						<Input
							id="password-{index}"
							type="password"
							bind:value={server.password}
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<Label for="maxConnections-{index}" class="mb-2">
							{$t("setup.servers.maxConnections")}
						</Label>
						<Input
							id="maxConnections-{index}"
							type="number"
							bind:value={server.maxConnections}
							min="1"
							max="50"
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div class="flex items-center pt-6">
						<Checkbox
							bind:checked={server.ssl}
							class="mr-2"
							onchange={() => onServerFieldChange(index)}
						/>
						<Label for="ssl-{index}">
							{$t("setup.servers.ssl")}
						</Label>
					</div>
				</div>
			</Card>
		{/each}
	</div>

	<Button
		color="alternative"
		outline
		onclick={addServer}
		class="w-full"
	>
		<PlusOutline class="w-4 h-4 mr-2" />
		{$t("setup.servers.addServer")}
	</Button>

	<div class="text-sm text-gray-500 dark:text-gray-400">
		<p>* {$t("setup.servers.requiredFields")}</p>
	</div>
</div>