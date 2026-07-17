<script lang="ts">
	import { listModels, pullModel, deleteModel, updateConfig, type ModelInfo, type ProgressEvent } from '$lib/api';

	let models = $state<ModelInfo[]>([]);
	let loaded = $state(false);
	let downloading = $state<string | null>(null);
	let progress = $state<ProgressEvent | null>(null);
	let errorMsg = $state('');

	$effect(() => {
		listModels().then((m) => { models = m; loaded = true; });
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

<div class="page">
	<h2 class="title">Models</h2>

	{#if errorMsg}
		<div class="error-banner">{errorMsg}</div>
	{/if}

	{#if loaded}
		<div class="list">
			{#each models as m}
				<div class="model-card">
					<div class="model-meta">
						<div class="model-quality">{m.quality}</div>
						<div class="model-size">{m.sizeGB.toFixed(1)} GB</div>
					</div>
					<div class="model-info">
						<h3 class="model-name">{m.name}</h3>
						<p class="model-tagline">{m.tagline}</p>
					</div>
					{#if downloading === m.id}
						<div class="progress-block">
							<div class="progress-track">
								<div class="progress-fill" style="width: {progress?.percent ?? 0}%;"></div>
							</div>
							<p class="progress-text">{Math.round(progress?.percent ?? 0)}%</p>
						</div>
					{:else if m.installed}
						<button class="btn-remove" onclick={() => remove(m.id)}>Remove</button>
					{:else}
						<button class="btn-install" onclick={() => install(m.id)}>Install</button>
					{/if}
				</div>
			{/each}
		</div>
	{:else}
		<p class="loading-text">Loading...</p>
	{/if}
</div>

<style>
	.page {
		padding: 1.5rem;
		max-width: 800px;
		margin: 0 auto;
	}

	.title {
		margin-bottom: 1rem;
	}

	.error-banner {
		background: #3b1a1a;
		border: 1px solid #ef4444;
		border-radius: 6px;
		padding: 0.6rem 0.8rem;
		margin-bottom: 1rem;
		font-size: 0.85rem;
		color: #fca5a5;
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.model-card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 10px;
		padding: 1rem;
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.model-meta {
		min-width: 80px;
		text-align: center;
	}

	.model-quality {
		color: #fbbf24;
		font-size: 0.85rem;
	}

	.model-size {
		color: var(--text2);
		font-size: 0.8rem;
	}

	.model-info {
		flex: 1;
	}

	.model-name {
		font-size: 1rem;
	}

	.model-tagline {
		color: var(--text2);
		font-size: 0.8rem;
	}

	.progress-block {
		width: 120px;
	}

	.progress-track {
		height: 6px;
		background: var(--border);
		border-radius: 3px;
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: var(--accent);
		transition: width 0.3s;
	}

	.progress-text {
		font-size: 0.75rem;
		color: var(--text2);
		text-align: center;
		margin-top: 0.25rem;
	}

	.btn-remove {
		padding: 0.4rem 1rem;
		border-radius: 6px;
		border: 1px solid var(--border);
		background: none;
		color: var(--text2);
		font-size: 0.85rem;
		cursor: pointer;
	}

	.btn-install {
		padding: 0.4rem 1.25rem;
		border-radius: 6px;
		border: none;
		background: var(--accent);
		color: #fff;
		font-weight: 600;
		font-size: 0.85rem;
		cursor: pointer;
	}

	.loading-text {
		color: var(--text2);
		font-size: 0.9rem;
		text-align: center;
		padding: 2rem;
	}
</style>
