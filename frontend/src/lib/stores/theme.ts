import { writable, get } from 'svelte/store';

export type Theme = 'light' | 'dark' | 'system';

function createThemeStore() {
	const stored =
		typeof localStorage !== 'undefined' ? (localStorage.getItem('theme') as Theme | null) : null;

	const { subscribe, set: _set } = writable<Theme>(stored ?? 'system');

	let mediaQuery: MediaQueryList | null = null;
	let mediaListener: (() => void) | null = null;

	function applyTheme(theme: Theme) {
		const isDark =
			theme === 'dark' ||
			(theme === 'system' &&
				typeof window !== 'undefined' &&
				window.matchMedia('(prefers-color-scheme: dark)').matches);

		document.documentElement.classList.toggle('dark', isDark);
	}

	function watchSystem(theme: Theme) {
		// Remove previous listener
		if (mediaQuery && mediaListener) {
			mediaQuery.removeEventListener('change', mediaListener);
			mediaQuery = null;
			mediaListener = null;
		}
		if (theme === 'system' && typeof window !== 'undefined') {
			mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
			mediaListener = () => applyTheme('system');
			mediaQuery.addEventListener('change', mediaListener);
		}
	}

	return {
		subscribe,

		set(theme: Theme) {
			_set(theme);
			localStorage.setItem('theme', theme);
			applyTheme(theme);
			watchSystem(theme);
		},

		/** Call once on app mount to apply the persisted/system theme. */
		init() {
			const t = (localStorage.getItem('theme') as Theme | null) ?? 'system';
			_set(t);
			applyTheme(t);
			watchSystem(t);
		},

		cycle() {
			const current: Theme = get({ subscribe });
			const next: Theme = current === 'light' ? 'dark' : current === 'dark' ? 'system' : 'light';
			this.set(next);
		}
	};
}

export const theme = createThemeStore();
