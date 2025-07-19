<script lang="ts">
// Types
interface ComponentProps {
	value?: number;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number; unit: string }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
	showBytes?: boolean;
	onchange?: (value: number) => void;
}

// Props
let {
	value = $bindable(100),
	label = "",
	description = "",
	placeholder = "100",
	presets = [],
	minValue = 1,
	maxValue = undefined,
	id = "",
	showBytes = false,
	onchange,
}: ComponentProps = $props();

// State - local state for form inputs (following best practices)
function parseSize(val: number | undefined): { value: number; unit: string } {
	if (!val || typeof val !== "number" || Number.isNaN(val)) {
		return { value: 100, unit: "MB" };
	}
	
	if (val >= 1073741824 && val % 1073741824 === 0) {
		return { value: val / 1073741824, unit: "GB" };
	}

	if (val >= 1048576 && val % 1048576 === 0) {
		return { value: val / 1048576, unit: "MB" };
	}

	return { value: val / 1024, unit: "KB" };	;	
}

// Initialize with current value
const initial = parseSize(value);
let sizeValue = $state(initial.value);
let sizeUnit = $state(initial.unit);
let lastUpdatedValue = $state(value);

// Effects - sync local state back to bound value when user changes inputs
$effect(() => {
	const newValue = unitToBytes(sizeValue, sizeUnit);
	if (newValue !== lastUpdatedValue) {
		lastUpdatedValue = newValue;
		value = newValue;
		onchange?.(newValue);
	}
});

// Effect to update local state when value prop changes externally (avoid infinite loop)
$effect(() => {
	if (value !== lastUpdatedValue) {
		const parsed = parseSize(value);
		sizeValue = parsed.value;
		sizeUnit = parsed.unit;
		lastUpdatedValue = value;
	}
});

// Derived state - reactive calculations
const sizeUnitOptions = $derived([
	{ value: "MB", name: "MB" },
	{ value: "GB", name: "GB" },
	{ value: "KB", name: "KB" },
]);

const dynamicMaxValue = $derived(
	maxValue
		? sizeUnit === "GB"
			? Math.ceil(maxValue / 1024)
			: maxValue
		: undefined,
);

const byteDisplay = $derived(
	showBytes ? `(${value.toLocaleString()} bytes)` : "",
);

function unitToBytes(val: number, unit: string): number {
	if (unit === "GB") {
		return val * 1024 * 1024 * 1024;
	}
	if (unit === "MB") {
		return val * 1024 * 1024;
	}
	if (unit === "KB") {
		return val * 1024;
	}
	return val;
}

function setPreset(presetValue: number, presetUnit: string) {
	sizeValue = presetValue;
	sizeUnit = presetUnit;
}
</script>

<!-- Size Input Component -->
<div class="form-control w-full">
	<!-- Label -->
	{#if label}
		<label class="label" for={id}>
			<span class="label-text font-medium text-base-content">{label}</span>
		</label>
	{/if}
	
	<!-- Input with Unit Selector -->
	<div class="flex gap-2">
		<div class="flex-1">
			<input
				{id}
				type="number"
				class="input input-bordered w-full focus:input-primary transition-colors"
				bind:value={sizeValue}
				min={minValue}
				max={dynamicMaxValue || undefined}
				{placeholder}
			/>
		</div>
		<div class="w-20">
			<select
				class="select select-bordered focus:select-primary transition-colors"
				bind:value={sizeUnit}
			>
				{#each sizeUnitOptions as option}
					<option value={option.value}>{option.name}</option>
				{/each}
			</select>
		</div>
	</div>
	
	<!-- Description with Bytes Display -->
	{#if description || byteDisplay}
		<p class="text-sm text-base-content/70 mt-1">
			{description} {byteDisplay}
		</p>
	{/if}
	
	<!-- Preset Buttons -->
	{#if presets.length > 0}
		<div class="mt-2 flex flex-wrap gap-2">
			{#each presets as preset}
				<button
					type="button"
					class="btn btn-xs btn-outline hover:btn-primary transition-colors"
					onclick={() => setPreset(preset.value, preset.unit)}
				>
					{preset.value}{preset.unit}
				</button>
			{/each}
		</div>
	{/if}
</div> 