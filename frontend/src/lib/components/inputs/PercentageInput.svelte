<script lang="ts">
import { Button, Input, Label, P } from "flowbite-svelte";

export let value: string = "10%";
export let label: string = "";
export let description: string = "";
export let placeholder: string = "10";
export let presets: Array<{ label: string; value: number }> = [];
export let minValue: number = 1;
export let maxValue: number = 100;
export let id: string = "";

let percentageValue: number;

// Parse existing value
function parseValue(valueString: string) {
	if (!valueString || typeof valueString !== 'string') {
		percentageValue = 10;
		return;
	}
	
	percentageValue = parseInt(valueString.replace('%', '')) || 10;
}

// Initialize values when value prop changes
$: if (value) {
	parseValue(value);
}

// Update value when internal value changes
function updateValue() {
	if (percentageValue !== undefined && percentageValue > 0) {
		value = `${percentageValue}%`;
	}
}

function setPreset(presetValue: number) {
	percentageValue = presetValue;
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
				bind:value={percentageValue}
				min={minValue}
				max={maxValue}
				{placeholder}
				on:input={updateValue}
			/>
		</div>
		<div class="w-12 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-400">
			%
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
					onclick={() => setPreset(preset.value)}
				>
					{preset.label}
				</Button>
			{/each}
		</div>
	{/if}
</div> 