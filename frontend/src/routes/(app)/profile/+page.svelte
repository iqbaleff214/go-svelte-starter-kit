<script lang="ts">
	import { onMount } from 'svelte';
	import Button from '$components/ui/Button.svelte';
	import Input from '$components/ui/Input.svelte';
	import { userApi } from '$api/user';
	import { authStore, currentUser } from '$stores/auth';
	import { toast } from '$stores/toast';
	import type { ApiError, ProfileResponse } from '$types';

	let profile = $state<ProfileResponse | null>(null);
	let displayName = $state('');
	let bio = $state('');
	let saving = $state(false);
	let uploading = $state(false);
	let avatarInput: HTMLInputElement;

	onMount(async () => {
		try {
			profile = await userApi.getProfile();
			displayName = profile.display_name;
			bio = profile.bio ?? '';
		} catch {
			toast.error('Error', 'Could not load profile.');
		}
	});

	async function handleSave(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		try {
			const updated = await userApi.updateProfile({
				display_name: displayName || undefined,
				bio: bio || undefined
			});
			profile = updated;
			authStore.updateUser({ display_name: updated.display_name });
			toast.success('Saved', 'Profile updated successfully.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Save failed', apiErr.message);
		} finally {
			saving = false;
		}
	}

	async function handleAvatarChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		uploading = true;
		try {
			const res = await userApi.uploadAvatar(file);
			if (profile) profile = { ...profile, avatar_url: res.avatar_url };
			authStore.updateUser({ avatar_url: res.avatar_url });
			toast.success('Avatar updated', 'Your profile picture has been updated.');
		} catch (err) {
			const apiErr = err as ApiError;
			toast.error('Upload failed', apiErr.message);
		} finally {
			uploading = false;
			input.value = '';
		}
	}

	function getInitials(name: string) {
		return name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2);
	}
</script>

<svelte:head>
	<title>Profile — StarterKit</title>
</svelte:head>

<div class="max-w-2xl">
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-[var(--color-foreground)]">Profile</h1>
		<p class="mt-1 text-sm text-[var(--color-muted-fg)]">Manage your public profile information.</p>
	</div>

	<!-- Avatar -->
	<div class="mb-8 rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6">
		<h2 class="text-base font-semibold text-[var(--color-foreground)] mb-4">Avatar</h2>
		<div class="flex items-center gap-4">
			<div class="h-16 w-16 rounded-full bg-[var(--color-primary)] flex items-center justify-center text-lg font-bold text-white shrink-0 overflow-hidden">
				{#if profile?.avatar_url}
					<img src={profile.avatar_url} alt="Avatar" class="h-16 w-16 object-cover" />
				{:else}
					{getInitials($currentUser?.display_name ?? 'U')}
				{/if}
			</div>
			<div class="flex flex-col gap-2">
				<input
					bind:this={avatarInput}
					type="file"
					accept="image/jpeg,image/png,image/webp"
					class="hidden"
					onchange={handleAvatarChange}
				/>
				<Button
					variant="outline"
					size="sm"
					loading={uploading}
					onclick={() => avatarInput.click()}
				>
					{uploading ? 'Uploading…' : 'Change avatar'}
				</Button>
				<p class="text-xs text-[var(--color-muted-fg)]">JPEG, PNG or WebP. Max 2 MB.</p>
			</div>
		</div>
	</div>

	<!-- Profile details -->
	<div class="rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-card)] p-6">
		<h2 class="text-base font-semibold text-[var(--color-foreground)] mb-4">Details</h2>
		<form onsubmit={handleSave} class="flex flex-col gap-4">
			<div>
				<p class="text-sm font-medium text-[var(--color-foreground)] mb-1.5">Email</p>
				<p class="text-sm text-[var(--color-muted-fg)] py-2">{profile?.email ?? '…'}</p>
			</div>

			<Input
				label="Display name"
				type="text"
				placeholder="Your name"
				bind:value={displayName}
			/>

			<div class="flex flex-col gap-1.5">
				<label class="text-sm font-medium text-[var(--color-foreground)]" for="bio">Bio</label>
				<textarea
					id="bio"
					rows="3"
					maxlength="500"
					placeholder="Tell us a little about yourself…"
					bind:value={bio}
					class="w-full rounded-[var(--radius)] border border-[var(--color-border)] bg-[var(--color-background)] px-3 py-2 text-sm text-[var(--color-foreground)] placeholder:text-[var(--color-muted-fg)] focus:outline-none focus:ring-2 focus:ring-[var(--color-ring)] resize-none"
				></textarea>
				<p class="text-xs text-[var(--color-muted-fg)] text-right">{bio.length}/500</p>
			</div>

			<div class="flex items-center justify-between pt-2">
				<a href="/profile/security" class="text-sm text-[var(--color-primary)] hover:underline">
					Security settings →
				</a>
				<Button type="submit" variant="primary" loading={saving}>
					{saving ? 'Saving…' : 'Save changes'}
				</Button>
			</div>
		</form>
	</div>
</div>
