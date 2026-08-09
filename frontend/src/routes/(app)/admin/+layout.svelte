<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { currentUser, isLoading } from '$stores/auth';
	import { onMount } from 'svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	const tabs = [
		{ href: '/admin', label: 'Overview' },
		{ href: '/admin/users', label: 'Users' },
		{ href: '/admin/roles', label: 'Roles' },
		{ href: '/admin/audit-logs', label: 'Audit Logs' },
		{ href: '/admin/emails', label: 'Email Logs' },
		{ href: '/admin/whatsapp', label: 'WhatsApp', superadminOnly: true }
	];

	onMount(() => {
		let unsub: () => void;
		unsub = isLoading.subscribe((loading) => {
			if (!loading) {
				unsub?.();
				const roles = $currentUser?.roles ?? [];
				if (!roles.includes('admin') && !roles.includes('superadmin')) {
					goto('/dashboard');
				}
			}
		});
	});

	function isActiveTab(href: string) {
		if (href === '/admin') return $page.url.pathname === '/admin';
		return $page.url.pathname.startsWith(href);
	}
</script>

<div class="max-w-6xl">
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-[var(--color-foreground)] mb-4">Admin Panel</h1>
		<div class="flex gap-1 border-b border-[var(--color-border)]">
			{#each tabs as tab}
				{#if !tab.superadminOnly || $currentUser?.roles?.includes('superadmin')}
					<a
						href={tab.href}
						class="
							px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors
							{isActiveTab(tab.href)
								? 'border-[var(--color-primary)] text-[var(--color-primary)]'
								: 'border-transparent text-[var(--color-muted-fg)] hover:text-[var(--color-foreground)]'}
						"
					>
						{tab.label}
					</a>
				{/if}
			{/each}
		</div>
	</div>

	{@render children()}
</div>
