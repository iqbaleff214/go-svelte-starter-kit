<script lang="ts">
	import '../app.css';
	import Toast from '$components/ui/Toast.svelte';
	import { onMount } from 'svelte';
	import { authStore } from '$stores/auth';
	import { authApi } from '$api/auth';
	import { api } from '$api/client';

	let { children }: { children: import('svelte').Snippet } = $props();

	// Keep the API client's Bearer token in sync with the auth store.
	// auth.ts no longer imports api/client directly — this breaks the chunk
	// circular dependency that caused "Cannot access before initialization".
	authStore.subscribe((state) => api.setAccessToken(state.accessToken));

	// Attempt silent token refresh on mount
	onMount(async () => {
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
