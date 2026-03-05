<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi } from '$api/admin';
	import { toast } from '$stores/toast';
	import type { AuditLog } from '$types';

	let loading = $state(true);
	let logs = $state<AuditLog[]>([]);
	let total = $state(0);
	let page = $state(1);
	let loadingMore = $state(false);
	const limit = 20;

	const actionLabels: Record<string, string> = {
		'role.assign': 'Role Assigned',
		'role.revoke': 'Role Revoked',
		'role.create': 'Role Created',
		'role.update': 'Role Updated',
		'role.delete': 'Role Deleted',
		'role.set_permissions': 'Permissions Updated',
		'user.delete': 'User Deleted',
	};

	const actionColors: Record<string, string> = {
		'role.assign': 'bg-[var(--color-success)]/10 text-[var(--color-success)]',
		'role.revoke': 'bg-amber-100 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400',
		'role.create': 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]',
		'role.update': 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]',
		'role.delete': 'bg-[var(--color-destructive)]/10 text-[var(--color-destructive)]',
		'role.set_permissions': 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]',
		'user.delete': 'bg-[var(--color-destructive)]/10 text-[var(--color-destructive)]',
	};

	onMount(async () => {
		try {
			const res = await adminApi.listAuditLogs(1, limit);
			logs = res.logs;
			total = res.total;
		} catch {
			toast.error('Failed to load', 'Could not fetch audit logs.');
		} finally {
			loading = false;
		}
	});

	async function loadMore() {
		loadingMore = true;
		try {
			const nextPage = page + 1;
			const res = await adminApi.listAuditLogs(nextPage, limit);
			logs = [...logs, ...res.logs];
			total = res.total;
			page = nextPage;
		} catch {
			toast.error('Error', 'Could not load more logs.');
		} finally {
			loadingMore = false;
		}
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleString(undefined, {
			month: 'short', day: 'numeric',
			hour: '2-digit', minute: '2-digit'
		});
	}

	function formatMetadata(meta: Record<string, unknown>): string {
		return Object.entries(meta)
			.map(([k, v]) => `${k}: ${v}`)
			.join(' · ');
	}
</script>

<svelte:head>
	<title>Audit Logs — Admin — StarterKit</title>
</svelte:head>

<div class="flex items-center justify-between mb-4">
	<span class="text-sm text-[var(--color-muted-fg)]">{total} event{total !== 1 ? 's' : ''}</span>
</div>

<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] overflow-hidden">
	{#if loading}
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Action</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Actor</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Details</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Time</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-[var(--color-border)]">
				{#each Array.from({ length: 5 }) as _, i (i)}
					<tr class="bg-[var(--color-card)]">
						<td class="px-4 py-3"><div class="animate-pulse h-5 w-32 rounded-full bg-[var(--color-muted)]"></div></td>
						<td class="px-4 py-3"><div class="animate-pulse h-3 rounded bg-[var(--color-muted)]" style="width: {i % 2 === 0 ? '140px' : '110px'}"></div></td>
						<td class="px-4 py-3"><div class="animate-pulse h-3 w-24 rounded bg-[var(--color-muted)]"></div></td>
						<td class="px-4 py-3"><div class="animate-pulse h-3 w-20 rounded bg-[var(--color-muted)]"></div></td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else if logs.length === 0}
		<div class="text-center py-16 text-[var(--color-muted-fg)] text-sm">No audit events yet.</div>
	{:else}
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Action</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Actor</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Details</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Time</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-[var(--color-border)]">
				{#each logs as log (log.id)}
					<tr class="bg-[var(--color-card)] hover:bg-[var(--color-muted)]/20 transition-colors">
						<td class="px-4 py-3">
							<span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {actionColors[log.action] ?? 'bg-[var(--color-muted)] text-[var(--color-muted-fg)]'}">
								{actionLabels[log.action] ?? log.action}
							</span>
						</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)] truncate max-w-[180px]">{log.actor_email}</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)] text-xs font-mono">
							{#if Object.keys(log.metadata).length > 0}
								{formatMetadata(log.metadata)}
							{:else}
								—
							{/if}
						</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)] whitespace-nowrap">{formatDate(log.created_at)}</td>
					</tr>
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
