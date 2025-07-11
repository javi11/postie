<script lang="ts">
import { Button, Input, Label, P } from "flowbite-svelte";

interface ComponentProps {
	value?: number;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
	unitLabel?: string;
}

let {
	value = $bindable(0),
	label = "",
	description = "",
	placeholder = "0",
	presets = [],
	minValue = 0,
	maxValue = 1000,
	id = "",
	unitLabel = "MB/s",
}: ComponentProps = $props();

function setPreset(presetValue: number) {
	value = presetValue;
}
</script>

<div>
	<Label for={id} class="mb-2">{label}</Label>
	<div class="flex gap-2">
		<div class="flex-1">
			<Input
				{id}
				type="number"
				bind:value
				min={minValue}
				max={maxValue}
				{placeholder}
			/>
		</div>
		<div class="w-16 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-400">
			{unitLabel}
		</div>
	</div>
	<P class="text-sm text-gray-600 dark:text-gray-400 mt-1">
		{description}
	</P>
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