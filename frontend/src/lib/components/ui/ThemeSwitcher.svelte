<script lang="ts">
	import { Sun, Moon, Monitor } from 'lucide-svelte';
	import { theme } from '$stores/theme';

	const labels: Record<string, string> = {
		light: 'Light',
		dark: 'Dark',
		system: 'System'
	};

	const icons: Record<string, typeof Sun> = {
		light: Sun,
		dark: Moon,
		system: Monitor
	};
</script>

<button
	onclick={() => theme.cycle()}
	title="Theme: {labels[$theme]} (click to switch)"
	class="shrink-0 rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
>
	<svelte:component this={icons[$theme]} class="h-4 w-4" />
</button>
