<script lang="ts">
	import StatsChart from "./StatsChart.svelte";
	import StatsBrowseList from "./StatsBrowseList.svelte";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsExpansion from "./StatsExpansion.svelte";
	import StatsTooltip from "./StatsTooltip.svelte";
	import { averageRating, meanRating } from "./format";
	import { statsTooltipPosition } from "./tooltipPosition";
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
	const momentumMaximum = $derived(
		Math.max(
			1,
			...(library?.momentum.flatMap((p) => [
				p.planned.length,
				p.watched.length,
			]) ?? []),
		),
	);
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
	const momentumBrowsePoints: ChartPoint[] = $derived(
		(library?.momentum ?? []).flatMap((point) =>
			(["planned", "watched"] as const).flatMap((kind) => {
				const items = kind === "planned" ? point.planned : point.watched;
				if (!items.length) return [];
				const label = `${kind === "planned" ? "First planned" : `First ${conversion} from ${isGame ? "backlog" : "watchlist"}`} · ${point.period}`;
				return [
					{
						label,
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
	type HoverTooltip = {
		label: string;
		titleCount: number;
		averageRating: string;
		detail: string;
	};
	let hoveredTooltip = $state<HoverTooltip>();
	let tooltipLeft = $state(0);
	let tooltipTop = $state(0);
	function showTooltip(
		label: string,
		items: StatsMediaCard[],
		detail: string,
		event: PointerEvent | FocusEvent,
	) {
		const element = event.currentTarget;
		if (!(element instanceof HTMLElement)) return;
		if (event.type === "focus" && !element.matches(":focus-visible")) return;
		const position = statsTooltipPosition(event, element);
		tooltipLeft = position.left;
		tooltipTop = position.top;
		hoveredTooltip = {
			label,
			titleCount: items.length,
			averageRating: averageRating(bucketRating(items), settings),
			detail,
		};
	}
	function hideTooltip() {
		hoveredTooltip = undefined;
	}
</script>

<svelte:window onscrollcapture={hideTooltip} onresize={hideTooltip} />

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
				<div
					class="momentum"
					class:annual={lifetime}
					role="group"
					aria-label={`First planned versus first ${conversion}`}
				>
					{#each library.momentum as point (point.period)}
						<div class="momentum-period">
							<div class="pair">
								{#each ["planned", "watched"] as kind (kind)}
									{@const items =
										kind === "planned" ? point.planned : point.watched}
									{@const detail = kind === "planned" ? "" : ""}
									{@const label = `${kind === "planned" ? "First planned" : `First ${conversion}`} · ${point.period}`}
									<button
										class="plain momentum-bar"
										class:planned={kind === "planned"}
										class:watched={kind === "watched"}
										style:height={`${items.length ? Math.max(8, (items.length / momentumMaximum) * 110) : 0}px`}
										disabled={!items.length}
										onpointerenter={(event) =>
											showTooltip(label, items, detail, event)}
										onpointermove={(event) =>
											showTooltip(label, items, detail, event)}
										onpointerleave={hideTooltip}
										onpointerup={(event) => {
											if (event.pointerType !== "mouse") hideTooltip();
										}}
										onpointercancel={hideTooltip}
										onfocus={(event) =>
											showTooltip(label, items, detail, event)}
										onblur={hideTooltip}
										onkeydown={(event) =>
											event.key === "Escape" && hideTooltip()}
										aria-describedby={hoveredTooltip?.label === label
											? "stats-library-tooltip"
											: undefined}
										aria-label={`Browse ${items.length} ${titleUnit}: ${label}`}
										onclick={() => {
											hideTooltip();
											onSelect({
												label,
												items,
												description: detail,
												period: point.period,
											});
										}}
									>
										<span>{items.length || ""}</span>
									</button>
								{/each}
							</div>
							<span class="period-label">{periodLabel(point.period)}</span>
						</div>
					{/each}
				</div>
				<StatsBrowseList
					valueUnit={titleUnit}
					points={momentumBrowsePoints}
					{settings}
					onSelect={(point) =>
						onSelect({
							label: point.label,
							items: point.items ?? [],
							description: point.detail,
							period: point.label.slice(point.label.lastIndexOf(" · ") + 3),
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
					{isGame ? "Time in your backlog" : "Time on your watchlist"}
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
{#if hoveredTooltip}
	<StatsTooltip
		label={hoveredTooltip.label}
		{titleUnit}
		titleCount={hoveredTooltip.titleCount}
		averageRating={hoveredTooltip.averageRating}
		detail={hoveredTooltip.detail}
		id="stats-library-tooltip"
		position={{ left: tooltipLeft, top: tooltipTop }}
	/>
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
	.momentum {
		display: grid;
		grid-template-columns: repeat(12, minmax(0, 1fr));
		gap: 10px;
		margin-top: 25px;
	}
	.momentum.annual {
		grid-template-columns: repeat(auto-fit, minmax(58px, 1fr));
	}
	.momentum-period {
		min-width: 0;
		text-align: center;
	}
	.pair {
		height: 132px;
		display: flex;
		align-items: end;
		justify-content: center;
		gap: 5px;
		border-bottom: 1px solid var(--stats-border);
	}
	.momentum-bar {
		position: relative;
		width: 24px;
		max-width: 42%;
		min-height: 0;
		padding: 0;
		border-radius: 3px 3px 0 0;
	}
	.momentum-bar > span {
		position: absolute;
		bottom: calc(100% + 3px);
		left: 50%;
		transform: translateX(-50%);
		font-size: 11px;
		color: var(--stats-muted);
	}
	.period-label {
		display: block;
		font-size: 12px;
		margin-top: 7px;
		color: var(--stats-muted);
	}
	button:not(:disabled) {
		cursor: pointer;
	}
	button:not(:disabled):hover {
		filter: brightness(1.15);
	}
	button:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	@media (max-width: 600px) {
		.momentum {
			grid-template-columns: repeat(6, minmax(0, 1fr));
			gap: 18px 8px;
		}
	}
</style>
