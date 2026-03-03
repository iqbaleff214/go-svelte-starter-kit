<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore, currentUser, isLoading } from '$stores/auth';
	import { authApi } from '$api/auth';
	import { toast } from '$stores/toast';
	import { onMount } from 'svelte';
	import {
		LayoutDashboard, User, Bell, Key, Bot, Shield, LogOut, Menu, X
	} from 'lucide-svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	let sidebarOpen = $state(false);

	const navItems = [
		{ href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
		{ href: '/notifications', icon: Bell, label: 'Notifications' },
		{ href: '/ai', icon: Bot, label: 'AI Assistant' },
		{ href: '/profile', icon: User, label: 'Profile' },
		{ href: '/profile/api-keys', icon: Key, label: 'API Keys' }
	];

	const adminItems = [
		{ href: '/admin', icon: Shield, label: 'Admin' }
	];

	onMount(() => {
		const unsub = isLoading.subscribe((loading) => {
			if (!loading && !$currentUser) {
				goto('/login');
				unsub();
			}
		});
	});

	async function handleLogout() {
		try {
			await authApi.logout();
		} finally {
			authStore.clearAuth();
			toast.info('Signed out', 'See you next time!');
			goto('/login');
		}
	}

	function isActive(href: string) {
		return $page.url.pathname === href || $page.url.pathname.startsWith(href + '/');
	}

	function getInitials(name: string) {
		return name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2);
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
	<div class="min-h-screen bg-[var(--color-background)] flex">
		<!-- Sidebar overlay (mobile) -->
		{#if sidebarOpen}
			<button
				class="fixed inset-0 z-20 bg-black/40 lg:hidden"
				onclick={() => (sidebarOpen = false)}
				aria-label="Close sidebar"
			></button>
		{/if}

		<!-- Sidebar -->
		<aside
			class="
				fixed top-0 left-0 z-30 h-full w-64 border-r border-[var(--color-border)]
				bg-[var(--color-card)] flex flex-col transition-transform duration-200
				{sidebarOpen ? 'translate-x-0' : '-translate-x-full'} lg:translate-x-0 lg:static lg:z-auto
			"
		>
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
						onclick={() => (sidebarOpen = false)}
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
					</a>
				{/each}

				{#if $currentUser && ['admin', 'superadmin'].some((r) => ($currentUser as any).roles?.includes(r))}
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
			<!-- Top bar (mobile) -->
			<header class="flex h-16 items-center gap-3 border-b border-[var(--color-border)] px-4 lg:hidden shrink-0">
				<button
					onclick={() => (sidebarOpen = true)}
					class="rounded p-2 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] transition-colors"
					aria-label="Open sidebar"
				>
					<Menu class="h-5 w-5" />
				</button>
				<span class="font-semibold text-[var(--color-foreground)]">StarterKit</span>
			</header>

			<main class="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
