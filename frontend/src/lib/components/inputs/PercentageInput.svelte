<script lang="ts">
interface ComponentProps {
	value?: string;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
}

let {
	value = $bindable("10%"),
	label = "",
	description = "",
	placeholder = "10",
	presets = [],
	minValue = 1,
	maxValue = 100,
	id = "",
}: ComponentProps = $props();

let percentageValue = $state(10);

// Parse existing value
function parseValue(valueString: string) {
	if (!valueString || typeof valueString !== "string") {
		percentageValue = 10;
		return;
	}

	percentageValue = Number.parseInt(valueString.replace("%", "")) || 10;
}

// Initialize values when value prop changes
$effect(() => {
	if (value) {
		parseValue(value);
	}
});

// Update value when internal value changes
$effect(() => {
	if (percentageValue !== undefined) {
		value = `${percentageValue}%`;
	}
});

function setPreset(presetValue: number) {
	percentageValue = presetValue;
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
				bind:value={percentageValue}
				min={minValue}
				max={maxValue}
				{placeholder}
			/>
		</div>
		<div class="w-12 flex items-center justify-center text-sm font-medium text-gray-600 dark:text-gray-400">
			%
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
					onclick={() => setPreset(preset.value)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 