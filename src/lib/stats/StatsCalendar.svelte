<script lang="ts">
	import type { StatsDay, StatsResponse, StatsSelection } from "./types";
	import StatsSegmentedControl from "./StatsSegmentedControl.svelte";
	import StatsTooltip from "./StatsTooltip.svelte";
	import { averageRating, meanRating, statsUnitLabel } from "./format";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import { statsTooltipPosition } from "./tooltipPosition";
	let {
		data,
		settings,
		onSelect,
		calendarYear = $bindable<number | undefined>(),
		activityKind = $bindable<"progress" | "completions">("progress"),
	}: {
		data: StatsResponse;
		settings: RatingSettings;
		onSelect: (selection: StatsSelection) => void;
		calendarYear?: number;
		activityKind?: "progress" | "completions";
	} = $props();
	const isGame = $derived(data.media === "game");
	const calendarDays = $derived(
		isGame && activityKind === "completions"
			? (data.games?.calendarCompletions ?? [])
			: (data.calendar ?? []),
	);
	const lifetime = $derived(data.scope === "lifetime");
	const years = $derived(
		[
			...new Set(
				[
					...(data.calendar ?? []),
					...(isGame ? (data.games?.calendarCompletions ?? []) : []),
				].map((d) => Number(d.date.slice(0, 4))),
			),
		].sort((a, b) => b - a),
	);
	const year = $derived(
		lifetime
			? calendarYear !== undefined && years.includes(calendarYear)
				? calendarYear
				: (years[0] ?? new Date().getUTCFullYear())
			: (data.year ?? new Date().getUTCFullYear()),
	);
	const days = $derived(
		calendarDays.filter((d) => Number(d.date.slice(0, 4)) === year),
	);
	const byDate = $derived(new Map(days.map((d) => [d.date, d])));
	const unit = $derived(
		isGame
			? activityKind === "progress"
				? "progress events"
				: "completions"
			: data.media === "tv"
				? "episodes"
				: "watches",
	);
	$effect(() => {
		// Metric and year changes invalidate any previously displayed day tooltip.
		void activityKind;
		void year;
		hoveredDay = undefined;
	});
	type CalendarTooltip = {
		label: string;
		plays: number;
		averageRating: string;
		detail: string;
	};
	let hoveredDay = $state<CalendarTooltip>();
	let tooltipLeft = $state(0);
	let tooltipTop = $state(0);
	const summary = $derived.by(() => {
		let streak = 0,
			longest = 0,
			previous: number | undefined;
		let busiest: (typeof days)[number] | undefined;
		for (const day of days) {
			const current = new Date(`${day.date}T00:00:00Z`).getTime();
			streak =
				previous !== undefined && current - previous === 86400000
					? streak + 1
					: 1;
			longest = Math.max(longest, streak);
			previous = current;
			if (!busiest || day.plays > busiest.plays) busiest = day;
		}
		return {
			longest,
			busiest,
			total: days.reduce((sum, d) => sum + d.plays, 0),
		};
	});
	function countLabel(count: number) {
		return `${count} ${statsUnitLabel(unit, count)}`;
	}
	function dateKey(month: number, day: number) {
		return `${String(year).padStart(4, "0")}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
	}
	function monthInfo(month: number) {
		const date = new Date(`${dateKey(month, 1)}T00:00:00Z`);
		const leapYear = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
		const monthDays = [
			31,
			leapYear ? 29 : 28,
			31,
			30,
			31,
			30,
			31,
			31,
			30,
			31,
			30,
			31,
		];
		return {
			label: date.toLocaleDateString(undefined, {
				month: "long",
				timeZone: "UTC",
			}),
			offset: (date.getUTCDay() + 6) % 7,
			count: monthDays[month],
		};
	}
	function prettyDate(date: string) {
		return new Date(`${date}T00:00:00Z`).toLocaleDateString(undefined, {
			month: "long",
			day: "numeric",
			year: "numeric",
			timeZone: "UTC",
		});
	}
	function explore(date: string) {
		const day = byDate.get(date);
		if (day)
			onSelect({
				label: prettyDate(date),
				items: day.items,
				description: isGame
					? activityKind === "progress"
						? "with recorded progress"
						: "completed"
					: data.media === "tv"
						? "episodes watched"
						: "watched",
				period: countLabel(day.plays),
			});
	}
	function showTooltip(day: StatsDay, event: PointerEvent | FocusEvent) {
		const element = event.currentTarget;
		if (!(element instanceof HTMLElement)) return;
		if (event.type === "focus" && !element.matches(":focus-visible")) return;
		const position = statsTooltipPosition(event, element);
		tooltipLeft = position.left;
		tooltipTop = position.top;
		hoveredDay = {
			label: prettyDate(day.date),
			plays: day.plays,
			averageRating: averageRating(
				meanRating(day.items.map((item) => item.rating ?? 0)),
				settings,
			),
			detail: `${day.items.length} unique ${isGame ? (day.items.length === 1 ? "game" : "games") : day.items.length === 1 ? "title" : "titles"}`,
		};
	}
	function hideTooltip() {
		hoveredDay = undefined;
	}
</script>

<svelte:window onscrollcapture={hideTooltip} onresize={hideTooltip} />

<section class="calendar-section">
	<div class="section-heading">
		<h2 class="norm">{isGame ? "Gaming" : "Viewing"} calendar</h2>
		{#if isGame}
			<StatsSegmentedControl label="Gaming calendar metric">
				<button
					class:active={activityKind === "progress"}
					aria-pressed={activityKind === "progress"}
					onclick={() => (activityKind = "progress")}>Progress</button
				>
				<button
					class:active={activityKind === "completions"}
					aria-pressed={activityKind === "completions"}
					onclick={() => (activityKind = "completions")}>Completions</button
				>
			</StatsSegmentedControl>
		{/if}
		{#if lifetime && years.length}<label
				>Calendar year <select
					aria-label="Calendar year"
					value={year}
					onchange={(event) =>
						(calendarYear = Number(event.currentTarget.value))}
					>{#each years as y (y)}<option value={y}>{y}</option>{/each}</select
				></label
			>{:else}<span>{year} · dates in UTC</span>{/if}
	</div>

	<div class="summary">
		<span><strong>{summary.total.toLocaleString()}</strong> {unit}</span>
		<span><strong>{days.length}</strong> active days</span>
		<span
			><strong>{summary.longest}</strong>
			{summary.longest === 1 ? "day" : "days"} · longest streak</span
		>
		{#if summary.busiest}<button
				class="plain busiest"
				onclick={() => summary.busiest && explore(summary.busiest.date)}
				>Busiest: {prettyDate(summary.busiest.date)} · {countLabel(
					summary.busiest.plays,
				)}</button
			>{/if}
	</div>
	<div class="legend" role="img" aria-label={`Daily ${unit}: 0, 1, 2 or 3+`}>
		<span>0</span><i class="level-one"></i><span>1</span><i class="level-two"
		></i><span>2</span><i class="level-three"></i><span>3+</span>
	</div>
	{#if !days.length}<p class="note">
			No recorded {unit} for this calendar year.
		</p>{/if}
	<div class="months">
		{#each Array.from({ length: 12 }, (_, i) => i) as month (month)}
			{@const info = monthInfo(month)}
			<div class="month" role="group" aria-label={`${info.label} ${year}`}>
				<h3 class="norm">{info.label}</h3>
				<div class="month-grid">
					{#each ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"] as weekday (weekday)}<span
							class="weekday">{weekday}</span
						>{/each}
					{#each Array.from({ length: info.offset }, (_, i) => i) as blank (blank)}<span
							aria-hidden="true"
						></span>{/each}
					{#each Array.from({ length: info.count }, (_, i) => i + 1) as day (day)}
						{@const date = dateKey(month, day)}
						{@const count = byDate.get(date)?.plays ?? 0}
						<button
							class="plain day"
							class:active={count > 0}
							class:level-one={count === 1}
							class:level-two={count === 2}
							class:level-three={count >= 3}
							disabled={!count}
							aria-label={`${prettyDate(date)}: ${countLabel(count)}${count ? `. Browse recorded ${isGame ? "games" : "titles"}` : ""}`}
							aria-describedby={hoveredDay?.label === prettyDate(date)
								? "stats-calendar-tooltip"
								: undefined}
							onpointerenter={(event) => {
								const dayData = byDate.get(date);
								if (dayData) showTooltip(dayData, event);
							}}
							onpointermove={(event) => {
								const dayData = byDate.get(date);
								if (dayData) showTooltip(dayData, event);
							}}
							onpointerleave={hideTooltip}
							onpointerup={(event) => {
								if (event.pointerType !== "mouse") hideTooltip();
							}}
							onpointercancel={hideTooltip}
							onfocus={(event) => {
								const dayData = byDate.get(date);
								if (dayData) showTooltip(dayData, event);
							}}
							onblur={hideTooltip}
							onkeydown={(event) => event.key === "Escape" && hideTooltip()}
							onclick={() => {
								hideTooltip();
								explore(date);
							}}>{day}</button
						>
					{/each}
				</div>
			</div>
		{/each}
	</div>
	<p class="note">
		{isGame
			? activityKind === "progress"
				? "Recorded progress, once per game per UTC day"
				: "Recorded completions, including replays"
			: data.media === "tv"
				? "Completed episodes"
				: "Recorded movie plays, including rewatches"}. Select a highlighted day
		to browse. Streaks are measured within the displayed year.
	</p>
</section>
{#if hoveredDay}
	<StatsTooltip
		label={hoveredDay.label}
		titleCount={hoveredDay.plays}
		titleUnit={unit}
		averageRating={hoveredDay.averageRating}
		detail={hoveredDay.detail}
		id="stats-calendar-tooltip"
		position={{ left: tooltipLeft, top: tooltipTop }}
	/>
{/if}

<style>
	.calendar-section {
		min-width: 0;
		padding: 26px 0 30px;
		border-top: 1px solid var(--stats-border);
	}
	h3 {
		font-size: 15px;
		margin-bottom: 10px;
	}
	.section-heading label,
	.note {
		font-size: 13px;
		color: var(--stats-muted);
	}
	select {
		margin-left: 8px;
		padding: 5px 8px;
		color: var(--stats-text);
		background: var(--stats-surface);
		border: 1px solid var(--stats-border);
		border-radius: 5px;
	}
	.summary {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 10px 24px;
		color: var(--stats-muted);
		font-size: 13px;
	}
	.summary strong {
		font-size: 22px;
		color: var(--stats-text);
		padding-right: 3px;
	}
	.busiest {
		color: var(--stats-accent);
		text-align: left;
	}
	.legend {
		display: flex;
		align-items: center;
		gap: 5px;
		margin: 18px 0 24px;
		color: var(--stats-muted);
		font-size: 12px;
	}
	.legend i {
		display: block;
		width: 12px;
		height: 12px;
		border-radius: 2px;
	}
	.legend .level-one {
		background: color-mix(
			in srgb,
			var(--stats-accent) 32%,
			var(--stats-surface)
		);
	}
	.legend .level-two {
		background: color-mix(
			in srgb,
			var(--stats-accent) 66%,
			var(--stats-surface)
		);
	}
	.legend .level-three {
		background: var(--stats-accent);
	}
	.months {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 24px;
	}
	.month-grid {
		display: grid;
		grid-template-columns: repeat(7, minmax(0, 1fr));
		gap: 5px;
	}
	.weekday {
		text-align: center;
		font-size: 11px;
		color: var(--stats-muted);
		padding-bottom: 3px;
	}
	.day {
		width: 100%;
		aspect-ratio: 1;
		border-radius: 4px;
		color: var(--stats-muted);
		font-size: 12px;
		padding: 0;
		background: var(--stats-border);
	}
	.day.level-one {
		background: color-mix(
			in srgb,
			var(--stats-accent) 32%,
			var(--stats-surface)
		);
	}
	.day.level-two {
		background: color-mix(
			in srgb,
			var(--stats-accent) 66%,
			var(--stats-surface)
		);
	}
	.day.level-three {
		background: var(--stats-accent);
	}
	.day.active {
		color: var(--stats-text);
		cursor: pointer;
	}
	.day.active:hover {
		outline: 1px solid var(--stats-accent);
		outline-offset: 2px;
	}
	.day:disabled {
		opacity: 0.55;
	}
	button:focus-visible,
	select:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.note {
		margin-top: 20px;
	}
	.busiest {
		cursor: pointer;
	}
	@media (max-width: 800px) {
		.months {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 20px;
		}
	}
	@media (max-width: 500px) {
		.months {
			grid-template-columns: minmax(0, 1fr);
		}
		.day {
			max-height: 40px;
		}
	}
</style>
