<script lang="ts">
	import { statsUnitLabel, type StatsUnit } from "./format";
	let {
		label,
		titleCount,
		titleUnit = "titles",
		averageRating,
		detail,
		id,
		position,
	}: {
		label: string;
		titleCount: number;
		titleUnit?: StatsUnit;
		averageRating: string;
		detail?: string;
		id?: string;
		position?: { left: number; top: number };
	} = $props();
	const unitLabel = $derived(statsUnitLabel(titleUnit, titleCount));
</script>

<div
	class="tooltip"
	class:floating={position !== undefined}
	role="tooltip"
	{id}
	style={position ? `left:${position.left}px;top:${position.top}px` : undefined}
>
	<div class="heading">
		<strong>
			{#each label.split(/(\d{4}-\d{2}(?:-\d{2})?)/) as part, index (index)}
				{#if /^\d{4}-\d{2}/.test(part)}<span class="date">{part}</span
					>{:else}{part}{/if}
			{/each}
		</strong>
		<span>{titleCount.toLocaleString()} {unitLabel}</span>
	</div>
	<p>Average rating: {averageRating}</p>
	{#if detail}<p>{detail}</p>{/if}
</div>

<style>
	.tooltip {
		background: var(--bg-color);
		color: var(--text-color);
		border: 1px solid #7775;
		border-radius: 8px;
		padding: 10px 12px;
		width: max-content;
		max-width: min(280px, calc(100vw - 32px));
		box-sizing: border-box;
		box-shadow: 0 6px 24px #0005;
		overflow-wrap: anywhere;
		font-size: 14px;
	}
	.floating {
		position: fixed;
		z-index: 60;
		pointer-events: none;
	}
	.heading {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 6px 12px;
	}
	.heading > span {
		white-space: nowrap;
		color: #086fa8;
	}
	.date {
		white-space: nowrap;
	}
	:global(:root.theme-dark) .heading > span {
		color: #29acf4;
	}
	.tooltip p {
		margin: 6px 0 0;
		font-size: 14px;
		max-height: 180px;
		overflow: auto;
	}
</style>
