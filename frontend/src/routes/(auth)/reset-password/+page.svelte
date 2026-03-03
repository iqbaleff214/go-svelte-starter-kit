<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import Button from '$components/ui/Button.svelte';
	import Input from '$components/ui/Input.svelte';
	import { authApi } from '$api/auth';
	import { toast } from '$stores/toast';

	let token = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let loading = $state(false);
	let done = $state(false);
	let passwordError = $state('');

	onMount(() => {
		token = $page.url.searchParams.get('token') ?? '';
		if (!token) {
			toast.error('Invalid link', 'This reset link is missing a token.');
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		passwordError = '';

		if (password !== confirmPassword) {
			passwordError = 'Passwords do not match';
			return;
		}
		if (password.length < 8) {
			passwordError = 'Password must be at least 8 characters';
			return;
		}

		loading = true;
		try {
			await authApi.resetPassword(token, password);
			done = true;
		} catch (err: any) {
			toast.error('Reset failed', err?.message ?? 'The reset link may have expired.');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Reset password — StarterKit</title>
</svelte:head>

<div>
	{#if done}
		<div class="text-center">
			<div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-[var(--color-success)]/10 text-2xl">
				✅
			</div>
			<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Password updated!</h1>
			<p class="mt-2 text-sm text-[var(--color-muted-fg)]">
				Your password has been changed successfully.
			</p>
			<a href="/login" class="mt-6 inline-block text-sm font-medium text-[var(--color-primary)] hover:underline">
				Sign in with your new password
			</a>
		</div>
	{:else}
		<div class="mb-6">
			<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Choose a new password</h1>
			<p class="mt-1 text-sm text-[var(--color-muted-fg)]">
				Enter a strong password for your account.
			</p>
		</div>

		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			<Input
				label="New password"
				type="password"
				placeholder="At least 8 characters"
				autocomplete="new-password"
				required
				bind:value={password}
			/>
			<Input
				label="Confirm new password"
				type="password"
				placeholder="Repeat your new password"
				autocomplete="new-password"
				required
				bind:value={confirmPassword}
				error={passwordError}
			/>

			<Button type="submit" variant="primary" class="mt-2 w-full" {loading} disabled={!token}>
				{loading ? 'Updating…' : 'Update password'}
			</Button>
		</form>

		<p class="mt-6 text-center text-sm text-[var(--color-muted-fg)]">
			<a href="/login" class="font-medium text-[var(--color-primary)] hover:underline">Back to sign in</a>
		</p>
	{/if}
</div>
