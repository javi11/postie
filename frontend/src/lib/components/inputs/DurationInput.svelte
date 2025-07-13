<script lang="ts">
import { t } from "$lib/i18n";

interface ComponentProps {
	value?: string;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number; unit: string }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
	onchange?: (value: string) => void;
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
	onchange,
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
	onchange?.(value);
}

function setPreset(presetValue: number, presetUnit: string) {
	numberValue = presetValue;
	unitValue = presetUnit;
	updateValue();
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
				bind:value={numberValue}
				min={minValue}
				max={maxValue}
				{placeholder}
				oninput={updateValue}
			/>
		</div>
		<div class="w-24">
			<select
				class="select select-bordered"
				bind:value={unitValue}
				onchange={updateValue}
			>
				{#each timeUnitOptions as option}
					<option value={option.value}>{option.name}</option>
				{/each}
			</select>
		</div>
	</div>
	{#if description}
		<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
			{description}
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