import { writable } from "svelte/store";

export type ToastType = "success" | "error" | "warning" | "info";

export interface ToastMessage {
	id: string;
	type: ToastType;
	title: string;
	message?: string;
	duration?: number;
}

function createToastStore() {
	const { subscribe, set, update } = writable<ToastMessage[]>([]);

	const addToast = (toast: Omit<ToastMessage, "id">) => {
		const id = crypto.randomUUID();
		const newToast: ToastMessage = {
			id,
			...toast,
			duration: 8000,
		};

		update((toasts) => [...toasts, newToast]);
		// Auto-remove after duration
		if (newToast.duration && newToast.duration > 0) {
			setTimeout(() => {
				update((toasts) => toasts.filter((t) => t.id !== id));
			}, newToast.duration);
		}

		return id;
	};

	return {
		subscribe,
		add: addToast,
		remove: (id: string) => {
			update((toasts) => toasts.filter((t) => t.id !== id));
		},
		clear: () => {
			set([]);
		},
		success: (title: string, message?: string, duration?: number) =>
			addToast({ type: "success", title, message, duration }),
		error: (title: string, message?: string, duration?: number) =>
			addToast({ type: "error", title, message, duration }),
		warning: (title: string, message?: string, duration?: number) =>
			addToast({ type: "warning", title, message, duration }),
		info: (title: string, message?: string, duration?: number) =>
			addToast({ type: "info", title, message, duration }),
	};
}

export const toastStore = createToastStore();
