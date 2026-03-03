<script lang="ts">
	import { goto } from '$app/navigation';
	import { isAuthenticated, isLoading } from '$stores/auth';
	import { onMount } from 'svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	onMount(() => {
		// Redirect to dashboard if already authenticated
		const unsubscribe = isLoading.subscribe((loading) => {
			if (!loading) {
				const authed = $isAuthenticated;
				if (authed) goto('/dashboard');
				unsubscribe();
			}
		});
	});
</script>

<div class="min-h-screen bg-[var(--color-muted)] flex flex-col items-center justify-center p-4">
	<!-- Logo -->
	<a href="/" class="mb-8 flex items-center gap-2 text-xl font-bold text-[var(--color-foreground)]">
		<span class="text-[var(--color-primary)]">⚡</span>
		StarterKit
	</a>

	<!-- Card -->
	<div class="w-full max-w-md rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-8 shadow-[var(--shadow-md)]">
		{@render children()}
	</div>

	<p class="mt-6 text-xs text-[var(--color-muted-fg)]">
		© {new Date().getFullYear()} StarterKit. All rights reserved.
	</p>
</div>
