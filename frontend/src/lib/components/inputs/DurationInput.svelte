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

const timeUnitOptions = [
	{ value: "s", name: $t("common.inputs.time_units.seconds") },
	{ value: "m", name: $t("common.inputs.time_units.minutes") },
	{ value: "h", name: $t("common.inputs.time_units.hours") },
];

// Local state for the inputs
let numberValue = $state(5);
let unitValue = $state("s");

// Parse the bound value and sync with local state
function parseAndSync() {
	if (!value || typeof value !== "string") {
		numberValue = 5;
		unitValue = "s";
		return;
	}

	const match = value.match(/^(\d+)([smh])$/);
	if (match) {
		numberValue = Number.parseInt(match[1]);
		unitValue = match[2];
	} else {
		// Fallback for invalid format
		numberValue = 5;
		unitValue = "s";
	}
}

// Sync local state when bound value changes
$effect(() => {
	parseAndSync();
});

// Update bound value when local state changes
function updateValue() {
	value = `${numberValue}${unitValue}`;
}

function setPreset(presetValue: number, presetUnit: string) {
	numberValue = presetValue;
	unitValue = presetUnit;
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
				bind:value={numberValue}
				min={minValue}
				max={maxValue}
				{placeholder}
				oninput={updateValue}
			/>
		</div>
		<div class="w-24">
			<Select
				items={timeUnitOptions}
				bind:value={unitValue}
				onchange={updateValue}
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