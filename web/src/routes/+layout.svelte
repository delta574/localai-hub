<script lang="ts">
	import '../app.css';
	let { children } = $props();
</script>

<svelte:boundary>
	{@render children()}
	{#snippet failed(error, reset)}
		<div class="error-boundary">
			<h2>Something went wrong</h2>
			<p>{error instanceof Error ? error.message : String(error)}</p>
			<button onclick={reset}>Try again</button>
		</div>
	{/snippet}
</svelte:boundary>

<style>
	.error-boundary {
		padding: 2rem;
		text-align: center;
	}
	.error-boundary h2 {
		color: #ef4444;
		margin-bottom: 0.5rem;
	}
	.error-boundary p {
		color: var(--text2);
		margin-bottom: 1rem;
	}
	.error-boundary button {
		padding: 0.5rem 1.5rem;
		border-radius: 6px;
		border: none;
		background: var(--accent);
		color: #fff;
		font-weight: 600;
		cursor: pointer;
	}
</style>
