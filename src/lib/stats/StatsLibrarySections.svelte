<script lang="ts">
	import StatsChart from "./StatsChart.svelte";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsExpansion from "./StatsExpansion.svelte";
	import { meanRating } from "./format";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import type {
		ChartPoint,
		StatsMediaCard,
		StatsResponse,
		StatsSelection,
	} from "./types";
	let {
		data,
		section,
		owner,
		settings,
		onSelect,
		waitingCount = $bindable(5),
	}: {
		data: StatsResponse;
		section: "library-momentum" | "library-waiting";
		owner?: { id: string; username: string };
		settings: RatingSettings;
		onSelect: (selection: StatsSelection) => void;
		waitingCount?: number;
	} = $props();
	const library = $derived(data.library);
	const isGame = $derived(data.media === "game");
	const titleUnit = $derived(isGame ? "games" : "titles");
	const conversion = $derived(isGame ? "completed" : "watched");
	const convertedFromList = $derived(
		isGame ? "first completed from backlog" : "first watched from watchlist",
	);
	const lifetime = $derived(data.scope === "lifetime");
	function bucketRating(items: StatsMediaCard[]) {
		return meanRating(items.map((item) => item.rating ?? 0));
	}
	function periodLabel(period: string) {
		return lifetime
			? period
			: new Date(`${period}-01T00:00:00Z`).toLocaleDateString(undefined, {
					month: "short",
					timeZone: "UTC",
				});
	}
	const waitingDays = $derived(
		new Map(
			library?.waiting.longest.map((w) => [
				`${w.item.type}:${w.item.id}`,
				w.days,
			]) ?? [],
		),
	);
	const momentumPoints: ChartPoint[] = $derived(
		(library?.momentum ?? []).flatMap((point) =>
			(["planned", "watched"] as const).flatMap((kind) => {
				const items = kind === "planned" ? point.planned : point.watched;
				const label = `${kind === "planned" ? "First planned" : `First ${conversion} from ${isGame ? "backlog" : "watchlist"}`} · ${point.period}`;
				return [
					{
						label,
						group: periodLabel(point.period),
						seriesKey: kind,
						period: point.period,
						value: items.length,
						titleCount: items.length,
						averageRating: bucketRating(items),
						items,
						detail: kind === "planned" ? "first planned" : convertedFromList,
					},
				];
			}),
		),
	);
</script>

{#if library}
	{#if section === "library-momentum"}<section
			class="library-section"
			data-stats-section={section}
		>
			<div class="section-heading">
				<h2 class="norm">{isGame ? "Backlog" : "Watchlist"} momentum</h2>
				<span
					>{lifetime ? "Through the years" : data.year} · first-time {titleUnit}</span
				>
			</div>
			<div class="totals">
				<span><strong>{library.planned.toLocaleString()}</strong> added</span
				><span
					><strong>{library.watched.toLocaleString()}</strong>
					{conversion}</span
				>
			</div>
			<div class="legend">
				<span><i class="planned"></i>First added</span><span
					><i class="watched"></i>First {conversion}</span
				>
			</div>
			{#if library.planned || library.watched}
				<StatsChart
					title={`${isGame ? "Backlog" : "Watchlist"} momentum`}
					kind="grouped"
					valueUnit={titleUnit}
					{titleUnit}
					points={momentumPoints}
					barSeries={[
						{ key: "planned", label: "First added", color: "#f5b85a" },
						{ key: "watched", label: `First ${conversion}`, color: "#29acf4" },
					]}
					{settings}
					onSelect={(point) =>
						onSelect({
							label: point.label,
							items: point.items ?? [],
							description: point.detail,
							period: point.period,
						})}
				/>
			{:else}<p class="note">
					No qualifying {isGame ? "backlog" : "watchlist"} additions or first {isGame
						? "completions"
						: "watches"} in this period.
				</p>{/if}
		</section>{/if}
	{#if section === "library-waiting"}<section
			class="library-section"
			data-stats-section={section}
		>
			<div class="section-heading">
				<h2 class="norm">
					Time {isGame ? "in" : "on"}
					{owner ? `${data.owner.username}'s` : "your"}
					{isGame ? "backlog" : "watchlist"}
				</h2>
				<span
					>Time before the first recorded {isGame
						? "completion"
						: "watch"}</span
				>
			</div>
			<div class="totals">
				<span
					><strong
						>{library.waiting.medianDays === null
							? "—"
							: library.waiting.medianDays.toLocaleString()}</strong
					> median days waiting</span
				><span
					><strong>{library.watched.toLocaleString()}</strong>
					{titleUnit} previously planned</span
				>
			</div>
			<StatsChart
				title={`${isGame ? "Backlog" : "Watchlist"} waiting time`}
				valueUnit={titleUnit}
				{titleUnit}
				points={library.waiting.buckets.map((bucket) => ({
					label: bucket.label,
					value: bucket.items.length,
					titleCount: bucket.items.length,
					averageRating: bucketRating(bucket.items),
					items: bucket.items,
				}))}
				{settings}
				onSelect={(point) =>
					onSelect({
						label: `Waited ${point.label.toLowerCase()}`,
						items: point.items ?? [],
						description: convertedFromList,
					})}
			/>
			{#if library.waiting.excluded}<p class="note">
					{library.waiting.excluded.toLocaleString()} first-{conversion}
					{titleUnit} excluded.
				</p>{/if}
			{#if library.waiting.longest.length}
				<h3 class="norm">The longest waits</h3>
				<StatsPosters
					items={library.waiting.longest
						.slice(0, waitingCount)
						.map((w) => w.item)}
					{owner}
					{settings}
					leftAligned
					detail={(c) =>
						`${waitingDays.get(`${c.type}:${c.id}`)} days before first ${isGame ? "completion" : "watch"}`}
				/>
				<StatsExpansion
					count={waitingCount}
					total={library.waiting.longest.length}
					onChange={(count) => (waitingCount = count)}
				/>
			{/if}
		</section>{/if}
{/if}

<style>
	.library-section {
		min-width: 0;
		padding: 26px 0 30px;
		border-top: 1px solid var(--stats-border);
	}
	h3 {
		font-size: 17px;
		margin: 22px 0 14px;
	}
	.note {
		color: var(--stats-muted);
		font-size: 13px;
	}
	.note {
		margin: 10px 0 18px;
	}
	.totals {
		display: flex;
		flex-wrap: wrap;
		gap: 10px 28px;
		color: var(--stats-muted);
		font-size: 14px;
	}
	.totals strong {
		color: var(--stats-text);
		font-size: 22px;
		padding-right: 4px;
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: 18px;
		font-size: 12px;
		margin: 20px 0;
		color: var(--stats-muted);
	}
	.legend span {
		display: flex;
		align-items: center;
		gap: 7px;
	}
	.legend i {
		width: 10px;
		height: 10px;
		border-radius: 2px;
	}
	.planned {
		background: #f5b85a;
	}
	.watched {
		background: #29acf4;
	}
</style>
