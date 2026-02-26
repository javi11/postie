<script lang="ts">
import { t } from "$lib/i18n";
import { currentTheme, groupedThemes, SYSTEM_THEME, type ThemeValue } from "$lib/stores/theme";
import { Check, Monitor } from "lucide-svelte";

// Track the stored preference (which may be "system") separately from the applied theme
let storedPreference = $state(
	typeof localStorage !== "undefined" ? (localStorage.getItem("theme") ?? SYSTEM_THEME) : SYSTEM_THEME,
);

function selectTheme(value: ThemeValue) {
	currentTheme.setTheme(value);
	storedPreference = value;
}

const lightThemes = $derived(groupedThemes["Light"] ?? []);
const darkThemes = $derived(groupedThemes["Dark"] ?? []);
</script>

<div class="space-y-4">
	<!-- System Default -->
	<button
		type="button"
		class="w-full flex items-center gap-3 p-3 rounded-xl border-2 transition-all duration-150 cursor-pointer text-left
			{storedPreference === SYSTEM_THEME
			? 'border-primary bg-primary/10'
			: 'border-base-300 bg-base-200 hover:border-primary/50 hover:bg-base-300'}"
		onclick={() => selectTheme(SYSTEM_THEME)}
	>
		<div class="flex-shrink-0 w-9 h-9 rounded-lg bg-base-300 flex items-center justify-center">
			<Monitor class="w-5 h-5 text-base-content/70" />
		</div>
		<div class="flex-1 min-w-0">
			<div class="font-medium text-sm text-base-content">{$t("settings.general.system_default")}</div>
			<div class="text-xs text-base-content/60 truncate">{$t("settings.general.system_default_description")}</div>
		</div>
		{#if storedPreference === SYSTEM_THEME}
			<Check class="flex-shrink-0 w-4 h-4 text-primary" />
		{/if}
	</button>

	<!-- Light Themes -->
	<div>
		<p class="text-xs font-semibold text-base-content/50 uppercase tracking-wider mb-2">
			{$t("settings.general.light_themes")}
		</p>
		<div class="grid grid-cols-3 gap-2 sm:grid-cols-4">
			{#each lightThemes as theme}
				<button
					type="button"
					class="group flex flex-col gap-1.5 p-2 rounded-xl border-2 transition-all duration-150 cursor-pointer
						{storedPreference === theme.value
						? 'border-primary bg-primary/10'
						: 'border-base-300 bg-base-200 hover:border-primary/40 hover:bg-base-300'}"
					onclick={() => selectTheme(theme.value)}
					title={theme.name}
				>
					<!-- Color swatches -->
					<div class="flex gap-0.5 h-5">
						{#each theme.colors as color}
							<span
								class="flex-1 rounded-sm"
								style="background-color: {color};"
							></span>
						{/each}
					</div>
					<!-- Theme name + check -->
					<div class="flex items-center justify-between gap-1 min-w-0">
						<span class="text-xs text-base-content/80 truncate leading-tight">{theme.name}</span>
						{#if storedPreference === theme.value}
							<Check class="flex-shrink-0 w-3 h-3 text-primary" />
						{/if}
					</div>
				</button>
			{/each}
		</div>
	</div>

	<!-- Dark Themes -->
	<div>
		<p class="text-xs font-semibold text-base-content/50 uppercase tracking-wider mb-2">
			{$t("settings.general.dark_themes")}
		</p>
		<div class="grid grid-cols-3 gap-2 sm:grid-cols-4">
			{#each darkThemes as theme}
				<button
					type="button"
					class="group flex flex-col gap-1.5 p-2 rounded-xl border-2 transition-all duration-150 cursor-pointer
						{storedPreference === theme.value
						? 'border-primary bg-primary/10'
						: 'border-base-300 bg-base-200 hover:border-primary/40 hover:bg-base-300'}"
					onclick={() => selectTheme(theme.value)}
					title={theme.name}
				>
					<!-- Color swatches -->
					<div class="flex gap-0.5 h-5">
						{#each theme.colors as color}
							<span
								class="flex-1 rounded-sm"
								style="background-color: {color};"
							></span>
						{/each}
					</div>
					<!-- Theme name + check -->
					<div class="flex items-center justify-between gap-1 min-w-0">
						<span class="text-xs text-base-content/80 truncate leading-tight">{theme.name}</span>
						{#if storedPreference === theme.value}
							<Check class="flex-shrink-0 w-3 h-3 text-primary" />
						{/if}
					</div>
				</button>
			{/each}
		</div>
	</div>
</div>
