<script lang="ts">
// Imports
import { t } from "$lib/i18n";

// Types
interface ComponentProps {
	value?: number;
	label?: string;
	description?: string;
	placeholder?: string;
	presets?: Array<{ label: string; value: number }>;
	minValue?: number;
	maxValue?: number;
	id?: string;
}

// Props
let {
	value = $bindable(750000),
	label = "",
	description = "",
	placeholder = "750000",
	presets = [],
	minValue = 1000,
	maxValue = 10000000,
	id = "",
}: ComponentProps = $props();

// Derived state
const unitLabel = $derived($t("common.inputs.bytes"));

// Functions
function setPreset(presetValue: number) {
	value = presetValue;
}
</script>

<!-- ByteSize Input Component -->
<div class="form-control w-full">
	<!-- Label -->
	{#if label}
		<label class="label" for={id}>
			<span class="label-text font-medium text-base-content">{label}</span>
		</label>
	{/if}
	
	<!-- Input with Unit Display -->
	<div class="flex gap-2">
		<div class="flex-1">
			<input
				{id}
				type="number"
				class="input input-bordered w-full focus:input-primary transition-colors"
				bind:value
				min={minValue}
				max={maxValue}
				{placeholder}
			/>
		</div>
		<div class="w-20 flex items-center justify-center text-sm font-medium text-base-content/60">
			{unitLabel}
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
					onclick={() => setPreset(preset.value)}
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
</div> 