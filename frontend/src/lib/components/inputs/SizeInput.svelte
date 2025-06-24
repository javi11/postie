<script lang="ts">
import { Button, Input, Label, P, Select } from "flowbite-svelte";

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
}

let {
	value = $bindable(100),
	label = "",
	description = "",
	placeholder = "100",
	presets = [],
	minValue = 1,
	maxValue = 10000,
	id = "",
	showBytes = false,
}: ComponentProps = $props();

const sizeUnitOptions = [
	{ value: "MB", name: "MB" },
	{ value: "GB", name: "GB" },
];

let sizeValue = $state(100);
let sizeUnit = $state("MB");
let initialized = $state(false);

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

// Initialize from value prop (only once to avoid cycles)
$effect(() => {
	if (!initialized && value !== undefined) {
		if (value >= 1073741824 && value % 1073741824 === 0) {
			sizeValue = value / 1073741824;
			sizeUnit = "GB";
		} else {
			sizeValue = Math.round(value / 1048576);
			sizeUnit = "MB";
		}
		initialized = true;
	}
});

// Update maxValue based on unit
let dynamicMaxValue = $derived(
	sizeUnit === "GB" ? Math.ceil(maxValue / 1024) : maxValue,
);

// Update value when user changes input
$effect(() => {
	if (sizeValue !== undefined && sizeUnit !== undefined && initialized) {
		value = unitToBytes(sizeValue, sizeUnit);
	}
});

function setPreset(presetValue: number, presetUnit: string) {
	sizeValue = presetValue;
	sizeUnit = presetUnit;
}

// Get byte display text
let byteDisplay = $derived(
	showBytes ? `(${value.toLocaleString()} bytes)` : "",
);
</script>

<div>
	{#if label}
		<Label for={id} class="mb-2">{label}</Label>
	{/if}
	<div class="flex gap-2">
		<div class="flex-1">
			<Input
				{id}
				type="number"
				bind:value={sizeValue}
				min={minValue}
				max={dynamicMaxValue}
				{placeholder}
			/>
		</div>
		<div class="w-20">
			<Select
				items={sizeUnitOptions}
				bind:value={sizeUnit}
			/>
		</div>
	</div>
	{#if description || byteDisplay}
		<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
			{description} {byteDisplay}
		</P>
	{/if}
	{#if presets.length > 0}
		<div class="mt-2 flex flex-wrap gap-2">
			{#each presets as preset}
				<Button
					type="button"
					class="cursor-pointer px-2 py-1 text-xs"
					onclick={() => setPreset(preset.value, preset.unit)}
				>
					{preset.label}
				</Button>
			{/each}
		</div>
	{/if}
</div> 