<script lang="ts">
	import StatsSegmentedControl from "./StatsSegmentedControl.svelte";
	import StatsChart from "./StatsChart.svelte";
	import StatsRankedChart from "./StatsRankedChart.svelte";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsExpansion from "./StatsExpansion.svelte";
	import { decimal, meanRating } from "./format";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import type { StatsResponse, ChartPoint } from "./types";

	let {
		data,
		section,
		owner,
		settings,
		onSelect,
		onExplore,
		counts = $bindable<Record<string, number>>({
			genres: 5,
			modes: 5,
			themes: 5,
			perspectives: 5,
			developers: 5,
			publishers: 5,
			playtime: 5,
		}),
		sortBy = $bindable<"count" | "rating">("count"),
		interval = $bindable<"week" | "month">("week"),
		activityKind = $bindable<"progress" | "completions">("progress"),
	}: {
		data: StatsResponse;
		section: "activity" | "categories" | "companies" | "playtime";
		owner?: { id: string; username: string };
		settings: RatingSettings;
		onSelect: (point: ChartPoint) => void;
		onExplore: (label: string, keys: string[]) => void;
		counts?: Record<string, number>;
		sortBy?: "count" | "rating";
		interval?: "week" | "month";
		activityKind?: "progress" | "completions";
	} = $props();
	const games = $derived(data.games);
	const playtime = $derived(games?.playtime);
	const activity = $derived(
		activityKind === "completions" ? games?.completions : data.activity,
	);
	const activityUnit = $derived(
		activityKind === "completions" ? "completions" : "progress events",
	);
	const rankingColors = ["#29acf4", "#51ad79", "#f5b85a", "#f47983", "#b19bea"];
	function activityAxisLabel(
		date: Date,
		compactFormat: Intl.DateTimeFormatOptions,
	) {
		return date.toLocaleDateString(undefined, {
			...compactFormat,
			...(data.scope === "lifetime" ? { year: "2-digit" as const } : {}),
			timeZone: "UTC",
		});
	}
	const activityPoints: ChartPoint[] = $derived(
		interval === "week"
			? (activity?.weeks ?? []).map((w) => {
					const start = new Date(`${w.start}T00:00:00Z`);
					const end = new Date(start.getTime() + 6 * 86400000);
					const format = (d: Date) =>
						d.toLocaleDateString(undefined, {
							month: "short",
							day: "numeric",
							year: data.scope === "lifetime" ? "numeric" : undefined,
							timeZone: "UTC",
						});
					return {
						label:
							data.scope === "lifetime"
								? activityAxisLabel(start, {
										month: "short",
										day: "numeric",
									})
								: w.start.slice(5),
						tooltipLabel: `${format(start)} – ${format(end)}`,
						value: w.plays,
						titleCount: w.uniqueTitles,
						averageRating: w.averageRating,
						items: w.items ?? [],
						detail: `${w.plays} ${activityUnit}`,
					};
				})
			: (activity?.months ?? []).map((m) => ({
					label: activityAxisLabel(new Date(`${m.month}-01T00:00:00Z`), {
						month: "short",
					}),
					tooltipLabel: new Date(`${m.month}-01T00:00:00Z`).toLocaleDateString(
						undefined,
						{ month: "long", year: "numeric", timeZone: "UTC" },
					),
					value: m.plays,
					titleCount: m.items?.length ?? 0,
					averageRating: m.averageRating,
					items: m.items ?? [],
					detail: `${m.plays} ${activityUnit}`,
				})),
	);
	const rankings = $derived(
		section === "companies"
			? [
					{
						key: "developers",
						title: "Developers",
						items: games?.developers ?? [],
						color: rankingColors[0],
					},
					{
						key: "publishers",
						title: "Publishers",
						items: games?.publishers ?? [],
						color: rankingColors[1],
					},
				]
			: [
					{
						key: "genres",
						title: "Genres",
						items: data.genres,
						color: rankingColors[0],
					},
					{
						key: "modes",
						title: "Game modes",
						items: games?.modes ?? [],
						color: rankingColors[1],
					},
					{
						key: "themes",
						title: "Themes",
						items: games?.themes ?? [],
						color: rankingColors[2],
					},
					{
						key: "perspectives",
						title: "Player perspectives",
						items: games?.perspectives ?? [],
						color: rankingColors[3],
					},
				],
	);
	function distributionPoints(): ChartPoint[] {
		return (playtime?.distribution ?? []).map((b) => ({
			label: b.label,
			value: b.count,
			titleCount: b.count,
			averageRating: (() => {
				const keys = new Set(b.titleKeys ?? []);
				return meanRating(
					data.posters
						.filter((c) => keys.has(`game:${c.id}`))
						.map((c) => c.rating ?? 0),
				);
			})(),
			titleKeys: b.titleKeys ?? [],
		}));
	}
