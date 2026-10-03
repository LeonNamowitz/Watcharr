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

{#if count < total || (total > 5 && count > 5)}
	<button
		class="plain more"
		onclick={() => onChange(count < total ? Math.min(total, count + 5) : 5)}
	>
		{#if count < total}Show more ({total - count} remaining){:else}Show less{/if}
	</button>
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
