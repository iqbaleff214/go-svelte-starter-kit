<script lang="ts">
	import Button from '$components/ui/Button.svelte';
	import Input from '$components/ui/Input.svelte';
	import { toast } from '$stores/toast';

	let email = $state('');
	let loading = $state(false);
	let sent = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		try {
			// TODO: wire to POST /api/auth/forgot-password in Phase 2
			await new Promise((r) => setTimeout(r, 800));
			sent = true;
		} catch {
			toast.error('Something went wrong', 'Please try again later.');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Forgot password — StarterKit</title>
</svelte:head>

<div>
	{#if sent}
		<div class="text-center">
			<div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-[var(--color-success)]/10 text-2xl">
				✉️
			</div>
			<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Check your email</h1>
			<p class="mt-2 text-sm text-[var(--color-muted-fg)]">
				If <strong>{email}</strong> is associated with an account, we've sent a password reset link.
			</p>
			<a href="/login" class="mt-6 inline-block text-sm text-[var(--color-primary)] hover:underline">
				Back to sign in
			</a>
		</div>
	{:else}
		<div class="mb-6">
			<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Forgot your password?</h1>
			<p class="mt-1 text-sm text-[var(--color-muted-fg)]">
				Enter your email and we'll send you a reset link.
			</p>
		</div>

		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			<Input
				label="Email address"
				type="email"
				placeholder="you@example.com"
				autocomplete="email"
				required
				bind:value={email}
			/>

			<Button type="submit" variant="primary" class="mt-2 w-full" {loading}>
				{loading ? 'Sending…' : 'Send reset link'}
			</Button>
		</form>

		<p class="mt-6 text-center text-sm text-[var(--color-muted-fg)]">
			Remember your password?
			<a href="/login" class="font-medium text-[var(--color-primary)] hover:underline">Sign in</a>
		</p>
	{/if}
</div>
