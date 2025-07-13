<script lang="ts">
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

const sizeUnitOptions = [
	{ value: "MB", name: "MB" },
	{ value: "GB", name: "GB" },
];

let sizeValue = $state(100);
let sizeUnit = $state("MB");

// Helper functions for size conversion
function bytesToUnit(bytes: number, unit: string): number {
	switch (unit) {
		case "GB":
			return Math.round((bytes / 1024 / 1024 / 1024) * 100) / 100;
		case "MB":
			return Math.round(bytes / 1024 / 1024);
		default:
			return bytes;
	}
}

function unitToBytes(val: number, unit: string): number {
	switch (unit) {
		case "GB":
			return val * 1024 * 1024 * 1024;
		case "MB":
			return val * 1024 * 1024;
		default:
			return val;
	}
}

// Update maxValue based on unit
const dynamicMaxValue = $derived(
	maxValue
		? sizeUnit === "GB"
			? Math.ceil(maxValue / 1024)
			: maxValue
		: undefined,
);

function setPreset(presetValue: number, presetUnit: string) {
	sizeValue = presetValue;
	sizeUnit = presetUnit;
	updateValue();
}

// Parse the bound value and sync with local state
function parseAndSync() {
	if (!value || typeof value !== "number") {
		sizeValue = 100;
		sizeUnit = "MB";
		return;
	}

	if (value >= 1073741824 && value % 1073741824 === 0) {
		sizeValue = value / 1073741824;
		sizeUnit = "GB";
	} else {
		sizeValue = Math.round(value / 1048576);
		sizeUnit = "MB";
	}
}

// Sync local state when bound value changes
$effect(() => {
	parseAndSync();
});

// Update bound value when local state changes
function updateValue() {
	value = unitToBytes(sizeValue, sizeUnit);
	onchange?.(value);
}

// Get byte display text
const byteDisplay = $derived(
	showBytes ? `(${value.toLocaleString()} bytes)` : "",
);
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
				bind:value={sizeValue}
				min={minValue}
				max={dynamicMaxValue || undefined}
				{placeholder}
				oninput={updateValue}
			/>
		</div>
		<div class="w-20">
			<select
				class="select select-bordered"
				bind:value={sizeUnit}
				onchange={updateValue}
			>
				{#each sizeUnitOptions as option}
					<option value={option.value}>{option.name}</option>
				{/each}
			</select>
		</div>
	</div>
	{#if description || byteDisplay}
		<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
			{description} {byteDisplay}
		</p>
	{/if}
	{#if presets.length > 0}
		<div class="mt-2 flex flex-wrap gap-2">
			{#each presets as preset}
				<button
					type="button"
					class="btn btn-xs btn-outline"
					onclick={() => setPreset(preset.value, preset.unit)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 