<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi } from '$api/admin';
	import { currentUser } from '$stores/auth';
	import { toast } from '$stores/toast';
	import type { AdminUser, Role } from '$types';
	import { Trash2, X } from 'lucide-svelte';

	let loading = $state(true);
	let users = $state<AdminUser[]>([]);
	let total = $state(0);
	let page = $state(1);
	const limit = 20;

	let roles = $state<Role[]>([]);
	let roleFilter = $state('');

	// Assign role modal state
	let assignTarget = $state<AdminUser | null>(null);
	let assignRoleId = $state('');
	let assignLoading = $state(false);

	// Delete confirmation
	let deleteTarget = $state<AdminUser | null>(null);
	let deleteLoading = $state(false);

	onMount(async () => {
		try {
			const [usersRes, rolesRes] = await Promise.all([
				adminApi.listUsers(1, limit),
				adminApi.listRoles()
			]);
			users = usersRes.users;
			total = usersRes.total;
			roles = rolesRes;
		} catch {
			toast.error('Failed to load', 'Could not fetch users.');
		} finally {
			loading = false;
		}
	});

	async function applyFilter() {
		loading = true;
		page = 1;
		try {
			const res = await adminApi.listUsers(1, limit, roleFilter);
			users = res.users;
			total = res.total;
		} catch {
			toast.error('Error', 'Could not filter users.');
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		const nextPage = page + 1;
		try {
			const res = await adminApi.listUsers(nextPage, limit, roleFilter);
			users = [...users, ...res.users];
			total = res.total;
			page = nextPage;
		} catch {
			toast.error('Error', 'Could not load more users.');
		}
	}

	async function handleAssignRole() {
		if (!assignTarget || !assignRoleId) return;
		assignLoading = true;
		try {
			await adminApi.assignRole(assignTarget.id, assignRoleId);
			const role = roles.find((r) => r.id === assignRoleId);
			if (role) {
				users = users.map((u) =>
					u.id === assignTarget!.id
						? { ...u, roles: [...new Set([...u.roles, role.name])] }
						: u
				);
			}
			toast.success('Role assigned', `Role added to ${assignTarget.display_name}.`);
			assignTarget = null;
			assignRoleId = '';
		} catch {
			toast.error('Error', 'Could not assign role.');
		} finally {
			assignLoading = false;
		}
	}

	async function handleRevokeRole(user: AdminUser, roleName: string) {
		const role = roles.find((r) => r.name === roleName);
		if (!role) return;
		try {
			await adminApi.revokeRole(user.id, role.id);
			users = users.map((u) =>
				u.id === user.id ? { ...u, roles: u.roles.filter((r) => r !== roleName) } : u
			);
			toast.success('Role revoked', `Removed ${roleName} from ${user.display_name}.`);
		} catch {
			toast.error('Error', 'Could not revoke role.');
		}
	}

	async function handleDeleteUser() {
		if (!deleteTarget) return;
		deleteLoading = true;
		try {
			await adminApi.deleteUser(deleteTarget.id);
			users = users.filter((u) => u.id !== deleteTarget!.id);
			total -= 1;
			toast.success('User deleted', `${deleteTarget.display_name} has been removed.`);
			deleteTarget = null;
		} catch {
			toast.error('Error', 'Could not delete user.');
		} finally {
			deleteLoading = false;
		}
	}

	function initials(name: string) {
		return name
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>Users — Admin — StarterKit</title>
</svelte:head>

<!-- Filter bar -->
<div class="flex items-center gap-3 mb-4">
	<select
		bind:value={roleFilter}
		onchange={applyFilter}
		class="text-sm rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] text-[var(--color-foreground)] px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
	>
		<option value="">All roles</option>
		{#each roles as role}
			<option value={role.name}>{role.name}</option>
		{/each}
	</select>
	<span class="text-sm text-[var(--color-muted-fg)]">{total} user{total !== 1 ? 's' : ''}</span>
</div>

<!-- Table -->
<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] overflow-hidden">
	{#if loading}
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">User</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Roles</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Joined</th>
					<th class="px-4 py-3"></th>
				</tr>
			</thead>
			<tbody class="divide-y divide-[var(--color-border)]">
				{#each Array.from({ length: 5 }) as _, i (i)}
					<tr class="bg-[var(--color-card)]">
						<td class="px-4 py-3">
							<div class="flex items-center gap-3">
								<div class="animate-pulse h-8 w-8 rounded-full bg-[var(--color-muted)] shrink-0"></div>
								<div class="space-y-1.5">
									<div class="animate-pulse h-3 rounded bg-[var(--color-muted)]" style="width: {i % 2 === 0 ? '120px' : '90px'}"></div>
									<div class="animate-pulse h-2.5 rounded bg-[var(--color-muted)]" style="width: {i % 3 === 0 ? '150px' : '120px'}"></div>
								</div>
							</div>
						</td>
						<td class="px-4 py-3">
							<div class="animate-pulse h-5 w-16 rounded-full bg-[var(--color-muted)]"></div>
						</td>
						<td class="px-4 py-3">
							<div class="animate-pulse h-3 w-20 rounded bg-[var(--color-muted)]"></div>
						</td>
						<td class="px-4 py-3 text-right">
							<div class="animate-pulse h-4 w-4 rounded bg-[var(--color-muted)] ml-auto"></div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{:else if users.length === 0}
		<div class="text-center py-16 text-[var(--color-muted-fg)] text-sm">No users found.</div>
	{:else}
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b border-[var(--color-border)] bg-[var(--color-muted)]/40">
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">User</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Roles</th>
					<th class="text-left px-4 py-3 font-medium text-[var(--color-muted-fg)]">Joined</th>
					<th class="px-4 py-3"></th>
				</tr>
			</thead>
			<tbody class="divide-y divide-[var(--color-border)]">
				{#each users as user (user.id)}
					<tr class="bg-[var(--color-card)] hover:bg-[var(--color-muted)]/20 transition-colors">
						<td class="px-4 py-3">
							<div class="flex items-center gap-3">
								{#if user.avatar_url}
									<img src={user.avatar_url} alt="" class="h-8 w-8 rounded-full object-cover shrink-0" />
								{:else}
									<div class="h-8 w-8 rounded-full bg-[var(--color-primary)]/10 text-[var(--color-primary)] text-xs font-semibold flex items-center justify-center shrink-0">
										{initials(user.display_name)}
									</div>
								{/if}
								<div class="min-w-0">
									<p class="font-medium text-[var(--color-foreground)] truncate">{user.display_name}</p>
									<p class="text-xs text-[var(--color-muted-fg)] truncate">{user.email}</p>
								</div>
							</div>
						</td>
						<td class="px-4 py-3">
							<div class="flex flex-wrap gap-1">
								{#each user.roles as role}
									<span class="inline-flex items-center gap-1 rounded-full bg-[var(--color-primary)]/10 text-[var(--color-primary)] px-2 py-0.5 text-xs font-medium">
										{role}
										{#if user.id !== $currentUser?.id}
											<button
												onclick={() => handleRevokeRole(user, role)}
												class="hover:text-[var(--color-destructive)] transition-colors"
												title="Revoke {role}"
											>
												<X class="h-3 w-3" />
											</button>
										{/if}
									</span>
								{/each}
								{#if user.id !== $currentUser?.id}
									<button
										onclick={() => { assignTarget = user; assignRoleId = ''; }}
										class="rounded-full border border-dashed border-[var(--color-border)] px-2 py-0.5 text-xs text-[var(--color-muted-fg)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)] transition-colors"
									>
										+ role
									</button>
								{/if}
							</div>
						</td>
						<td class="px-4 py-3 text-[var(--color-muted-fg)]">{formatDate(user.created_at)}</td>
						<td class="px-4 py-3 text-right">
							{#if user.id !== $currentUser?.id}
								<button
									onclick={() => { deleteTarget = user; }}
									class="text-[var(--color-muted-fg)] hover:text-[var(--color-destructive)] transition-colors p-1 rounded"
									title="Delete user"
								>
									<Trash2 class="h-4 w-4" />
								</button>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>

		{#if users.length < total}
			<div class="flex justify-center py-4 border-t border-[var(--color-border)]">
				<button
					onclick={loadMore}
					class="text-sm font-medium text-[var(--color-primary)] hover:underline"
				>
					Load more
				</button>
			</div>
		{/if}
	{/if}
</div>

<!-- Assign Role Modal -->
{#if assignTarget}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="presentation"
		onclick={() => { assignTarget = null; }}
		onkeydown={(e) => { if (e.key === 'Escape') assignTarget = null; }}
	>
		<div
			class="w-full max-w-sm rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6 shadow-[var(--shadow-lg)]"
			role="presentation"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<h3 class="text-base font-semibold text-[var(--color-foreground)] mb-1">Assign role</h3>
			<p class="text-sm text-[var(--color-muted-fg)] mb-4">
				Add a role to <strong>{assignTarget.display_name}</strong>.
			</p>
			<select
				bind:value={assignRoleId}
				class="w-full text-sm rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
			>
				<option value="">Select a role…</option>
				{#each roles.filter((r) => !assignTarget!.roles.includes(r.name)) as role}
					<option value={role.id}>{role.name}{role.description ? ` — ${role.description}` : ''}</option>
				{/each}
			</select>
			<div class="flex justify-end gap-2">
				<button
					onclick={() => { assignTarget = null; }}
					class="px-4 py-2 text-sm rounded-[var(--radius)] border border-[var(--color-border)] text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={handleAssignRole}
					disabled={!assignRoleId || assignLoading}
					class="px-4 py-2 text-sm rounded-[var(--radius)] bg-[var(--color-primary)] text-white font-medium disabled:opacity-50 hover:opacity-90 transition-opacity"
				>
					{assignLoading ? 'Assigning…' : 'Assign'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Delete Confirmation Modal -->
{#if deleteTarget}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="presentation"
		onclick={() => { deleteTarget = null; }}
		onkeydown={(e) => { if (e.key === 'Escape') deleteTarget = null; }}
	>
		<div
			class="w-full max-w-sm rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6 shadow-[var(--shadow-lg)]"
			role="presentation"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<h3 class="text-base font-semibold text-[var(--color-foreground)] mb-1">Delete user</h3>
			<p class="text-sm text-[var(--color-muted-fg)] mb-4">
				Are you sure you want to delete <strong>{deleteTarget.display_name}</strong>? This action cannot be undone.
			</p>
			<div class="flex justify-end gap-2">
				<button
					onclick={() => { deleteTarget = null; }}
					class="px-4 py-2 text-sm rounded-[var(--radius)] border border-[var(--color-border)] text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={handleDeleteUser}
					disabled={deleteLoading}
					class="px-4 py-2 text-sm rounded-[var(--radius)] bg-[var(--color-destructive)] text-white font-medium disabled:opacity-50 hover:opacity-90 transition-opacity"
				>
					{deleteLoading ? 'Deleting…' : 'Delete'}
				</button>
			</div>
		</div>
	</div>
{/if}