</script>

{#if games}
	{#if section === "activity" && activity}
		<section class="stats-game-section" data-stats-section={section}>
			<div class="section-heading">
				<h2 class="norm" data-stats-jump="activity" tabindex="-1">Activity</h2>
				<StatsSegmentedControl label="Activity metric" wrap>
					<button
						class="plain"
						class:active={activityKind === "progress"}
						aria-pressed={activityKind === "progress"}
						onclick={() => (activityKind = "progress")}>Progress</button
					>
					<button
						class="plain"
						class:active={activityKind === "completions"}
						aria-pressed={activityKind === "completions"}
						onclick={() => (activityKind = "completions")}>Completions</button
					>
				</StatsSegmentedControl>
				<StatsSegmentedControl label="Activity interval" wrap>
					<button
						class="plain"
						class:active={interval === "week"}
						aria-pressed={interval === "week"}
						onclick={() => (interval = "week")}>Week</button
					>
					<button
						class="plain"
						class:active={interval === "month"}
						aria-pressed={interval === "month"}
						onclick={() => (interval = "month")}>Month</button
					>
				</StatsSegmentedControl>
			</div>
			<div class="activity-averages">
				<span
					><strong>{activity.total.toLocaleString()}</strong>
					{activityUnit}</span
				>
				{#if data.scope === "year"}<span
						><strong>{decimal(activity.averagePerWeek)}</strong> / week</span
					><span
						><strong>{decimal(activity.averagePerMonth)}</strong> / month</span
					>{/if}
			</div>
			{#key `${interval}:${activityKind}`}<StatsChart
					titleUnit="games"
					title={`${activityKind === "progress" ? "Recorded progress" : "Completions"} by ${interval}`}
					points={activityPoints}
					valueUnit={activityUnit}
					{settings}
					alignFirstXAxisTick={true}
					{onSelect}
				/>{/key}
			<p class="muted">
				Progress records starts and non-planned status changes, once per game
				per UTC day. Completions retain separately recorded finishes.
			</p>
		</section>
	{:else if section === "categories" || section === "companies"}
		<section class="stats-game-section" data-stats-section={section}>
			<div class="section-heading">
				<h2 class="norm">
					{section === "companies"
						? "Developers & publishers"
						: "Genres & play styles"}
				</h2>
				<StatsSegmentedControl label="Sort game rankings" wrap>
					<button
						class="plain"
						class:active={sortBy === "count"}
						aria-pressed={sortBy === "count"}
						onclick={() => (sortBy = "count")}>Most played</button
					>
					<button
						class="plain"
						class:active={sortBy === "rating"}
						aria-pressed={sortBy === "rating"}
						onclick={() => (sortBy = "rating")}>Highest rated</button
					>
				</StatsSegmentedControl>
			</div>
			<div class="rankings">
				{#each rankings as group (group.key)}<StatsRankedChart
						title={group.title}
						items={group.items}
						color={group.color}
						unit="played games"
						{settings}
						{sortBy}
						bind:count={counts[group.key]}
						onSelect={(item) => onExplore(item.label, item.titleKeys)}
					/>{/each}
			</div>
		</section>
	{:else if section === "playtime" && playtime && data.scope === "lifetime"}
		<section class="stats-game-section" data-stats-section={section}>
			<div class="section-heading playtime-heading">
				<div class="playtime-intro">
					<h2 class="norm" data-stats-jump="playtime" tabindex="-1">
						Lifetime game stats
					</h2>
				</div>
			</div>
			<div class="totals" role="group" aria-label="Lifetime game playtime">
				<div class="total-card">
					<span class="metric-label">Total hours</span>
					<strong>{playtime.totalHours.toLocaleString()}</strong>
					<span class="metric-detail">recorded playtime</span>
				</div>
				<div class="total-card">
					<span class="metric-label">Average hours</span>
					<strong
						>{playtime.recordedGames
							? decimal(playtime.averageHours)
							: "—"}</strong
					>
					<span class="metric-detail">per recorded game</span>
				</div>
				<div class="total-card">
					<span class="metric-label">Median hours</span>
					<strong
						>{playtime.recordedGames
							? decimal(playtime.medianHours)
							: "—"}</strong
					>
					<span class="metric-detail">per recorded game</span>
				</div>
			</div>
			<h3 class="norm">Most-played games</h3>
			<StatsPosters
				items={playtime.mostPlayed.slice(0, counts.playtime ?? 5)}
				{owner}
				{settings}
				detail={(c) => `${c.playtimeHours?.toLocaleString()} hours`}
			/>
			<StatsExpansion
				count={counts.playtime ?? 5}
				total={playtime.mostPlayed.length}
				onChange={(value) => (counts.playtime = value)}
			/>
			<h3 class="norm">Games by playtime</h3>
			<StatsChart
				titleUnit="games"
				title="Games by recorded playtime"
				points={distributionPoints()}
				valueUnit="games"
				{settings}
				{onSelect}
			/>
			<h3 class="norm">Hours by personal rating</h3>
			<StatsChart
				titleUnit="games"
				title="Hours by personal rating"
				valueUnit="hours"
				points={playtime.byRating.map((b) => ({
					label: b.rating ? String(b.rating) : "Unrated",
					tooltipLabel: b.rating ? `Rated ${b.rating}/10` : "Unrated",
					value: b.hours,
					titleCount: b.items.length,
					averageRating: b.rating,
					items: b.items,
					detail: `${b.hours.toLocaleString()} recorded hours`,
				}))}
				{settings}
				{onSelect}
			/>
			<p class="muted coverage-note">
				Saved hours are available for {playtime.recordedGames} of {data.summary
					.titles}
				played games.
			</p>
		</section>
	{/if}
{/if}

<style>
	h3 {
		font-size: 17px;
		margin: 22px 0 14px;
	}
	.muted {
		color: var(--stats-muted);
		font-size: 13px;
	}
	.rankings {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 24px;
	}
	.totals {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 12px;
	}
	.total-card {
		display: flex;
		flex-direction: column;
		gap: 3px;
		min-width: 0;
		padding: 14px 16px;
		border: 1px solid var(--stats-border);
		border-radius: 10px;
		background: var(--stats-surface);
	}
	.playtime-intro p {
		margin: 6px 0 0;
		max-width: 680px;
		color: var(--stats-muted);
		font-size: 15px;
		line-height: 1.5;
	}
	.total-card .metric-label {
		color: var(--stats-text);
		font-size: 18px;
		font-weight: 600;
	}
	.total-card strong {
		font-size: 32px;
		line-height: 1.2;
		color: var(--stats-accent);
	}
	.total-card .metric-detail {
		font-size: 13px;
		color: var(--stats-muted);
	}
	.coverage-note {
		margin-top: 18px;
	}
	@media (max-width: 600px) {
		.rankings {
			grid-template-columns: minmax(0, 1fr);
		}
		.totals {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 8px;
		}
		.total-card:first-child {
			grid-column: 1 / -1;
		}
	}
</style>
