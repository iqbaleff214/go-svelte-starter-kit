import { writable } from "svelte/store";
import type { Toast, ToastType } from "$types";

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);

  function add(
    type: ToastType,
    title: string,
    message?: string,
    duration = 4000,
  ): string {
    const id = crypto.randomUUID();
    const toast: Toast = { id, type, title, message, duration };

    update((toasts) => [...toasts, toast]);

    if (duration > 0) {
      setTimeout(() => remove(id), duration);
    }

    return id;
  }

  function remove(id: string) {
    update((toasts) => toasts.filter((t) => t.id !== id));
  }

  return {
    subscribe,
    success: (title: string, message?: string) =>
      add("success", title, message),
    error: (title: string, message?: string) => add("error", title, message),
    info: (title: string, message?: string) => add("info", title, message),
    warning: (title: string, message?: string) =>
      add("warning", title, message),
    remove,
  };
}

export const toast = createToastStore();
