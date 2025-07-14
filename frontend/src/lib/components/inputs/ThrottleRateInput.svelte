<script lang="ts">
// Types
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
	onchange?: (value: number) => void;
}

// Props
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
	onchange,
}: ComponentProps = $props();

// Effects - trigger callback when value changes
$effect(() => {
	onchange?.(value);
});

// Functions
function setPreset(presetValue: number) {
	value = presetValue;
}
</script>

<!-- Throttle Rate Input Component -->
<div class="form-control w-full">
	<!-- Label -->
	{#if label}
		<label class="label" for={id}>
			<span class="label-text font-medium text-base-content">{label}</span>
		</label>
	{/if}
	
	<!-- Input with Unit Display -->
	<div class="flex gap-2">
		<div class="flex-1">
			<input
				{id}
				type="number"
				class="input input-bordered w-full focus:input-primary transition-colors"
				bind:value
				min={minValue}
				max={maxValue}
				{placeholder}
			/>
		</div>
		<div class="w-16 flex items-center justify-center text-sm font-medium text-base-content/60">
			{unitLabel}
		</div>
	</div>
	
	<!-- Description -->
	{#if description}
		<p class="text-sm text-base-content/70 mt-1">
			{description}
		</p>
	{/if}
	
	<!-- Preset Buttons -->
	{#if presets.length > 0}
		<div class="mt-2 flex flex-wrap gap-2">
			{#each presets as preset}
				<button
					type="button"
					class="btn btn-xs btn-outline hover:btn-primary transition-colors"
					onclick={() => setPreset(preset.value)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 