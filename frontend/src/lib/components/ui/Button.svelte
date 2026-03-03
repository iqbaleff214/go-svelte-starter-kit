<script lang="ts">
	interface Props {
		variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'destructive';
		size?: 'sm' | 'md' | 'lg';
		type?: 'button' | 'submit' | 'reset';
		loading?: boolean;
		disabled?: boolean;
		class?: string;
		onclick?: (e: MouseEvent) => void;
	}

	let {
		variant = 'primary',
		size = 'md',
		type = 'button',
		loading = false,
		disabled = false,
		class: className = '',
		onclick,
		children
	}: Props & { children?: import('svelte').Snippet } = $props();

	const base =
		'inline-flex items-center justify-center gap-2 font-medium rounded-[var(--radius)] transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 disabled:pointer-events-none disabled:opacity-50 cursor-pointer select-none';

	const variants: Record<string, string> = {
		primary:
			'bg-[var(--color-primary)] text-[var(--color-primary-fg)] hover:bg-[var(--color-primary-hover)] focus-visible:outline-[var(--color-ring)]',
		secondary:
			'bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)]',
		outline:
			'border border-[var(--color-border)] bg-transparent text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
		ghost: 'bg-transparent text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
		destructive:
			'bg-[var(--color-destructive)] text-[var(--color-destructive-fg)] hover:opacity-90'
	};

	const sizes: Record<string, string> = {
		sm: 'h-8 px-3 text-sm',
		md: 'h-10 px-4 text-sm',
		lg: 'h-12 px-6 text-base'
	};
</script>

<button
	{type}
	{onclick}
	disabled={disabled || loading}
	class="{base} {variants[variant]} {sizes[size]} {className}"
>
	{#if loading}
		<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
			/>
		</svg>
	{/if}
	{#if children}
		{@render children()}
	{/if}
</button>
