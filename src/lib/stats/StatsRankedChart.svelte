<script lang="ts">
	import { untrack } from "svelte";
	import StatsTooltip from "./StatsTooltip.svelte";
	import StatsExpansion from "./StatsExpansion.svelte";
	import { BarChart } from "layerchart/svg";
	import { averageRating, decimal } from "./format";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import { RatingSystem } from "@/types";
	import type { StatsBar } from "./types";
	let {
		title,
		items,
		unit = "watched titles",
		color = "#29acf4",
		sortBy = "count",
		settings,
		onSelect,
		count = $bindable(5),
	}: {
		title: string;
		unit?: string;
		items: StatsBar[];
		color?: string;
		sortBy?: "count" | "rating";
		settings: RatingSettings;
		onSelect: (item: StatsBar) => void;
		count?: number;
	} = $props();
	let hovered = $state<StatsBar>();
	let tooltipAnchor: HTMLElement | undefined;
	let tooltipLeft = $state(0);
	let tooltipTop = $state(0);
	function showTooltip(item: StatsBar, element: EventTarget | null) {
		if (!(element instanceof HTMLElement)) return;
		tooltipAnchor = element;
		const rect = element.getBoundingClientRect();
		tooltipLeft = Math.max(16, Math.min(rect.left, window.innerWidth - 296));
		tooltipTop =
			rect.bottom + 8 + 116 < window.innerHeight
				? rect.bottom + 8
				: Math.max(16, rect.top - 116);
		hovered = item;
	}
	function repositionTooltip() {
		if (hovered && tooltipAnchor) {
			const rect = tooltipAnchor.getBoundingClientRect();
			if (rect.bottom < 0 || rect.top > window.innerHeight) hideTooltip();
			else showTooltip(hovered, tooltipAnchor);
		}
	}
	function hideTooltip() {
		tooltipAnchor = undefined;
		hovered = undefined;
	}

	$effect(() => {
		const sort = sortBy;
		untrack(() => {
			if (sort) count = 5;
		});
	});
	const ranked = $derived(
		[...items].sort((a, b) =>
			sortBy === "rating"
				? b.averageRating - a.averageRating ||
					b.count - a.count ||
					a.label.localeCompare(b.label)
				: b.count - a.count ||
					b.averageRating - a.averageRating ||
					a.label.localeCompare(b.label),
		),
	);
	const ratingScale = $derived(
		settings.ratingSystem === RatingSystem.OutOf5
			? 5
			: settings.ratingSystem === RatingSystem.OutOf100
				? 100
				: 10,
	);
	const maximum = $derived(
		sortBy === "rating"
			? ratingScale
			: Math.max(1, ...items.map((item) => item.count)),
	);
	function ratingValue(value: number) {
		if (settings.ratingSystem === RatingSystem.OutOf5) return value / 2;
		if (settings.ratingSystem === RatingSystem.OutOf100) return value * 10;
		return value;
	}
</script>

<svelte:window onscroll={repositionTooltip} onresize={repositionTooltip} />

<div
	class="ranking"
	role="group"
	aria-label={title}
	style={`--rank-base:${color}`}
>
	<h3 class="norm">{title}</h3>
	<div class="rows">
		{#each ranked.slice(0, count) as item (item.label)}
			{@const barValue =
				sortBy === "rating" ? ratingValue(item.averageRating) : item.count}
			<button
				class="plain row"
				onclick={() => {
					hideTooltip();
					onSelect(item);
				}}
				onpointerenter={(event) => showTooltip(item, event.currentTarget)}
				onpointermove={(event) => showTooltip(item, event.currentTarget)}
				onpointerleave={hideTooltip}
				onfocus={(event) => showTooltip(item, event.currentTarget)}
				onblur={hideTooltip}
				onkeydown={(event) => {
					if (event.key === "Escape") hideTooltip();
				}}
				aria-describedby={hovered === item
					? `stats-${title}-tooltip`
					: undefined}
				aria-label={`${item.label}: ${item.count} ${unit}, ${averageRating(item.averageRating, settings)} average. Bar shows ${sortBy === "rating" ? "rating" : "title count"}. Explore titles.`}
			>
				<span class="label"
					><span>{item.label}</span><span class="values"
						><strong
							>{sortBy === "rating"
								? decimal(ratingValue(item.averageRating)).replace(
										/([.,])0$/,
										"",
									)
								: item.count.toLocaleString()}</strong
						></span
					></span
				>
				<div class="track" aria-hidden="true">
					<BarChart
						data={[{ label: item.label, value: barValue }]}
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

	<StatsExpansion
		{count}
		total={ranked.length}
		onChange={(value) => (count = value)}
	/>
</div>

{#if hovered}
	<StatsTooltip
		label={hovered.label}
		titleCount={hovered.count}
		titleUnit={unit === "played games" ? "games" : "titles"}
		averageRating={averageRating(hovered.averageRating, settings)}
		id={`stats-${title}-tooltip`}
		position={{ left: tooltipLeft, top: tooltipTop }}
	/>
{/if}

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
	.label > span:first-child {
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
	.values {
		display: flex;
		align-items: baseline;
		justify-content: end;
		gap: 8px;
		flex: none;
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
	.empty {
		color: var(--stats-muted);
	}
</style>
