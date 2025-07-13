<script lang="ts">
import { t } from "$lib/i18n";

interface ComponentProps {
	value?: number;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
}

let {
	value = $bindable(750000),
	label = "",
	description = "",
	placeholder = "750000",
	presets = [],
	minValue = 1000,
	maxValue = 10000000,
	id = "",
}: ComponentProps = $props();

const unitLabel = $derived($t("common.inputs.bytes"));

function setPreset(presetValue: number) {
	value = presetValue;
}
</script>

<div class="form-control w-full">
	{#if label}
		<label class="label" for={id}>
			<span class="label-text">{label}</span>
		</label>
	{/if}
	<div class="flex gap-2">
		<div class="flex-1">
			<input
				{id}
				type="number"
				class="input input-bordered w-full"
				bind:value
				min={minValue}
				max={maxValue}
				{placeholder}
			/>
		</div>
		<div class="w-20 flex items-center justify-center text-sm font-medium opacity-60">
			{unitLabel}
		</div>
	</div>
	{#if description}
		<div class="label">
			<span class="label-text-alt opacity-70">{description}</span>
		</div>
	{/if}
	{#if presets.length > 0}
		<div class="mt-2 flex flex-wrap gap-2">
			{#each presets as preset}
				<button
					type="button"
					class="btn btn-xs btn-outline"
					onclick={() => setPreset(preset.value)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 