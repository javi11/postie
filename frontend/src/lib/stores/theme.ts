import { browser } from "$app/environment";
import { writable } from "svelte/store";

export type ThemeCategory = "Light" | "Dark";

export interface ThemeEntry {
	name: string;
	value: string;
	category: ThemeCategory;
	colors: [string, string, string, string];
}

// All available daisyUI themes with representative color swatches
export const availableThemes: ThemeEntry[] = [
	// Light themes
	{ name: "Cupcake", value: "cupcake", category: "Light", colors: ["#faf7f5", "#65c3c8", "#ef9fbc", "#291334"] },
	{ name: "Light", value: "light", category: "Light", colors: ["#ffffff", "#570df8", "#f000b8", "#3d4451"] },
	{ name: "Bumblebee", value: "bumblebee", category: "Light", colors: ["#ffffff", "#e0a82e", "#f9d72f", "#181830"] },
	{ name: "Emerald", value: "emerald", category: "Light", colors: ["#ffffff", "#66cc8a", "#377cfb", "#333c4d"] },
	{ name: "Corporate", value: "corporate", category: "Light", colors: ["#ffffff", "#4b6bfb", "#7b92b2", "#1d2734"] },
	{ name: "Retro", value: "retro", category: "Light", colors: ["#e4d8b4", "#ef9995", "#a4cbb4", "#282425"] },
	{ name: "Garden", value: "garden", category: "Light", colors: ["#e9e7e7", "#5c7f67", "#ecf4e7", "#100f0f"] },
	{ name: "Lofi", value: "lofi", category: "Light", colors: ["#ffffff", "#0d0d0d", "#1a1a1a", "#000000"] },
	{ name: "Pastel", value: "pastel", category: "Light", colors: ["#ffffff", "#d1c1d7", "#f6cbd1", "#403c3d"] },
	{ name: "Fantasy", value: "fantasy", category: "Light", colors: ["#ffffff", "#6e0b75", "#007ebd", "#1f2937"] },
	{ name: "Wireframe", value: "wireframe", category: "Light", colors: ["#ffffff", "#b8b8b8", "#b8b8b8", "#000000"] },
	{ name: "CMYK", value: "cmyk", category: "Light", colors: ["#ffffff", "#45aeee", "#e8488a", "#3d4451"] },
	{ name: "Autumn", value: "autumn", category: "Light", colors: ["#f9f1e7", "#8c0327", "#d85251", "#201826"] },
	{ name: "Lemonade", value: "lemonade", category: "Light", colors: ["#ffffff", "#519903", "#e9e92e", "#1c2b15"] },
	{ name: "Winter", value: "winter", category: "Light", colors: ["#ffffff", "#047aff", "#463aa2", "#021431"] },
	{ name: "Nord", value: "nord", category: "Light", colors: ["#eceff4", "#5e81ac", "#81a1c1", "#2e3440"] },
	{ name: "Caramellatte", value: "caramellatte", category: "Light", colors: ["#f5e6d3", "#c47a45", "#d4956e", "#2c1a0e"] },
	// Dark themes
	{ name: "Dim", value: "dim", category: "Dark", colors: ["#2a303c", "#9fb3d1", "#f9b625", "#e6e6e6"] },
	{ name: "Forest", value: "forest", category: "Dark", colors: ["#171212", "#1eb854", "#1db88e", "#d1fae5"] },
	{ name: "Dark", value: "dark", category: "Dark", colors: ["#1d232a", "#661ae6", "#d926a9", "#a6adbb"] },
	{ name: "Dracula", value: "dracula", category: "Dark", colors: ["#282a36", "#ff79c6", "#bd93f9", "#f8f8f2"] },
	{ name: "Synthwave", value: "synthwave", category: "Dark", colors: ["#1a103c", "#e779c1", "#58c7f3", "#f9f7fd"] },
	{ name: "Halloween", value: "halloween", category: "Dark", colors: ["#212121", "#f28c18", "#6d3a9c", "#f8f8f2"] },
	{ name: "Luxury", value: "luxury", category: "Dark", colors: ["#09090b", "#ffffff", "#c08e5d", "#d4d4d8"] },
	{ name: "Night", value: "night", category: "Dark", colors: ["#0f1729", "#38bdf8", "#818cf8", "#b3c5ef"] },
	{ name: "Coffee", value: "coffee", category: "Dark", colors: ["#20161f", "#db924b", "#263e3f", "#e3cbaf"] },
	{ name: "Business", value: "business", category: "Dark", colors: ["#1b1b1b", "#1c4f82", "#7b92b2", "#ffffff"] },
	{ name: "Sunset", value: "sunset", category: "Dark", colors: ["#1b1a1e", "#ff865b", "#fd6f9c", "#f0ddd8"] },
	{ name: "Abyss", value: "abyss", category: "Dark", colors: ["#060f16", "#00d8ff", "#0090cc", "#d3e8f0"] },
];

