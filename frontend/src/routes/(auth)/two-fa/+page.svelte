<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import Button from '$components/ui/Button.svelte';
	import Input from '$components/ui/Input.svelte';
	import { authApi } from '$api/auth';
	import { authStore, preAuthToken } from '$stores/auth';
	import { toast } from '$stores/toast';
	import type { ApiError } from '$types';

	let token = $state('');
	let code = $state('');
	let backupCode = $state('');
	let useBackup = $state(false);
	let loading = $state(false);
	let error = $state('');

	onMount(() => {
		const unsubscribe = preAuthToken.subscribe((t) => {
			if (!t) {
				goto('/login');
				return;
			}
			token = t;
		});
		return unsubscribe;
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		loading = true;

		try {
			const res = await authApi.twoFaVerify(
				token,
				useBackup ? undefined : code || undefined,
				useBackup ? backupCode || undefined : undefined
			);
			preAuthToken.set(null);
			authStore.setAuth(res.user, res.token.access_token);
			toast.success('Welcome back!', `Signed in as ${res.user.display_name}`);
			goto('/dashboard');
		} catch (err) {
			const apiErr = err as ApiError;
			error = apiErr.message ?? 'Verification failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Two-factor verification — StarterKit</title>
</svelte:head>

<div>
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Two-factor verification</h1>
		<p class="mt-1 text-sm text-[var(--color-muted-fg)]">
			{useBackup
				? 'Enter one of your backup codes to continue.'
				: 'Enter the 6-digit code from your authenticator app.'}
		</p>
	</div>

	<form onsubmit={handleSubmit} class="flex flex-col gap-4">
		{#if useBackup}
			<Input
				label="Backup code"
				type="text"
				placeholder="XXXX-XXXX-XX"
				autocomplete="one-time-code"
				bind:value={backupCode}
				{error}
			/>
		{:else}
			<Input
				label="Authentication code"
				type="text"
				inputmode="numeric"
				pattern="[0-9]*"
				maxlength={6}
				placeholder="000000"
				autocomplete="one-time-code"
				bind:value={code}
				{error}
			/>
		{/if}

		<Button type="submit" variant="primary" class="w-full" {loading}>
			{loading ? 'Verifying…' : 'Verify'}
		</Button>
	</form>

	<button
		type="button"
		onclick={() => { useBackup = !useBackup; code = ''; backupCode = ''; error = ''; }}
		class="mt-4 w-full text-center text-sm text-[var(--color-primary)] hover:underline"
	>
		{useBackup ? 'Use authenticator app instead' : 'Use a backup code instead'}
	</button>

	<p class="mt-4 text-center text-sm text-[var(--color-muted-fg)]">
		<a href="/login" class="text-[var(--color-primary)] hover:underline">Back to sign in</a>
	</p>
</div>
