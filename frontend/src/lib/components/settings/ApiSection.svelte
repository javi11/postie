<script lang="ts">
import apiClient from "$lib/api/client";
import { t } from "$lib/i18n";
import { toastStore } from "$lib/stores/toast";
import type { config as configType } from "$lib/wailsjs/go/models";
import { Copy, Eye, EyeOff, KeyRound, RefreshCw } from "lucide-svelte";
import { onMount } from "svelte";

interface Props {
	config: configType.ConfigData;
}

let { config = $bindable() }: Props = $props();

let enabled = $state(config.api?.enabled ?? false);
let apiKey = $state("");
let revealed = $state(false);
let loading = $state(false);
let regenerating = $state(false);
let confirmRegenerate = $state(false);

const displayedKey = $derived(
	apiKey ? (revealed ? apiKey : "•".repeat(Math.min(apiKey.length, 32))) : ""
);

const curlExample = $derived(
	apiKey
		? `curl -X POST -H "X-API-Key: ${apiKey}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"file":"/mnt/local/Media/Movie/Movie 1/movie.mkv","relative_path":"/mnt/local","delete_after_upload":false}' \\\n  http://localhost:8080/api/v1/queue/upload`
		: ""
);

$effect(() => {
	if (!config.api) {
		config.api = { enabled: false };
	}
	config.api.enabled = enabled;
});

onMount(async () => {
	if (!enabled) return;
	await loadKey();
});

async function loadKey() {
	loading = true;
	try {
		apiKey = await apiClient.getApiKey();
	} catch (error) {
		console.error("Failed to load API key:", error);
		toastStore.error($t("settings.api.load_failed"), String(error));
	} finally {
		loading = false;
	}
}

async function regenerate() {
	regenerating = true;
	try {
		apiKey = await apiClient.regenerateApiKey();
		toastStore.success(
			$t("settings.api.regenerated_success"),
			$t("settings.api.regenerated_success_description")
		);
	} catch (error) {
		console.error("Failed to regenerate API key:", error);
		toastStore.error($t("settings.api.regenerate_failed"), String(error));
	} finally {
		regenerating = false;
		confirmRegenerate = false;
	}
}

async function copyKey() {
	if (!apiKey) return;
	try {
		await navigator.clipboard.writeText(apiKey);
		toastStore.success($t("settings.api.copied"), "");
	} catch (error) {
		console.error("Failed to copy:", error);
	}
}

$effect(() => {
	if (enabled && !apiKey && !loading) {
		void loadKey();
	}
});
</script>

<div class="card bg-base-100 shadow-xl">
	<div class="card-body space-y-6">
		<div class="flex items-center gap-3">
			<KeyRound class="w-5 h-5 text-base-content/60" />
			<h2 class="card-title text-lg">{$t("settings.api.title")}</h2>
		</div>

		<div class="form-control">
			<label class="label cursor-pointer justify-start gap-3">
				<input type="checkbox" class="toggle" bind:checked={enabled} />
				<span class="label-text">{$t("settings.api.enabled")}</span>
			</label>
			<div class="label">
				<span class="label-text-alt ml-8">
					{$t("settings.api.enabled_description")}
				</span>
			</div>
		</div>

		{#if enabled}
			<div class="space-y-4 pl-4 border-l-2 border-blue-200 dark:border-blue-700">
				<div class="form-control">
					<label class="label" for="api-key">
						<span class="label-text">{$t("settings.api.key_label")}</span>
					</label>
					<div class="join w-full">
						<input
							id="api-key"
							type="text"
							readonly
							class="input input-bordered join-item flex-1 font-mono text-sm"
							value={displayedKey}
							placeholder={loading ? "…" : ""}
						/>
						<button
							type="button"
							class="btn join-item"
							onclick={() => (revealed = !revealed)}
							title={revealed ? $t("settings.api.hide") : $t("settings.api.reveal")}
							disabled={!apiKey}
						>
							{#if revealed}
								<EyeOff class="w-4 h-4" />
							{:else}
								<Eye class="w-4 h-4" />
							{/if}
						</button>
						<button
							type="button"
							class="btn join-item"
							onclick={copyKey}
							title={$t("settings.api.copy")}
							disabled={!apiKey}
						>
							<Copy class="w-4 h-4" />
						</button>
						<button
							type="button"
							class="btn btn-warning join-item"
							onclick={() => (confirmRegenerate = true)}
							disabled={regenerating}
							title={$t("settings.api.regenerate")}
						>
							<RefreshCw class="w-4 h-4 {regenerating ? 'animate-spin' : ''}" />
						</button>
					</div>
					<div class="label">
						<span class="label-text-alt">{$t("settings.api.key_description")}</span>
					</div>
				</div>

				{#if apiKey}
					<div class="form-control">
						<label class="label" for="api-curl">
							<span class="label-text">{$t("settings.api.curl_example_label")}</span>
						</label>
						<textarea
							id="api-curl"
							readonly
							class="textarea textarea-bordered font-mono text-xs h-32"
							value={curlExample}
						></textarea>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

{#if confirmRegenerate}
	<div class="modal modal-open">
		<div class="modal-box">
			<h3 class="font-bold text-lg mb-4">{$t("settings.api.regenerate")}</h3>
			<p class="py-4">{$t("settings.api.regenerate_confirm")}</p>
			<div class="modal-action">
				<button
					type="button"
					class="btn btn-ghost"
					onclick={() => (confirmRegenerate = false)}
					disabled={regenerating}
				>
					{$t("common.common.cancel")}
				</button>
				<button
					type="button"
					class="btn btn-warning"
					onclick={regenerate}
					disabled={regenerating}
				>
					<RefreshCw class="w-4 h-4 {regenerating ? 'animate-spin' : ''}" />
					{regenerating ? $t("common.common.saving") : $t("settings.api.regenerate")}
				</button>
			</div>
		</div>
	</div>
{/if}
