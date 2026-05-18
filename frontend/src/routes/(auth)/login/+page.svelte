<script lang="ts">
	import Button from '$components/ui/Button.svelte';
	import Input from '$components/ui/Input.svelte';
	import { authApi } from '$api/auth';
	import { authStore, preAuthToken } from '$stores/auth';
	import { toast } from '$stores/toast';
	import { goto } from '$app/navigation';
	import type { ApiError } from '$types';

	let email = $state('');
	let password = $state('');
	let loading = $state(false);
	let errors = $state<Record<string, string>>({});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		errors = {};
		loading = true;

		try {
			const res = await authApi.login({ email, password });
			if (res.two_fa_required && res.pre_auth_token) {
				preAuthToken.set(res.pre_auth_token);
				goto('/two-fa');
			} else {
				authStore.setAuth(res.user, res.token.access_token);
				toast.success('Welcome back!', `Signed in as ${res.user.display_name}`);
				goto('/dashboard');
			}
		} catch (err) {
			const apiErr = err as ApiError;
			if (apiErr.details) {
				errors = Object.fromEntries(apiErr.details.map((e) => [e.field, e.message]));
			} else {
				toast.error('Sign in failed', apiErr.message);
			}
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Sign in — StarterKit</title>
</svelte:head>

<div>
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Welcome back</h1>
		<p class="mt-1 text-sm text-[var(--color-muted-fg)]">Sign in to your account to continue.</p>
	</div>

	<form onsubmit={handleSubmit} class="flex flex-col gap-4">
		<Input
			label="Email address"
			type="email"
			placeholder="you@example.com"
			autocomplete="email"
			required
			bind:value={email}
			error={errors.email}
		/>

		<div class="flex flex-col gap-1.5">
			<div class="flex items-center justify-between">
				<label class="text-sm font-medium text-[var(--color-foreground)]" for="password">
					Password <span class="text-[var(--color-destructive)]">*</span>
				</label>
				<a
					href="/forgot-password"
					class="text-xs text-[var(--color-primary)] hover:underline"
				>
					Forgot password?
				</a>
			</div>
			<Input
				id="password"
				type="password"
				placeholder="••••••••"
				autocomplete="current-password"
				required
				bind:value={password}
				error={errors.password}
			/>
		</div>

		<Button type="submit" variant="primary" class="mt-2 w-full" {loading}>
			{loading ? 'Signing in…' : 'Sign in'}
		</Button>
	</form>

	<div class="relative my-6">
		<div class="absolute inset-0 flex items-center">
			<div class="w-full border-t border-[var(--color-border)]"></div>
		</div>
		<div class="relative flex justify-center text-xs text-[var(--color-muted-fg)]">
			<span class="bg-[var(--color-card)] px-2">or continue with</span>
		</div>
	</div>

	<a
		href="/api/auth/google"
		class="flex w-full items-center justify-center gap-2 rounded-[var(--radius)] border border-[var(--color-border)] bg-transparent px-4 py-2.5 text-sm font-medium text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
	>
		<svg class="h-4 w-4" viewBox="0 0 24 24" aria-hidden="true">
			<path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
			<path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
			<path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
			<path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
		</svg>
		Continue with Google
	</a>

	<p class="mt-6 text-center text-sm text-[var(--color-muted-fg)]">
		Don't have an account?
		<a href="/register" class="font-medium text-[var(--color-primary)] hover:underline">Create one</a>
	</p>
</div>
