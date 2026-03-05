<script lang="ts">
	let { class: cls = '' }: { class?: string } = $props();
</script>

<div class="animate-pulse rounded bg-[var(--color-muted)] {cls}"></div>
