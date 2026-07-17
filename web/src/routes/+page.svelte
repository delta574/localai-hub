<script lang="ts">
	import { getSystemInfo, listConversations, createConversation, deleteConversation, type SystemInfo, type ConversationSummary } from '$lib/api';
	import SetupWizard from '$lib/components/SetupWizard.svelte';
	import Chat from '$lib/components/Chat.svelte';
	import ModelsPage from '../routes/models/+page.svelte';
	import SettingsPage from './settings/+page.svelte';

	let sysInfo = $state<SystemInfo | null>(null);
	let showWizard = $state(false);
	let page: 'chat' | 'models' | 'settings' = $state('chat');
	let conversations = $state<ConversationSummary[]>([]);
	let activeConv = $state<string | null>(null);
	let loading = $state(false);

	async function loadConvs() {
		conversations = await listConversations();
	}

	function onTitleChange(id: string, title: string) {
		const c = conversations.find(c => c.id === id);
		if (c) c.title = title;
	}

	$effect(() => {
		getSystemInfo().then((info) => {
			sysInfo = info;
			showWizard = info.isFirstLaunch || (info.installedModels.length === 0 && !info.llamaServerRunning);
		});
		loadConvs();
	});

	$effect(() => {
		const interval = setInterval(() => {
			getSystemInfo()
				.then((info) => { sysInfo = info; })
				.catch(() => {});
		}, 5000);
		return () => clearInterval(interval);
	});

	async function newChat() {
		activeConv = null;
		page = 'chat';
	}

	function selectConv(id: string) {
		activeConv = id;
		page = 'chat';
	}

	async function removeConv(id: string, e: Event) {
		e.stopPropagation();
		await deleteConversation(id);
		if (activeConv === id) activeConv = null;
		loadConvs();
	}
</script>

{#if showWizard}
	<SetupWizard onDone={() => { showWizard = false; getSystemInfo().then(i => sysInfo = i); }} />
{/if}

<nav class="sidebar">
	<div class="logo">LocalAI Hub</div>
	<div class="nav-items">
		<button class="nav-item" class:active={page === 'chat'} onclick={() => page = 'chat'}>
			Chat
		</button>
		<button class="nav-item" class:active={page === 'models'} onclick={() => page = 'models'}>Models</button>
		<button class="nav-item" class:active={page === 'settings'} onclick={() => page = 'settings'}>Settings</button>
	</div>
	{#if page === 'chat'}
		<div class="conv-header">
			<span>Conversations</span>
			<button class="btn-new" onclick={newChat} title="New chat">+</button>
		</div>
		<div class="conv-list">
			{#each conversations as c}
				<div class="conv-item" class:active={activeConv === c.id} onclick={() => selectConv(c.id)} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && selectConv(c.id)}>
					<span class="conv-title">{c.title}</span>
					<button class="btn-del" onclick={(e) => removeConv(c.id, e)} title="Delete">&times;</button>
				</div>
			{/each}
			{#if conversations.length === 0}
				<div class="conv-empty">No conversations yet</div>
			{/if}
		</div>
	{/if}
	<div class="status">
		<div class="dot" class:green={sysInfo?.llamaServerRunning} class:red={!sysInfo?.llamaServerRunning}></div>
		<span>{sysInfo?.llamaServerRunning ? 'Running' : 'Idle'}</span>
	</div>
</nav>

<main class="main">
	{#if page === 'chat'}
		<Chat conversationId={activeConv} onTitleChange={onTitleChange} />
	{:else if page === 'models'}
		<ModelsPage />
	{:else if page === 'settings'}
		<SettingsPage />
	{/if}
</main>

<style>
	:global(body) {
		display: flex;
		height: 100dvh;
		margin: 0;
	}

	.sidebar {
		width: 200px; background: var(--surface); border-right: 1px solid var(--border);
		padding: 1rem; display: flex; flex-direction: column; height: 100dvh;
		flex-shrink: 0;
	}
	.logo { font-weight: 700; font-size: 1.1rem; margin-bottom: 1.5rem; }
	.nav-items { display: flex; flex-direction: column; gap: 0.25rem; flex: 1; }
	.nav-item {
		padding: 0.5rem 0.75rem; border-radius: 6px; border: none;
		background: none; color: var(--text2); text-align: left; font-size: 0.9rem;
	}
	.nav-item.active { background: var(--surface2); color: var(--text); }
	.nav-item:hover { background: var(--hover); color: var(--text); }
	.status {
		display: flex; align-items: center; gap: 0.5rem;
		font-size: 0.8rem; color: var(--text2);
	}
	.dot { width: 8px; height: 8px; border-radius: 50%; }
	.dot.green { background: #22c55e; }
	.dot.red { background: #ef4444; }
	.main { flex: 1; overflow-y: auto; }
	.conv-header {
		display: flex; align-items: center; justify-content: space-between;
		font-size: 0.75rem; color: var(--text2); text-transform: uppercase;
		margin-top: 1rem; margin-bottom: 0.5rem; letter-spacing: 0.05em;
	}
	.btn-new {
		width: 20px; height: 20px; border-radius: 4px; border: none;
		background: var(--surface2); color: var(--text); font-size: 1rem;
		line-height: 1; cursor: pointer; display: flex; align-items: center; justify-content: center;
	}
	.conv-list { display: flex; flex-direction: column; gap: 2px; margin-bottom: 1rem; overflow-y: auto; flex: 1; }
	.conv-item {
		padding: 0.4rem 0.5rem; border-radius: 4px; cursor: pointer;
		display: flex; align-items: center; gap: 0.25rem; font-size: 0.8rem;
		color: var(--text2); transition: background 0.15s;
	}
	.conv-item:hover { background: var(--hover); }
	.conv-item.active { background: var(--surface2); color: var(--text); }
	.conv-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.btn-del {
		background: none; border: none; color: var(--text2); font-size: 1rem;
		line-height: 1; padding: 0 2px; opacity: 0; cursor: pointer;
	}
	.conv-item:hover .btn-del { opacity: 0.6; }
	.conv-item .btn-del:hover { opacity: 1; color: #ef4444; }
	.conv-empty { font-size: 0.75rem; color: var(--text2); padding: 0.5rem; text-align: center; }
</style>
