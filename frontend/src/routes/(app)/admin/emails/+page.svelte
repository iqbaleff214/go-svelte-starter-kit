<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi } from '$api/admin';
	import { toast } from '$stores/toast';
	import type { EmailLog } from '$types';

	let loading = $state(true);
	let logs = $state<EmailLog[]>([]);
	let total = $state(0);
	let page = $state(1);
	let loadingMore = $state(false);
	const limit = 20;

	const statusClasses: Record<string, string> = {
		queued: 'bg-[var(--color-muted)] text-[var(--color-muted-fg)]',
		sent: 'bg-[var(--color-success)]/10 text-[var(--color-success)]',
		failed: 'bg-amber-100 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400',
		dead: 'bg-[var(--color-destructive)]/10 text-[var(--color-destructive)]'
	};

	onMount(async () => {
		try {
			const res = await adminApi.listEmailLogs(1, limit);
			logs = res.logs;
			total = res.total;
		} catch {
			toast.error('Failed to load', 'Could not fetch email logs.');
		} finally {
			loading = false;
		}
	});

	async function loadMore() {
		loadingMore = true;
		try {
			const nextPage = page + 1;
			const res = await adminApi.listEmailLogs(nextPage, limit);
			logs = [...logs, ...res.logs];
			total = res.total;
			page = nextPage;
		} catch {
			toast.error('Error', 'Could not load more logs.');
		} finally {
			loadingMore = false;
		}
	}

	function formatDate(iso: string | null) {
		if (!iso) return '—';
		return new Date(iso).toLocaleString(undefined, {
			month: 'short', day: 'numeric',
			hour: '2-digit', minute: '2-digit'
		});
	}
</script>

<svelte:head>
	<title>Email Logs — Admin — StarterKit</title>
</svelte:head>

<div class="flex items-center justify-between mb-4">
	<span class="text-sm text-[var(--color-muted-fg)]">{total} log{total !== 1 ? 's' : ''}</span>
</div>

<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] overflow-hidden">
	{#if loading}
		<div class="flex justify-center py-16">
			<svg class="h-6 w-6 animate-spin text-[var(--color-primary)]" viewBox="0 0 24 24" fill="none">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
			</svg>
		</div>
	{:else if logs.length === 0}
		<div class="text-center py-16 text-[var(--color-muted-fg)] text-sm">No email logs found.</div>
	{:else}
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Template</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Recipient</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Status</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Attempts</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Sent at</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Created</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-[var(--color-border)]">
				{#each logs as log (log.id)}
					<tr class="bg-[var(--color-card)] hover:bg-[var(--color-muted)]/20 transition-colors">
						<td class="px-4 py-3 font-medium text-[var(--color-foreground)]">{log.template}</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)] truncate max-w-[200px]">{log.recipient}</td>
						<td class="px-4 py-3">
							<span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {statusClasses[log.status] ?? statusClasses.queued}">
								{log.status}
							</span>
						</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)]">{log.attempts}</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)] whitespace-nowrap">{formatDate(log.sent_at)}</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)] whitespace-nowrap">{formatDate(log.created_at)}</td>
					</tr>
					{#if log.error}
						<tr class="bg-[var(--color-destructive)]/5">
							<td colspan="6" class="px-4 py-2 text-xs text-[var(--color-destructive)] font-mono">{log.error}</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>

		{#if logs.length < total}
			<div class="flex justify-center py-4 border-t border-[var(--color-border)]">
				<button
					onclick={loadMore}
					disabled={loadingMore}
					class="text-sm font-medium text-[var(--color-primary)] hover:underline disabled:opacity-50"
				>
					{loadingMore ? 'Loading…' : 'Load more'}
				</button>
			</div>
		{/if}
	{/if}
</div>
