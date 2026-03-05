<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { authApi } from '$api/auth';

	type VerifyStatus = 'loading' | 'success' | 'error';
	let status: VerifyStatus = $state('loading');
	let errorMsg = $state('');

	onMount(async () => {
		const token = $page.url.searchParams.get('token') ?? '';
		if (!token) {
			errorMsg = 'No verification token found in the link.';
			status = 'error';
			return;
		}

		try {
			await authApi.verifyEmail(token);
			status = 'success';
		} catch (err: any) {
			errorMsg = err?.message ?? 'The verification link is invalid or has expired.';
			status = 'error';
		}
	});
</script>

<svelte:head>
	<title>Verify email — StarterKit</title>
</svelte:head>

<div class="text-center">
	{#if status === 'loading'}
		<svg class="mx-auto mb-4 h-10 w-10 animate-spin text-[var(--color-primary)]" viewBox="0 0 24 24" fill="none">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
			<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
		</svg>
		<p class="text-sm text-[var(--color-muted-fg)]">Verifying your email address…</p>

	{:else if status === 'success'}
		<div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-[var(--color-success)]/10 text-2xl">
			✅
		</div>
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Email verified!</h1>
		<p class="mt-2 text-sm text-[var(--color-muted-fg)]">
			Your email address has been verified successfully.
		</p>
		<a href="/dashboard" class="mt-6 inline-block text-sm font-medium text-[var(--color-primary)] hover:underline">
			Go to dashboard
		</a>

	{:else}
		<div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-[var(--color-destructive)]/10 text-2xl">
			❌
		</div>
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Verification failed</h1>
		<p class="mt-2 text-sm text-[var(--color-muted-fg)]">{errorMsg}</p>
		<a href="/login" class="mt-6 inline-block text-sm font-medium text-[var(--color-primary)] hover:underline">
			Back to sign in
		</a>
	{/if}
</div>
