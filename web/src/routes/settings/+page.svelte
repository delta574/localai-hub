<script lang="ts">
	import { onDestroy } from 'svelte';
	import { getSystemInfo, listModels, updateConfig, listApiKeys, createApiKey, deleteApiKey, toggleApiKey, type SystemInfo, type ModelInfo, type ApiKeyInfo } from '$lib/api';

	let info = $state<SystemInfo | null>(null);
	let models = $state<ModelInfo[]>([]);
	let prompt = $state('');
	let temp = $state(0.7);
	let saved = $state(false);
	let savedTimer: ReturnType<typeof setTimeout> | undefined;

	let keys = $state<ApiKeyInfo[]>([]);
	let showNewKey = $state(false);
	let newKeyName = $state('');
	let newKeyRaw = $state('');
	let newKeyCopied = $state(false);

	onDestroy(() => clearTimeout(savedTimer));

	$effect(() => {
		getSystemInfo().then((i) => { info = i; prompt = i.systemPrompt; temp = i.temperature; }).catch(() => {});
		listModels().then((m) => models = m).catch(() => {});
		loadKeys();
	});

	async function loadKeys() {
		try { keys = await listApiKeys(); } catch {}
	}

	async function save() {
		await updateConfig({ systemPrompt: prompt, temperature: temp });
		saved = true;
		clearTimeout(savedTimer);
		savedTimer = setTimeout(() => saved = false, 2000);
	}

	async function selectModel(id: string) {
		await updateConfig({ activeModel: id });
		if (info) info.activeModel = id;
	}

	async function genKey() {
		if (!newKeyName.trim()) return;
		const result = await createApiKey(newKeyName.trim());
		newKeyRaw = result.key;
		newKeyCopied = false;
		newKeyName = '';
		loadKeys();
	}

	async function copyKey() {
		await navigator.clipboard.writeText(newKeyRaw);
		newKeyCopied = true;
	}

	function closeNewKey() {
		newKeyRaw = '';
		showNewKey = false;
	}

	async function removeKey(id: string) {
		await deleteApiKey(id);
		loadKeys();
	}

	async function toggleKey(id: string) {
		await toggleApiKey(id);
		loadKeys();
	}

	function fmtDate(s: string) {
		if (!s) return 'never';
		const d = new Date(s);
		const now = new Date();
		const diff = now.getTime() - d.getTime();
		if (diff < 60000) return 'just now';
		if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
		if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
		return d.toLocaleDateString();
	}
</script>

