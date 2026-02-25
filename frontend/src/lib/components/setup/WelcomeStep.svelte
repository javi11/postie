<script lang="ts">
import LanguageSwitcher from "$lib/components/LanguageSwitcher.svelte";
import ThemeSwitcher from "$lib/components/ThemeSwitcher.svelte";
import { t } from "$lib/i18n";
</script>

<div class="w-full h-full flex flex-col items-center justify-center space-y-16 px-4">
	<!-- Welcome Title -->
	<div class="text-center space-y-6">
		<h2 class="text-4xl sm:text-5xl font-bold text-base-content">
			{$t("setup.welcome.title")}
		</h2>
		<p class="text-xl sm:text-2xl text-base-content/70 max-w-4xl mx-auto leading-relaxed">
			{$t("setup.welcome.subtitle")}
		</p>
	</div>

	<!-- Language and Theme Controls - Modern daisyUI Cards -->
	<div class="w-full max-w-3xl space-y-4">
		<h3 class="text-2xl font-semibold text-base-content text-center mb-8">
			{$t("setup.welcome.preferences_title")}
		</h3>

		<!-- Language Selector - compact horizontal card -->
		<div class="card bg-base-100/90 backdrop-blur-sm shadow-xl border border-base-300/60">
			<div class="card-body py-4">
				<div class="flex items-center gap-4">
					<div class="w-10 h-10 bg-primary rounded-lg flex items-center justify-center shrink-0">
						<svg class="w-5 h-5 text-primary-content" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"></path>
						</svg>
					</div>
					<div class="flex-1 min-w-0">
						<h4 class="font-semibold text-base-content">{$t("setup.welcome.language_title")}</h4>
						<p class="text-sm text-base-content/70">{$t("setup.welcome.language_description")}</p>
					</div>
					<div class="ml-auto shrink-0">
						<LanguageSwitcher />
					</div>
				</div>
			</div>
		</div>

		<!-- Theme Selector - full-width card -->
		<div class="card bg-base-100/90 backdrop-blur-sm shadow-xl border border-base-300/60">
			<div class="card-body">
				<div class="flex items-center gap-4 mb-4">
					<div class="w-10 h-10 bg-secondary rounded-lg flex items-center justify-center shrink-0">
						<svg class="w-5 h-5 text-secondary-content" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>
						</svg>
					</div>
					<div>
						<h4 class="font-semibold text-base-content">{$t("setup.welcome.theme_title")}</h4>
						<p class="text-sm text-base-content/70">{$t("setup.welcome.theme_description")}</p>
					</div>
				</div>
				<ThemeSwitcher />
			</div>
		</div>
	</div>
</div>