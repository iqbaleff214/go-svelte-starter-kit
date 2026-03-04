import { writable, derived } from 'svelte/store';
import type { User } from '$types';

interface AuthState {
	user: User | null;
	accessToken: string | null;
	loading: boolean;
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		user: null,
		accessToken: null,
		loading: true
	});

	return {
		subscribe,

		setAuth(user: User, accessToken: string) {
			// Decode JWT payload to extract roles (browser-side; no signature verification needed)
			try {
				const payload = JSON.parse(atob(accessToken.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
				user = { ...user, roles: payload.roles ?? [] };
			} catch {
				user = { ...user, roles: [] };
			}
			set({ user, accessToken, loading: false });
		},

		clearAuth() {
			set({ user: null, accessToken: null, loading: false });
		},

		setLoading(loading: boolean) {
			update((s) => ({ ...s, loading }));
		},

		updateUser(user: Partial<User>) {
			update((s) => ({
				...s,
				user: s.user ? { ...s.user, ...user } : null
			}));
		}
	};
}

export const authStore = createAuthStore();

export const isAuthenticated = derived(authStore, ($auth) => !!$auth.user);
export const currentUser = derived(authStore, ($auth) => $auth.user);
export const isLoading = derived(authStore, ($auth) => $auth.loading);

// Temporary store for the pre-auth token during 2FA verification
export const preAuthToken = writable<string | null>(null);
