<script lang="ts">
	import Button from '$components/ui/Button.svelte';
	import { goto } from '$app/navigation';
	import { isAuthenticated } from '$stores/auth';

	const features = [
		{ icon: '🔐', title: 'Authentication', desc: 'Email/password, Google OAuth, 2FA, and secure session management.' },
		{ icon: '📧', title: 'Email Worker', desc: 'Async email delivery via Redis queue with retry and templating.' },
		{ icon: '🔔', title: 'Real-time Notifications', desc: 'WebSocket-powered push notifications with persistent history.' },
		{ icon: '🛡️', title: 'RBAC', desc: 'Granular role-based access control with admin panel.' },
		{ icon: '🤖', title: 'AI Agent', desc: 'Anthropic Claude integration with streaming and tool calling.' },
		{ icon: '🔑', title: 'Public API', desc: 'Versioned REST API with API key auth and OpenAPI docs.' }
	];
</script>

<svelte:head>
	<title>Go + Svelte Starter Kit</title>
	<meta name="description" content="A production-ready full-stack starter kit with SvelteKit and Go." />
</svelte:head>

<div class="min-h-screen bg-[var(--color-background)]">
	<!-- Nav -->
	<nav class="border-b border-[var(--color-border)] px-4 sm:px-6">
		<div class="mx-auto flex max-w-6xl h-16 items-center justify-between">
			<a href="/" class="flex items-center gap-2 font-bold text-lg text-[var(--color-foreground)]">
				<span class="text-[var(--color-primary)]">⚡</span>
				StarterKit
			</a>
			<div class="flex items-center gap-3">
				{#if $isAuthenticated}
					<Button variant="primary" size="sm" onclick={() => goto('/dashboard')}>
						Dashboard
					</Button>
				{:else}
					<Button variant="ghost" size="sm" onclick={() => goto('/login')}>Sign in</Button>
					<Button variant="primary" size="sm" onclick={() => goto('/register')}>Get started</Button>
				{/if}
			</div>
		</div>
	</nav>

	<!-- Hero -->
	<section class="px-4 sm:px-6 py-20 sm:py-28 text-center">
		<div class="mx-auto max-w-3xl">
			<div class="mb-4 inline-flex items-center gap-2 rounded-full border border-[var(--color-border)] px-3 py-1 text-xs text-[var(--color-muted-fg)]">
				<span class="inline-block h-1.5 w-1.5 rounded-full bg-[var(--color-success)]"></span>
				v1.0 · Phase 1 scaffold
			</div>
			<h1 class="text-4xl sm:text-5xl md:text-6xl font-bold tracking-tight text-[var(--color-foreground)] leading-tight">
				Build faster with<br />
				<span class="text-[var(--color-primary)]">Go + SvelteKit</span>
			</h1>
			<p class="mt-6 text-lg text-[var(--color-muted-fg)] max-w-xl mx-auto">
				A production-ready starter kit with auth, real-time notifications, RBAC, AI integration,
				and a public API — all set up so you can ship faster.
			</p>
			<div class="mt-10 flex flex-col sm:flex-row gap-3 justify-center">
				<Button variant="primary" size="lg" onclick={() => goto('/register')}>
					Get started — it's free
				</Button>
				<a
					href="https://github.com/404nfid/go-svelte-starter-kit"
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center justify-center gap-2 h-12 px-6 text-base font-medium rounded-[var(--radius)] border border-[var(--color-border)] hover:bg-[var(--color-muted)] transition-colors"
				>
					View on GitHub
				</a>
			</div>
		</div>
	</section>

	<!-- Features grid -->
	<section class="px-4 sm:px-6 pb-20 sm:pb-28">
		<div class="mx-auto max-w-6xl">
			<h2 class="text-center text-2xl font-bold text-[var(--color-foreground)] mb-12">
				Everything you need, nothing you don't
			</h2>
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
				{#each features as f}
					<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6 hover:shadow-[var(--shadow-md)] transition-shadow">
						<div class="text-3xl mb-3">{f.icon}</div>
						<h3 class="font-semibold text-[var(--color-card-fg)] mb-2">{f.title}</h3>
						<p class="text-sm text-[var(--color-muted-fg)] leading-relaxed">{f.desc}</p>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Stack badges -->
	<section class="border-t border-[var(--color-border)] px-4 sm:px-6 py-12">
		<div class="mx-auto max-w-6xl text-center">
			<p class="text-xs uppercase tracking-widest text-[var(--color-muted-fg)] mb-6">Built with</p>
			<div class="flex flex-wrap justify-center gap-3">
				{#each ['SvelteKit 2', 'Go 1.22', 'PostgreSQL 16', 'Redis 7', 'TailwindCSS v4', 'Claude AI'] as tech}
					<span class="rounded-full border border-[var(--color-border)] px-3 py-1 text-sm text-[var(--color-muted-fg)]">
						{tech}
					</span>
				{/each}
			</div>
		</div>
	</section>
</div>
