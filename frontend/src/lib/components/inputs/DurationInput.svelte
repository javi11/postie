<script lang="ts">
import { Input, Label, P, Select } from "flowbite-svelte";
import { createEventDispatcher } from "svelte";

export let value: string = "5s";
export let label: string = "";
export let description: string = "";
export let placeholder: string = "5";
export let presets: Array<{ label: string; value: number; unit: string }> = [];
export let minValue: number = 1;
export let maxValue: number = 3600;
export let id: string = "";

const dispatch = createEventDispatcher();

const timeUnitOptions = [
	{ value: "s", name: "Seconds" },
	{ value: "m", name: "Minutes" },
	{ value: "h", name: "Hours" },
];

let durationValue: number;
let durationUnit: string;

// Parse existing value
function parseValue(valueString: string) {
	if (!valueString || typeof valueString !== 'string') {
		durationValue = 5;
		durationUnit = "s";
		return;
	}
	
	const match = valueString.match(/^(\d+)([smh])$/);
	if (match) {
		durationValue = parseInt(match[1]);
		durationUnit = match[2];
	} else {
		// Fallback for invalid format
		durationValue = 5;
		durationUnit = "s";
	}
}

// Initialize values when value prop changes
$: if (value) {
	parseValue(value);
}

// Update value and dispatch change event
function updateValue() {
	if (durationValue !== undefined && durationUnit && durationValue > 0) {
		const newValue = `${durationValue}${durationUnit}`;
		value = newValue;
		dispatch('change', newValue);
	}
}

function setPreset(presetValue: number, presetUnit: string) {
	durationValue = presetValue;
	durationUnit = presetUnit;
	updateValue();
}
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
				bind:value={durationValue}
				min={minValue}
				max={maxValue}
				{placeholder}
				on:input={updateValue}
			/>
		</div>
		<div class="w-24">
			<Select
				items={timeUnitOptions}
				bind:value={durationUnit}
				on:change={updateValue}
			/>
		</div>
	</div>
	{#if description}
		<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
			{description}
		</P>
	{/if}
	{#if presets.length > 0}
		<div class="mt-2 flex flex-wrap gap-2">
			{#each presets as preset}
				<button
					type="button"
					class="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
					onclick={() => setPreset(preset.value, preset.unit)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 