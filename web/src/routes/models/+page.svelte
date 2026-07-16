<script lang="ts">
	import { listModels, pullModel, deleteModel, updateConfig, type ModelInfo, type ProgressEvent } from '$lib/api';

	let models = $state<ModelInfo[]>([]);
	let downloading = $state<string | null>(null);
	let progress = $state<ProgressEvent | null>(null);
	let errorMsg = $state('');

	$effect(() => {
		listModels().then((m) => models = m);
	});

	function install(id: string) {
		errorMsg = '';
		downloading = id;
		pullModel(id, (evt) => {
			progress = evt;
			if (evt.type === 'done') {
				updateConfig({ activeModel: id });
				downloading = null;
				listModels().then((m) => models = m);
			}
			if (evt.type === 'error') {
				errorMsg = evt.message || 'Download failed';
				downloading = null;
			}
		});
	}

	async function remove(id: string) {
		await deleteModel(id);
		listModels().then((m) => models = m);
	}
</script>

<div style="padding: 1.5rem; max-width: 800px; margin: 0 auto;">
	<h2 style="margin-bottom: 1rem;">Models</h2>

	{#if errorMsg}
		<div style="background: #3b1a1a; border: 1px solid #ef4444; border-radius: 6px; padding: 0.6rem 0.8rem; margin-bottom: 1rem; font-size: 0.85rem; color: #fca5a5;">{errorMsg}</div>
	{/if}

	<div style="display: flex; flex-direction: column; gap: 0.75rem;">
		{#each models as m}
			<div style="background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1rem; display: flex; align-items: center; gap: 1rem;">
				<div style="min-width: 80px; text-align: center;">
					<div style="color: #fbbf24; font-size: 0.85rem;">{m.quality}</div>
					<div style="color: var(--text2); font-size: 0.8rem;">{m.sizeGB.toFixed(1)} GB</div>
				</div>
				<div style="flex: 1;">
					<h3 style="font-size: 1rem;">{m.name}</h3>
					<p style="color: var(--text2); font-size: 0.8rem;">{m.tagline}</p>
				</div>
				{#if downloading === m.id}
					<div style="width: 120px;">
						<div style="height: 6px; background: var(--border); border-radius: 3px; overflow: hidden;">
							<div style="height: 100%; background: var(--accent); width: {progress?.percent ?? 0}%; transition: width 0.3s;"></div>
						</div>
						<p style="font-size: 0.75rem; color: var(--text2); text-align: center; margin-top: 0.25rem;">{Math.round(progress?.percent ?? 0)}%</p>
					</div>
				{:else if m.installed}
					<button onclick={() => remove(m.id)} style="padding: 0.4rem 1rem; border-radius: 6px; border: 1px solid var(--border); background: none; color: var(--text2); font-size: 0.85rem;">Remove</button>
				{:else}
					<button onclick={() => install(m.id)} style="padding: 0.4rem 1.25rem; border-radius: 6px; border: none; background: var(--accent); color: #fff; font-weight: 600; font-size: 0.85rem;">Install</button>
				{/if}
			</div>
		{/each}
	</div>
</div>
