<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authApi } from '$api/auth';
	import { authStore } from '$stores/auth';
	import { toast } from '$stores/toast';

	onMount(async () => {
		const code = $page.url.searchParams.get('code');
		const error = $page.url.searchParams.get('error');

		if (error || !code) {
			toast.error('Google sign-in failed', 'Please try again.');
			goto('/login');
			return;
		}

		try {
			const res = await authApi.googleExchange(code);
			authStore.setAuth(res.user, res.token.access_token);
			toast.success('Welcome!', `Signed in as ${res.user.display_name}`);
			goto('/dashboard');
		} catch {
			toast.error('Google sign-in failed', 'Could not complete sign-in. Please try again.');
			goto('/login');
		}
	});
</script>

<svelte:head>
	<title>Signing in… — StarterKit</title>
</svelte:head>

<div class="flex flex-col items-center gap-4 py-8">
	<svg class="h-8 w-8 animate-spin text-[var(--color-primary)]" viewBox="0 0 24 24" fill="none">
		<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
		<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
	</svg>
	<p class="text-sm text-[var(--color-muted-fg)]">Completing sign-in…</p>
</div>
