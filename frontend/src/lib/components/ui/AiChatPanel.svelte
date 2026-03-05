<script lang="ts">
	import { tick } from 'svelte';
	import { fly } from 'svelte/transition';
	import { marked } from 'marked';
	import { goto } from '$app/navigation';
	import { aiApi } from '$api/ai';
	import { aiChat } from '$stores/aiChat';
	import { toast } from '$stores/toast';
	import { X, Maximize2, Plus, Send, Square, Bot } from 'lucide-svelte';
	import type { ChatMessage } from '$types';

	let messages = $state<ChatMessage[]>([]);
	let streamingText = $state('');
	let streaming = $state(false);
	let input = $state('');
	let activeConvId = $state<string | null>(null);
	let provider = $state<'anthropic' | 'gemini'>('anthropic');
	let messagesEl = $state<HTMLDivElement | null>(null);
	let textareaEl = $state<HTMLTextAreaElement | null>(null);

	let abortCtrl: AbortController | null = null;

	$effect(() => {
		if ($aiChat) {
			tick().then(() => textareaEl?.focus());
		}
	});

	function newChat() {
		if (streaming) return;
		activeConvId = null;
		messages = [];
		streamingText = '';
		input = '';
	}

	function toggleProvider() {
		if (streaming) return;
		provider = provider === 'anthropic' ? 'gemini' : 'anthropic';
		newChat();
	}

	function close() {
		if (streaming) abortCtrl?.abort();
		aiChat.close();
	}

	function openFullPage() {
		aiChat.close();
		goto('/ai');
	}

	async function send() {
		const msg = input.trim();
		if (!msg || streaming) return;

		messages = [...messages, { role: 'user', content: msg }];
		input = '';
		if (textareaEl) { textareaEl.style.height = 'auto'; }
		streaming = true;
		streamingText = '';
		await tick();
		scrollToBottom();

		abortCtrl = new AbortController();
		try {
			const resp = await aiApi.streamChat(msg, activeConvId, abortCtrl.signal, provider);
			if (!resp.ok || !resp.body) throw new Error('Stream failed');

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
					try { await handleSSE(JSON.parse(part.slice(6))); } catch { /* ignore */ }
				}
			}
		} catch (err: unknown) {
			if (err instanceof Error && err.name === 'AbortError') {
				if (streamingText) messages = [...messages, { role: 'assistant', content: streamingText }];
			} else {
				toast.error('Error', 'Could not send message.');
				messages = messages.slice(0, -1);
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
				break;
			case 'error':
				toast.error('AI Error', (data.message as string) ?? 'Unknown error');
				streaming = false;
				break;
		}
	}

	function stopStreaming() { abortCtrl?.abort(); }

	function scrollToBottom() {
		if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') { close(); return; }
		if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
	}

	function renderMarkdown(text: string): string {
		return marked.parse(text) as string;
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if $aiChat}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px]"
		onclick={close}
		aria-hidden="true"
	></div>

	<!-- Panel -->
	<div
		class="fixed right-0 top-0 bottom-0 z-50 flex w-full flex-col sm:w-[420px] bg-[var(--color-card)] border-l border-[var(--color-border)] shadow-[var(--shadow-xl)]"
		transition:fly={{ x: 420, duration: 250 }}
	>
		<!-- Header -->
		<div class="flex h-14 shrink-0 items-center gap-2 border-b border-[var(--color-border)] px-4">
			<Bot class="h-4 w-4 text-[var(--color-primary)] shrink-0" />
			<span class="text-sm font-semibold text-[var(--color-foreground)]">AI Assistant</span>
			<!-- Provider toggle -->
			<button
				onclick={toggleProvider}
				disabled={streaming}
				title="Switch provider"
				class="ml-1 flex items-center gap-1 rounded-full border border-[var(--color-border)] bg-[var(--color-muted)] px-2 py-0.5 text-[10px] font-medium text-[var(--color-muted-fg)] hover:text-[var(--color-foreground)] transition-colors disabled:opacity-40"
			>
				{#if provider === 'anthropic'}
					<span class="h-1.5 w-1.5 rounded-full bg-orange-400 shrink-0"></span>
					Anthropic
				{:else}
					<span class="h-1.5 w-1.5 rounded-full bg-blue-400 shrink-0"></span>
					Gemini
				{/if}
			</button>
			<span class="flex-1"></span>
			<button
				onclick={newChat}
				disabled={streaming}
				title="New chat"
				class="rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors disabled:opacity-40"
			>
				<Plus class="h-4 w-4" />
			</button>
			<button
				onclick={openFullPage}
				title="Open full page"
				class="rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
			>
				<Maximize2 class="h-4 w-4" />
			</button>
			<button
				onclick={close}
				title="Close"
				class="rounded p-1.5 text-[var(--color-muted-fg)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)] transition-colors"
			>
				<X class="h-4 w-4" />
			</button>
		</div>

		<!-- Messages -->
		<div bind:this={messagesEl} class="flex-1 overflow-y-auto p-4 flex flex-col gap-3">
			{#if messages.length === 0 && !streaming}
				<div class="flex-1 flex flex-col items-center justify-center text-center py-16">
					<div class="h-12 w-12 rounded-full bg-[var(--color-primary)]/10 flex items-center justify-center mb-3">
						<Bot class="h-6 w-6 text-[var(--color-primary)]" />
					</div>
					<p class="text-sm font-medium text-[var(--color-foreground)]">Ask me anything</p>
					<p class="text-xs text-[var(--color-muted-fg)] mt-1">I can help with your account, data, and more.</p>
				</div>
			{:else}
				{#each messages as msg, i (i)}
					{#if msg.role === 'user'}
						<div class="flex justify-end">
							<div class="max-w-[80%] rounded-[var(--radius-lg)] bg-[var(--color-primary)] text-white px-3 py-2 text-sm whitespace-pre-wrap">
								{msg.content}
							</div>
						</div>
					{:else}
						<div class="flex justify-start">
							<div class="max-w-[85%] rounded-[var(--radius-lg)] bg-[var(--color-muted)] px-3 py-2 text-sm prose prose-sm dark:prose-invert max-w-none">
								<!-- eslint-disable-next-line svelte/no-at-html-tags -->
								{@html renderMarkdown(msg.content)}
							</div>
						</div>
					{/if}
				{/each}

				{#if streaming && streamingText}
					<div class="flex justify-start">
						<div class="max-w-[85%] rounded-[var(--radius-lg)] bg-[var(--color-muted)] px-3 py-2 text-sm prose prose-sm dark:prose-invert max-w-none">
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html renderMarkdown(streamingText)}<span class="inline-block w-1 h-4 ml-0.5 bg-[var(--color-primary)] align-middle animate-pulse rounded-sm"></span>
						</div>
					</div>
				{:else if streaming}
					<div class="flex justify-start">
						<div class="rounded-[var(--radius-lg)] bg-[var(--color-muted)] px-3 py-3">
							<div class="flex gap-1">
								<span class="h-1.5 w-1.5 rounded-full bg-[var(--color-muted-fg)] animate-bounce" style="animation-delay:0ms"></span>
								<span class="h-1.5 w-1.5 rounded-full bg-[var(--color-muted-fg)] animate-bounce" style="animation-delay:150ms"></span>
								<span class="h-1.5 w-1.5 rounded-full bg-[var(--color-muted-fg)] animate-bounce" style="animation-delay:300ms"></span>
							</div>
						</div>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Input -->
		<div class="shrink-0 border-t border-[var(--color-border)] p-3">
			<div class="flex gap-2 items-end">
				<textarea
					bind:this={textareaEl}
					bind:value={input}
					onkeydown={handleKeydown}
					disabled={streaming}
					placeholder="Message…"
					rows={1}
					class="flex-1 resize-none rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-background)] text-[var(--color-foreground)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:opacity-50 min-h-[38px] max-h-32 overflow-y-auto"
					oninput={(e) => {
						const el = e.currentTarget as HTMLTextAreaElement;
						el.style.height = 'auto';
						el.style.height = Math.min(el.scrollHeight, 128) + 'px';
					}}
				></textarea>
				{#if streaming}
					<button
						onclick={stopStreaming}
						class="shrink-0 flex items-center gap-1 px-3 py-2 rounded-[var(--radius-lg)] bg-[var(--color-muted)] text-[var(--color-foreground)] text-sm font-medium hover:bg-[var(--color-border)] transition-colors"
					>
						<Square class="h-3.5 w-3.5" />
					</button>
				{:else}
					<button
						onclick={send}
						disabled={!input.trim()}
						class="shrink-0 flex items-center gap-1 px-3 py-2 rounded-[var(--radius-lg)] bg-[var(--color-primary)] text-white text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-40"
					>
						<Send class="h-3.5 w-3.5" />
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}
