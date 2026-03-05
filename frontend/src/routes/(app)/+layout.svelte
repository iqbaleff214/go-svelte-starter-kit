<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore, currentUser, isLoading } from '$stores/auth';
	import { authApi } from '$api/auth';
	import { toast } from '$stores/toast';
	import { notificationStore, unreadCount } from '$stores/notification';
	import { wsStore } from '$stores/ws';
	import { notificationApi } from '$api/notification';
	import { onMount, onDestroy } from 'svelte';
	import {
		LayoutDashboard, User, Bell, Key, Bot, Shield, Lock, LogOut,
		Sun, Moon, Monitor, ChevronUp, Search, PanelLeftClose, PanelLeftOpen
	} from 'lucide-svelte';
	import { theme } from '$stores/theme';
	import { commandPalette } from '$stores/commandPalette';
	import { aiChat } from '$stores/aiChat';
	import CommandPalette from '$components/ui/CommandPalette.svelte';
	import AiChatPanel from '$components/ui/AiChatPanel.svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	const navItems = [
		{ href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' }
	];

	const themeLabels: Record<string, string> = { light: 'Light', dark: 'Dark', system: 'System' };
	const themeIcons: Record<string, typeof Sun> = { light: Sun, dark: Moon, system: Monitor };
	const ThemeIcon = $derived(themeIcons[$theme]);

	let userMenuOpen = $state(false);
	let mobileUserMenuOpen = $state(false);
	let sidebarCollapsed = $state(
		typeof localStorage !== 'undefined' ? localStorage.getItem('sidebar-collapsed') === 'true' : false
	);
	$effect(() => {
		localStorage.setItem('sidebar-collapsed', String(sidebarCollapsed));
	});

	const adminItems = [
		{ href: '/admin', icon: Shield, label: 'Admin' }
	];

	// Condensed items for mobile bottom tab bar
	const bottomTabItems = [
		{ href: '/dashboard', icon: LayoutDashboard, label: 'Home' },
		{ href: '/notifications', icon: Bell, label: 'Alerts' }
	];

	onMount(() => {
		const unsub = isLoading.subscribe((loading) => {
			if (!loading) {
				if (!$currentUser) {
					goto('/login');
					unsub();
				} else {
					wsStore.connect($authStore.accessToken!);
					notificationApi.unreadCount().then((r) => notificationStore.setCount(r.count)).catch(() => {});
					unsub();
				}
			}
		});

		function handleKeydown(e: KeyboardEvent) {
			if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
				e.preventDefault();
				commandPalette.toggle();
			}
		}
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	onDestroy(() => wsStore.disconnect());

	async function handleLogout() {
		try {
			await authApi.logout();
		} finally {
			authStore.clearAuth();
			toast.info('Signed out', 'See you next time!');
			goto('/login');
		}
	}

	const exactMatchRoutes = ['/profile'];

	function isActive(href: string) {
		if (exactMatchRoutes.includes(href)) {
			return $page.url.pathname === href;
		}
		return $page.url.pathname === href || $page.url.pathname.startsWith(href + '/');
	}

	function getInitials(name: string) {
		return name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2);
	}

	function isAdmin() {
		return $currentUser && ['admin', 'superadmin'].some((r) => ($currentUser as any).roles?.includes(r));
	}
</script>

