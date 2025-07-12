<script lang="ts">
import { page } from "$app/stores";
import { loadTranslations, locale, t } from "$lib/i18n";
import { availableLocales, setStoredLocale } from "$lib/i18n";
import { Dropdown, DropdownItem, Button } from "flowbite-svelte";
import { ChevronDownOutline } from "flowbite-svelte-icons";

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

<Button class="flex items-center space-x-2 cursor-pointer">
	<span class="text-lg">{currentLocaleData.flag}</span>
	<span>{currentLocaleData.name}</span>
	<ChevronDownOutline class="w-3 h-3 ms-2 text-sm" />
</Button>
<Dropdown class="w-44 p-3 space-y-1 text-sm list-none">
	{#each availableLocales as langOption}
		<DropdownItem 
			onclick={() => changeLocale(langOption.code)}
			class="cursor-pointer w-full"
		>
			<div class="flex items-center gap-1 text-primary-600 dark:text-primary-400">
				<span class="text-lg mr-3">{langOption.flag}</span>
				<span>{langOption.name}</span>
			{#if currentLocale === langOption.code}
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
					<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
				</svg>
			{/if}
      </div>
		</DropdownItem>
	{/each}
</Dropdown>