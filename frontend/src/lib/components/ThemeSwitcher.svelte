<script lang="ts">
import {
	availableThemes,
	currentTheme,
	type ThemeValue,
} from "$lib/stores/theme";

function changeTheme(event: Event) {
	const target = event.target as HTMLSelectElement;
	const newTheme = target.value as ThemeValue;
	currentTheme.setTheme(newTheme);
}
</script>

<select
	class="select select-bordered w-full"
	value={$currentTheme}
	onchange={changeTheme}
>
	{#each availableThemes as theme}
		<option value={theme.value}>{theme.name}</option>
	{/each}
</select>