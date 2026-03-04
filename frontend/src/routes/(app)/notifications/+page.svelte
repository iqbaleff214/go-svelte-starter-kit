<script lang="ts">
	import { onMount } from 'svelte';
	import { notificationApi } from '$api/notification';
	import { notificationStore, notificationsList } from '$stores/notification';
	import { toast } from '$stores/toast';
	import type { Notification } from '$types';

	let loading = $state(true);
	let loadingMore = $state(false);
	let page = $state(1);
	let total = $state(0);
	const limit = 20;

	const typeBorderClass: Record<Notification['type'], string> = {
		info: 'border-l-[var(--color-primary)]',
		success: 'border-l-[var(--color-success)]',
		warning: 'border-l-amber-400',
		alert: 'border-l-[var(--color-destructive)]'
	};

	onMount(async () => {
		try {
			const res = await notificationApi.list(1, limit);
			notificationStore.setList(res.notifications);
			total = res.total;
			page = 1;
		} catch {
			toast.error('Failed to load', 'Could not fetch notifications.');
		} finally {
			loading = false;
		}
	});

	async function loadMore() {
		loadingMore = true;
		try {
			const nextPage = page + 1;
			const res = await notificationApi.list(nextPage, limit);
			notificationsList.update((list) => [...list, ...res.notifications]);
			total = res.total;
			page = nextPage;
		} catch {
			toast.error('Failed to load', 'Could not load more notifications.');
		} finally {
			loadingMore = false;
		}
	}

	async function handleMarkRead(n: Notification) {
		if (n.read_at) return;
		try {
			await notificationApi.markRead(n.id);
			notificationStore.markRead(n.id);
		} catch {
			toast.error('Error', 'Could not mark notification as read.');
		}
	}

	async function handleMarkAllRead() {
		try {
			await notificationApi.markAllRead();
			notificationStore.markAllRead();
		} catch {
			toast.error('Error', 'Could not mark all as read.');
		}
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<svelte:head>
	<title>Notifications — StarterKit</title>
</svelte:head>

<div class="max-w-2xl">
	<div class="mb-6">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Notifications</h1>
				<p class="mt-1 text-sm text-[var(--color-muted-fg)]">Your activity and system alerts.</p>
			</div>
			{#if $notificationsList.some((n) => !n.read_at)}
				<button
					onclick={handleMarkAllRead}
					class="text-sm font-medium text-[var(--color-primary)] hover:underline"
				>
					Mark all as read
				</button>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="flex justify-center py-16">
			<svg class="h-6 w-6 animate-spin text-[var(--color-primary)]" viewBox="0 0 24 24" fill="none">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
			</svg>
		</div>
	{:else if $notificationsList.length === 0}
		<div class="text-center py-16 text-[var(--color-muted-fg)]">
			<p class="text-4xl mb-3">🔔</p>
			<p class="text-sm">No notifications yet.</p>
		</div>
	{:else}
		<div class="flex flex-col gap-2">
			{#each $notificationsList as n (n.id)}
				<button
					onclick={() => handleMarkRead(n)}
					class="
						w-full text-left rounded-[var(--radius)] border border-[var(--color-border)]
						border-l-4 {typeBorderClass[n.type]}
						bg-[var(--color-card)] px-4 py-3 transition-colors
						{n.read_at ? 'opacity-60' : 'hover:bg-[var(--color-muted)]'}
					"
				>
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<p class="text-sm font-semibold text-[var(--color-foreground)] {n.read_at ? '' : 'font-bold'}">
								{n.title}
							</p>
							{#if n.body}
								<p class="mt-0.5 text-sm text-[var(--color-muted-fg)]">{n.body}</p>
							{/if}
							{#if n.link}
								<a
									href={n.link}
									onclick={(e) => e.stopPropagation()}
									class="mt-1 inline-block text-xs text-[var(--color-primary)] hover:underline"
								>
									View →
								</a>
							{/if}
						</div>
						<div class="flex flex-col items-end gap-1 shrink-0">
							<span class="text-xs text-[var(--color-muted-fg)]">{formatDate(n.created_at)}</span>
							{#if !n.read_at}
								<span class="h-2 w-2 rounded-full bg-[var(--color-primary)]"></span>
							{/if}
						</div>
					</div>
				</button>
			{/each}
		</div>

		{#if $notificationsList.length < total}
			<div class="mt-4 flex justify-center">
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
