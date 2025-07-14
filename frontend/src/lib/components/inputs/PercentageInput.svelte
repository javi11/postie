<script lang="ts">
// Types
interface ComponentProps {
	value?: string;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
}

// Props
let {
	value = $bindable("10%"),
	label = "",
	description = "",
	placeholder = "10",
	presets = [],
	minValue = 1,
	maxValue = 100,
	id = "",
}: ComponentProps = $props();

// State - local state for form input (following best practices)
function parsePercentage(val: string | undefined): number {
	if (!val || typeof val !== "string" || val.trim() === "") {
		return 10;
	}
	return Number.parseInt(val.replace("%", "")) || 10;
}

// Initialize with current value
let percentageValue = $state(parsePercentage(value));
let lastUpdatedValue = $state(value);

// Effects - sync local state back to bound value when user changes input
$effect(() => {
	const newValue = `${percentageValue}%`;
	if (newValue !== lastUpdatedValue) {
		lastUpdatedValue = newValue;
		value = newValue;
	}
});

// Effect to update local state when value prop changes externally (avoid infinite loop)
$effect(() => {
	if (value !== lastUpdatedValue) {
		percentageValue = parsePercentage(value);
		lastUpdatedValue = value;
	}
});

// Functions
function setPreset(presetValue: number) {
	percentageValue = presetValue;
}
</script>

<!-- Percentage Input Component -->
<div class="form-control w-full">
	<!-- Label -->
	{#if label}
		<label class="label" for={id}>
			<span class="label-text font-medium text-base-content">{label}</span>
		</label>
	{/if}
	
	<!-- Input with Percentage Symbol -->
	<div class="flex gap-2">
		<div class="flex-1">
			<input
				{id}
				type="number"
				class="input input-bordered w-full focus:input-primary transition-colors"
				bind:value={percentageValue}
				min={minValue}
				max={maxValue}
				{placeholder}
			/>
		</div>
		<div class="w-12 flex items-center justify-center text-sm font-medium text-base-content/60">
			%
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