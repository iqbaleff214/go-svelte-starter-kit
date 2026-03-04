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
		LayoutDashboard, User, Bell, Key, Bot, Shield, Lock, LogOut
	} from 'lucide-svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	const navItems = [
		{ href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
		{ href: '/notifications', icon: Bell, label: 'Notifications' },
		{ href: '/ai', icon: Bot, label: 'AI Assistant' },
		{ href: '/profile', icon: User, label: 'Profile' },
		{ href: '/profile/security', icon: Lock, label: 'Security' },
		{ href: '/profile/api-keys', icon: Key, label: 'API Keys' }
	];

	const adminItems = [
		{ href: '/admin', icon: Shield, label: 'Admin' }
	];

	// Condensed items for mobile bottom tab bar (max 5 slots)
	const bottomTabItems = [
		{ href: '/dashboard', icon: LayoutDashboard, label: 'Home' },
		{ href: '/notifications', icon: Bell, label: 'Alerts' },
		{ href: '/ai', icon: Bot, label: 'AI' },
		{ href: '/profile', icon: User, label: 'Profile' }
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
		<aside class="hidden lg:flex lg:flex-col w-64 border-r border-[var(--color-border)] bg-[var(--color-card)] shrink-0">
			<!-- Logo -->
			<div class="flex h-16 items-center px-5 border-b border-[var(--color-border)] shrink-0">
				<a href="/" class="flex items-center gap-2 font-bold text-[var(--color-foreground)]">
					<span class="text-[var(--color-primary)]">⚡</span>
					StarterKit
				</a>
			</div>

			<!-- Nav -->
			<nav class="flex-1 overflow-y-auto p-3 space-y-0.5">
				{#each navItems as item}
					<a
						href={item.href}
						class="
							flex items-center gap-3 rounded-[var(--radius-sm)] px-3 py-2.5 text-sm font-medium
							transition-colors
							{isActive(item.href)
								? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
								: 'text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)]'}
						"
					>
						<svelte:component this={item.icon} class="h-4 w-4 shrink-0" />
						{item.label}
						{#if item.href === '/notifications' && $unreadCount > 0}
							<span class="ml-auto text-xs font-bold bg-[var(--color-destructive)] text-white rounded-full px-1.5 py-0.5 min-w-[18px] text-center leading-tight">
								{$unreadCount > 99 ? '99+' : $unreadCount}
							</span>
						{/if}
					</a>
				{/each}

				{#if isAdmin()}
					<div class="my-2 border-t border-[var(--color-border)]"></div>
					{#each adminItems as item}
						<a
							href={item.href}
							class="flex items-center gap-3 rounded-[var(--radius-sm)] px-3 py-2.5 text-sm font-medium text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
						>
							<svelte:component this={item.icon} class="h-4 w-4 shrink-0" />
							{item.label}
						</a>
					{/each}
				{/if}
			</nav>

			<!-- User section -->
			<div class="border-t border-[var(--color-border)] p-3 shrink-0">
				<div class="flex items-center gap-3 px-2 py-2">
					<div class="h-8 w-8 rounded-full bg-[var(--color-primary)] flex items-center justify-center text-xs font-bold text-white shrink-0">
						{#if $currentUser.avatar_url}
							<img src={$currentUser.avatar_url} alt="" class="h-8 w-8 rounded-full object-cover" />
						{:else}
							{getInitials($currentUser.display_name)}
						{/if}
					</div>
					<div class="min-w-0 flex-1">
						<p class="text-sm font-medium text-[var(--color-foreground)] truncate">{$currentUser.display_name}</p>
						<p class="text-xs text-[var(--color-muted-fg)] truncate">{$currentUser.email}</p>
					</div>
					<button
						onclick={handleLogout}
						class="shrink-0 rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-destructive)] transition-colors"
						title="Sign out"
					>
						<LogOut class="h-4 w-4" />
					</button>
				</div>
			</div>
		</aside>

		<!-- Main content -->
		<div class="flex-1 flex flex-col min-w-0">
			<!-- Extra bottom padding on mobile to avoid overlap with bottom tab bar -->
			<main class="flex-1 overflow-auto p-4 sm:p-6 lg:p-8 pb-20 lg:pb-8">
				{@render children()}
			</main>
		</div>
	</div>

	<!-- Mobile bottom tab bar (hidden on lg+) -->
	<nav class="fixed bottom-0 left-0 right-0 z-30 flex border-t border-[var(--color-border)] bg-[var(--color-card)] lg:hidden">
		{#each bottomTabItems as item}
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
					<svelte:component this={item.icon} class="h-5 w-5" />
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
