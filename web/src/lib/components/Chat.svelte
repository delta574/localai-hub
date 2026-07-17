<script lang="ts">
	import SvelteMarkdown from '@humanspeak/svelte-markdown';
	import { chatCompletion, createConversation, updateConversation, getConversation } from '$lib/api';

	interface Message {
		role: 'user' | 'assistant';
		content: string;
	}

	let { conversationId, onTitleChange, systemPrompt = '', temperature = 0.7, activeModel = '' } = $props<{ conversationId: string | null; onTitleChange?: (id: string, title: string) => void; systemPrompt?: string; temperature?: number; activeModel?: string }>();

	let messages = $state<Message[]>([]);
	let input = $state('');
	let streaming = $state(false);
	let abortCtrl = $state<AbortController | null>(null);
	let currentId = $state<string | null>(null);
	let pendingSave = $state(false);

	$effect(() => {
		if (conversationId && conversationId !== currentId) {
			const expectedId = conversationId;
			currentId = conversationId;
			getConversation(conversationId).then((c) => {
				if (currentId !== expectedId) return;
				messages = c.messages.map((m: { role: string; content: string }) => ({ role: m.role as 'user' | 'assistant', content: m.content }));
			}).catch(() => {
				if (currentId === expectedId) messages = [];
			});
		}
	});

	function save() {
		if (!currentId) return;
		if (pendingSave) { pendingSave = false; return; }
		pendingSave = true;
		const id = currentId;
		const title = messages.length > 0 ? messages[0].content.slice(0, 60) : 'New Chat';
		updateConversation(id, messages, title).then(() => {
			onTitleChange?.(id, title);
		}).finally(() => { if (currentId === id) pendingSave = false; });
	}

	async function send() {
		const text = input.trim();
		if (!text || streaming) return;
		input = '';

		if (!currentId) {
			currentId = await createConversation();
		}

		messages.push({ role: 'user', content: text });
		const idx = messages.length;
		messages.push({ role: 'assistant', content: '' });
		streaming = true;

		abortCtrl = new AbortController();
		try {
			await chatCompletion(activeModel, messages.slice(0, idx), (token) => {
				messages[idx].content += token;
				messages = messages;
			}, { signal: abortCtrl.signal, systemPrompt, temperature });
	} catch (e) {
		const errMsg = 'Error: ' + (e instanceof Error ? e.message : 'connection failed');
		messages[idx].content = messages[idx].content ? messages[idx].content + '\n\n' + errMsg : errMsg;
		messages = messages;
	}
	streaming = false;
		abortCtrl = null;
		save();
	}

	function stop() {
		abortCtrl?.abort();
		streaming = false;
	}

	function clear() {
		messages = [];
		currentId = null;
	}

	function handleKey(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}

	let chatEnd: HTMLDivElement;
	$effect(() => {
		messages;
		chatEnd?.scrollIntoView({ behavior: 'smooth' });
	});
</script>

<div class="chat-layout">
	<div class="header">
		<h2>Chat</h2>
		<button class="btn-text" onclick={clear}>Clear</button>
	</div>

	<div class="messages">
		{#each messages as msg, i}
			<div class="msg {msg.role}">
				<div class="msg-label">{msg.role === 'user' ? 'You' : 'AI'}</div>
				<div class="msg-content"><SvelteMarkdown source={msg.content} /></div>
			</div>
		{/each}

		{#if messages.length === 0}
			<div class="empty">
				<h3>LocalAI Hub</h3>
				<p>Your private, local AI assistant. Type a message to start.</p>
			</div>
		{/if}

		<div bind:this={chatEnd}></div>
	</div>

	<div class="input-area">
		<textarea
			bind:value={input}
			onkeydown={handleKey}
			placeholder="Type a message..."
			rows="1"
			disabled={streaming}
			name="chat-input"
		></textarea>
		{#if streaming}
			<button class="btn-stop" onclick={stop}>Stop</button>
		{:else}
			<button class="btn-send" onclick={send} disabled={!input.trim()}>Send</button>
		{/if}
	</div>
</div>

<style>
	.chat-layout {
		display: flex; flex-direction: column; height: 100dvh;
	}
	.header {
		padding: 0.75rem 1rem;
		display: flex; align-items: center; justify-content: space-between;
		border-bottom: 1px solid var(--border);
		background: var(--surface);
	}
	.header h2 { font-size: 1rem; font-weight: 600; }
	.btn-text { background: none; border: none; color: var(--text2); font-size: 0.85rem; }
	.btn-text:hover { color: var(--text); }

	.messages {
		flex: 1; overflow-y: auto; padding: 1rem;
		display: flex; flex-direction: column; gap: 1rem;
	}
	.msg { max-width: 80%; }
	.msg.user { align-self: flex-end; }
	.msg.assistant { align-self: flex-start; }
	.msg-label { font-size: 0.75rem; color: var(--text2); margin-bottom: 0.25rem; }
	.msg-content {
		background: var(--surface); padding: 0.75rem 1rem;
		border-radius: 12px; line-height: 1.5; font-size: 0.9rem;
		white-space: pre-wrap; word-break: break-word;
	}
	.msg.user .msg-content { background: var(--accent); color: #fff; }

	.empty {
		margin: auto; text-align: center; color: var(--text2);
	}
	.empty h3 { font-size: 1.5rem; margin-bottom: 0.5rem; color: var(--text); }

	.input-area {
		padding: 0.75rem 1rem; border-top: 1px solid var(--border);
		background: var(--surface);
		display: flex; gap: 0.5rem; align-items: flex-end;
	}
	textarea {
		flex: 1; padding: 0.6rem 0.75rem; border-radius: 8px;
		border: 1px solid var(--border); background: var(--bg);
		color: var(--text); resize: none; min-height: 40px;
		max-height: 120px; outline: none;
	}
	textarea:focus { border-color: var(--accent); }
	.btn-send, .btn-stop {
		padding: 0.5rem 1rem; border-radius: 6px; border: none;
		font-weight: 600; font-size: 0.85rem;
	}
	.btn-send { background: var(--accent); color: #fff; }
	.btn-send:disabled { opacity: 0.4; cursor: not-allowed; }
	.btn-stop { background: #666; color: #fff; }
</style>
