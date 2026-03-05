<script lang="ts">
	import { goto } from '$app/navigation';
	import { commandPalette } from '$stores/commandPalette';
	import { currentUser } from '$stores/auth';
	import { searchApi, type SearchResult } from '$api/search';
	import {
		LayoutDashboard, Bot, Bell, User, Lock, Key, Shield,
		Search, Users, Tag, Loader
	} from 'lucide-svelte';

	type NavAction = {
		type: 'action';
		id: string;
		title: string;
		subtitle: string;
		icon: typeof Search;
		href: string;
		adminOnly?: boolean;
	};

	const navActions: NavAction[] = [
		{ type: 'action', id: 'dashboard', title: 'Dashboard', subtitle: 'Go to dashboard', icon: LayoutDashboard, href: '/dashboard' },
		{ type: 'action', id: 'ai', title: 'AI Assistant', subtitle: 'Open AI chat', icon: Bot, href: '/ai' },
		{ type: 'action', id: 'notifications', title: 'Notifications', subtitle: 'View notifications', icon: Bell, href: '/notifications' },
		{ type: 'action', id: 'profile', title: 'Profile', subtitle: 'Edit your profile', icon: User, href: '/profile' },
		{ type: 'action', id: 'security', title: 'Security', subtitle: 'Password & 2FA settings', icon: Lock, href: '/profile/security' },
		{ type: 'action', id: 'api-keys', title: 'API Keys', subtitle: 'Manage API keys', icon: Key, href: '/profile/api-keys' },
		{ type: 'action', id: 'admin', title: 'Admin Panel', subtitle: 'System administration', icon: Shield, href: '/admin', adminOnly: true },
	];

	let query = $state('');
	let loading = $state(false);
	let users = $state<SearchResult[]>([]);
	let roles = $state<SearchResult[]>([]);
	let selectedIndex = $state(0);
	let debounceTimer: ReturnType<typeof setTimeout>;

	function isAdmin() {
		return $currentUser && ['admin', 'superadmin'].some((r) => ($currentUser as any).roles?.includes(r));
	}

	function filteredActions(): NavAction[] {
		const visible = navActions.filter((a) => !a.adminOnly || isAdmin());
		if (!query.trim()) return visible;
		const q = query.toLowerCase();
		return visible.filter((a) => a.title.toLowerCase().includes(q) || a.subtitle.toLowerCase().includes(q));
	}

	type Group = { label: string; items: { title: string; subtitle: string; href: string; icon?: typeof Search; avatarUrl?: string | null; type: string }[] };

	function groups(): Group[] {
		const result: Group[] = [];
		const actions = filteredActions();
		if (actions.length) result.push({ label: 'Quick Actions', items: actions.map((a) => ({ ...a, icon: a.icon })) });
		if (users.length) result.push({ label: 'Users', items: users.map((u) => ({ title: u.title, subtitle: u.subtitle, href: u.href, avatarUrl: u.avatar_url, type: 'user' })) });
		if (roles.length) result.push({ label: 'Roles', items: roles.map((r) => ({ title: r.title, subtitle: r.subtitle, href: r.href, type: 'role' })) });
		return result;
	}

	function flatItems() {
		return groups().flatMap((g) => g.items);
	}

	function totalCount() {
		return flatItems().length;
	}

	$effect(() => {
		selectedIndex = 0;
	});

	$effect(() => {
		if (!$commandPalette) {
			query = '';
			users = [];
			roles = [];
			selectedIndex = 0;
		}
	});

	async function doSearch(q: string) {
		if (!isAdmin() || q.trim().length < 2) {
			users = [];
			roles = [];
			return;
		}
		loading = true;
		try {
			const res = await searchApi.search(q.trim());
			users = res.users;
			roles = res.roles;
		} catch {
			users = [];
			roles = [];
		} finally {
			loading = false;
		}
	}

	function onInput() {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => doSearch(query), 300);
	}

	function onKeydown(e: KeyboardEvent) {
		const total = totalCount();
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = (selectedIndex + 1) % total;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = (selectedIndex - 1 + total) % total;
		} else if (e.key === 'Enter') {
			const item = flatItems()[selectedIndex];
			if (item) navigate(item.href);
		} else if (e.key === 'Escape') {
			commandPalette.close();
		}
	}

	function navigate(href: string) {
		commandPalette.close();
		goto(href);
	}

	function getInitials(name: string) {
		return name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2);
	}

	let globalIndex = 0;
