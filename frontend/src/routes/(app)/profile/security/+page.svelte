<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$components/ui/Button.svelte';
	import Input from '$components/ui/Input.svelte';
	import { authApi } from '$api/auth';
	import { userApi } from '$api/user';
	import { authStore, currentUser } from '$stores/auth';
	import { toast } from '$stores/toast';
	import { goto } from '$app/navigation';
	import type { ApiError, Session, TwoFASetupResponse } from '$types';

	// ── Password ────────────────────────────────────────────────────────────────

	let currentPassword = $state('');
	let newPassword = $state('');
	let savingPassword = $state(false);

	async function handleChangePassword(e: SubmitEvent) {
		e.preventDefault();
		savingPassword = true;
		try {
			await userApi.changePassword(currentPassword, newPassword);
			currentPassword = '';
			newPassword = '';
			toast.success('Password changed', 'Your password has been updated.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Failed', apiErr.message);
		} finally {
			savingPassword = false;
		}
	}

	// ── 2FA ─────────────────────────────────────────────────────────────────────

	let twoFAEnabled = $derived($currentUser?.two_fa_enabled ?? false);
	let setupData = $state<TwoFASetupResponse | null>(null);
	let confirmCode = $state('');
	let backupCodes = $state<string[]>([]);
	let settingUp2FA = $state(false);
	let confirming2FA = $state(false);
	let disableCode = $state('');
	let disableBackup = $state('');
	let disabling2FA = $state(false);

	async function startSetup() {
		settingUp2FA = true;
		try {
			setupData = await authApi.twoFaSetup();
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Setup failed', apiErr.message);
		} finally {
			settingUp2FA = false;
		}
	}

	async function confirmSetup(e: SubmitEvent) {
		e.preventDefault();
		confirming2FA = true;
		try {
			const res = await authApi.twoFaConfirm(confirmCode);
			backupCodes = res.backup_codes;
			setupData = null;
			confirmCode = '';
			authStore.updateUser({ two_fa_enabled: true });
			toast.success('2FA enabled', 'Two-factor authentication is now active.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Confirmation failed', apiErr.message);
		} finally {
			confirming2FA = false;
		}
	}

	async function handleDisable2FA(e: SubmitEvent) {
		e.preventDefault();
		disabling2FA = true;
		try {
			await authApi.twoFaDisable(disableCode || undefined, disableBackup || undefined);
			disableCode = '';
			disableBackup = '';
			authStore.updateUser({ two_fa_enabled: false });
			toast.success('2FA disabled', 'Two-factor authentication has been disabled.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Failed', apiErr.message);
		} finally {
			disabling2FA = false;
		}
	}

	// ── Sessions ────────────────────────────────────────────────────────────────

	let sessions = $state<Session[]>([]);
	let loadingSessions = $state(true);
	let revokingId = $state<string | null>(null);
	let revokingAll = $state(false);

	onMount(async () => {
		try {
			sessions = await userApi.listSessions();
		} catch {
			toast.error('Error', 'Could not load sessions.');
		} finally {
			loadingSessions = false;
		}
	});

	async function revokeSession(id: string) {
		revokingId = id;
		try {
			await userApi.revokeSession(id);
			sessions = sessions.filter((s) => s.id !== id);
			toast.success('Session revoked', 'The session has been signed out.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Failed', apiErr.message);
		} finally {
			revokingId = null;
		}
	}

	async function revokeAll() {
		revokingAll = true;
		try {
			await userApi.revokeAllOtherSessions();
			sessions = sessions.filter((s) => s.is_current);
			toast.success('Sessions revoked', 'All other sessions have been signed out.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Failed', apiErr.message);
		} finally {
			revokingAll = false;
		}
	}

	// ── Danger zone ─────────────────────────────────────────────────────────────

	let deletingAccount = $state(false);

	async function handleDeleteAccount() {
		if (!confirm('Are you sure? This permanently deletes your account and all data. This cannot be undone.')) return;
		deletingAccount = true;
		try {
			await userApi.deleteAccount();
			authStore.clearAuth();
			toast.info('Account deleted', 'Your account has been permanently deleted.');
			goto('/login');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Failed', apiErr.message);
			deletingAccount = false;
		}
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleString();
	}
</script>

<svelte:head>
	<title>Security — StarterKit</title>
</svelte:head>

