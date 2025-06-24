<script lang="ts">
import { t } from "$lib/i18n";
import { Button, Input, Label, P, Select } from "flowbite-svelte";

interface ComponentProps {
	value?: string;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number; unit: string }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
}

let {
	value = $bindable("5s"),
	label = "",
	description = "",
	placeholder = "5",
	presets = [],
	minValue = 1,
	maxValue = 3600,
	id = "",
}: ComponentProps = $props();

let timeUnitOptions = $derived([
	{ value: "s", name: $t("common.inputs.time_units.seconds") },
	{ value: "m", name: $t("common.inputs.time_units.minutes") },
	{ value: "h", name: $t("common.inputs.time_units.hours") },
]);

let durationValue = $state(5);
let durationUnit = $state("s");

// Parse existing value
function parseValue(valueString: string) {
	if (!valueString || typeof valueString !== "string") {
		durationValue = 5;
		durationUnit = "s";
		return;
	}

	const match = valueString.match(/^(\d+)([smh])$/);
	if (match) {
		durationValue = Number.parseInt(match[1]);
		durationUnit = match[2];
	} else {
		// Fallback for invalid format
		durationValue = 5;
		durationUnit = "s";
	}
}

// Initialize values when value prop changes
$effect(() => {
	if (value) {
		parseValue(value);
	}
});

// Update value when internal values change
$effect(() => {
	if (durationValue !== undefined && durationUnit !== undefined) {
		value = `${durationValue}${durationUnit}`;
	}
});

function setPreset(presetValue: number, presetUnit: string) {
	durationValue = presetValue;
	durationUnit = presetUnit;
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
			/>
		</div>
		<div class="w-24">
			<Select
				items={timeUnitOptions}
				bind:value={durationUnit}
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