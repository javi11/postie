import { register, init, locale, locales, t, isLoading } from 'svelte-i18n';
import { browser } from '$app/environment';
import { get } from 'svelte/store';

export { t, locale, locales, isLoading as loading } from 'svelte-i18n';

// Register the translation files
register('en', () => import('./locales/en/common.json'));
register('en', () => import('./locales/en/setup.json'));
register('en', () => import('./locales/en/dashboard.json'));
register('en', () => import('./locales/en/settings.json'));

register('es', () => import('./locales/es/common.json'));
register('es', () => import('./locales/es/setup.json'));
register('es', () => import('./locales/es/dashboard.json'));
register('es', () => import('./locales/es/settings.json'));

register('fr', () => import('./locales/fr/common.json'));
register('fr', () => import('./locales/fr/setup.json'));
register('fr', () => import('./locales/fr/dashboard.json'));
register('fr', () => import('./locales/fr/settings.json'));

// Initialize the i18n library
init({
  fallbackLocale: 'en',
  initialLocale: getInitialLocale(),
});

// Helper function to get the initial locale
function getInitialLocale(): string {
  if (browser) {
    // Check localStorage first
    const stored = localStorage.getItem('locale');
    if (stored && ['en', 'es', 'fr'].includes(stored)) {
      return stored;
    }
    
    // Fall back to browser language
    return getBrowserLocale();
  }
  
  return 'en';
}

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

// Custom getLocale function
export function getLocale(): string {
  return get(locale) || 'en';
}

// Custom function to change locale and persist it
export function setLocale(newLocale: string): void {
  if (['en', 'es', 'fr'].includes(newLocale)) {
    locale.set(newLocale);
    setStoredLocale(newLocale);
  }
}