export type ThemeValue = string;

// Special "system" value stored in localStorage to follow OS preference
const SYSTEM_THEME = "system";
const defaultTheme = "cupcake";
const defaultDarkTheme = "dracula";

function resolveSystemTheme(): string {
	if (!browser) return defaultTheme;
	return window.matchMedia("(prefers-color-scheme: dark)").matches ? defaultDarkTheme : defaultTheme;
}

function getInitialTheme(): string {
	if (!browser) return defaultTheme;

	const saved = localStorage.getItem("theme");
	if (saved === SYSTEM_THEME) return resolveSystemTheme();
	if (saved && availableThemes.some((t) => t.value === saved)) return saved;

	// Fallback: check system preference
	return resolveSystemTheme();
}

function getStoredPreference(): string {
	if (!browser) return defaultTheme;
	return localStorage.getItem("theme") ?? SYSTEM_THEME;
}

function applyTheme(theme: string) {
	if (!browser) return;
	document.documentElement.setAttribute("data-theme", theme);
}

function createThemeStore() {
	// Store the stored preference (may be "system") separately from active theme
	const { subscribe, set } = writable<string>(getInitialTheme());

	// Track whether the stored preference is "system"
	const { subscribe: subscribeStored, set: setStored } = writable<string>(getStoredPreference());

	return {
		subscribe,
		subscribeStored,
		setTheme: (themeValue: string) => {
			if (!browser) return;

			document.documentElement.classList.add("theme-transition");

			if (themeValue === SYSTEM_THEME) {
				localStorage.setItem("theme", SYSTEM_THEME);
				setStored(SYSTEM_THEME);
				const resolved = resolveSystemTheme();
				applyTheme(resolved);
				set(resolved);
			} else if (availableThemes.some((t) => t.value === themeValue)) {
				localStorage.setItem("theme", themeValue);
				setStored(themeValue);
				applyTheme(themeValue);
				set(themeValue);
			}

			setTimeout(() => {
				document.documentElement.classList.remove("theme-transition");
			}, 250);
		},
		set: (theme: string) => {
			if (browser) {
				applyTheme(theme);
				localStorage.setItem("theme", theme);
			}
			set(theme);
		},
		reset: () => {
			if (browser) {
				localStorage.removeItem("theme");
				const resolved = resolveSystemTheme();
				applyTheme(resolved);
				setStored(SYSTEM_THEME);
				set(resolved);
			}
		},
	};
}

export const currentTheme = createThemeStore();
export { SYSTEM_THEME };

// Group themes by category
export const groupedThemes = availableThemes.reduce(
	(acc, theme) => {
		if (!acc[theme.category]) {
			acc[theme.category] = [];
		}
		acc[theme.category].push(theme);
		return acc;
	},
	{} as Record<ThemeCategory, ThemeEntry[]>,
);

export function getThemeInfo(themeValue: string): ThemeEntry {
	return availableThemes.find((t) => t.value === themeValue) ?? availableThemes[0];
}

// Initialize theme on first load
if (browser) {
	const preference = localStorage.getItem("theme");
	if (preference === SYSTEM_THEME || !preference) {
		applyTheme(resolveSystemTheme());
	} else if (availableThemes.some((t) => t.value === preference)) {
		applyTheme(preference);
	} else {
		applyTheme(defaultTheme);
	}
}
