import { writable } from "svelte/store";
import type { Notification } from "$types";

export const unreadCount = writable<number>(0);
export const notificationsList = writable<Notification[]>([]);

export const notificationStore = {
  setCount(n: number) {
    unreadCount.set(n);
  },

  setList(ns: Notification[]) {
    notificationsList.set(ns);
  },

  /** Prepend a new notification (received via WebSocket) and increment unread count. */
  add(n: Notification) {
    notificationsList.update((list) => [n, ...list]);
    if (!n.read_at) {
      unreadCount.update((c) => c + 1);
    }
  },

  /** Mark a single notification as read and decrement unread count if it was unread. */
  markRead(id: string) {
    notificationsList.update((list) =>
      list.map((n) => {
        if (n.id === id && !n.read_at) {
          unreadCount.update((c) => Math.max(0, c - 1));
          return { ...n, read_at: new Date().toISOString() };
        }
        return n;
      }),
    );
  },

  /** Mark all notifications as read and zero out the badge. */
  markAllRead() {
    const now = new Date().toISOString();
    notificationsList.update((list) =>
      list.map((n) => (n.read_at ? n : { ...n, read_at: now })),
    );
    unreadCount.set(0);
  },
};