<div class="max-w-2xl space-y-6">
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Security</h1>
		<p class="mt-1 text-sm text-[var(--color-muted-fg)]">
			Manage your password, two-factor authentication, and active sessions.
		</p>
	</div>

	<!-- Change password -->
	<section class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6">
		<h2 class="text-base font-semibold text-[var(--color-foreground)] mb-4">Change password</h2>
		<form onsubmit={handleChangePassword} class="flex flex-col gap-4">
			<Input
				label="Current password"
				type="password"
				autocomplete="current-password"
				bind:value={currentPassword}
			/>
			<Input
				label="New password"
				type="password"
				autocomplete="new-password"
				bind:value={newPassword}
			/>
			<div class="flex justify-end">
				<Button type="submit" variant="primary" loading={savingPassword}>
					{savingPassword ? 'Saving…' : 'Update password'}
				</Button>
			</div>
		</form>
	</section>

	<!-- Two-factor authentication -->
	<section class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6">
		<div class="flex items-center justify-between mb-4">
			<div>
				<h2 class="text-base font-semibold text-[var(--color-foreground)]">Two-factor authentication</h2>
				<p class="text-sm text-[var(--color-muted-fg)] mt-0.5">
					{twoFAEnabled ? 'Enabled — your account has an extra layer of security.' : 'Add an extra layer of security to your account.'}
				</p>
			</div>
			<span class="text-xs font-medium px-2 py-1 rounded-full {twoFAEnabled ? 'bg-[var(--color-success)]/10 text-[var(--color-success)]' : 'bg-[var(--color-muted)] text-[var(--color-muted-fg)]'}">
				{twoFAEnabled ? 'Enabled' : 'Disabled'}
			</span>
		</div>

		{#if backupCodes.length > 0}
			<!-- Show backup codes after successful setup -->
			<div class="rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-muted)] p-4 mb-4">
				<p class="text-sm font-medium text-[var(--color-foreground)] mb-2">
					Save these backup codes in a safe place. Each can only be used once.
				</p>
				<div class="grid grid-cols-2 gap-1.5">
					{#each backupCodes as code}
						<code class="text-xs font-mono text-[var(--color-foreground)] bg-[var(--color-card)] rounded px-2 py-1">{code}</code>
					{/each}
				</div>
				<Button variant="outline" size="sm" class="mt-3" onclick={() => (backupCodes = [])}>
					I've saved my codes
				</Button>
			</div>
		{:else if setupData}
			<!-- QR code setup flow -->
			<div class="space-y-4">
				<p class="text-sm text-[var(--color-muted-fg)]">
					Scan this QR code with your authenticator app (e.g. Google Authenticator, Authy), then enter the 6-digit code to confirm.
				</p>
				<div class="flex flex-col items-center gap-3 rounded-[var(--radius)] border border-[var(--color-border)] p-4">
					<img
						src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&data={encodeURIComponent(setupData.otpauth_url)}"
						alt="QR code"
						class="h-44 w-44"
					/>
					<p class="text-xs text-[var(--color-muted-fg)] text-center">
						Can't scan? Enter this secret manually:<br />
						<code class="font-mono text-[var(--color-foreground)]">{setupData.secret}</code>
					</p>
				</div>
				<form onsubmit={confirmSetup} class="flex gap-2">
					<Input
						type="text"
						inputmode="numeric"
						pattern="[0-9]*"
						maxlength={6}
						placeholder="000000"
						autocomplete="one-time-code"
						bind:value={confirmCode}
					/>
					<Button type="submit" variant="primary" loading={confirming2FA}>
						{confirming2FA ? 'Confirming…' : 'Confirm'}
					</Button>
				</form>
				<button
					type="button"
					onclick={() => (setupData = null)}
					class="text-sm text-[var(--color-muted-fg)] hover:underline"
				>
					Cancel
				</button>
			</div>
		{:else if !twoFAEnabled}
			<Button variant="outline" loading={settingUp2FA} onclick={startSetup}>
				{settingUp2FA ? 'Setting up…' : 'Enable 2FA'}
			</Button>
		{:else}
			<!-- Disable 2FA -->
			<form onsubmit={handleDisable2FA} class="space-y-3">
				<p class="text-sm text-[var(--color-muted-fg)]">
					Enter your current authenticator code or a backup code to disable 2FA.
				</p>
				<div class="flex gap-2">
					<Input
						type="text"
						inputmode="numeric"
						pattern="[0-9]*"
						maxlength={6}
						placeholder="6-digit code"
						bind:value={disableCode}
					/>
					<span class="self-center text-sm text-[var(--color-muted-fg)]">or</span>
					<Input
						type="text"
						placeholder="backup code"
						bind:value={disableBackup}
					/>
				</div>
				<Button type="submit" variant="destructive" loading={disabling2FA}>
					{disabling2FA ? 'Disabling…' : 'Disable 2FA'}
				</Button>
			</form>
		{/if}
	</section>

	<!-- Active sessions -->
	<section class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6">
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-base font-semibold text-[var(--color-foreground)]">Active sessions</h2>
			{#if sessions.filter((s) => !s.is_current).length > 0}
				<Button variant="outline" size="sm" loading={revokingAll} onclick={revokeAll}>
					{revokingAll ? 'Signing out…' : 'Sign out all others'}
				</Button>
			{/if}
		</div>

		{#if loadingSessions}
			<p class="text-sm text-[var(--color-muted-fg)]">Loading…</p>
		{:else if sessions.length === 0}
			<p class="text-sm text-[var(--color-muted-fg)]">No active sessions.</p>
		{:else}
			<ul class="space-y-3">
				{#each sessions as session}
					<li class="flex items-start justify-between gap-3 rounded-[var(--radius)] border border-[var(--color-border)] p-3">
						<div class="min-w-0 flex-1">
							<p class="text-sm font-medium text-[var(--color-foreground)] truncate">
								{session.user_agent || 'Unknown browser'}
								{#if session.is_current}
									<span class="ml-1.5 text-xs font-normal text-[var(--color-success)]">(this device)</span>
								{/if}
							</p>
							<p class="text-xs text-[var(--color-muted-fg)] mt-0.5">
								{session.ip_address} · Last seen {formatDate(session.last_seen_at)}
							</p>
						</div>
						{#if !session.is_current}
							<Button
								variant="ghost"
								size="sm"
								loading={revokingId === session.id}
								onclick={() => revokeSession(session.id)}
							>
								Revoke
							</Button>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<!-- Danger zone -->
	<section class="rounded-[var(--radius-lg)] border border-[var(--color-destructive)]/30 bg-[var(--color-card)] p-6">
		<h2 class="text-base font-semibold text-[var(--color-destructive)] mb-1">Danger zone</h2>
		<p class="text-sm text-[var(--color-muted-fg)] mb-4">
			Permanently delete your account and all associated data. This action cannot be undone.
		</p>
		<Button variant="destructive" loading={deletingAccount} onclick={handleDeleteAccount}>
			{deletingAccount ? 'Deleting…' : 'Delete account'}
		</Button>
	</section>
</div>
