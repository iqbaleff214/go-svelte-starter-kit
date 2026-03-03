<script lang="ts">
	import '../app.css';
	import Toast from '$components/ui/Toast.svelte';
	import { onMount } from 'svelte';
	import { authStore } from '$stores/auth';
	import { authApi } from '$api/auth';

	let { children }: { children: import('svelte').Snippet } = $props();

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
