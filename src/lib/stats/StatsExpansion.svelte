<script lang="ts">
	let {
		count,
		total,
		onChange,
		step = 5,
		separateCollapse = false,
	}: {
		count: number;
		total: number;
		step?: number;
		separateCollapse?: boolean;
		onChange: (count: number) => void;
	} = $props();
</script>

{#if separateCollapse}
	{#if count < total || count > 5}
		<div class="expansion">
			{#if count < total}<button
					class="plain more"
					onclick={() => onChange(Math.min(total, count + step))}
					>Show more ({total - count} remaining)</button
				>{/if}
			{#if count > 5}<button class="plain more less" onclick={() => onChange(5)}
					>Show less</button
				>{/if}
		</div>
	{/if}
{:else if count < total || (total > 5 && count > 5)}
	<button
		class="plain more"
		onclick={() => onChange(count < total ? Math.min(total, count + step) : 5)}
	>
		{#if count < total}Show more ({total - count} remaining){:else}Show less{/if}
	</button>
{/if}

<style>
	.expansion {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20px;
	}
	.less {
		margin-left: auto;
	}
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
