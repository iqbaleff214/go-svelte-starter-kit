<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi } from '$api/admin';
	import { toast } from '$stores/toast';
	import { Users, Shield, Lock, Mail } from 'lucide-svelte';

	let loading = $state(true);
	let stats = $state({ users: 0, roles: 0, permissions: 0, emails: 0 });

	onMount(async () => {
		try {
			const [usersRes, rolesRes, permsRes, emailsRes] = await Promise.all([
				adminApi.listUsers(1, 1),
				adminApi.listRoles(),
				adminApi.listPermissions(),
				adminApi.listEmailLogs(1, 1)
			]);
			stats = {
				users: usersRes.total,
				roles: rolesRes.length,
				permissions: permsRes.length,
				emails: emailsRes.total
			};
		} catch {
			toast.error('Failed to load', 'Could not fetch admin stats.');
		} finally {
			loading = false;
		}
	});

	const cards = [
		{ key: 'users' as const, label: 'Total Users', icon: Users, href: '/admin/users', color: 'text-blue-600 bg-blue-50 dark:bg-blue-900/20 dark:text-blue-400' },
		{ key: 'roles' as const, label: 'Roles', icon: Shield, href: '/admin/roles', color: 'text-purple-600 bg-purple-50 dark:bg-purple-900/20 dark:text-purple-400' },
		{ key: 'permissions' as const, label: 'Permissions', icon: Lock, href: '/admin/roles', color: 'text-green-600 bg-green-50 dark:bg-green-900/20 dark:text-green-400' },
		{ key: 'emails' as const, label: 'Email Logs', icon: Mail, href: '/admin/emails', color: 'text-amber-600 bg-amber-50 dark:bg-amber-900/20 dark:text-amber-400' }
	];
</script>

<svelte:head>
	<title>Admin — StarterKit</title>
</svelte:head>

<div>
	<p class="text-sm text-[var(--color-muted-fg)] mb-6">System overview and quick access to admin tools.</p>

	{#if loading}
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
			{#each Array(4) as _}
				<div class="rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] p-5 animate-pulse">
					<div class="h-8 w-8 rounded-[var(--radius)] bg-[var(--color-muted)] mb-3"></div>
					<div class="h-7 w-12 bg-[var(--color-muted)] rounded mb-1"></div>
					<div class="h-4 w-20 bg-[var(--color-muted)] rounded"></div>
				</div>
			{/each}
		</div>
	{:else}
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
			{#each cards as card}
				<a
					href={card.href}
					class="group rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] p-5 hover:shadow-[var(--shadow-md)] transition-shadow"
				>
					<div class="flex h-8 w-8 items-center justify-center rounded-[var(--radius)] {card.color} mb-3">
						<card.icon class="h-4 w-4" />
					</div>
					<p class="text-2xl font-bold text-[var(--color-foreground)]">{stats[card.key]}</p>
					<p class="text-xs text-[var(--color-muted-fg)] mt-0.5">{card.label}</p>
				</a>
			{/each}
		</div>
	{/if}
</div>
