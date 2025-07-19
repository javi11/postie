<script lang="ts">
// Imports
import { t } from "$lib/i18n";

// Types
interface ComponentProps {
	value?: string | undefined;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number; unit: string }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
	onchange?: (value: string) => void;
}

// Props
let {
	value = $bindable(),
	label = "",
	description = "",
	placeholder = "5",
	presets = [],
	minValue = 1,
	maxValue = 3600,
	id = "",
	onchange,
}: ComponentProps = $props();

// State - local state for form inputs (following best practices)
function parseValue(val: string | undefined): { number: number; unit: string } {
	if (!val || typeof val !== "string" || val.trim() === "") {
		return { number: 5, unit: "s" };
	}
	// Match patterns like "1m0s" or "1h30m" and extract the first meaningful element
	const match = val.match(/^(\d+)([smh])/);
	return match
		? { number: Number.parseInt(match[1]), unit: match[2] }
		: { number: 5, unit: "s" };
}

// Initialize with current value
const initial = parseValue(value);
let numberValue = $state(initial.number);
let unitValue = $state(initial.unit);
let lastUpdatedValue = $state(value);

// Effects - sync local state back to bound value when user changes inputs
$effect(() => {
	const newValue = `${numberValue}${unitValue}`;
	if (newValue !== lastUpdatedValue) {
		lastUpdatedValue = newValue;
		value = newValue;
		onchange?.(newValue);
	}
});

// Effect to update local state when value prop changes externally (avoid infinite loop)
$effect(() => {
	if (value !== lastUpdatedValue) {
		const parsed = parseValue(value);
		numberValue = parsed.number;
		unitValue = parsed.unit;
		lastUpdatedValue = value;
	}
});

// Derived state - reactive dropdown options
const timeUnitOptions = $derived([
	{ value: "s", name: $t("common.inputs.time_units.seconds") },
	{ value: "m", name: $t("common.inputs.time_units.minutes") },
	{ value: "h", name: $t("common.inputs.time_units.hours") },
]);

// Functions
function setPreset(presetValue: number, presetUnit: string) {
	numberValue = presetValue;
	unitValue = presetUnit;
}
</script>

<!-- Duration Input Component -->
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
				bind:value={numberValue}
				min={minValue}
				max={maxValue}
				{placeholder}
				oninput={(e) => {
					const target = e.target as HTMLInputElement;
					target.value = target.value.replace(/[^0-9]/g, '');
				}}
			/>
		</div>
		<div class="w-24">
			<select
				class="select select-bordered focus:select-primary transition-colors"
				bind:value={unitValue}
			>
				{#each timeUnitOptions as option}
					<option value={option.value}>{option.name}</option>
				{/each}
			</select>
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
					onclick={() => setPreset(preset.value, preset.unit)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 