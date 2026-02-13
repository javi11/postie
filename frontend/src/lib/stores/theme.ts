import { browser } from "$app/environment";
import { writable } from "svelte/store";

// Available DaisyUI themes
export const availableThemes = [
	{ name: "Light", value: "cupcake", category: "Light" },
	{ name: "Bumblebee", value: "bumblebee", category: "Light" },
	{ name: "Emerald", value: "emerald", category: "Light" },
	{ name: "Dracula", value: "dracula", category: "Dark" },
	{ name: "Corporate", value: "corporate", category: "Light" },
	{ name: "Dark", value: "dim", category: "Light" },
] as const;

export type ThemeValue = (typeof availableThemes)[number]["value"];

// Default theme
const defaultTheme: ThemeValue = "cupcake";

// Get initial theme from localStorage or system preference
function getInitialTheme(): ThemeValue {
	if (!browser) return defaultTheme;

	const saved = localStorage.getItem("theme") as ThemeValue;
	if (saved && availableThemes.some((t) => t.value === saved)) {
		return saved;
	}

	// Check system preference
	const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
	return prefersDark ? "dim" : "cupcake";
}

// Create the theme store
function createThemeStore() {
	const { subscribe, set } = writable<ThemeValue>(getInitialTheme());

	return {
		subscribe,
		set: (theme: ThemeValue) => {
			if (browser) {
				// Apply theme to document
				document.documentElement.setAttribute("data-theme", theme);

				// Save to localStorage
				localStorage.setItem("theme", theme);
			}
			set(theme);
		},
		setTheme: (theme: ThemeValue) => {
			if (availableThemes.some((t) => t.value === theme)) {
				if (browser) {
					// Add transition class for smooth theme switching
					document.documentElement.classList.add("theme-transition");
					document.documentElement.setAttribute("data-theme", theme);
					localStorage.setItem("theme", theme);
					// Remove transition class after animation completes
					setTimeout(() => {
						document.documentElement.classList.remove("theme-transition");
					}, 250);
				}
				set(theme);
			}
		},
		reset: () => {
			if (browser) {
				localStorage.removeItem("theme");
				document.documentElement.setAttribute("data-theme", defaultTheme);
			}
			set(defaultTheme);
		},
	};
}

export const currentTheme = createThemeStore();

// Group themes by category
export const groupedThemes = availableThemes.reduce(
	(acc, theme) => {
		if (!acc[theme.category]) {
			acc[theme.category] = [];
		}
		acc[theme.category].push(theme);
		return acc;
	},
	{} as Record<string, (typeof availableThemes)[number][]>,
);

// Helper to get theme info
export function getThemeInfo(themeValue: ThemeValue) {
	return (
		availableThemes.find((t) => t.value === themeValue) || availableThemes[0]
	);
}

// Initialize theme on first load
if (browser) {
	const initialTheme = getInitialTheme();
	document.documentElement.setAttribute("data-theme", initialTheme);
}