</script>

{#if $commandPalette}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-start justify-center pt-[10vh] px-4"
		onclick={() => commandPalette.close()}
		role="dialog"
		aria-modal="true"
		aria-label="Command palette"
	>
		<!-- Panel -->
		<div
			class="w-full max-w-xl rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] shadow-[var(--shadow-xl)] overflow-hidden"
			onclick={(e) => e.stopPropagation()}
		>
			<!-- Search input -->
			<div class="flex items-center gap-3 px-4 border-b border-[var(--color-border)]">
				{#if loading}
					<Loader class="h-4 w-4 text-[var(--color-muted-fg)] shrink-0 animate-spin" />
				{:else}
					<Search class="h-4 w-4 text-[var(--color-muted-fg)] shrink-0" />
				{/if}
				<input
					type="text"
					placeholder="Search pages, users, roles…"
					class="flex-1 h-12 bg-transparent text-sm text-[var(--color-foreground)] placeholder:text-[var(--color-muted-fg)] border-0 outline-none focus:outline-none focus:ring-0"
					bind:value={query}
					oninput={onInput}
					onkeydown={onKeydown}
					autofocus
				/>
				<kbd class="hidden sm:inline-flex items-center gap-1 rounded border border-[var(--color-border)] bg-[var(--color-muted)] px-1.5 py-0.5 text-[10px] text-[var(--color-muted-fg)] font-mono shrink-0">
					ESC
				</kbd>
			</div>

			<!-- Results -->
			<div class="max-h-[60vh] overflow-y-auto py-2">
				{#if totalCount() === 0}
					<p class="px-4 py-8 text-center text-sm text-[var(--color-muted-fg)]">
						{query.trim() ? 'No results found.' : 'Type to search…'}
					</p>
				{:else}
					{@const _ = (globalIndex = 0)}
					{#each groups() as group}
						<div class="px-2">
							<p class="px-2 py-1.5 text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-fg)]">
								{group.label}
							</p>
							{#each group.items as item}
								{@const idx = globalIndex++}
								<button
									class="
										w-full flex items-center gap-3 px-2 py-2 rounded-[var(--radius-sm)] text-left transition-colors
										{selectedIndex === idx
											? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
											: 'text-[var(--color-foreground)] hover:bg-[var(--color-muted)]'}
									"
									onclick={() => navigate(item.href)}
									onmouseenter={() => (selectedIndex = idx)}
								>
									<!-- Icon / avatar -->
									<div class="h-7 w-7 shrink-0 flex items-center justify-center rounded-[var(--radius-sm)]
										{selectedIndex === idx ? 'bg-[var(--color-primary)]/15' : 'bg-[var(--color-muted)]'}">
										{#if item.type === 'user'}
											{#if item.avatarUrl}
												<img src={item.avatarUrl} alt="" class="h-7 w-7 rounded-[var(--radius-sm)] object-cover" />
											{:else}
												<span class="text-[10px] font-bold text-[var(--color-muted-fg)]">{getInitials(item.title)}</span>
											{/if}
										{:else if item.type === 'role'}
											<Tag class="h-3.5 w-3.5 text-[var(--color-muted-fg)]" />
										{:else if item.icon}
											<svelte:component this={item.icon} class="h-3.5 w-3.5 {selectedIndex === idx ? 'text-[var(--color-primary)]' : 'text-[var(--color-muted-fg)]'}" />
										{/if}
									</div>
									<div class="min-w-0 flex-1">
										<p class="text-sm font-medium truncate">{item.title}</p>
										<p class="text-xs text-[var(--color-muted-fg)] truncate">{item.subtitle}</p>
									</div>
								</button>
							{/each}
						</div>
					{/each}
				{/if}
			</div>

			<!-- Footer -->
			<div class="flex items-center gap-4 px-4 py-2 border-t border-[var(--color-border)] text-[10px] text-[var(--color-muted-fg)]">
				<span class="flex items-center gap-1"><kbd class="font-mono">↑↓</kbd> navigate</span>
				<span class="flex items-center gap-1"><kbd class="font-mono">↵</kbd> open</span>
				<span class="flex items-center gap-1"><kbd class="font-mono">ESC</kbd> close</span>
			</div>
		</div>
	</div>
{/if}
