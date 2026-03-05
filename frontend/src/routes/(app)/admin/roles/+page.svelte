<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi } from '$api/admin';
	import { currentUser } from '$stores/auth';
	import { toast } from '$stores/toast';
	import type { Role, Permission } from '$types';
	import { Plus, Trash2, Save } from 'lucide-svelte';

	const systemRoles = ['superadmin', 'admin', 'user'];
	const isSuperAdmin = $derived($currentUser?.roles?.includes('superadmin') ?? false);

	let loading = $state(true);
	let roles = $state<Role[]>([]);
	let allPermissions = $state<Permission[]>([]);
	let selectedRole = $state<Role | null>(null);
	let checkedPermIds = $state<Set<string>>(new Set());
	let saving = $state(false);

	// New role form
	let showNewRole = $state(false);
	let newRoleName = $state('');
	let newRoleDesc = $state('');
	let creating = $state(false);

	// Delete state
	let deleteTarget = $state<Role | null>(null);
	let deleting = $state(false);

	onMount(async () => {
		try {
			const [rolesRes, permsRes] = await Promise.all([
				adminApi.listRoles(),
				adminApi.listPermissions()
			]);
			roles = rolesRes;
			allPermissions = permsRes;
			if (roles.length > 0) selectRole(roles[0]);
		} catch {
			toast.error('Failed to load', 'Could not fetch roles.');
		} finally {
			loading = false;
		}
	});

	function selectRole(role: Role) {
		selectedRole = role;
		checkedPermIds = new Set(role.permissions.map((p) => p.id));
	}

	// Group permissions by resource prefix (e.g. "users:read" → "users")
	function groupPermissions(perms: Permission[]): Record<string, Permission[]> {
		const groups: Record<string, Permission[]> = {};
		for (const p of perms) {
			const key = p.name.includes(':') ? p.name.split(':')[0] : 'other';
			if (!groups[key]) groups[key] = [];
			groups[key].push(p);
		}
		return groups;
	}

	const permGroups = $derived(groupPermissions(allPermissions));

	function togglePerm(id: string) {
		const next = new Set(checkedPermIds);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		checkedPermIds = next;
	}

	async function handleSavePermissions() {
		if (!selectedRole) return;
		saving = true;
		try {
			const updated = await adminApi.setRolePermissions(selectedRole.id, [...checkedPermIds]);
			roles = roles.map((r) => (r.id === updated.id ? updated : r));
			selectedRole = updated;
			checkedPermIds = new Set(updated.permissions.map((p) => p.id));
			toast.success('Saved', 'Permissions updated successfully.');
		} catch {
			toast.error('Error', 'Could not save permissions.');
		} finally {
			saving = false;
		}
	}

	async function handleCreateRole() {
		if (!newRoleName.trim()) return;
		creating = true;
		try {
			const role = await adminApi.createRole({
				name: newRoleName.trim(),
				description: newRoleDesc.trim() || undefined
			});
			roles = [role, ...roles];
			showNewRole = false;
			newRoleName = '';
			newRoleDesc = '';
			selectRole(role);
			toast.success('Created', `Role "${role.name}" created.`);
		} catch {
			toast.error('Error', 'Could not create role.');
		} finally {
			creating = false;
		}
	}

	async function handleDeleteRole() {
		if (!deleteTarget) return;
		deleting = true;
		try {
			await adminApi.deleteRole(deleteTarget.id);
			const removed = deleteTarget;
			roles = roles.filter((r) => r.id !== removed.id);
			if (selectedRole?.id === removed.id) {
				selectedRole = roles[0] ?? null;
				if (selectedRole) checkedPermIds = new Set(selectedRole.permissions.map((p) => p.id));
			}
			toast.success('Deleted', `Role "${removed.name}" deleted.`);
			deleteTarget = null;
		} catch {
			toast.error('Error', 'Could not delete role. It may still be in use.');
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>Roles — Admin — StarterKit</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-16">
		<svg class="h-6 w-6 animate-spin text-[var(--color-primary)]" viewBox="0 0 24 24" fill="none">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
			<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
		</svg>
	</div>
{:else}
	<div class="flex gap-6">
		<!-- Left panel: role list -->
		<div class="w-60 shrink-0">
			<div class="flex items-center justify-between mb-3">
				<h2 class="text-xs font-semibold text-[var(--color-muted-fg)] uppercase tracking-wide">Roles</h2>
				{#if isSuperAdmin}
					<button
						onclick={() => { showNewRole = true; }}
						class="text-[var(--color-primary)] hover:opacity-70 transition-opacity"
						title="New role"
					>
						<Plus class="h-4 w-4" />
					</button>
				{/if}
			</div>

			{#if showNewRole && isSuperAdmin}
				<div class="mb-3 rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] p-3">
					<input
						type="text"
						bind:value={newRoleName}
						placeholder="Role name"
						class="w-full text-sm rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-2 py-1.5 mb-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
					/>
					<input
						type="text"
						bind:value={newRoleDesc}
						placeholder="Description (optional)"
						class="w-full text-sm rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-2 py-1.5 mb-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
					/>
					<div class="flex gap-2">
						<button
							onclick={handleCreateRole}
							disabled={!newRoleName.trim() || creating}
							class="flex-1 text-xs py-1.5 rounded-[var(--radius)] bg-[var(--color-primary)] text-white font-medium disabled:opacity-50"
						>
							{creating ? 'Creating…' : 'Create'}
						</button>
						<button
							onclick={() => { showNewRole = false; newRoleName = ''; newRoleDesc = ''; }}
							class="text-xs py-1.5 px-2 rounded-[var(--radius)] border border-[var(--color-border)] text-[var(--color-foreground)]"
						>
							Cancel
						</button>
					</div>
				</div>
			{/if}

			<div class="flex flex-col gap-1">
				{#each roles as role (role.id)}
					<button
						onclick={() => selectRole(role)}
						class="w-full text-left rounded-[var(--radius)] px-3 py-2.5 transition-colors {selectedRole?.id === role.id
							? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
							: 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)]'}"
					>
						<p class="text-sm font-medium">{role.name}</p>
						<p class="text-xs opacity-70">{role.permissions.length} permission{role.permissions.length !== 1 ? 's' : ''}</p>
					</button>
				{/each}
			</div>
		</div>

		<!-- Right panel: permissions -->
		<div class="flex-1 min-w-0">
			{#if selectedRole}
				<div class="flex items-start justify-between mb-4">
					<div>
						<h2 class="text-base font-semibold text-[var(--color-foreground)]">{selectedRole.name}</h2>
						{#if selectedRole.description}
							<p class="text-sm text-[var(--color-muted-fg)]">{selectedRole.description}</p>
						{/if}
					</div>
					<div class="flex items-center gap-2">
						{#if isSuperAdmin && !systemRoles.includes(selectedRole.name)}
							<button
								onclick={() => { deleteTarget = selectedRole; }}
								class="flex items-center gap-1.5 text-sm text-[var(--color-destructive)] hover:opacity-70 transition-opacity px-3 py-1.5 rounded-[var(--radius)] border border-[var(--color-destructive)]/30"
							>
								<Trash2 class="h-3.5 w-3.5" />
								Delete
							</button>
						{/if}
						<button
							onclick={handleSavePermissions}
							disabled={saving}
							class="flex items-center gap-1.5 text-sm bg-[var(--color-primary)] text-white font-medium px-3 py-1.5 rounded-[var(--radius)] disabled:opacity-50 hover:opacity-90 transition-opacity"
						>
							<Save class="h-3.5 w-3.5" />
							{saving ? 'Saving…' : 'Save permissions'}
						</button>
					</div>
				</div>

				{#if allPermissions.length === 0}
					<p class="text-sm text-[var(--color-muted-fg)]">No permissions defined.</p>
				{:else}
					<div class="flex flex-col gap-6">
						{#each Object.entries(permGroups) as [group, perms]}
							<div>
								<h3 class="text-xs font-semibold text-[var(--color-muted-fg)] uppercase tracking-wide mb-2">{group}</h3>
								<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] divide-y divide-[var(--color-border)]">
									{#each perms as perm (perm.id)}
										<label class="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-[var(--color-muted)]/30 transition-colors">
											<input
												type="checkbox"
												checked={checkedPermIds.has(perm.id)}
												onchange={() => togglePerm(perm.id)}
												class="h-4 w-4 rounded border-[var(--color-border)] text-[var(--color-primary)] accent-[var(--color-primary)]"
											/>
											<div class="min-w-0">
												<p class="text-sm font-medium text-[var(--color-foreground)]">{perm.name}</p>
												{#if perm.description}
													<p class="text-xs text-[var(--color-muted-fg)]">{perm.description}</p>
												{/if}
											</div>
										</label>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			{:else}
				<div class="text-center py-16 text-[var(--color-muted-fg)] text-sm">
					Select a role to manage its permissions.
				</div>
			{/if}
		</div>
	</div>
{/if}

<!-- Delete Role Modal -->
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
			<h3 class="text-base font-semibold text-[var(--color-foreground)] mb-1">Delete role</h3>
			<p class="text-sm text-[var(--color-muted-fg)] mb-4">
				Delete <strong>{deleteTarget.name}</strong>? Users with this role will lose its permissions.
			</p>
			<div class="flex justify-end gap-2">
				<button
					onclick={() => { deleteTarget = null; }}
					class="px-4 py-2 text-sm rounded-[var(--radius)] border border-[var(--color-border)] text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={handleDeleteRole}
					disabled={deleting}
					class="px-4 py-2 text-sm rounded-[var(--radius)] bg-[var(--color-destructive)] text-white font-medium disabled:opacity-50 hover:opacity-90 transition-opacity"
				>
					{deleting ? 'Deleting…' : 'Delete'}
				</button>
			</div>
		</div>
	</div>
{/if}
