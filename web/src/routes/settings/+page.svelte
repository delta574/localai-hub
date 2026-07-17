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
		getSystemInfo().then((i) => { info = i; prompt = i.systemPrompt; temp = i.temperature; });
		listModels().then((m) => models = m);
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

<div style="padding: 1.5rem; max-width: 600px; margin: 0 auto;">
	<h2 style="margin-bottom: 1.5rem;">Settings</h2>

	{#if info}
		<div style="background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1rem; margin-bottom: 1.5rem;">
			<h3 style="font-size: 0.9rem; color: var(--text2); margin-bottom: 0.75rem;">System Info</h3>
			<div style="font-size: 0.85rem; display: flex; flex-direction: column; gap: 0.35rem;">
				<div><span style="color: var(--text2); display: inline-block; width: 100px;">RAM</span>{info.ram.total} GB total / {info.ram.free} GB free</div>
				<div><span style="color: var(--text2); display: inline-block; width: 100px;">Free Disk</span>{info.diskFreeGB} GB</div>
				<div><span style="color: var(--text2); display: inline-block; width: 100px;">CPU</span>{info.cpu} cores</div>
				<div><span style="color: var(--text2); display: inline-block; width: 100px;">Platform</span>{info.os} {info.arch}</div>
				<div><span style="color: var(--text2); display: inline-block; width: 100px;">Engine</span>{info.llamaServerRunning ? 'Running' : 'Idle'}</div>
			</div>
		</div>

		<div style="background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1rem; margin-bottom: 1.5rem;">
			<h3 style="font-size: 0.9rem; color: var(--text2); margin-bottom: 0.75rem;">Active Model</h3>
			<select onchange={(e) => selectModel((e.target as HTMLSelectElement).value)} style="width: 100%; padding: 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface); color: var(--text); font-size: 0.9rem;">
				<option value="">-- Select a model --</option>
				{#each models.filter(m => m.installed) as m}
					<option value={m.id} selected={info.activeModel === m.id}>{m.name}</option>
				{/each}
			</select>
			{#if models.filter(m => m.installed).length === 0}
				<p style="font-size: 0.8rem; color: var(--text2); margin-top: 0.5rem;">Download a model from the Models page first.</p>
			{/if}
		</div>
	{/if}

	<div style="display: flex; flex-direction: column; gap: 1rem;">
		<div>
			<label style="display: block; font-size: 0.85rem; color: var(--text2); margin-bottom: 0.35rem;">System Prompt</label>
			<textarea bind:value={prompt} rows="4" style="width: 100%; padding: 0.6rem; border-radius: 8px; border: 1px solid var(--border); background: var(--surface); color: var(--text); resize: vertical; outline: none;"></textarea>
		</div>
		<div>
			<label style="display: block; font-size: 0.85rem; color: var(--text2); margin-bottom: 0.35rem;">Temperature: {temp.toFixed(1)}</label>
			<input type="range" bind:value={temp} min="0" max="2" step="0.1" style="width: 100%;" />
		</div>
		<button onclick={save} style="padding: 0.5rem 1.5rem; border-radius: 6px; border: none; background: var(--accent); color: #fff; font-weight: 600; align-self: flex-start; font-size: 0.9rem;">
			{saved ? 'Saved!' : 'Save Settings'}
		</button>
	</div>

	<div style="margin-top: 2.5rem;">
		<h3 style="font-size: 1rem; margin-bottom: 1rem;">&#128273; API Keys</h3>
		<p style="font-size: 0.8rem; color: var(--text2); margin-bottom: 1rem;">
			API keys allow external tools (opencode, VS Code, Open WebUI, curl) to connect to the OpenAI-compatible endpoint at <code>http://localhost:8080/v1</code>.
			When at least one key exists, all <code>/v1/*</code> requests require <code>Authorization: Bearer &lt;key&gt;</code>.
		</p>

		{#if keys.length === 0}
			<div style="background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1rem; margin-bottom: 1rem; text-align: center; color: var(--text2); font-size: 0.85rem;">
				No API keys yet. Generate one to enable authenticated access.
			</div>
		{:else}
			<div style="display: flex; flex-direction: column; gap: 0.5rem; margin-bottom: 1rem;">
				{#each keys as k}
					<div style="background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.75rem; display: flex; align-items: center; gap: 0.75rem;">
						<div style="flex: 1; min-width: 0;">
							<div style="display: flex; align-items: center; gap: 0.5rem;">
								<span style="font-size: 0.9rem; font-weight: 500;">{k.name}</span>
								<span style="font-size: 0.75rem; color: var(--text2); font-family: monospace;">{k.prefix}</span>
								<span style="font-size: 0.75rem; padding: 1px 6px; border-radius: 4px; {k.enabled ? 'background: #14532d; color: #86efac;' : 'background: #450a0a; color: #fca5a5;'}">{k.enabled ? 'Enabled' : 'Disabled'}</span>
							</div>
							<div style="font-size: 0.75rem; color: var(--text2); margin-top: 0.2rem;">
								Created {new Date(k.createdAt).toLocaleDateString()} &middot; Last used {fmtDate(k.lastUsedAt)}
							</div>
						</div>
						<button onclick={() => toggleKey(k.id)} style="padding: 0.3rem 0.6rem; border-radius: 4px; border: 1px solid var(--border); background: none; color: var(--text2); font-size: 0.8rem; cursor: pointer;">{k.enabled ? 'Disable' : 'Enable'}</button>
						<button onclick={() => removeKey(k.id)} style="padding: 0.3rem 0.6rem; border-radius: 4px; border: 1px solid transparent; background: none; color: #ef4444; font-size: 0.8rem; cursor: pointer;">Delete</button>
					</div>
				{/each}
			</div>
		{/if}

		{#if showNewKey}
			<div style="background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1rem; margin-bottom: 1rem;">
				{#if newKeyRaw}
					<h4 style="font-size: 0.85rem; margin-bottom: 0.5rem;">New API Key</h4>
					<p style="font-size: 0.8rem; color: #fbbf24; margin-bottom: 0.5rem;">Copy this key now. You won't be able to see it again.</p>
					<div style="display: flex; gap: 0.5rem; align-items: center;">
						<code style="flex: 1; padding: 0.5rem; background: var(--bg); border-radius: 6px; font-size: 0.8rem; word-break: break-all;">{newKeyRaw}</code>
						<button onclick={copyKey} style="padding: 0.4rem 1rem; border-radius: 6px; border: none; background: var(--accent); color: #fff; font-weight: 600; font-size: 0.85rem; white-space: nowrap;">{newKeyCopied ? 'Copied!' : 'Copy'}</button>
					</div>
					<button onclick={closeNewKey} style="margin-top: 0.75rem; padding: 0.4rem 1rem; border-radius: 6px; border: 1px solid var(--border); background: none; color: var(--text2); font-size: 0.85rem;">Done</button>
				{:else}
					<h4 style="font-size: 0.85rem; margin-bottom: 0.5rem;">Generate New Key</h4>
					<div style="display: flex; gap: 0.5rem; align-items: center;">
						<input bind:value={newKeyName} placeholder="e.g. opencode, VS Code, Open WebUI" style="flex: 1; padding: 0.5rem; border-radius: 6px; border: 1px solid var(--border); background: var(--bg); color: var(--text); font-size: 0.85rem; outline: none;" />
						<button onclick={genKey} disabled={!newKeyName.trim()} style="padding: 0.4rem 1rem; border-radius: 6px; border: none; background: var(--accent); color: #fff; font-weight: 600; font-size: 0.85rem; {!newKeyName.trim() ? 'opacity: 0.4;' : ''}">Generate</button>
					</div>
					<button onclick={() => showNewKey = false} style="margin-top: 0.5rem; background: none; border: none; color: var(--text2); font-size: 0.8rem; cursor: pointer;">Cancel</button>
				{/if}
			</div>
		{:else}
			<button onclick={() => { showNewKey = true; newKeyRaw = ''; }} style="padding: 0.5rem 1.25rem; border-radius: 6px; border: none; background: var(--accent); color: #fff; font-weight: 600; font-size: 0.85rem;">+ Generate New Key</button>
		{/if}
	</div>
</div>
