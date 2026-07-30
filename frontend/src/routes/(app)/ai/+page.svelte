<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { marked } from 'marked';
	import { aiApi } from '$api/ai';
	import { toast } from '$stores/toast';
	import { Trash2, Plus, Send, Square } from 'lucide-svelte';
	import type { ChatMessage, ConversationSummary } from '$types';

	let conversations = $state<ConversationSummary[]>([]);
	let activeConvId = $state<string | null>(null);
	let messages = $state<ChatMessage[]>([]);
	let streamingText = $state('');
	let streaming = $state(false);
	let input = $state('');
	let loading = $state(true);
	let loadingConv = $state(false);
	let messagesEl = $state<HTMLDivElement | null>(null);

	let abortCtrl: AbortController | null = null;

	onMount(async () => {
		try {
			conversations = await aiApi.listConversations();
		} catch {
			toast.error('Error', 'Could not load conversations.');
		} finally {
			loading = false;
		}
	});

	async function loadConversation(id: string) {
		if (streaming) return;
		loadingConv = true;
		activeConvId = id;
		messages = [];
		try {
			const conv = await aiApi.getConversation(id);
			messages = conv.messages ?? [];
		} catch {
			toast.error('Error', 'Could not load conversation.');
		} finally {
			loadingConv = false;
			await tick();
			scrollToBottom();
		}
	}

	function newChat() {
		if (streaming) return;
		activeConvId = null;
		messages = [];
		streamingText = '';
		input = '';
	}

	async function deleteConversation(id: string, e: MouseEvent) {
		e.stopPropagation();
		if (streaming && activeConvId === id) return;
		try {
			await aiApi.deleteConversation(id);
			conversations = conversations.filter((c) => c.id !== id);
			if (activeConvId === id) {
				activeConvId = null;
				messages = [];
			}
		} catch {
			toast.error('Error', 'Could not delete conversation.');
		}
	}

	async function send() {
		const msg = input.trim();
		if (!msg || streaming) return;

		messages = [...messages, { role: 'user', content: msg }];
		input = '';
		streaming = true;
		streamingText = '';
		await tick();
		scrollToBottom();

		abortCtrl = new AbortController();
		try {
			const resp = await aiApi.streamChat(msg, activeConvId, abortCtrl.signal);
			if (!resp.ok || !resp.body) {
				throw new Error('Stream failed');
			}

			const reader = resp.body.getReader();
			const decoder = new TextDecoder();
			let buffer = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const parts = buffer.split('\n\n');
				buffer = parts.pop() ?? '';

				for (const part of parts) {
					if (!part.startsWith('data: ')) continue;
					try {
						const data = JSON.parse(part.slice(6));
						await handleSSE(data);
					} catch {
						// ignore malformed events
					}
				}
			}
		} catch (err: unknown) {
			if (err instanceof Error && err.name === 'AbortError') {
				// user stopped
				if (streamingText) {
					messages = [...messages, { role: 'assistant', content: streamingText }];
				}
			} else {
				toast.error('Error', 'Could not send message.');
				messages = messages.slice(0, -1); // remove optimistic user msg
			}
		} finally {
			streaming = false;
			streamingText = '';
			abortCtrl = null;
		}
	}

	async function handleSSE(data: Record<string, unknown>) {
		switch (data.type) {
			case 'start':
				activeConvId = data.conversation_id as string;
				break;
			case 'delta':
				streamingText += data.text as string;
				await tick();
				scrollToBottom();
				break;
			case 'done':
				messages = [...messages, { role: 'assistant', content: streamingText }];
				streamingText = '';
				streaming = false;
				await tick();
				scrollToBottom();
				// refresh sidebar
				conversations = await aiApi.listConversations().catch(() => conversations);
				break;
			case 'error':
				toast.error('AI Error', (data.message as string) ?? 'Unknown error');
				streaming = false;
				break;
		}
	}

	function stopStreaming() {
		abortCtrl?.abort();
	}

	function scrollToBottom() {
		if (messagesEl) {
			messagesEl.scrollTop = messagesEl.scrollHeight;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}

	function renderMarkdown(text: string): string {
		return marked.parse(text) as string;
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}
</script>

<svelte:head>
	<title>AI Assistant — StarterKit</title>
</svelte:head>

<div class="flex h-full overflow-hidden">
	<!-- Sidebar -->
	<div class="w-56 shrink-0 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)]">
		<div class="p-3 border-b border-[var(--color-border)] flex flex-col gap-2">
			<button
				onclick={newChat}
				disabled={streaming}
				class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-[var(--radius)] bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
			>
				<Plus class="h-4 w-4" />
				New chat
			</button>
		</div>

		<div class="flex-1 overflow-y-auto p-2 flex flex-col gap-1">
			{#if loading}
				{#each Array.from({ length: 5 }) as _, i (i)}
					<div class="rounded-[var(--radius)] px-3 py-2 flex items-start justify-between gap-1">
						<div class="min-w-0 flex-1 space-y-1.5">
							<div class="animate-pulse h-3 rounded bg-[var(--color-muted)]" style="width: {i % 2 === 0 ? '80%' : '60%'}"></div>
							<div class="animate-pulse h-2.5 w-10 rounded bg-[var(--color-muted)]"></div>
						</div>
					</div>
				{/each}
			{:else if conversations.length === 0}
				<div class="text-center py-8 text-xs text-[var(--color-muted-fg)]">No conversations yet</div>
			{:else}
				{#each conversations as conv (conv.id)}
					<div
						role="button"
						tabindex="0"
						onclick={() => loadConversation(conv.id)}
						onkeydown={(e) => e.key === 'Enter' && loadConversation(conv.id)}
						class="group w-full text-left rounded-[var(--radius)] px-3 py-2 transition-colors flex items-start justify-between gap-1 cursor-pointer
							{activeConvId === conv.id
							? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
							: 'hover:bg-[var(--color-muted)] text-[var(--color-foreground)]'}"
					>
						<div class="min-w-0 flex-1">
							<p class="text-sm font-medium truncate">{conv.title}</p>
							<p class="text-xs opacity-60">{formatDate(conv.updated_at)}</p>
						</div>
						<button
							onclick={(e) => deleteConversation(conv.id, e)}
							class="opacity-0 group-hover:opacity-100 shrink-0 text-[var(--color-muted-fg)] hover:text-[var(--color-destructive)] transition-all mt-0.5"
							title="Delete"
						>
							<Trash2 class="h-3.5 w-3.5" />
						</button>
					</div>
				{/each}
			{/if}
		</div>
	</div>

	<!-- Chat area -->
	<div class="flex-1 flex flex-col min-w-0 bg-[var(--color-background)]">
		<!-- Messages -->
		<div bind:this={messagesEl} class="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
			{#if loadingConv}
				<div class="flex flex-col gap-4">
					<div class="flex justify-end">
						<div class="animate-pulse h-10 w-48 rounded-[var(--radius-lg)] bg-[var(--color-muted)]"></div>
					</div>
					<div class="flex justify-start">
						<div class="animate-pulse h-16 w-64 rounded-[var(--radius-lg)] bg-[var(--color-muted)]"></div>
					</div>
					<div class="flex justify-end">
						<div class="animate-pulse h-8 w-36 rounded-[var(--radius-lg)] bg-[var(--color-muted)]"></div>
					</div>
				</div>
			{:else if messages.length === 0 && !streaming}
				<div class="flex-1 flex flex-col items-center justify-center text-center">
					<p class="text-2xl font-semibold text-[var(--color-foreground)] mb-2">Ask me anything</p>
					<p class="text-sm text-[var(--color-muted-fg)]">I can help with your account, notifications, and more.</p>
				</div>
			{:else}
				{#each messages as msg, i (i)}
					{#if msg.role === 'user'}
						<div class="flex justify-end">
							<div class="max-w-[75%] rounded-[var(--radius-lg)] bg-[var(--color-primary)] text-white px-4 py-2.5 text-sm whitespace-pre-wrap">
								{msg.content}
							</div>
						</div>
					{:else}
						<div class="flex justify-start">
							<div class="max-w-[80%] rounded-[var(--radius-lg)] bg-[var(--color-card)] border border-[var(--color-border)] px-4 py-2.5 text-sm prose prose-sm dark:prose-invert max-w-none">
								<!-- eslint-disable-next-line svelte/no-at-html-tags -->
								{@html renderMarkdown(msg.content)}
							</div>
						</div>
					{/if}
				{/each}

				<!-- Live streaming bubble -->
				{#if streaming && streamingText}
					<div class="flex justify-start">
						<div class="max-w-[80%] rounded-[var(--radius-lg)] bg-[var(--color-card)] border border-[var(--color-border)] px-4 py-2.5 text-sm prose prose-sm dark:prose-invert max-w-none">
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html renderMarkdown(streamingText)}<span class="inline-block w-1.5 h-4 ml-0.5 bg-[var(--color-primary)] align-middle animate-pulse rounded-sm"></span>
						</div>
					</div>
				{:else if streaming && !streamingText}
					<div class="flex justify-start">
						<div class="rounded-[var(--radius-lg)] bg-[var(--color-card)] border border-[var(--color-border)] px-4 py-3">
							<div class="flex gap-1">
								<span class="h-2 w-2 rounded-full bg-[var(--color-muted-fg)] animate-bounce" style="animation-delay:0ms"></span>
								<span class="h-2 w-2 rounded-full bg-[var(--color-muted-fg)] animate-bounce" style="animation-delay:150ms"></span>
								<span class="h-2 w-2 rounded-full bg-[var(--color-muted-fg)] animate-bounce" style="animation-delay:300ms"></span>
							</div>
						</div>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Input area -->
		<div class="border-t border-[var(--color-border)] bg-[var(--color-card)] p-4">
			<div class="flex gap-3 items-end max-w-3xl mx-auto">
				<textarea
					bind:value={input}
					onkeydown={handleKeydown}
					disabled={streaming}
					placeholder="Message…"
					rows={1}
					class="flex-1 resize-none rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:opacity-50 min-h-[42px] max-h-40 overflow-y-auto"
					oninput={(e) => {
						const el = e.currentTarget as HTMLTextAreaElement;
						el.style.height = 'auto';
						el.style.height = Math.min(el.scrollHeight, 160) + 'px';
					}}
				></textarea>
				{#if streaming}
					<button
						onclick={stopStreaming}
						class="flex items-center gap-1.5 px-4 py-2.5 rounded-[var(--radius-lg)] bg-[var(--color-muted)] text-[var(--color-foreground)] text-sm font-medium hover:bg-[var(--color-border)] transition-colors shrink-0"
					>
						<Square class="h-4 w-4" />
						Stop
					</button>
				{:else}
					<button
						onclick={send}
						disabled={!input.trim()}
						class="flex items-center gap-1.5 px-4 py-2.5 rounded-[var(--radius-lg)] bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50 shrink-0"
					>
						<Send class="h-4 w-4" />
						Send
					</button>
				{/if}
			</div>
		</div>
	</div>
</div>
