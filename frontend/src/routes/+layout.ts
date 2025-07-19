import { waitLocale } from 'svelte-i18n'
import { setupConsoleInterceptor } from "$lib/stores/logs";

export const prerender = true;
export const ssr = false;

/** @type {import('@sveltejs/kit').Load} */
export const load = async ({ url }: { url: URL }) => {
	setupConsoleInterceptor();

	return waitLocale()
};
