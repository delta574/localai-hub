<script lang="ts">
	import { onDestroy } from 'svelte';
	import { getSystemInfo, listModels, updateConfig, type SystemInfo, type ModelInfo } from '$lib/api';

	let info = $state<SystemInfo | null>(null);
	let models = $state<ModelInfo[]>([]);
	let prompt = $state('');
	let temp = $state(0.7);
	let saved = $state(false);
	let savedTimer: ReturnType<typeof setTimeout> | undefined;

	onDestroy(() => clearTimeout(savedTimer));

	$effect(() => {
		getSystemInfo().then((i) => { info = i; prompt = i.systemPrompt; temp = i.temperature; });
		listModels().then((m) => models = m);
	});

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
</div>
