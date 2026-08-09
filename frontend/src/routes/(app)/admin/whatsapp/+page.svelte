<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import QRCode from 'qrcode';
	import { whatsappApi, type WASession, type WAMessage, type QREvent } from '$api/whatsapp';
	import { toast } from '$stores/toast';
	import { Plus, Trash2, Pause, Play, QrCode, Phone, RefreshCw, Send, X } from 'lucide-svelte';

	// ---- Sessions ----
	let sessions = $state<WASession[]>([]);
	let loadingSessions = $state(true);

	// ---- Add session modal ----
	let showAddModal = $state(false);
	let newName = $state('');
	let addLoading = $state(false);

	// ---- QR modal ----
	let qrSession = $state<WASession | null>(null);
	let qrDataURL = $state('');
	let qrStatus = $state<'scanning' | 'connected' | 'timeout' | 'error'>('scanning');
	let qrES: EventSource | null = null;

	// ---- Pair modal ----
	let pairSession = $state<WASession | null>(null);
	let pairPhone = $state('');
	let pairCode = $state('');
	let pairLoading = $state(false);

	// ---- Send test modal ----
	let sendSession = $state<WASession | null>(null);
	let sendRecipient = $state('');
	let sendBody = $state('');
	let sendLoading = $state(false);

	// ---- Messages tab ----
	let activeTab = $state<'sessions' | 'messages'>('sessions');
	let messages = $state<WAMessage[]>([]);
	let msgTotal = $state(0);
	let msgOffset = $state(0);
	const msgLimit = 20;
	let msgStatus = $state('');
	let loadingMessages = $state(false);

	// ---- Delete confirm ----
	let deleteTarget = $state<WASession | null>(null);
	let deleteLoading = $state(false);

	onMount(loadSessions);

	onDestroy(() => qrES?.close());

	async function loadSessions() {
		loadingSessions = true;
		try {
			sessions = (await whatsappApi.listSessions()) ?? [];
		} catch {
			toast.error('Error', 'Could not load WhatsApp sessions.');
		} finally {
			loadingSessions = false;
		}
	}

	async function addSession() {
		if (!newName.trim()) return;
		addLoading = true;
		try {
			await whatsappApi.createSession(newName.trim());
			showAddModal = false;
			newName = '';
			await loadSessions();
			toast.success('Session created', 'Now pair it with a WhatsApp number.');
		} catch {
			toast.error('Error', 'Failed to create session.');
		} finally {
			addLoading = false;
		}
	}

	async function togglePause(sess: WASession) {
		try {
			if (sess.paused) {
				await whatsappApi.resumeSession(sess.id);
				toast.success('Resumed', `${sess.name} is active.`);
			} else {
				await whatsappApi.pauseSession(sess.id);
				toast.success('Paused', `${sess.name} paused.`);
			}
			await loadSessions();
		} catch {
			toast.error('Error', 'Could not update session.');
		}
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		deleteLoading = true;
		try {
			await whatsappApi.deleteSession(deleteTarget.id);
			sessions = sessions.filter((s) => s.id !== deleteTarget!.id);
			toast.success('Deleted', `${deleteTarget.name} removed.`);
			deleteTarget = null;
		} catch {
			toast.error('Error', 'Could not delete session.');
		} finally {
			deleteLoading = false;
		}
	}

	function openQR(sess: WASession) {
		qrES?.close();
		qrSession = sess;
		qrDataURL = '';
		qrStatus = 'scanning';

		qrES = whatsappApi.streamQR(
			sess.id,
			async (evt: QREvent) => {
				if (evt.type === 'qr') {
					qrDataURL = await QRCode.toDataURL(evt.code, { width: 280, margin: 2 });
				} else if (evt.type === 'connected') {
					qrStatus = 'connected';
					qrES?.close();
					await loadSessions();
				} else if (evt.type === 'timeout') {
					qrStatus = 'timeout';
				} else {
					qrStatus = 'error';
				}
			},
			() => {
				if (qrStatus === 'scanning') qrStatus = 'error';
			}
		);
	}

	function closeQR() {
		qrES?.close();
		qrES = null;
		qrSession = null;
	}

	async function requestPairCode() {
		if (!pairSession || !pairPhone.trim()) return;
		pairLoading = true;
		try {
			const res = await whatsappApi.getPairingCode(pairSession.id, pairPhone.trim());
			pairCode = res.code;
		} catch {
			toast.error('Error', 'Could not get pairing code.');
		} finally {
			pairLoading = false;
		}
	}

	async function sendTest() {
		if (!sendRecipient.trim() || !sendBody.trim()) return;
		sendLoading = true;
		try {
			await whatsappApi.sendMessage(sendRecipient.trim(), sendBody.trim());
			toast.success('Queued', 'Message added to send queue.');
			sendSession = null;
			sendRecipient = '';
			sendBody = '';
		} catch {
			toast.error('Error', 'Could not queue message.');
		} finally {
			sendLoading = false;
		}
	}

	async function loadMessages() {
		loadingMessages = true;
		try {
			const res = await whatsappApi.listMessages({
				limit: msgLimit,
				offset: msgOffset,
				status: msgStatus || undefined
			});
			messages = (res as unknown as WAMessage[]) ?? [];
		} catch {
			toast.error('Error', 'Could not load messages.');
		} finally {
			loadingMessages = false;
		}
	}

	$effect(() => {
		if (activeTab === 'messages') loadMessages();
	});

	function statusColor(status: string) {
		switch (status) {
			case 'connected':
				return 'text-green-600 dark:text-green-400';
			case 'disconnected':
				return 'text-yellow-600 dark:text-yellow-400';
			case 'banned':
				return 'text-red-600 dark:text-red-400';
			default:
				return 'text-[var(--color-text-muted)]';
		}
	}

	function msgStatusColor(status: string) {
		switch (status) {
			case 'sent':
				return 'text-green-600 dark:text-green-400';
			case 'failed':
				return 'text-red-600 dark:text-red-400';
			default:
				return 'text-[var(--color-text-muted)]';
		}
	}
