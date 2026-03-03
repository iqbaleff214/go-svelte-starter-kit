<script lang="ts">
	interface Props {
		label?: string;
		type?: string;
		placeholder?: string;
		value?: string;
		error?: string;
		disabled?: boolean;
		required?: boolean;
		autocomplete?: string;
		inputmode?: 'none' | 'text' | 'decimal' | 'numeric' | 'tel' | 'search' | 'email' | 'url';
		pattern?: string;
		maxlength?: number;
		class?: string;
		id?: string;
		oninput?: (e: Event) => void;
		onchange?: (e: Event) => void;
	}

	let {
		label,
		type = 'text',
		placeholder,
		value = $bindable(''),
		error,
		disabled = false,
		required = false,
		autocomplete,
		inputmode,
		pattern,
		maxlength,
		class: className = '',
		id,
		oninput,
		onchange
	}: Props = $props();

	const inputId = id ?? `input-${Math.random().toString(36).slice(2)}`;
</script>

<div class="flex flex-col gap-1.5 {className}">
	{#if label}
		<label for={inputId} class="text-sm font-medium text-[var(--color-foreground)]">
			{label}
			{#if required}
				<span class="text-[var(--color-destructive)] ml-0.5">*</span>
			{/if}
		</label>
	{/if}

	<input
		{type}
		{placeholder}
		{disabled}
		{required}
		autocomplete={autocomplete as any}
		{inputmode}
		{pattern}
		{maxlength}
		id={inputId}
		bind:value
		{oninput}
		{onchange}
		class="
			h-10 w-full rounded-[var(--radius-sm)] border px-3 text-sm
			bg-[var(--color-background)] text-[var(--color-foreground)]
			placeholder:text-[var(--color-muted-fg)]
			transition-colors
			focus:outline-2 focus:outline-[var(--color-ring)] focus:outline-offset-0
			disabled:cursor-not-allowed disabled:opacity-50
			{error
			? 'border-[var(--color-destructive)]'
			: 'border-[var(--color-input)] hover:border-[var(--color-muted-fg)]'}
		"
	/>

	{#if error}
		<p class="text-xs text-[var(--color-destructive)]">{error}</p>
	{/if}
</div>
