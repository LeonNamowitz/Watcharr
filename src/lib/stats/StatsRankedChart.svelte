<script lang="ts">
	import { BarChart } from "layerchart/svg";
	import { averageRating } from "./format";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import type { StatsBar } from "./types";
	let {
		title,
		items,
		color = "#29acf4",
		settings,
		onSelect,
	}: {
		title: string;
		items: StatsBar[];
		color?: string;
		settings: RatingSettings;
		onSelect: (item: StatsBar) => void;
	} = $props();
	let count = $state(5);
	const maximum = $derived(Math.max(1, ...items.map((item) => item.count)));
</script>

<div
	class="ranking"
	role="group"
	aria-label={title}
	style={`--rank-base:${color}`}
>
	<h3 class="norm">{title}</h3>
	<div class="rows">
		{#each items.slice(0, count) as item (item.label)}
			<button
				class="plain row"
				onclick={() => onSelect(item)}
				aria-label={`${item.label}: ${item.count} watched titles. Explore titles.`}
				title={`${item.label} · ${averageRating(item.averageRating, settings)} average`}
			>
				<span class="label"
					><span>{item.label}</span><strong
						>{item.count.toLocaleString()}</strong
					></span
				>
				<div class="track" aria-hidden="true">
					<BarChart
						data={[{ label: item.label, value: item.count }]}
						orientation="horizontal"
						x="value"
						y="label"
						xDomain={[0, maximum]}
						height={10}
						padding={{ left: 0, right: 0, top: 0, bottom: 0 }}
						bandPadding={0}
						axis={false}
						grid={false}
						rule={false}
						legend={false}
						highlight={false}
						tooltipContext={false}
						motion="none"
						series={[
							{ key: "value", value: "value", color: "var(--rank-color)" },
						]}
						props={{
							bars: { radius: 4, stroke: "transparent", strokeWidth: 0 },
						}}
					/>
				</div>
			</button>
		{:else}<p class="empty">No recorded metadata yet.</p>{/each}
	</div>
	{#if items.length}<div class="scale" aria-hidden="true">
			<span>0</span><span>{maximum.toLocaleString()} titles</span>
		</div>{/if}
	{#if count < items.length}<button
			class="plain more"
			onclick={() => (count += 5)}
			>Show more <span>({items.length - count} remaining)</span></button
		>{/if}
</div>

<style>
	.ranking {
		min-width: 0;
	}
	h3 {
		font-size: 18px;
		margin-bottom: 18px;
	}
	.rows {
		display: grid;
		gap: 10px;
	}
	.row {
		--rank-color: var(--rank-base);
		display: grid;
		gap: 8px;
		width: 100%;
		padding: 4px 0;
		text-align: left;
		color: inherit;
		border-radius: 5px;
	}
	.label {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		font-size: 14px;
		line-height: 1.3;
	}
	.label > span {
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.label strong {
		flex: none;
		font-size: 14px;
		color: var(--stats-accent);
	}
	.track {
		height: 10px;
		border-radius: 4px;
		background: color-mix(in srgb, var(--rank-base) 10%, transparent);
		pointer-events: none;
		overflow: hidden;
	}
	.row:hover,
	.row:focus-visible {
		--rank-color: color-mix(in srgb, var(--rank-base) 80%, white);
		color: var(--stats-accent);
	}
	.row:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 4px;
	}
	.scale {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
		color: var(--stats-muted);
		margin-top: 8px;
	}
	.more {
		margin-top: 16px;
		font-size: 14px;
		color: var(--stats-accent);
	}
	.more:hover {
		text-decoration: underline;
	}
	.more span,
	.empty {
		color: var(--stats-muted);
	}
</style>
