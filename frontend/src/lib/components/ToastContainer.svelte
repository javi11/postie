<script lang="ts">
import { AlertTriangle, CheckCircle, Info, X } from "lucide-svelte";
import { onMount } from "svelte";
import { slide } from "svelte/transition";
import { type ToastMessage, toastStore } from "../stores/toast";

let toasts: ToastMessage[] = [];

onMount(() => {
	const unsubscribe = toastStore.subscribe((value) => {
		toasts = value;
	});

	return unsubscribe;
});

function getToastClass(type: ToastMessage["type"]) {
	switch (type) {
		case "success":
			return "alert-success";
		case "error":
			return "alert-error";
		case "warning":
			return "alert-warning";
		case "info":
			return "alert-info";
		default:
			return "alert-info";
	}
}

function getToastIcon(type: ToastMessage["type"]) {
	switch (type) {
		case "success":
			return CheckCircle;
		case "error":
			return X;
		case "warning":
			return AlertTriangle;
		case "info":
			return Info;
		default:
			return Info;
	}
}

function dismissToast(id: string) {
	toastStore.remove(id);
}
</script>

<!-- Toast Container positioned at bottom-right -->
<div class="toast toast-bottom toast-end z-50" aria-live="polite" aria-relevant="additions removals">
  {#each toasts as toast (toast.id)}
	<div
	  transition:slide={{ duration: 300 }}
	  class="alert {getToastClass(toast.type)} shadow-lg"
	  role="alert"
	  aria-atomic="true"
	>
	  <svelte:component this={getToastIcon(toast.type)} class="h-5 w-5" />
	  <div class="flex-1">
		<div class="font-semibold">{toast.title}</div>
		{#if toast.message}
		  <div class="text-sm opacity-90">{toast.message}</div>
		{/if}
	  </div>
	  <button
		class="btn btn-ghost btn-sm btn-circle"
		onclick={() => dismissToast(toast.id)}
		aria-label="Close notification"
	  >
		<X class="h-4 w-4" />
	  </button>
	</div>
  {/each}
</div>
