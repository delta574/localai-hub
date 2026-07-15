<script lang="ts">
	import { getSystemInfo, listModels, pullModel, type SystemInfo, type ModelInfo, type ProgressEvent } from '$lib/api';

	let sysInfo = $state<SystemInfo | null>(null);
	let models = $state<ModelInfo[]>([]);
	let downloading = $state<string | null>(null);
	let progress = $state<ProgressEvent | null>(null);
	let done = $state(false);

	$effect(() => {
		getSystemInfo().then((i) => sysInfo = i);
		listModels().then((m) => models = m);
	});

	function startDownload(id: string) {
		downloading = id;
		pullModel(id, (evt) => {
			progress = evt;
			if (evt.type === 'done') {
				done = true;
			}
		});
	}

	function skip() {
		done = true;
	}
</script>

{#if !done}
	<div class="wizard-overlay">
		<div class="wizard">
			<h1>Welcome to LocalAI Hub</h1>
			{#if sysInfo}
				<p class="specs">Your PC: <strong>{sysInfo.ram.total} GB RAM</strong> &middot; <strong>{sysInfo.cpu} cores</strong> &middot; <strong>{sysInfo.os}</strong></p>
			{/if}
			<p class="subtitle">Pick a model to install. Your PC can run any of these:</p>

			<div class="model-grid">
				{#each models as m}
					<div class="model-card" class:installed={m.installed} class:downloading={downloading === m.id}>
						<div class="card-header">
							<span class="quality">{m.quality}</span>
							<span class="size">{m.sizeGB.toFixed(1)} GB</span>
						</div>
						<h3>{m.name}</h3>
						<p class="tagline">{m.tagline}</p>
						{#if downloading === m.id}
							<div class="progress-bar">
								<div class="progress-fill" style="width: {progress?.percent ?? 0}%"></div>
							</div>
							<p class="progress-text">{Math.round(progress?.percent ?? 0)}%</p>
						{:else if m.installed}
							<button class="btn installed-btn" disabled>Installed</button>
						{:else}
							<button class="btn install-btn" onclick={() => startDownload(m.id)}>Install</button>
						{/if}
					</div>
				{/each}
			</div>

			<button class="skip-btn" onclick={skip}>Skip &mdash; I'll choose later</button>
		</div>
	</div>
{/if}

<style>
	.wizard-overlay {
		position: fixed; inset: 0;
		background: var(--bg);
		display: flex; align-items: center; justify-content: center;
		z-index: 1000;
		padding: 2rem;
	}
	.wizard {
		max-width: 700px; width: 100%;
	}
	h1 { font-size: 1.8rem; margin-bottom: 0.5rem; }
	.specs { color: var(--text2); margin-bottom: 0.25rem; font-size: 0.95rem; }
	.subtitle { color: var(--text2); margin-bottom: 1.5rem; }
	.model-grid { display: flex; flex-direction: column; gap: 0.75rem; }
	.model-card {
		background: var(--surface); border: 1px solid var(--border);
		border-radius: 10px; padding: 1rem;
		display: flex; align-items: center; gap: 1rem;
	}
	.model-card.installed { opacity: 0.6; }
	.card-header { display: flex; flex-direction: column; align-items: center; min-width: 80px; }
	.quality { color: #fbbf24; font-size: 0.85rem; }
	.size { color: var(--text2); font-size: 0.8rem; }
	h3 { flex: 1; font-size: 1rem; }
	.tagline { color: var(--text2); font-size: 0.8rem; display: none; }
	.btn {
		padding: 0.5rem 1.25rem; border-radius: 6px; border: none;
		font-weight: 600; font-size: 0.85rem; white-space: nowrap;
	}
	.install-btn { background: var(--accent); color: #fff; }
	.installed-btn { background: var(--surface2); color: var(--text2); }
	.progress-bar { width: 100px; height: 6px; background: var(--border); border-radius: 3px; overflow: hidden; }
	.progress-fill { height: 100%; background: var(--accent); transition: width 0.3s; }
	.progress-text { font-size: 0.8rem; color: var(--text2); min-width: 3rem; }
	.skip-btn {
		display: block; margin: 1.5rem auto 0;
		background: none; border: none; color: var(--text2); font-size: 0.85rem;
	}
	.skip-btn:hover { color: var(--text); }
</style>
