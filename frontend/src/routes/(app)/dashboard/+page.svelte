<script lang="ts">
	import { currentUser } from '$stores/auth';
	import { User, Bell, Key, Bot, ArrowRight } from 'lucide-svelte';

	const quickActions = [
		{
			href: '/profile',
			icon: User,
			title: 'Complete your profile',
			desc: 'Add a bio and avatar.',
			color: 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400'
		},
		{
			href: '/notifications',
			icon: Bell,
			title: 'View notifications',
			desc: 'Check your latest updates.',
			color: 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400'
		},
		{
			href: '/ai',
			icon: Bot,
			title: 'Try the AI assistant',
			desc: 'Chat with Claude.',
			color: 'bg-purple-50 text-purple-600 dark:bg-purple-900/20 dark:text-purple-400'
		},
		{
			href: '/profile/api-keys',
			icon: Key,
			title: 'Create an API key',
			desc: 'Integrate with external apps.',
			color: 'bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400'
		}
	];
</script>

<svelte:head>
	<title>Dashboard — StarterKit</title>
</svelte:head>

<div class="max-w-4xl">
	<!-- Header -->
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">
			Good {new Date().getHours() < 12 ? 'morning' : new Date().getHours() < 17 ? 'afternoon' : 'evening'},
			{$currentUser?.display_name?.split(' ')[0] ?? 'there'} 👋
		</h1>
		<p class="mt-1 text-sm text-[var(--color-muted-fg)]">
			Here's a quick overview of your account.
		</p>
	</div>

	<!-- Stats row -->
	<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-8">
		{#each [
			{ label: 'Email', value: $currentUser?.email_verified_at ? 'Verified' : 'Unverified', ok: !!$currentUser?.email_verified_at },
			{ label: '2FA', value: $currentUser?.two_fa_enabled ? 'Enabled' : 'Disabled', ok: !!$currentUser?.two_fa_enabled },
			{ label: 'Notifications', value: '0 unread', ok: true },
			{ label: 'API Keys', value: '0 active', ok: true }
		] as stat}
			<div class="rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
				<p class="text-xs text-[var(--color-muted-fg)] mb-1">{stat.label}</p>
				<p class="text-sm font-semibold {stat.ok ? 'text-[var(--color-success)]' : 'text-[var(--color-warning)]'}">
					{stat.value}
				</p>
			</div>
		{/each}
	</div>

	<!-- Quick actions -->
	<h2 class="text-base font-semibold text-[var(--color-foreground)] mb-4">Quick actions</h2>
	<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
		{#each quickActions as action}
			<a
				href={action.href}
				class="group flex items-center gap-4 rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-5 hover:shadow-[var(--shadow-md)] transition-shadow"
			>
				<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--radius)] {action.color}">
					<svelte:component this={action.icon} class="h-5 w-5" />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-sm font-medium text-[var(--color-foreground)]">{action.title}</p>
					<p class="text-xs text-[var(--color-muted-fg)] mt-0.5">{action.desc}</p>
				</div>
				<ArrowRight class="h-4 w-4 text-[var(--color-muted-fg)] group-hover:text-[var(--color-foreground)] transition-colors shrink-0" />
			</a>
		{/each}
	</div>

	<!-- Account info -->
	<div class="mt-8 rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6">
		<h2 class="text-base font-semibold text-[var(--color-foreground)] mb-4">Account details</h2>
		<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-3">
			{#each [
				{ label: 'User ID', value: $currentUser?.id },
				{ label: 'Email', value: $currentUser?.email },
				{ label: 'Display name', value: $currentUser?.display_name },
				{ label: 'Member since', value: $currentUser?.created_at
					? new Date($currentUser.created_at).toLocaleDateString()
					: '—' }
			] as field}
				<div>
					<dt class="text-xs text-[var(--color-muted-fg)]">{field.label}</dt>
					<dd class="mt-0.5 text-sm font-medium text-[var(--color-foreground)] truncate">{field.value ?? '—'}</dd>
				</div>
			{/each}
		</dl>
	</div>
</div>
