import i18n from "sveltekit-i18n";

/** @type {import('sveltekit-i18n').Config} */
const config = {
	loaders: [
		// English translations
		{
			locale: "en",
			key: "common",
			loader: async () =>
				(await import("./translations/en/common.json")).default,
		},
		{
			locale: "en",
			key: "dashboard",
			routes: ["/"],
			loader: async () =>
				(await import("./translations/en/dashboard.json")).default,
		},
		{
			locale: "en",
			key: "settings",
			routes: ["/settings"],
			loader: async () =>
				(await import("./translations/en/settings.json")).default,
		},
		// Spanish translations
		{
			locale: "es",
			key: "common",
			loader: async () =>
				(await import("./translations/es/common.json")).default,
		},
		{
			locale: "es",
			key: "dashboard",
			routes: ["/"],
			loader: async () =>
				(await import("./translations/es/dashboard.json")).default,
		},
		{
			locale: "es",
			key: "settings",
			routes: ["/settings"],
			loader: async () =>
				(await import("./translations/es/settings.json")).default,
		},
		// French translations
		{
			locale: "fr",
			key: "common",
			loader: async () =>
				(await import("./translations/fr/common.json")).default,
		},
		{
			locale: "fr",
			key: "dashboard",
			routes: ["/"],
			loader: async () =>
				(await import("./translations/fr/dashboard.json")).default,
		},
		{
			locale: "fr",
			key: "settings",
			routes: ["/settings"],
			loader: async () =>
				(await import("./translations/fr/settings.json")).default,
		},
	],
};

export const { t, locale, locales, loading, loadTranslations } = new i18n(
	config,
);

// Helper function to get the browser's preferred language
export function getBrowserLocale(): string {
	if (typeof navigator !== "undefined") {
		const language = navigator.language;
		if (language.startsWith("es")) return "es";
		if (language.startsWith("fr")) return "fr";
	}
	return "en"; // Default to English
}

// Helper function to get locale from localStorage
export function getStoredLocale(): string | null {
	if (typeof localStorage !== "undefined") {
		return localStorage.getItem("locale");
	}
	return null;
}

// Helper function to store locale in localStorage
export function setStoredLocale(newLocale: string): void {
	if (typeof localStorage !== "undefined") {
		localStorage.setItem("locale", newLocale);
	}
}

// Available locales for language switcher
export const availableLocales = [
	{ code: "en", name: "English", flag: "🇺🇸" },
	{ code: "es", name: "Español", flag: "🇪🇸" },
	{ code: "fr", name: "Français", flag: "🇫🇷" },
];
