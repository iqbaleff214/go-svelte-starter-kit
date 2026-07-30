<script lang="ts">
	import '../app.css';
	import Toast from '$components/ui/Toast.svelte';
	import { onMount } from 'svelte';
	import { authStore } from '$stores/auth';
	import { authApi } from '$api/auth';
	import { api } from '$api/client';
	import { theme } from '$stores/theme';
	import { goto } from '$app/navigation';

	let { children }: { children: import('svelte').Snippet } = $props();

	// Keep the API client's Bearer token in sync with the auth store.
	// auth.ts no longer imports api/client directly — this breaks the chunk
	// circular dependency that caused "Cannot access before initialization".
	authStore.subscribe((state) => api.setAccessToken(state.accessToken));

	// On any 401, clear auth state and boot user to login.
	api.setOnUnauthorized(() => {
		authStore.clearAuth();
		const path = window.location.pathname;
		const isAuthPage = path === '/login' || path === '/register' || path.startsWith('/forgot-password') || path.startsWith('/reset-password');
		if (!isAuthPage) {
			goto('/login');
		}
	});

	onMount(async () => {
		// Apply persisted/system theme before any rendering.
		theme.init();

		// Attempt silent token refresh.
		try {
			const res = await authApi.refresh();
			authStore.setAuth(res.user, res.token.access_token);
		} catch {
			authStore.clearAuth();
		}
	});
</script>

{@render children()}
<Toast />
