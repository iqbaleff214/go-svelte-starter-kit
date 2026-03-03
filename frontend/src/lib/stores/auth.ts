import { writable, derived } from 'svelte/store';
import type { User } from '$types';
import { api } from '$api/client';

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
			api.setAccessToken(accessToken);
			set({ user, accessToken, loading: false });
		},

		clearAuth() {
			api.setAccessToken(null);
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