<div class="settings-page">
	<h2 class="page-title">Settings</h2>

	{#if info !== null}
		<div class="card">
			<h3 class="card-title">System Info</h3>
			<div class="info-grid">
				<div><span class="info-label">RAM</span>{info.ram.total} GB total / {info.ram.free} GB free</div>
				<div><span class="info-label">Free Disk</span>{info.diskFreeGB} GB</div>
				<div><span class="info-label">CPU</span>{info.cpu} cores</div>
				<div><span class="info-label">Platform</span>{info.os} {info.arch}</div>
				<div><span class="info-label">Engine</span>{info.llamaServerRunning ? 'Running' : 'Idle'}</div>
			</div>
		</div>

		<div class="card">
			<h3 class="card-title">Active Model</h3>
			<select class="select" onchange={(e) => selectModel((e.target as HTMLSelectElement).value)}>
				<option value="">-- Select a model --</option>
				{#each models.filter(m => m.installed) as m}
					<option value={m.id} selected={info.activeModel === m.id}>{m.name}</option>
				{/each}
			</select>
			{#if models.filter(m => m.installed).length === 0}
				<p class="hint">Download a model from the Models page first.</p>
			{/if}
		</div>
	{:else}
		<div class="card">
			<div class="skeleton-line" style="width: 40%; height: 1rem; margin-bottom: 0.75rem;"></div>
			<div class="skeleton-line" style="width: 70%; height: 0.85rem; margin-bottom: 0.35rem;"></div>
			<div class="skeleton-line" style="width: 55%; height: 0.85rem; margin-bottom: 0.35rem;"></div>
			<div class="skeleton-line" style="width: 60%; height: 0.85rem; margin-bottom: 0.35rem;"></div>
			<div class="skeleton-line" style="width: 45%; height: 0.85rem; margin-bottom: 0.35rem;"></div>
			<div class="skeleton-line" style="width: 50%; height: 0.85rem;"></div>
		</div>
		<div class="card">
			<div class="skeleton-line" style="width: 30%; height: 1rem; margin-bottom: 0.75rem;"></div>
			<div class="skeleton-line" style="width: 100%; height: 2.2rem; border-radius: 6px;"></div>
		</div>
	{/if}

	<div class="form-section">
		<div>
			<label class="label" for="system-prompt">System Prompt</label>
			<textarea class="textarea" bind:value={prompt} rows="4" id="system-prompt"></textarea>
		</div>
		<div>
			<label class="label" for="temperature">Temperature: {temp.toFixed(1)}</label>
			<input type="range" bind:value={temp} min="0" max="2" step="0.1" class="range" id="temperature" />
		</div>
		<button class="btn-primary" onclick={save}>
			{saved ? 'Saved!' : 'Save Settings'}
		</button>
	</div>

	<div class="api-keys-section">
		<h3 class="section-title">&#128273; API Keys</h3>
		<p class="section-desc">
			API keys allow external tools (opencode, VS Code, Open WebUI, curl) to connect to the OpenAI-compatible endpoint at <code>http://localhost:8080/v1</code>.
			When at least one key exists, all <code>/v1/*</code> requests require <code>Authorization: Bearer &lt;key&gt;</code>.
		</p>

		{#if keys.length === 0}
			<div class="empty-state">No API keys yet. Generate one to enable authenticated access.</div>
		{:else}
			<div class="keys-list">
				{#each keys as k}
					<div class="key-card">
						<div class="key-card-body">
							<div class="key-card-header">
								<span class="key-name">{k.name}</span>
								<span class="key-prefix">{k.prefix}</span>
								<span class="badge" style="background: {k.enabled ? '#14532d' : '#450a0a'}; color: {k.enabled ? '#86efac' : '#fca5a5'};">{k.enabled ? 'Enabled' : 'Disabled'}</span>
							</div>
							<div class="key-date">
								Created {new Date(k.createdAt).toLocaleDateString()} &middot; Last used {fmtDate(k.lastUsedAt)}
							</div>
						</div>
						<button class="btn-sm" onclick={() => toggleKey(k.id)}>{k.enabled ? 'Disable' : 'Enable'}</button>
						<button class="btn-sm btn-danger" onclick={() => removeKey(k.id)}>Delete</button>
					</div>
				{/each}
			</div>
		{/if}

		{#if showNewKey}
			<div class="card">
				{#if newKeyRaw}
					<h4 class="card-subtitle">New API Key</h4>
					<p class="warning-text">Copy this key now. You won't be able to see it again.</p>
					<div class="key-display-row">
						<code class="key-display">{newKeyRaw}</code>
						<button class="btn-primary" onclick={copyKey}>{newKeyCopied ? 'Copied!' : 'Copy'}</button>
					</div>
					<button class="btn-outline" onclick={closeNewKey}>Done</button>
				{:else}
					<h4 class="card-subtitle">Generate New Key</h4>
					<div class="key-input-row">
						<input class="input" bind:value={newKeyName} placeholder="e.g. opencode, VS Code, Open WebUI" autocomplete="off" name="key-name" />
						<button class="btn-primary" onclick={genKey} disabled={!newKeyName.trim()}>Generate</button>
					</div>
					<button class="btn-cancel" onclick={() => showNewKey = false}>Cancel</button>
				{/if}
			</div>
		{:else}
			<button class="btn-primary" onclick={() => { showNewKey = true; newKeyRaw = ''; }}>+ Generate New Key</button>
		{/if}
	</div>
</div>

<style>
	.settings-page {
		padding: 1.5rem;
		max-width: 600px;
		margin: 0 auto;
	}

	.page-title {
		margin-bottom: 1.5rem;
	}

	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 1rem;
		margin-bottom: 1.5rem;
	}

	.card-title {
		font-size: 0.9rem;
		color: var(--text2);
		margin-bottom: 0.75rem;
	}

	.card-subtitle {
		font-size: 0.85rem;
		margin-bottom: 0.5rem;
	}

	.info-grid {
		font-size: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.info-label {
		color: var(--text2);
		display: inline-block;
		width: 100px;
	}

	.select {
		width: 100%;
		padding: 0.5rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		font-size: 0.9rem;
	}

	.hint {
		font-size: 0.8rem;
		color: var(--text2);
		margin-top: 0.5rem;
	}

	.form-section {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.label {
		display: block;
		font-size: 0.85rem;
		color: var(--text2);
		margin-bottom: 0.35rem;
	}

	.textarea {
		width: 100%;
		padding: 0.6rem;
		border-radius: 8px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		resize: vertical;
		outline: none;
		font-family: inherit;
		font-size: inherit;
	}

	.range {
		width: 100%;
	}

	.btn-primary {
		padding: 0.5rem 1.5rem;
		border-radius: 6px;
		border: none;
		background: var(--accent);
		color: #fff;
		font-weight: 600;
		align-self: flex-start;
		font-size: 0.9rem;
		cursor: pointer;
	}

	.btn-primary:disabled {
		opacity: 0.4;
	}

	.btn-outline {
		margin-top: 0.75rem;
		padding: 0.4rem 1rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: none;
		color: var(--text2);
		font-size: 0.85rem;
		cursor: pointer;
	}

	.btn-sm {
		padding: 0.3rem 0.6rem;
		border-radius: 4px;
		border: 1px solid var(--border);
		background: none;
		color: var(--text2);
		font-size: 0.8rem;
		cursor: pointer;
		flex-shrink: 0;
	}

	.btn-danger {
		border-color: transparent;
		color: #ef4444;
	}

	.btn-cancel {
		margin-top: 0.5rem;
		background: none;
		border: none;
		color: var(--text2);
		font-size: 0.8rem;
		cursor: pointer;
	}

	.api-keys-section {
		margin-top: 2.5rem;
	}

	.section-title {
		font-size: 1rem;
		margin-bottom: 1rem;
	}

	.section-desc {
		font-size: 0.8rem;
		color: var(--text2);
		margin-bottom: 1rem;
	}

	.empty-state {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 1rem;
		margin-bottom: 1rem;
		text-align: center;
		color: var(--text2);
		font-size: 0.85rem;
	}

	.keys-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.key-card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 0.75rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.key-card-body {
		flex: 1;
		min-width: 0;
	}

	.key-card-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.key-name {
		font-size: 0.9rem;
		font-weight: 500;
	}

	.key-prefix {
		font-size: 0.75rem;
		color: var(--text2);
		font-family: monospace;
	}

	.badge {
		font-size: 0.75rem;
		padding: 1px 6px;
		border-radius: 4px;
	}

	.key-date {
		font-size: 0.75rem;
		color: var(--text2);
		margin-top: 0.2rem;
	}

	.key-display-row {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.key-display {
		flex: 1;
		padding: 0.5rem;
		background: var(--bg);
		border-radius: 6px;
		font-size: 0.8rem;
		word-break: break-all;
	}

	.key-input-row {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.input {
		flex: 1;
		padding: 0.5rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: var(--bg);
		color: var(--text);
		font-size: 0.85rem;
		outline: none;
	}

	.warning-text {
		font-size: 0.8rem;
		color: #fbbf24;
		margin-bottom: 0.5rem;
	}

	@keyframes pulse {
		0%, 100% { opacity: 0.4; }
		50% { opacity: 0.8; }
	}

	.skeleton-line {
		background: var(--border);
		border-radius: 4px;
		animation: pulse 1.5s ease-in-out infinite;
	}
</style>
