<script lang="ts">
	import { averageRating, statsUnitLabel, type StatsUnit } from "./format";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import type { ChartPoint } from "./types";

	let {
		points,
		settings,
		valueUnit = "titles",
		onSelect,
	}: {
		points: ChartPoint[];
		settings?: RatingSettings;
		valueUnit?: StatsUnit;
		onSelect?: (point: ChartPoint) => void;
	} = $props();
	function countLabel(point: ChartPoint) {
		const value = point.value ?? 0;
		return `${value.toLocaleString(undefined, { maximumFractionDigits: 0 })} ${statsUnitLabel(valueUnit, value)}`;
	}
	function preview(point: ChartPoint) {
		const items = point.items ?? [];
		return [
			...items.slice(0, 3).map((item) => item.title),
			...(items.length > 3 ? [`+${items.length - 3} more`] : []),
		].join(" · ");
	}
</script>

<details>
	<summary>Browse chart titles <span>({points.length})</span></summary>
	<div class="data-list">
		{#each points as point (point.label)}
			<button
				type="button"
				class="plain data-row"
				disabled={!onSelect}
				aria-haspopup="dialog"
				onclick={() => onSelect?.(point)}
			>
				<span class="data-label">
					<span class="row-heading">{point.tooltipLabel ?? point.label}</span>
					<small
						>{point.averageRating
							? `${averageRating(point.averageRating, settings)} average rating`
							: "Unrated"}</small
					>
					{#if preview(point)}<small class="title-preview"
							>{preview(point)}</small
						>{/if}
				</span>
				<span class="row-action"
					><strong>{countLabel(point)}</strong><span aria-hidden="true">→</span
					></span
				>
			</button>
		{:else}<p class="empty">No recorded data to browse.</p>{/each}
	</div>
</details>

<style>
	summary {
		cursor: pointer;
		font-size: 13px;
		color: var(--stats-accent);
		margin-top: 12px;
		padding: 10px 0;
		min-height: 44px;
	}
	summary span {
		color: var(--stats-muted);
	}
	.data-list {
		max-height: 320px;
		overflow: auto;
		overscroll-behavior: contain;
		border: 1px solid var(--stats-border);
		border-radius: 8px;
	}
	.data-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		width: 100%;
		min-height: 60px;
		padding: 12px;
		text-align: left;
		font-size: 14px;
		color: inherit;
	}
	.data-row + .data-row {
		border-top: 1px solid var(--stats-border);
	}
	.row-heading {
		font-weight: 600;
	}
	.row-action {
		display: flex;
		align-items: center;
		gap: 8px;
		flex: none;
		color: var(--stats-accent);
		font-size: 13px;
		white-space: nowrap;
	}
	.data-row:not(:disabled) {
		cursor: pointer;
	}
	.data-row:not(:disabled):hover {
		color: var(--stats-accent);
		background: var(--stats-surface);
	}
	.data-row:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: -2px;
	}
	.data-label {
		display: grid;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.data-label small {
		margin-top: 2px;
		color: var(--stats-muted);
		font-size: 12px;
		line-height: 1.4;
	}
	.title-preview {
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		overflow: hidden;
	}
	summary:hover {
		color: var(--stats-accent);
	}
	summary:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.empty {
		color: var(--stats-muted);
		padding: 24px 12px;
		font-size: 13px;
	}
</style>
