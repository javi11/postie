<!-- Setup layout without navbar -->
<script lang="ts">
import { DarkMode } from "flowbite-svelte";
import ToastContainer from "$lib/components/ToastContainer.svelte";
import "../../style.css";
</script>

<div class="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800">
	<slot />
	<!-- Toast notifications -->
	<ToastContainer />
</div>