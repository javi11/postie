<script lang="ts">
  import { onMount } from "svelte";
  import { Toast } from "flowbite-svelte";
  import { slide } from "svelte/transition";
  import {
    CheckCircleSolid,
    ExclamationCircleSolid,
    CloseCircleSolid,
    InfoCircleSolid,
  } from "flowbite-svelte-icons";
  import { toastStore, type ToastMessage } from "../stores/toast";

  let toasts: ToastMessage[] = [];

  onMount(() => {
    const unsubscribe = toastStore.subscribe((value) => {
      toasts = value;
    });

    return unsubscribe;
  });

  function getToastColor(type: ToastMessage["type"]) {
    switch (type) {
      case "success":
        return "green";
      case "error":
        return "red";
      case "warning":
        return "yellow";
      case "info":
        return "blue";
      default:
        return "blue";
    }
  }

  function getToastIcon(type: ToastMessage["type"]) {
    switch (type) {
      case "success":
        return CheckCircleSolid;
      case "error":
        return CloseCircleSolid;
      case "warning":
        return ExclamationCircleSolid;
      case "info":
        return InfoCircleSolid;
      default:
        return InfoCircleSolid;
    }
  }

  function dismissToast(id: string) {
    toastStore.remove(id);
  }
</script>

<!-- Toast Container positioned at top-right -->
<div class="fixed top-20 right-5 z-50 space-y-3 max-w-sm w-full">
  {#each toasts as toast (toast.id)}
    <Toast
      transition={slide}
      params={{ duration: 300 }}
      color={getToastColor(toast.type)}
      onclick={() => dismissToast(toast.id)}
      class="shadow-lg"
    >
      {#snippet icon()}
        <svelte:component this={getToastIcon(toast.type)} class="h-5 w-5" />
        <span class="sr-only">{toast.type} icon</span>
      {/snippet}
      <div class="flex-1">
        <div class="text-sm font-semibold">{toast.title}</div>
        {#if toast.message}
          <div class="text-sm">{toast.message}</div>
        {/if}
      </div>
    </Toast>
  {/each}
</div>
