<script lang="ts">
	import { t } from '$lib/i18n';
	import apiClient, { type ArrInstance } from '$lib/api/client';
	import { toastStore } from '$lib/stores/toast';

	let instances = $state<ArrInstance[]>([]);
	let showAddForm = $state(false);
	let saving = $state(false);
	let testingId = $state<string | null>(null);

	function blankInstance(): ArrInstance {
		return {
			id: '',
			name: '',
			type: 'radarr',
			url: '',
			api_key: '',
			enabled: true,
			webhook_id: 0,
			delete_after_upload: false
		};
	}

	let draft = $state<ArrInstance>(blankInstance());

	async function load() {
		try {
			instances = await apiClient.getArrInstances();
		} catch (e) {
			toastStore.error($t('arr.title'), String(e));
		}
	}

	$effect(() => {
		load();
	});

	async function testConnection(instance: ArrInstance) {
		testingId = instance.id || 'draft';
		try {
			await apiClient.testArrConnection(instance);
			toastStore.success($t('arr.connection_ok'));
		} catch (e) {
			toastStore.error($t('arr.error_test'), String(e));
		} finally {
			testingId = null;
		}
	}

	async function setupWebhook() {
		saving = true;
		try {
			const saved = await apiClient.addArrInstance(draft);
			instances = [...instances, saved];
			draft = blankInstance();
			showAddForm = false;
			toastStore.success($t('arr.webhook_created'));
		} catch (e) {
			toastStore.error($t('arr.error_setup'), String(e));
		} finally {
			saving = false;
		}
	}

	async function removeInstance(instance: ArrInstance) {
		try {
			await apiClient.removeArrInstance(instance.id);
			instances = instances.filter((i) => i.id !== instance.id);
			toastStore.success($t('arr.webhook_removed'));
		} catch (e) {
			toastStore.error($t('arr.error_remove'), String(e));
		}
	}

	const ARR_TYPES: { value: ArrInstance['type']; label: string; port: number }[] = [
		{ value: 'radarr', label: 'Radarr', port: 7878 },
		{ value: 'sonarr', label: 'Sonarr', port: 8989 },
		{ value: 'lidarr', label: 'Lidarr', port: 8686 },
		{ value: 'readarr', label: 'Readarr', port: 8787 }
	];

	function applyTypeDefaults() {
		const found = ARR_TYPES.find((t) => t.value === draft.type);
		if (found && !draft.url) {
			draft.url = `http://localhost:${found.port}`;
		}
	}
</script>

<div class="card bg-base-100 shadow-sm">
	<div class="card-body space-y-4">
		<div class="flex items-center justify-between">
			<div>
				<h3 class="card-title">{$t('arr.title')}</h3>
				<p class="text-sm text-base-content/70">{$t('arr.description')}</p>
			</div>
			{#if !showAddForm}
				<button class="btn btn-primary btn-sm" onclick={() => (showAddForm = true)}>
					+ {$t('arr.add_instance')}
				</button>
			{/if}
		</div>

		{#if instances.length === 0 && !showAddForm}
			<p class="text-sm text-base-content/50 italic">{$t('arr.no_instances')}</p>
		{/if}

		{#each instances as instance (instance.id)}
			<div class="flex items-center justify-between rounded-lg border border-base-300 px-4 py-3">
				<div>
					<span class="font-medium">{instance.name}</span>
					<span class="badge badge-ghost badge-sm ml-2">{instance.type}</span>
					<p class="text-xs text-base-content/60 mt-0.5">{instance.url}</p>
				</div>
				<div class="flex gap-2">
					<button
						class="btn btn-ghost btn-xs"
						disabled={testingId === instance.id}
						onclick={() => testConnection(instance)}
					>
						{testingId === instance.id ? $t('arr.testing') : $t('arr.test_connection')}
					</button>
					<button
						class="btn btn-error btn-xs btn-outline"
						onclick={() => removeInstance(instance)}
					>
						{$t('arr.remove_instance')}
					</button>
				</div>
			</div>
		{/each}

		{#if showAddForm}
			<div class="rounded-lg border border-primary/30 bg-base-200 p-4 space-y-4">
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="label" for="arr-name">
							<span class="label-text">{$t('arr.instance_name')}</span>
						</label>
						<input
							id="arr-name"
							type="text"
							class="input input-bordered w-full"
							placeholder={$t('arr.instance_name_placeholder')}
							bind:value={draft.name}
						/>
					</div>

					<div>
						<label class="label" for="arr-type">
							<span class="label-text">{$t('arr.type')}</span>
						</label>
						<select
							id="arr-type"
							class="select select-bordered w-full"
							bind:value={draft.type}
							onchange={applyTypeDefaults}
						>
							{#each ARR_TYPES as arrType}
								<option value={arrType.value}>{arrType.label}</option>
							{/each}
						</select>
					</div>

					<div>
						<label class="label" for="arr-url">
							<span class="label-text">{$t('arr.url')}</span>
						</label>
						<input
							id="arr-url"
							type="url"
							class="input input-bordered w-full"
							placeholder={$t('arr.url_placeholder')}
							bind:value={draft.url}
						/>
					</div>

					<div>
						<label class="label" for="arr-apikey">
							<span class="label-text">{$t('arr.api_key')}</span>
						</label>
						<input
							id="arr-apikey"
							type="password"
							class="input input-bordered w-full"
							placeholder={$t('arr.api_key_placeholder')}
							bind:value={draft.api_key}
						/>
					</div>
				</div>

				<label class="flex items-center gap-2 cursor-pointer">
					<input type="checkbox" class="checkbox checkbox-sm" bind:checked={draft.delete_after_upload} />
					<span class="text-sm">{$t('arr.delete_after_upload')}</span>
				</label>

				<div class="flex gap-2 pt-2">
					<button
						class="btn btn-ghost btn-sm"
						disabled={testingId === 'draft'}
						onclick={() => testConnection(draft)}
					>
						{testingId === 'draft' ? $t('arr.testing') : $t('arr.test_connection')}
					</button>
					<button
						class="btn btn-primary btn-sm"
						disabled={saving || !draft.name || !draft.url || !draft.api_key}
						onclick={setupWebhook}
					>
						{saving ? $t('arr.setting_up') : $t('arr.save_instance')}
					</button>
					<button
						class="btn btn-ghost btn-sm"
						onclick={() => {
							showAddForm = false;
							draft = blankInstance();
						}}
					>
						Cancel
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>
