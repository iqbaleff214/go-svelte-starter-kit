<script lang="ts">
	import { onMount } from 'svelte';
	import { apikeyApi } from '$api/apikey';
	import { toast } from '$stores/toast';
	import { Copy, Trash2, ChevronDown, ChevronUp, Plus, Check } from 'lucide-svelte';
	import type { APIKey, APIKeyCreateResponse, APIKeyLog } from '$types';

	const ALL_SCOPES = [
		{ value: 'read:profile', label: 'Read profile', desc: 'View your profile information' },
		{ value: 'write:profile', label: 'Write profile', desc: 'Update your profile information' },
		{ value: 'read:notifications', label: 'Read notifications', desc: 'List your notifications' },
		{ value: 'write:notifications', label: 'Write notifications', desc: 'Mark notifications as read' },
		{ value: 'read:users', label: 'Read users', desc: 'List users (admin role required)' },
		{ value: 'write:users', label: 'Write users', desc: 'Manage users (admin role required)' },
		{ value: 'read:webhooks', label: 'Read webhooks', desc: 'List your webhooks' },
		{ value: 'write:webhooks', label: 'Write webhooks', desc: 'Create and delete webhooks' }
	];

	let keys = $state<APIKey[]>([]);
	let loading = $state(true);

	// Create form
	let showCreate = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newScopes = $state<string[]>([]);
	let newExpires = $state('');

	// Newly created key modal (one-time reveal)
	let revealedKey = $state<APIKeyCreateResponse | null>(null);
	let copied = $state(false);

	// Per-key log expansion
	let expandedLogs = $state<Record<string, boolean>>({});
	let logs = $state<Record<string, APIKeyLog[]>>({});
	let logsLoading = $state<Record<string, boolean>>({});

	// Revoke confirmation
	let confirmRevoke = $state<string | null>(null);

	onMount(async () => {
		try {
			keys = await apikeyApi.list();
		} catch {
			toast.error('Error', 'Could not load API keys.');
		} finally {
			loading = false;
		}
	});

	async function createKey() {
		if (!newName.trim() || newScopes.length === 0) return;
		creating = true;
		try {
			const resp = await apikeyApi.create({
				name: newName.trim(),
				scopes: newScopes,
				expires_at: newExpires || undefined
			});
			keys = [resp, ...keys];
			revealedKey = resp;
			showCreate = false;
			newName = '';
			newScopes = [];
			newExpires = '';
		} catch {
			toast.error('Error', 'Could not create API key.');
		} finally {
			creating = false;
		}
	}

	async function revokeKey(id: string) {
		try {
			await apikeyApi.revoke(id);
			keys = keys.map((k) =>
				k.id === id ? { ...k, revoked_at: new Date().toISOString() } : k
			);
			confirmRevoke = null;
		} catch {
			toast.error('Error', 'Could not revoke API key.');
		}
	}

	async function toggleLogs(id: string) {
		if (expandedLogs[id]) {
			expandedLogs = { ...expandedLogs, [id]: false };
			return;
		}
		expandedLogs = { ...expandedLogs, [id]: true };
		if (logs[id]) return;
		logsLoading = { ...logsLoading, [id]: true };
		try {
			const resp = await apikeyApi.listLogs(id);
			logs = { ...logs, [id]: resp.logs };
		} catch {
			toast.error('Error', 'Could not load logs.');
		} finally {
			logsLoading = { ...logsLoading, [id]: false };
		}
	}

	async function copyKey() {
		if (!revealedKey) return;
		await navigator.clipboard.writeText(revealedKey.key);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	function closeReveal() {
		revealedKey = null;
		copied = false;
	}

	function toggleScope(scope: string) {
		if (newScopes.includes(scope)) {
			newScopes = newScopes.filter((s) => s !== scope);
		} else {
			newScopes = [...newScopes, scope];
		}
	}

	function keyStatus(key: APIKey): 'revoked' | 'expired' | 'active' {
		if (key.revoked_at) return 'revoked';
		if (key.expires_at && new Date(key.expires_at) < new Date()) return 'expired';
		return 'active';
	}

	function formatDate(iso: string | null) {
		if (!iso) return '—';
		return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}

	function formatDateTime(iso: string) {
		return new Date(iso).toLocaleString(undefined, {
			month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
		});
	}
</script>

<svelte:head>
	<title>API Keys — StarterKit</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-[var(--color-foreground)]">API Keys</h1>
			<p class="text-sm text-[var(--color-muted-fg)] mt-1">
				Use API keys to authenticate against <code class="text-xs bg-[var(--color-muted)] px-1 py-0.5 rounded">/api/v1/*</code> endpoints.
			</p>
		</div>
		<button
			onclick={() => (showCreate = !showCreate)}
			class="flex items-center gap-2 px-4 py-2 rounded-[var(--radius)] bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity"
		>
			<Plus class="h-4 w-4" />
			New key
		</button>
	</div>

	<!-- Create form -->
	{#if showCreate}
		<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-5 space-y-4">
			<h2 class="font-semibold text-[var(--color-foreground)]">Create API Key</h2>

			<div>
				<label class="block text-sm font-medium text-[var(--color-foreground)] mb-1">Name</label>
				<input
					bind:value={newName}
					type="text"
					placeholder="e.g. My integration"
					class="w-full rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
				/>
			</div>

			<div>
				<label class="block text-sm font-medium text-[var(--color-foreground)] mb-2">Scopes</label>
				<div class="grid grid-cols-2 gap-2">
					{#each ALL_SCOPES as scope}
						<label class="flex items-start gap-2 cursor-pointer p-2 rounded-[var(--radius)] border border-[var(--color-border)] hover:bg-[var(--color-muted)] transition-colors {newScopes.includes(scope.value) ? 'border-[var(--color-primary)] bg-[var(--color-primary)]/5' : ''}">
							<input
								type="checkbox"
								checked={newScopes.includes(scope.value)}
								onchange={() => toggleScope(scope.value)}
								class="mt-0.5 accent-[var(--color-primary)]"
							/>
							<div>
								<p class="text-sm font-medium text-[var(--color-foreground)]">{scope.label}</p>
								<p class="text-xs text-[var(--color-muted-fg)]">{scope.desc}</p>
							</div>
						</label>
					{/each}
				</div>
			</div>

			<div>
				<label class="block text-sm font-medium text-[var(--color-foreground)] mb-1">
					Expiry <span class="text-[var(--color-muted-fg)] font-normal">(optional)</span>
				</label>
				<input
					bind:value={newExpires}
					type="datetime-local"
					class="rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
				/>
			</div>

			<div class="flex gap-2 justify-end">
				<button
					onclick={() => (showCreate = false)}
					class="px-4 py-2 rounded-[var(--radius)] border border-[var(--color-border)] text-sm hover:bg-[var(--color-muted)] transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={createKey}
					disabled={creating || !newName.trim() || newScopes.length === 0}
					class="px-4 py-2 rounded-[var(--radius)] bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
				>
					{creating ? 'Creating…' : 'Create key'}
				</button>
			</div>
		</div>
	{/if}

	<!-- Key list -->
	<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] divide-y divide-[var(--color-border)]">
		{#if loading}
			<div class="p-8 text-center text-sm text-[var(--color-muted-fg)]">Loading…</div>
		{:else if keys.length === 0}
			<div class="p-8 text-center text-sm text-[var(--color-muted-fg)]">No API keys yet. Create one above.</div>
		{:else}
			{#each keys as key (key.id)}
				{@const status = keyStatus(key)}
				<div class="p-4 space-y-2">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2 flex-wrap">
								<span class="font-medium text-[var(--color-foreground)]">{key.name}</span>
								<code class="text-xs bg-[var(--color-muted)] px-1.5 py-0.5 rounded font-mono">sk_{key.key_prefix}…</code>
								<span class="text-xs px-2 py-0.5 rounded-full font-medium
									{status === 'active' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : ''}
									{status === 'revoked' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : ''}
									{status === 'expired' ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' : ''}
								">{status}</span>
							</div>
							<div class="flex flex-wrap gap-1 mt-1.5">
								{#each key.scopes as scope}
									<span class="text-xs bg-[var(--color-muted)] text-[var(--color-muted-fg)] px-1.5 py-0.5 rounded">{scope}</span>
								{/each}
							</div>
							<p class="text-xs text-[var(--color-muted-fg)] mt-1">
								Created {formatDate(key.created_at)}
								{key.last_used_at ? `· Last used ${formatDate(key.last_used_at)}` : '· Never used'}
								{key.expires_at ? `· Expires ${formatDate(key.expires_at)}` : ''}
							</p>
						</div>

						<div class="flex items-center gap-2 shrink-0">
							<button
								onclick={() => toggleLogs(key.id)}
								class="text-xs text-[var(--color-muted-fg)] hover:text-[var(--color-foreground)] flex items-center gap-1 transition-colors"
							>
								Logs
								{#if expandedLogs[key.id]}
									<ChevronUp class="h-3.5 w-3.5" />
								{:else}
									<ChevronDown class="h-3.5 w-3.5" />
								{/if}
							</button>
							{#if status === 'active'}
								{#if confirmRevoke === key.id}
									<span class="text-xs text-[var(--color-muted-fg)]">Revoke?</span>
									<button
										onclick={() => revokeKey(key.id)}
										class="text-xs text-red-600 hover:text-red-700 font-medium transition-colors"
									>Yes</button>
									<button
										onclick={() => (confirmRevoke = null)}
										class="text-xs text-[var(--color-muted-fg)] hover:text-[var(--color-foreground)] transition-colors"
									>No</button>
								{:else}
									<button
										onclick={() => (confirmRevoke = key.id)}
										class="text-[var(--color-muted-fg)] hover:text-[var(--color-destructive)] transition-colors"
										title="Revoke key"
									>
										<Trash2 class="h-4 w-4" />
									</button>
								{/if}
							{/if}
						</div>
					</div>

					<!-- Logs panel -->
					{#if expandedLogs[key.id]}
						<div class="mt-2 rounded-[var(--radius)] border border-[var(--color-border)] overflow-hidden">
							{#if logsLoading[key.id]}
								<p class="p-3 text-xs text-[var(--color-muted-fg)]">Loading logs…</p>
							{:else if !logs[key.id] || logs[key.id].length === 0}
								<p class="p-3 text-xs text-[var(--color-muted-fg)]">No requests logged yet.</p>
							{:else}
								<table class="w-full text-xs">
									<thead class="bg-[var(--color-muted)] text-[var(--color-muted-fg)]">
										<tr>
											<th class="text-left px-3 py-2 font-medium">Method</th>
											<th class="text-left px-3 py-2 font-medium">Path</th>
											<th class="text-left px-3 py-2 font-medium">Status</th>
											<th class="text-left px-3 py-2 font-medium">IP</th>
											<th class="text-left px-3 py-2 font-medium">Time</th>
										</tr>
									</thead>
									<tbody class="divide-y divide-[var(--color-border)]">
										{#each logs[key.id] as log (log.id)}
											<tr class="hover:bg-[var(--color-muted)]/50">
												<td class="px-3 py-2 font-mono">{log.method}</td>
												<td class="px-3 py-2 font-mono truncate max-w-[200px]">{log.path}</td>
												<td class="px-3 py-2">
													<span class="{log.status_code < 400 ? 'text-green-600' : 'text-red-600'} font-medium">{log.status_code}</span>
												</td>
												<td class="px-3 py-2 font-mono">{log.ip}</td>
												<td class="px-3 py-2">{formatDateTime(log.created_at)}</td>
											</tr>
										{/each}
									</tbody>
								</table>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- One-time key reveal modal -->
{#if revealedKey}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
		<div class="w-full max-w-lg rounded-[var(--radius-lg)] bg-[var(--color-card)] border border-[var(--color-border)] p-6 space-y-4 shadow-xl">
			<div>
				<h2 class="text-lg font-semibold text-[var(--color-foreground)]">API Key Created</h2>
				<p class="text-sm text-[var(--color-muted-fg)] mt-1">
					Copy your key now — it won't be shown again.
				</p>
			</div>

			<div class="flex items-center gap-2">
				<code class="flex-1 text-sm font-mono bg-[var(--color-muted)] border border-[var(--color-border)] rounded-[var(--radius)] px-3 py-2 break-all select-all">
					{revealedKey.key}
				</code>
				<button
					onclick={copyKey}
					class="shrink-0 p-2 rounded-[var(--radius)] border border-[var(--color-border)] hover:bg-[var(--color-muted)] transition-colors text-[var(--color-foreground)]"
					title="Copy to clipboard"
				>
					{#if copied}
						<Check class="h-4 w-4 text-green-600" />
					{:else}
						<Copy class="h-4 w-4" />
					{/if}
				</button>
			</div>

			<button
				onclick={closeReveal}
				class="w-full px-4 py-2 rounded-[var(--radius)] bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity"
			>
				Done, I've copied the key
			</button>
		</div>
	</div>
{/if}
