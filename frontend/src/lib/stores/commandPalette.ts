import { writable } from 'svelte/store';

const { subscribe, update, set } = writable(false);

export const commandPalette = {
	subscribe,
	open: () => set(true),
	close: () => set(false),
	toggle: () => update((v) => !v)
};
