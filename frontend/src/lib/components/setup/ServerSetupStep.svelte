<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import { Check, Loader2, Plus, Trash2 } from "lucide-svelte";
import { createEventDispatcher } from "svelte";

const dispatch = createEventDispatcher();

export let servers = [];

// Track validation state for each server
let validationStates = {};

function addServer() {
	const serverIndex = servers.length;
	servers = [
		...servers,
		{
			host: "",
			port: 563,
			username: "",
			password: "",
			ssl: true,
			maxConnections: 10,
			enabled: true,
		},
	];
	validationStates = {
		...validationStates,
		[serverIndex]: { status: "pending", error: "" },
	};
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
		validationStates = {
			...validationStates,
			[index]: { status: "pending", error: "" },
		};
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
	return validationState.status === "valid";
}

// Check if any servers are valid and emit validation state
function checkValidationState() {
	const hasValid = servers.some((server, index) =>
		isServerComplete(server, index),
	);
	dispatch("validationChange", { hasValidServers: hasValid });
	return hasValid;
}

async function validateServer(index) {
	const server = servers[index];

	// Basic validation first
	if (!server.host || !server.port) {
		validationStates = {
			...validationStates,
			[index]: { status: "incomplete", error: "Host and port are required" },
		};
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
			username: server.username,
			password: server.password,
			ssl: server.ssl,
			maxConnections: server.maxConnections,
		});

		if (result.valid) {
			validationStates = {
				...validationStates,
				[index]: { status: "valid", error: "" },
			};
			toastStore.success($t("setup.servers.valid"));
		} else {
			console.log("Setting server", index, "as invalid:", result.error);
			validationStates = {
				...validationStates,
				[index]: { status: "invalid", error: result.error },
			};
			toastStore.error($t("setup.servers.invalid"), String(result.error));
		}

		// Emit validation state change
		checkValidationState();
	} catch (error) {
		validationStates = {
			...validationStates,
			[index]: {
				status: "invalid",
				error: `Validation failed: ${error.message}`,
			},
		};
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
		<h3 class="text-lg font-semibold text-base-content mb-2">
			{$t("setup.servers.title")}
		</h3>
		<p class="text-base-content/70 mb-4">
			{$t("setup.servers.description")}
		</p>
	</div>

	<div class="space-y-4">
		{#each servers as server, index}
			{@const validationState = getServerValidationState(index)}
			<div class="card bg-base-100 shadow-lg">
				<div class="card-body p-4">
					<div class="flex justify-between items-start mb-4">
						<div class="flex items-center gap-2">
							<h4 class="font-medium text-base-content">
								{$t("setup.servers.server")} {index + 1}
							</h4>
							{#if validationState.status === "validating"}
								<div class="badge badge-primary gap-1">
									<Loader2 class="w-3 h-3 animate-spin" />
									{$t("setup.servers.validating")}
								</div>
							{:else if validationState.status === "valid"}
								<div class="badge badge-success gap-1">
									<Check class="w-3 h-3" />
									{$t("setup.servers.valid")}
								</div>
							{:else if validationState.status === "invalid"}
								<div class="badge badge-error">{$t("setup.servers.invalid")}</div>
							{:else}
								<div class="badge badge-error">{$t("setup.servers.incomplete")}</div>
							{/if}
						</div>
						<div class="flex items-center gap-2">
							{#if validationState.status !== "validating"}
								<button
									class="btn btn-primary btn-outline btn-sm"
									onclick={() => validateServer(index)}
								>
									{$t("setup.servers.testConnection")}
								</button>
							{/if}
							{#if servers.length > 1}
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
					<div>
						<label for="host-{index}" class="label">
							<span class="label-text">{$t("setup.servers.host")} *</span>
						</label>
						<input
							id="host-{index}"
							class="input input-bordered w-full"
							bind:value={server.host}
							placeholder="news.example.com"
							required
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<label for="port-{index}" class="label">
							<span class="label-text">{$t("setup.servers.port")} *</span>
						</label>
						<input
							id="port-{index}"
							class="input input-bordered w-full"
							type="number"
							bind:value={server.port}
							min="1"
							max="65535"
							required
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<label for="username-{index}" class="label">
							<span class="label-text">{$t("setup.servers.username")}</span>
						</label>
						<input
							id="username-{index}"
							class="input input-bordered w-full"
							bind:value={server.username}
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<label for="password-{index}" class="label">
							<span class="label-text">{$t("setup.servers.password")}</span>
						</label>
						<input
							id="password-{index}"
							class="input input-bordered w-full"
							type="password"
							bind:value={server.password}
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div>
						<label for="maxConnections-{index}" class="label">
							<span class="label-text">{$t("setup.servers.maxConnections")}</span>
						</label>
						<input
							id="maxConnections-{index}"
							class="input input-bordered w-full"
							type="number"
							bind:value={server.maxConnections}
							min="1"
							max="50"
							oninput={() => onServerFieldChange(index)}
						/>
					</div>

					<div class="flex items-center pt-6">
						<input
							type="checkbox"
							class="checkbox mr-2"
							bind:checked={server.ssl}
							onchange={() => onServerFieldChange(index)}
						/>
						<label for="ssl-{index}" class="label-text cursor-pointer">
							{$t("setup.servers.ssl")}
						</label>
					</div>
				</div>
				</div>
			</div>
		{/each}
	</div>

	<button
		class="btn btn-outline w-full"
		onclick={addServer}
	>
		<Plus class="w-4 h-4 mr-2" />
		{$t("setup.servers.addServer")}
	</button>

	<div class="text-sm text-base-content/70">
		<p>* {$t("setup.servers.requiredFields")}</p>
	</div>
</div>