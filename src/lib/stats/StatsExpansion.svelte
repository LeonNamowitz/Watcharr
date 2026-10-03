<script lang="ts">
	let {
		count,
		total,
		onChange,
	}: {
		count: number;
		total: number;
		onChange: (count: number) => void;
	} = $props();
</script>

{#if count < total}
	<button
		class="plain more"
		onclick={() => onChange(Math.min(total, count + 5))}
	>
		Show more ({total - count} remaining)
	</button>
{:else if total > 5 && count > 5}
	<button class="plain more" onclick={() => onChange(5)}>Show less</button>
{/if}

<style>
	.more {
		margin-top: 18px;
		font-size: 14px;
		color: var(--stats-accent);
	}
	.more:hover {
		text-decoration: underline;
	}
	.more:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 4px;
	}
</style>
