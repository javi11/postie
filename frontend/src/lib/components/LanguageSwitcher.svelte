<script lang="ts">
import { page } from "$app/stores";
import { loadTranslations, locale, t } from "$lib/i18n";
import { availableLocales, setStoredLocale } from "$lib/i18n";

let isOpen = false;

// Get current locale
$: currentLocale = $locale;
$: currentLocaleData =
	availableLocales.find((l) => l.code === currentLocale) || availableLocales[0];

async function changeLocale(newLocale: string) {
	// Store the selected locale
	setStoredLocale(newLocale);

	// Load translations for the new locale
	await loadTranslations(newLocale, $page.url.pathname);

	// Close the dropdown
	isOpen = false;
}

function toggleDropdown() {
	isOpen = !isOpen;
}

// Close dropdown when clicking outside
function handleClickOutside(event: MouseEvent) {
	const target = event.target as HTMLElement;
	if (!target.closest(".language-switcher")) {
		isOpen = false;
	}
}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="language-switcher relative">
  <button
    onclick={toggleDropdown}
    class="flex items-center space-x-2 px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white transition-colors"
    aria-label={$t('common.nav.language')}
  >
    <span class="text-lg">{currentLocaleData.flag}</span>
    <span class="hidden sm:inline">{currentLocaleData.name}</span>
    <svg 
      class="w-4 h-4 transition-transform {isOpen ? 'rotate-180' : ''}" 
      fill="none" 
      stroke="currentColor" 
      viewBox="0 0 24 24"
    >
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
    </svg>
  </button>

  {#if isOpen}
    <div class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50">
      <div class="py-1">
        {#each availableLocales as langOption}
          <button
            onclick={() => changeLocale(langOption.code)}
            class="flex items-center w-full px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors {currentLocale === langOption.code ? 'bg-gray-50 dark:bg-gray-700' : ''}"
          >
            <span class="text-lg mr-3">{langOption.flag}</span>
            <span>{langOption.name}</span>
            {#if currentLocale === langOption.code}
              <svg class="w-4 h-4 ml-auto text-blue-600" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
              </svg>
            {/if}
          </button>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .language-switcher {
    @apply inline-block;
  }
</style> 