</script>

<div class="p-6 max-w-6xl mx-auto space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-[var(--color-text)]">WhatsApp</h1>
			<p class="text-sm text-[var(--color-text-muted)]">Manage WA numbers and message queue</p>
		</div>
		<button
			onclick={() => (showAddModal = true)}
			class="flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:opacity-90 transition-opacity"
		>
			<Plus class="w-4 h-4" />
			Add Number
		</button>
	</div>

	<!-- Tabs -->
	<div class="flex gap-1 border-b border-[var(--color-border)]">
		{#each [['sessions', 'Sessions'], ['messages', 'Message Logs']] as [key, label]}
			<button
				onclick={() => (activeTab = key as 'sessions' | 'messages')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === key
					? 'border-[var(--color-primary)] text-[var(--color-primary)]'
					: 'border-transparent text-[var(--color-text-muted)] hover:text-[var(--color-text)]'}"
			>
				{label}
			</button>
		{/each}
	</div>

	<!-- Sessions tab -->
	{#if activeTab === 'sessions'}
		{#if loadingSessions}
			<div class="flex justify-center py-12 text-[var(--color-text-muted)]">Loading...</div>
		{:else if sessions.length === 0}
			<div class="text-center py-12 text-[var(--color-text-muted)]">
				No sessions yet. Add a WhatsApp number to get started.
			</div>
		{:else}
			<div class="grid gap-4">
				{#each sessions as sess}
					<div
						class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)]"
					>
						<div class="space-y-1 min-w-0">
							<div class="flex items-center gap-2">
								<span class="font-semibold text-[var(--color-text)] truncate">{sess.name}</span>
								{#if sess.paused}
									<span class="text-xs px-2 py-0.5 rounded-full bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400">paused</span>
								{/if}
							</div>
							<div class="text-sm text-[var(--color-text-muted)] flex flex-wrap gap-x-4 gap-y-1">
								<span class={statusColor(sess.status)}>{sess.status}</span>
								{#if sess.phone}
									<span>+{sess.phone}</span>
								{/if}
								<span>Sent today: {sess.sent_today}</span>
							</div>
						</div>
						<div class="flex flex-wrap gap-2 shrink-0">
							{#if sess.status !== 'connected'}
								<button
									onclick={() => openQR(sess)}
									class="flex items-center gap-1 px-3 py-1.5 text-xs rounded-lg border border-[var(--color-border)] hover:bg-[var(--color-border)] transition-colors"
								>
									<QrCode class="w-3.5 h-3.5" /> QR
								</button>
								<button
									onclick={() => { pairSession = sess; pairCode = ''; pairPhone = ''; }}
									class="flex items-center gap-1 px-3 py-1.5 text-xs rounded-lg border border-[var(--color-border)] hover:bg-[var(--color-border)] transition-colors"
								>
									<Phone class="w-3.5 h-3.5" /> Pair Code
								</button>
							{/if}
							{#if sess.status === 'connected'}
								<button
									onclick={() => { sendSession = sess; sendRecipient = ''; sendBody = ''; }}
									class="flex items-center gap-1 px-3 py-1.5 text-xs rounded-lg border border-[var(--color-border)] hover:bg-[var(--color-border)] transition-colors"
								>
									<Send class="w-3.5 h-3.5" /> Test
								</button>
							{/if}
							<button
								onclick={() => togglePause(sess)}
								class="flex items-center gap-1 px-3 py-1.5 text-xs rounded-lg border border-[var(--color-border)] hover:bg-[var(--color-border)] transition-colors"
							>
								{#if sess.paused}
									<Play class="w-3.5 h-3.5" /> Resume
								{:else}
									<Pause class="w-3.5 h-3.5" /> Pause
								{/if}
							</button>
							<button
								onclick={() => (deleteTarget = sess)}
								class="flex items-center gap-1 px-3 py-1.5 text-xs rounded-lg border border-red-300 dark:border-red-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
							>
								<Trash2 class="w-3.5 h-3.5" />
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<button
			onclick={loadSessions}
			class="flex items-center gap-1 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
		>
			<RefreshCw class="w-3.5 h-3.5" /> Refresh
		</button>
	{/if}

	<!-- Messages tab -->
	{#if activeTab === 'messages'}
		<div class="space-y-4">
			<div class="flex gap-3 items-center flex-wrap">
				<select
					bind:value={msgStatus}
					onchange={() => { msgOffset = 0; loadMessages(); }}
					class="px-3 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-sm text-[var(--color-text)]"
				>
					<option value="">All statuses</option>
					<option value="queued">Queued</option>
					<option value="sent">Sent</option>
					<option value="failed">Failed</option>
				</select>
				<button
					onclick={() => { msgOffset = 0; loadMessages(); }}
					class="flex items-center gap-1 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
				>
					<RefreshCw class="w-3.5 h-3.5" /> Refresh
				</button>
			</div>

			{#if loadingMessages}
				<div class="flex justify-center py-8 text-[var(--color-text-muted)]">Loading...</div>
			{:else if messages.length === 0}
				<div class="text-center py-8 text-[var(--color-text-muted)]">No messages found.</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-[var(--color-border)]">
					<table class="w-full text-sm">
						<thead class="bg-[var(--color-surface)] border-b border-[var(--color-border)]">
							<tr>
								{#each ['Recipient', 'Body', 'Status', 'Queued', 'Sent'] as h}
									<th class="px-4 py-3 text-left font-medium text-[var(--color-text-muted)]">{h}</th>
								{/each}
							</tr>
						</thead>
						<tbody>
							{#each messages as msg}
								<tr class="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-surface)] transition-colors">
									<td class="px-4 py-3 text-[var(--color-text)]">{msg.recipient}</td>
									<td class="px-4 py-3 text-[var(--color-text-muted)] max-w-xs truncate">{msg.body}</td>
									<td class="px-4 py-3 font-medium {msgStatusColor(msg.status)}">{msg.status}</td>
									<td class="px-4 py-3 text-[var(--color-text-muted)] whitespace-nowrap">{new Date(msg.queued_at).toLocaleString()}</td>
									<td class="px-4 py-3 text-[var(--color-text-muted)] whitespace-nowrap">{msg.sent_at ? new Date(msg.sent_at).toLocaleString() : '—'}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination -->
				<div class="flex items-center gap-3 text-sm text-[var(--color-text-muted)]">
					<button
						onclick={() => { msgOffset = Math.max(0, msgOffset - msgLimit); loadMessages(); }}
						disabled={msgOffset === 0}
						class="px-3 py-1.5 rounded-lg border border-[var(--color-border)] disabled:opacity-40"
					>Prev</button>
					<span>Showing {msgOffset + 1}–{msgOffset + messages.length}</span>
					<button
						onclick={() => { msgOffset += msgLimit; loadMessages(); }}
						disabled={messages.length < msgLimit}
						class="px-3 py-1.5 rounded-lg border border-[var(--color-border)] disabled:opacity-40"
					>Next</button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Add session modal -->
{#if showAddModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="bg-[var(--color-background)] rounded-2xl p-6 w-full max-w-sm space-y-4 shadow-xl">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-text)]">Add WhatsApp Number</h2>
				<button onclick={() => (showAddModal = false)} class="text-[var(--color-text-muted)] hover:text-[var(--color-text)]"><X class="w-5 h-5" /></button>
			</div>
			<input
				bind:value={newName}
				placeholder="Label (e.g. Marketing WA)"
				class="w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text)] text-sm outline-none focus:border-[var(--color-primary)]"
			/>
			<div class="flex gap-3 justify-end">
				<button onclick={() => (showAddModal = false)} class="px-4 py-2 text-sm rounded-lg border border-[var(--color-border)] text-[var(--color-text-muted)]">Cancel</button>
				<button
					onclick={addSession}
					disabled={addLoading || !newName.trim()}
					class="px-4 py-2 text-sm rounded-lg bg-[var(--color-primary)] text-white font-medium disabled:opacity-50"
				>
					{addLoading ? 'Creating…' : 'Create'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- QR modal -->
{#if qrSession}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="bg-[var(--color-background)] rounded-2xl p-6 w-full max-w-sm space-y-4 shadow-xl text-center">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-text)]">Scan QR Code</h2>
				<button onclick={closeQR} class="text-[var(--color-text-muted)] hover:text-[var(--color-text)]"><X class="w-5 h-5" /></button>
			</div>
			<p class="text-sm text-[var(--color-text-muted)]">Open WhatsApp → Linked Devices → Link a Device</p>

			{#if qrStatus === 'scanning' && qrDataURL}
				<img src={qrDataURL} alt="QR code" class="mx-auto rounded-xl" />
			{:else if qrStatus === 'scanning'}
				<div class="h-64 flex items-center justify-center text-[var(--color-text-muted)] text-sm">Generating QR…</div>
			{:else if qrStatus === 'connected'}
				<div class="py-8 text-green-600 dark:text-green-400 font-semibold">Connected successfully!</div>
			{:else if qrStatus === 'timeout'}
				<div class="py-6 space-y-3">
					<p class="text-yellow-600 dark:text-yellow-400">QR code expired.</p>
					<button onclick={() => openQR(qrSession!)} class="px-4 py-2 text-sm rounded-lg bg-[var(--color-primary)] text-white">Try Again</button>
				</div>
			{:else}
				<div class="py-6 text-red-600 dark:text-red-400">Connection error. Try again.</div>
			{/if}
		</div>
	</div>
{/if}

<!-- Pair code modal -->
{#if pairSession}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="bg-[var(--color-background)] rounded-2xl p-6 w-full max-w-sm space-y-4 shadow-xl">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-text)]">Phone Pairing Code</h2>
				<button onclick={() => (pairSession = null)} class="text-[var(--color-text-muted)] hover:text-[var(--color-text)]"><X class="w-5 h-5" /></button>
			</div>
			<p class="text-sm text-[var(--color-text-muted)]">Enter the phone number to link (with country code, no +).</p>
			<input
				bind:value={pairPhone}
				placeholder="628123456789"
				class="w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text)] text-sm outline-none focus:border-[var(--color-primary)]"
			/>
			{#if pairCode}
				<div class="text-center py-3 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]">
					<p class="text-xs text-[var(--color-text-muted)] mb-1">Enter this code in WhatsApp → Linked Devices → Link with phone number</p>
					<p class="text-3xl font-mono font-bold tracking-widest text-[var(--color-text)]">{pairCode}</p>
				</div>
			{/if}
			<div class="flex gap-3 justify-end">
				<button onclick={() => (pairSession = null)} class="px-4 py-2 text-sm rounded-lg border border-[var(--color-border)] text-[var(--color-text-muted)]">Close</button>
				<button
					onclick={requestPairCode}
					disabled={pairLoading || !pairPhone.trim()}
					class="px-4 py-2 text-sm rounded-lg bg-[var(--color-primary)] text-white font-medium disabled:opacity-50"
				>
					{pairLoading ? 'Getting code…' : pairCode ? 'Refresh Code' : 'Get Code'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Send test modal -->
{#if sendSession}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="bg-[var(--color-background)] rounded-2xl p-6 w-full max-w-sm space-y-4 shadow-xl">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-[var(--color-text)]">Send Test Message</h2>
				<button onclick={() => (sendSession = null)} class="text-[var(--color-text-muted)] hover:text-[var(--color-text)]"><X class="w-5 h-5" /></button>
			</div>
			<input
				bind:value={sendRecipient}
				placeholder="Recipient (628xxx)"
				class="w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text)] text-sm outline-none focus:border-[var(--color-primary)]"
			/>
			<textarea
				bind:value={sendBody}
				rows="3"
				placeholder="Message body"
				class="w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text)] text-sm outline-none focus:border-[var(--color-primary)] resize-none"
			></textarea>
			<div class="flex gap-3 justify-end">
				<button onclick={() => (sendSession = null)} class="px-4 py-2 text-sm rounded-lg border border-[var(--color-border)] text-[var(--color-text-muted)]">Cancel</button>
				<button
					onclick={sendTest}
					disabled={sendLoading || !sendRecipient.trim() || !sendBody.trim()}
					class="px-4 py-2 text-sm rounded-lg bg-[var(--color-primary)] text-white font-medium disabled:opacity-50"
				>
					{sendLoading ? 'Queuing…' : 'Send'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Delete confirm modal -->
{#if deleteTarget}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
		<div class="bg-[var(--color-background)] rounded-2xl p-6 w-full max-w-sm space-y-4 shadow-xl">
			<h2 class="text-lg font-semibold text-[var(--color-text)]">Delete Session</h2>
			<p class="text-sm text-[var(--color-text-muted)]">Remove <strong class="text-[var(--color-text)]">{deleteTarget.name}</strong>? This disconnects the number and deletes all associated data.</p>
			<div class="flex gap-3 justify-end">
				<button onclick={() => (deleteTarget = null)} class="px-4 py-2 text-sm rounded-lg border border-[var(--color-border)] text-[var(--color-text-muted)]">Cancel</button>
				<button
					onclick={confirmDelete}
					disabled={deleteLoading}
					class="px-4 py-2 text-sm rounded-lg bg-red-600 text-white font-medium disabled:opacity-50"
				>
					{deleteLoading ? 'Deleting…' : 'Delete'}
				</button>
			</div>
		</div>
	</div>
{/if}
