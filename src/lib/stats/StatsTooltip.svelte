<script lang="ts">
	let {
		label,
		titleCount,
		averageRating,
		detail,
		id,
		position,
	}: {
		label: string;
		titleCount: number;
		averageRating: string;
		detail?: string;
		id?: string;
		position?: { left: number; top: number };
	} = $props();
</script>

<div
	class="tooltip"
	class:floating={position !== undefined}
	role="tooltip"
	{id}
	style={position ? `left:${position.left}px;top:${position.top}px` : undefined}
>
	<strong>{label}</strong><span
		>{titleCount.toLocaleString()} {titleCount === 1 ? "title" : "titles"}</span
	>
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
	.tooltip span {
		margin-left: 12px;
		color: #086fa8;
	}
	:global(:root.theme-dark) .tooltip span {
		color: #29acf4;
	}
	.tooltip p {
		margin: 6px 0 0;
		font-size: 14px;
		max-height: 180px;
		overflow: auto;
	}
</style>
