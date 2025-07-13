<script lang="ts">
import { page } from "$app/stores";
import { loadTranslations, locale, t } from "$lib/i18n";
import { availableLocales, setStoredLocale } from "$lib/i18n";
import { Check, ChevronDown } from "lucide-svelte";

// Get current locale
$: currentLocale = $locale;
$: currentLocaleData =
	availableLocales.find((l) => l.code === currentLocale) || availableLocales[0];

async function changeLocale(newLocale: string) {
	// Store the selected locale
	setStoredLocale(newLocale);

	// Load translations for the new locale
	await loadTranslations(newLocale, $page.url.pathname);
}
</script>

<div class="dropdown dropdown-end">
	<div tabindex="0" role="button" class="btn btn-ghost flex items-center space-x-2">
		<span class="text-lg">{currentLocaleData.flag}</span>
		<span>{currentLocaleData.name}</span>
		<ChevronDown class="w-3 h-3" />
	</div>
	<div tabindex="0" class="dropdown-content menu bg-base-100 rounded-box z-[1] w-44 p-2 shadow"
		 role="menu">
		{#each availableLocales as langOption}
			<button 
				onclick={() => changeLocale(langOption.code)}
				class="flex items-center gap-3 w-full text-left p-2 hover:bg-base-200 rounded"
				role="menuitem"
			>
				<span class="text-lg">{langOption.flag}</span>
				<span class="flex-1">{langOption.name}</span>
				{#if currentLocale === langOption.code}
					<Check class="w-4 h-4 text-primary" />
				{/if}
			</button>
		{/each}
	</div>
</div>