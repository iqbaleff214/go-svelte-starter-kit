<script lang="ts">
	import { toast } from '$stores/toast';
	import { fly } from 'svelte/transition';
	import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-svelte';

	const icons = {
		success: CheckCircle,
		error: AlertCircle,
		info: Info,
		warning: AlertTriangle
	};

	const colors = {
		success: 'border-[var(--color-success)] text-[var(--color-success)]',
		error: 'border-[var(--color-destructive)] text-[var(--color-destructive)]',
		info: 'border-[var(--color-primary)] text-[var(--color-primary)]',
		warning: 'border-[var(--color-warning)] text-[var(--color-warning)]'
	};
</script>

<div
	aria-live="polite"
	class="fixed top-4 right-4 z-50 flex flex-col gap-2 w-full max-w-sm pointer-events-none"
>
	{#each $toast as t (t.id)}
		<div
			role="alert"
			transition:fly={{ x: 20, duration: 200 }}
			class="
				pointer-events-auto flex items-start gap-3 rounded-[var(--radius)]
				border-l-4 bg-[var(--color-card)] p-4
				shadow-[var(--shadow-lg)] {colors[t.type]}
			"
		>
			<svelte:component this={icons[t.type]} class="h-5 w-5 mt-0.5 shrink-0" />
			<div class="flex-1 min-w-0">
				<p class="text-sm font-medium text-[var(--color-foreground)]">{t.title}</p>
				{#if t.message}
					<p class="text-xs text-[var(--color-muted-fg)] mt-0.5">{t.message}</p>
				{/if}
			</div>
			<button
				onclick={() => toast.remove(t.id)}
				class="shrink-0 p-0.5 rounded hover:bg-[var(--color-muted)] text-[var(--color-muted-fg)] transition-colors"
				aria-label="Dismiss"
			>
				<X class="h-4 w-4" />
			</button>
		</div>
	{/each}
</div>