{#if $isLoading}
	<div class="min-h-screen flex items-center justify-center bg-[var(--color-background)]">
		<svg class="h-8 w-8 animate-spin text-[var(--color-primary)]" viewBox="0 0 24 24" fill="none">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
			<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
		</svg>
	</div>
{:else if $currentUser}
	<div class="h-screen overflow-hidden bg-[var(--color-background)] flex">
		<!-- Desktop sidebar (hidden on mobile) -->
		<aside class="hidden lg:flex lg:flex-col border-r border-[var(--color-border)] bg-[var(--color-card)] shrink-0 transition-[width] duration-200 overflow-hidden {sidebarCollapsed ? 'w-16' : 'w-64'}">
			<!-- Logo -->
			<div class="flex h-16 shrink-0 items-center border-b border-[var(--color-border)] {sidebarCollapsed ? 'justify-center px-0' : 'px-4 justify-between'}">
				{#if sidebarCollapsed}
					<button
						onclick={() => sidebarCollapsed = false}
						title="Expand sidebar"
						class="text-[var(--color-primary)] text-lg font-bold hover:opacity-70 transition-opacity"
					>⚡</button>
				{:else}
					<a href="/" class="flex items-center gap-2 font-bold text-[var(--color-foreground)] min-w-0">
						<span class="text-[var(--color-primary)] shrink-0">⚡</span>
						<span class="truncate">StarterKit</span>
					</a>
					<button
						onclick={() => sidebarCollapsed = true}
						title="Collapse sidebar"
						class="shrink-0 rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
					>
						<PanelLeftClose class="h-4 w-4" />
					</button>
				{/if}
			</div>

			<!-- Nav -->
			<nav class="flex-1 overflow-y-auto p-2 space-y-0.5">
				{#each navItems as item}
					{@const NavIcon = item.icon}
					<a
						href={item.href}
						title={sidebarCollapsed ? item.label : undefined}
						class="
							flex items-center rounded-[var(--radius-sm)] py-2.5 text-sm font-medium transition-colors
							{sidebarCollapsed ? 'justify-center px-0' : 'gap-3 px-3'}
							{isActive(item.href)
								? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
								: 'text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)]'}
						"
					>
						<NavIcon class="h-4 w-4 shrink-0" />
						{#if !sidebarCollapsed}{item.label}{/if}
					</a>
				{/each}

				{#if isAdmin()}
					<div class="my-2 border-t border-[var(--color-border)]"></div>
					{#each adminItems as item}
						{@const AdminIcon = item.icon}
						<a
							href={item.href}
							title={sidebarCollapsed ? item.label : undefined}
							class="
								flex items-center rounded-[var(--radius-sm)] py-2.5 text-sm font-medium
								text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors
								{sidebarCollapsed ? 'justify-center px-0' : 'gap-3 px-3'}
							"
						>
							<AdminIcon class="h-4 w-4 shrink-0" />
							{#if !sidebarCollapsed}{item.label}{/if}
						</a>
					{/each}
				{/if}
			</nav>

			<!-- User section -->
			<div class="border-t border-[var(--color-border)] p-2 shrink-0 relative">
				{#if userMenuOpen}
					<!-- Backdrop -->
					<div class="fixed inset-0 z-10" role="presentation" onclick={() => userMenuOpen = false} onkeydown={() => {}}></div>
					<!-- Dropdown: upward when expanded, rightward when collapsed -->
					<div class="absolute z-20 rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] shadow-[var(--shadow-md)] py-1 overflow-hidden
						{sidebarCollapsed ? 'bottom-2 left-full ml-2 w-52' : 'bottom-full left-2 right-2 mb-1'}">
						<a
							href="/profile"
							onclick={() => userMenuOpen = false}
							class="flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
						>
							<User class="h-4 w-4 shrink-0" />
							Profile
						</a>
						<a
							href="/profile/security"
							onclick={() => userMenuOpen = false}
							class="flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
						>
							<Lock class="h-4 w-4 shrink-0" />
							Security
						</a>
						<a
							href="/profile/api-keys"
							onclick={() => userMenuOpen = false}
							class="flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
						>
							<Key class="h-4 w-4 shrink-0" />
							API Keys
						</a>
						<div class="my-1 border-t border-[var(--color-border)]"></div>
						<button
							onclick={() => theme.cycle()}
							class="w-full flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
						>
							<ThemeIcon class="h-4 w-4 shrink-0" />
							<span class="flex-1 text-left">Theme: {themeLabels[$theme]}</span>
						</button>
						<div class="my-1 border-t border-[var(--color-border)]"></div>
						<button
							onclick={handleLogout}
							class="w-full flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-destructive)] hover:bg-[var(--color-muted)] transition-colors"
						>
							<LogOut class="h-4 w-4 shrink-0" />
							Sign out
						</button>
					</div>
				{/if}
				<!-- User trigger button -->
				<button
					onclick={() => userMenuOpen = !userMenuOpen}
					title={sidebarCollapsed ? $currentUser.display_name : undefined}
					class="w-full flex items-center rounded-[var(--radius-sm)] hover:bg-[var(--color-muted)] transition-colors
						{sidebarCollapsed ? 'justify-center p-1' : 'gap-3 px-2 py-2 text-left'}"
				>
					<div class="h-8 w-8 rounded-full bg-[var(--color-primary)] flex items-center justify-center text-xs font-bold text-white shrink-0 overflow-hidden">
						{#if $currentUser.avatar_url}
							<img src={$currentUser.avatar_url} alt="" class="h-8 w-8 object-cover" />
						{:else}
							{getInitials($currentUser.display_name)}
						{/if}
					</div>
					{#if !sidebarCollapsed}
						<div class="min-w-0 flex-1">
							<p class="text-sm font-medium text-[var(--color-foreground)] truncate">{$currentUser.display_name}</p>
							<p class="text-xs text-[var(--color-muted-fg)] truncate">{$currentUser.email}</p>
						</div>
						<ChevronUp class="h-4 w-4 text-[var(--color-muted-fg)] shrink-0 transition-transform {userMenuOpen ? '' : 'rotate-180'}" />
					{/if}
				</button>
			</div>
		</aside>

		<!-- Main content -->
		<div class="flex-1 flex flex-col min-w-0">
			<!-- Topbar -->
			<header class="h-16 shrink-0 border-b border-[var(--color-border)] bg-[var(--color-card)] flex items-center px-4 sm:px-6 gap-2">
				{#if sidebarCollapsed}
					<button
						onclick={() => sidebarCollapsed = false}
						title="Expand sidebar"
						class="hidden lg:flex shrink-0 rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
					>
						<PanelLeftOpen class="h-4 w-4" />
					</button>
				{/if}
				<!-- Search trigger -->
				<button
					onclick={() => commandPalette.open()}
					class="flex-1 flex items-center gap-2 h-8 max-w-xs rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-muted)] px-3 text-sm text-[var(--color-muted-fg)] hover:border-[var(--color-primary)]/50 transition-colors"
				>
					<Search class="h-3.5 w-3.5 shrink-0" />
					<span class="flex-1 text-left text-xs">Search…</span>
					<kbd class="hidden sm:inline-flex items-center gap-0.5 text-[10px] font-mono opacity-60">⌘K</kbd>
				</button>
				<div class="flex items-center gap-1 ml-auto shrink-0">
				<a
					href="/notifications"
					class="relative p-2 rounded-[var(--radius-sm)] text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
					title="Notifications"
				>
					<Bell class="h-5 w-5" />
					{#if $unreadCount > 0}
						<span class="absolute top-1 right-1 h-2 w-2 rounded-full bg-[var(--color-destructive)]"></span>
					{/if}
				</a>
				<!-- Mobile user menu (hidden on desktop where sidebar handles it) -->
				<div class="relative lg:hidden">
					{#if mobileUserMenuOpen}
						<div class="fixed inset-0 z-10" role="presentation" onclick={() => mobileUserMenuOpen = false} onkeydown={() => {}}></div>
						<div class="absolute right-0 top-full mt-1 z-20 w-52 rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] shadow-[var(--shadow-md)] py-1 overflow-hidden">
							<a
								href="/profile"
								onclick={() => mobileUserMenuOpen = false}
								class="flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
							>
								<User class="h-4 w-4 shrink-0" />
								Profile
							</a>
							<a
								href="/profile/security"
								onclick={() => mobileUserMenuOpen = false}
								class="flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
							>
								<Lock class="h-4 w-4 shrink-0" />
								Security
							</a>
							<a
								href="/profile/api-keys"
								onclick={() => mobileUserMenuOpen = false}
								class="flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
							>
								<Key class="h-4 w-4 shrink-0" />
								API Keys
							</a>
							<div class="my-1 border-t border-[var(--color-border)]"></div>
							<button
								onclick={() => theme.cycle()}
								class="w-full flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
							>
								<ThemeIcon class="h-4 w-4 shrink-0" />
								<span class="flex-1 text-left">Theme: {themeLabels[$theme]}</span>
							</button>
							<div class="my-1 border-t border-[var(--color-border)]"></div>
							<button
								onclick={handleLogout}
								class="w-full flex items-center gap-3 px-3 py-2 text-sm text-[var(--color-destructive)] hover:bg-[var(--color-muted)] transition-colors"
							>
								<LogOut class="h-4 w-4 shrink-0" />
								Sign out
							</button>
						</div>
					{/if}
					<button
						onclick={() => mobileUserMenuOpen = !mobileUserMenuOpen}
						class="p-1 rounded-full transition-opacity hover:opacity-80"
					>
						<div class="h-8 w-8 rounded-full bg-[var(--color-primary)] flex items-center justify-center text-xs font-bold text-white shrink-0 overflow-hidden">
							{#if $currentUser.avatar_url}
								<img src={$currentUser.avatar_url} alt="" class="h-8 w-8 object-cover" />
							{:else}
								{getInitials($currentUser.display_name)}
							{/if}
						</div>
					</button>
				</div>
				</div><!-- end flex items-center gap-1 ml-auto -->
			</header>
			<main class="flex-1 overflow-auto p-4 sm:p-6 lg:p-8 pb-20 lg:pb-8">
				{@render children()}
			</main>
		</div>
	</div>

	<CommandPalette />

	<!-- Floating AI button -->
	<button
		onclick={() => aiChat.toggle()}
		title="AI Assistant"
		class="
			fixed right-5 z-40 flex h-12 w-12 items-center justify-center rounded-full
			bg-[var(--color-primary)] text-white shadow-[var(--shadow-lg)]
			hover:opacity-90 active:scale-95 transition-all
			bottom-20 lg:bottom-5
		"
	>
		<Bot class="h-5 w-5" />
	</button>

	<AiChatPanel />

	<!-- Mobile bottom tab bar (hidden on lg+) -->
	<nav class="fixed bottom-0 left-0 right-0 z-30 flex border-t border-[var(--color-border)] bg-[var(--color-card)] lg:hidden">
		{#each bottomTabItems as item}
			{@const TabIcon = item.icon}
			<a
				href={item.href}
				class="
					flex-1 flex flex-col items-center justify-center gap-0.5 py-2.5
					text-xs font-medium transition-colors min-h-[56px]
					{isActive(item.href)
						? 'text-[var(--color-primary)]'
						: 'text-[var(--color-muted-fg)]'}
				"
			>
				<div class="relative">
					<TabIcon class="h-5 w-5" />
					{#if item.href === '/notifications' && $unreadCount > 0}
						<span class="absolute -top-1 -right-1.5 text-[10px] font-bold bg-[var(--color-destructive)] text-white rounded-full w-4 h-4 flex items-center justify-center leading-none">
							{$unreadCount > 9 ? '9+' : $unreadCount}
						</span>
					{/if}
				</div>
				<span>{item.label}</span>
			</a>
		{/each}
		{#if isAdmin()}
			<a
				href="/admin"
				class="
					flex-1 flex flex-col items-center justify-center gap-0.5 py-2.5
					text-xs font-medium transition-colors min-h-[56px]
					{isActive('/admin') ? 'text-[var(--color-primary)]' : 'text-[var(--color-muted-fg)]'}
				"
			>
				<Shield class="h-5 w-5" />
				<span>Admin</span>
			</a>
		{/if}
	</nav>
{/if}
