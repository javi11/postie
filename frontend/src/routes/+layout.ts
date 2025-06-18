import { getBrowserLocale, getStoredLocale, loadTranslations } from "$lib/i18n";

export const prerender = true;
export const ssr = false;

/** @type {import('@sveltejs/kit').Load} */
export const load = async ({ url }: { url: URL }) => {
	const { pathname } = url;

	// Determine the initial locale
	const storedLocale = getStoredLocale();
	const initLocale = storedLocale || getBrowserLocale();

	// Load translations for the current route and locale
	await loadTranslations(initLocale, pathname);

	return {};
};
