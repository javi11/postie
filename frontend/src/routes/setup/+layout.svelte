<!-- Setup layout without navbar -->
<script lang="ts">
import ToastContainer from "$lib/components/ToastContainer.svelte";
import "../../style.css";
import apiClient from "$lib/api/client";
import { onMount } from "svelte";

onMount(async () => {
	// Initialize API client
	await apiClient.initialize();
});
</script>

<div class="min-h-screen bg-gradient-to-br from-primary/10 to-secondary/10">
	<slot />
	<!-- Toast notifications -->
	<ToastContainer />
</div>