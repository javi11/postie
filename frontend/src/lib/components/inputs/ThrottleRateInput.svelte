<script lang="ts">
import { Input, Label, P } from "flowbite-svelte";

export let value: number = 0;
export let label: string = "";
export let description: string = "";
export let placeholder: string = "0";
export let presets: Array<{ label: string; value: number }> = [];
export let minValue: number = 0;
export let maxValue: number = 1000;
export let id: string = "";
export let unitLabel: string = "MB/s";

function updateValue() {
	// This is just to trigger any parent reactive statements
	value = value;
}

function setPreset(presetValue: number) {
	value = presetValue;
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
				bind:value
				min={minValue}
				max={maxValue}
				{placeholder}
				on:input={updateValue}
			/>
		</div>
		<div class="w-16 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-400">
			{unitLabel}
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
					onclick={() => setPreset(preset.value)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 