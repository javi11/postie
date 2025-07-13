<script lang="ts">
import logo from "$lib/assets/images/logo.png";
import LanguageSwitcher from "$lib/components/LanguageSwitcher.svelte";
import { t } from "$lib/i18n";
import { Globe, Moon, Sun } from "lucide-svelte";

let isDarkMode = false;

function toggleDarkMode() {
	isDarkMode = !isDarkMode;
	if (isDarkMode) {
		document.documentElement.classList.add("dark");
		document.documentElement.setAttribute("data-theme", "dark");
	} else {
		document.documentElement.classList.remove("dark");
		document.documentElement.setAttribute("data-theme", "light");
	}
	localStorage.setItem("theme", isDarkMode ? "dark" : "light");
}
</script>

<div class="w-full h-full flex flex-col items-center justify-center space-y-16 px-4">
	<!-- Welcome Title -->
	<div class="text-center space-y-6">
		<h2 class="text-4xl sm:text-5xl font-bold text-gray-900 dark:text-white">
			{$t("setup.welcome.title")}
		</h2>
		<p class="text-xl sm:text-2xl text-gray-600 dark:text-gray-400 max-w-4xl mx-auto leading-relaxed">
			{$t("setup.welcome.subtitle")}
		</p>
	</div>

	<!-- Language and Theme Controls - Centered and prominent -->
	<div class="w-full max-w-2xl space-y-8">
		<h3 class="text-2xl font-semibold text-gray-900 dark:text-white text-center mb-8">
			{$t("setup.welcome.preferences_title")}
		</h3>
		
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-8">
			<!-- Language Selector -->
			<div class="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-xl border border-gray-200/60 dark:border-gray-700/60 p-6">
				<div class="text-center space-y-4">
					<div class="w-12 h-12 bg-blue-500 rounded-lg mx-auto mb-4 flex items-center justify-center">
						<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"></path>
						</svg>
					</div>
					<h4 class="font-semibold text-gray-900 dark:text-white text-lg">{$t("setup.welcome.language_title")}</h4>
					<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">{$t("setup.welcome.language_description")}</p>
					<div class="flex justify-center">
						<LanguageSwitcher />
					</div>
				</div>
			</div>

			<!-- Dark Mode Toggle -->
			<div class="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-xl border border-gray-200/60 dark:border-gray-700/60 p-6">
				<div class="text-center space-y-4">
					<div class="w-12 h-12 bg-purple-500 rounded-lg mx-auto mb-4 flex items-center justify-center">
						<svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>
						</svg>
					</div>
					<h4 class="font-semibold text-gray-900 dark:text-white text-lg">{$t("setup.welcome.theme_title")}</h4>
					<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">{$t("setup.welcome.theme_description")}</p>
					<div class="flex justify-center">
						<button 
							class="btn btn-ghost text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
							onclick={toggleDarkMode}
						>
							{#if isDarkMode}
								<Sun class="w-5 h-5" />
							{:else}
								<Moon class="w-5 h-5" />
							{/if}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>