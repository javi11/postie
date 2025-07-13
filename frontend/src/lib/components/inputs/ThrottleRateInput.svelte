<script lang="ts">
interface ComponentProps {
	value?: number;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
	unitLabel?: string;
}

let {
	value = $bindable(0),
	label = "",
	description = "",
	placeholder = "0",
	presets = [],
	minValue = 0,
	maxValue = 1000,
	id = "",
	unitLabel = "MB/s",
}: ComponentProps = $props();

function setPreset(presetValue: number) {
	value = presetValue;
}
</script>

<div class="form-control w-full">
	<label class="label" for={id}>
		<span class="label-text">{label}</span>
	</label>
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
		<div class="w-16 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-400">
			{unitLabel}
		</div>
	</div>
	<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
		{description}
	</p>
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