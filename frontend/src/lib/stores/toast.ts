import { writable } from "svelte/store";

export type ToastType = "success" | "error" | "warning" | "info";

export interface ToastMessage {
	id: string;
	type: ToastType;
	title: string;
	message?: string;
	duration?: number;
}

const MAX_VISIBLE_TOASTS = 5;

const DEFAULT_DURATIONS: Record<ToastType, number> = {
	success: 5000,
	info: 5000,
	warning: 10000,
	error: 0, // persistent — user must dismiss
};

function createToastStore() {
	const { subscribe, set, update } = writable<ToastMessage[]>([]);

	const addToast = (toast: Omit<ToastMessage, "id">) => {
		const id = `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
		const duration = toast.duration ?? DEFAULT_DURATIONS[toast.type];
		const newToast: ToastMessage = {
			id,
			...toast,
			duration,
		};

		update((toasts) => {
			const updated = [...toasts, newToast];
			// Cap visible toasts — remove oldest non-error toasts first
			if (updated.length > MAX_VISIBLE_TOASTS) {
				const excess = updated.length - MAX_VISIBLE_TOASTS;
				let removed = 0;
				return updated.filter((t) => {
					if (removed >= excess) return true;
					if (t.type !== "error") {
						removed++;
						return false;
					}
					return true;
				});
			}
			return updated;
		});

		// Auto-remove after duration (0 = persistent)
		if (duration > 0) {
			setTimeout(() => {
				update((toasts) => toasts.filter((t) => t.id !== id));
			}, duration);
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
