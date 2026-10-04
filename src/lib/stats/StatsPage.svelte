<script lang="ts">
	import StatsSegmentedControl from "./StatsSegmentedControl.svelte";
	import { getContext, tick } from "svelte";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import StatsExpansion from "./StatsExpansion.svelte";
	import { resolve } from "$app/paths";
	import StatsChart from "./StatsChart.svelte";
	import StatsRankedChart from "./StatsRankedChart.svelte";
	import StatsTitlesDialog from "./StatsTitlesDialog.svelte";
	import { averageRating, decimal, meanRating } from "./format";
	import "./stats.css";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsPeople from "./StatsPeople.svelte";
	import StatsGameSections from "./StatsGameSections.svelte";
	import StatsLibrarySections from "./StatsLibrarySections.svelte";
	import StatsCalendar from "./StatsCalendar.svelte";
	import StatsLayoutEditor from "./StatsLayoutEditor.svelte";
	import { normalizeSectionOrder, type StatsSectionId } from "./sectionOrder";
	import { type RatingSettings } from "@/lib/rating/helpers";
	import {
		STATS_BACKGROUND_CONTEXT,
		type StatsBackgroundState,
	} from "./backgroundContext";
	import type {
		StatsResponse,
		StatsMedia,
		StatsMediaCard,
		StatsPie,
		ChartPoint,
		StatsSelection,
	} from "./types";
	let {
		data,
		publicOwner,
		onSelectionChange,
		onSaveLayout,
		loading = false,
		initialLoading = false,
		error,
		requestedYear,
		requestedMedia,
	}: {
		loading?: boolean;
		initialLoading?: boolean;
		error?: unknown;
		requestedYear?: string;
		requestedMedia?: StatsMedia;
		data: StatsResponse;
		publicOwner?: { id: string; username: string };
		onSelectionChange: (year: string, media: StatsMedia) => void;
		onSaveLayout: (order: StatsSectionId[]) => Promise<void>;
	} = $props();
	const statsBackground = getContext<StatsBackgroundState>(
		STATS_BACKGROUND_CONTEXT,
	);
	let episodeCount = $state(5);
	let editingLayout = $state(false);
	const orderedSections = $derived(
		normalizeSectionOrder(data.media, data.sectionOrder),
	);
	function detailsKey(el: HTMLDetailsElement) {
		const section = el.closest<HTMLElement>("[data-stats-section]");
		return section
			? `${section.dataset.statsSection}:${Array.from(section.querySelectorAll("details")).indexOf(el)}`
			: "coverage";
	}
	let episodeTab = $state<"current" | "older" | "unknown">("current");
	let highestTab = $state<"current" | "older">("current");
	let yearlyFavoriteCount = $state(5);
	// MOD: Rows added per click for Cast and Directors & creators only.
	const peopleExpansionRows = 3;
	// MOD: Rows added per click in the title list popup.
	const titleDialogExpansionRows = 2;
	let calendarMonthsExpanded = $state(false);
	let peopleMode = $state<"most" | "rating">("most");
	let categorySort = $state<"count" | "rating">("count");
	let higherCount = $state(5);
	let lowerCount = $state(5);
	let mostWatchedCount = $state(5);
	let favoriteCount = $state(5);
	let decadeCounts = $state<Record<number, number>>({});
	let crewCounts = $state<Record<string, number>>({});
	let gameCounts = $state<Record<string, number>>({
		genres: 5,
		modes: 5,
		themes: 5,
		perspectives: 5,
		developers: 5,
		publishers: 5,
		playtime: 5,
	});
	let gameActivityKind = $state<"progress" | "completions">("progress");
	let activityMode = $state<"week" | "month">("week");
	let activityMetric = $state<"count" | "rating">("count");
	let calendarYear = $state<number>();
	let gameCalendarKind = $state<"progress" | "completions">("progress");
	let waitingCount = $state(5);
	let selection = $state<{
		period?: string;
		label: string;
		items: StatsMediaCard[];
		personId?: number;
		description?: string;
	}>();
	let categoryCounts = $state({ genres: 5, countries: 5, languages: 5 });
	let peopleCounts = $state({ cast: 5, directors: 5, studios: 5 });
	let root: HTMLDivElement;
	let scrollY = $state(0);
	let viewportHeight = $state(0);
	function backToTop() {
		window.scrollTo({
			top: 0,
			behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
				? "instant"
				: "smooth",
		});
	}
	let titlesDialog = $state<StatsTitlesDialog>();
	let dialogState = $state<ReturnType<StatsTitlesDialog["capture"]>>();
	export function capture() {
		const dialog = titlesDialog?.capture();
		return $state.snapshot({
			episodeCount,
			episodeTab,
			highestTab,
			yearlyFavoriteCount,
			peopleMode,
			categorySort,
			higherCount,
			lowerCount,
			mostWatchedCount,
			favoriteCount,
			decadeCounts,
			crewCounts,
			activityMode,
			activityMetric,
			calendarMonthsExpanded,
			calendarYear,
			gameCalendarKind,
			waitingCount,
			gameCounts,
			gameActivityKind,
			peopleCounts,
			categoryCounts,
			selection,
			dialog,
			openDetails: Object.fromEntries(
				Array.from(root.querySelectorAll("details"), (el) => [
					detailsKey(el),
					el.open,
				]),
			),
			scrollX: dialog?.scrollX ?? window.scrollX,
			scrollY: dialog?.scrollY ?? window.scrollY,
		});
	}
	export async function restore(saved: ReturnType<typeof capture>) {
		// Restore the full page height before scrolling or reopening a modal.
		selection = undefined;
		episodeCount = saved.episodeCount;
		episodeTab = saved.episodeTab;
		highestTab = saved.highestTab;
		yearlyFavoriteCount = saved.yearlyFavoriteCount;
		peopleMode = saved.peopleMode;
		categorySort = saved.categorySort;
		higherCount = saved.higherCount;
		lowerCount = saved.lowerCount;
		mostWatchedCount = saved.mostWatchedCount;
		favoriteCount = saved.favoriteCount;
		decadeCounts = saved.decadeCounts;
		crewCounts = saved.crewCounts;
		calendarYear = saved.calendarYear;
		gameCalendarKind = saved.gameCalendarKind ?? "progress";
		waitingCount = saved.waitingCount ?? 5;
		activityMode = saved.activityMode;
		activityMetric = saved.activityMetric ?? "count";
		gameActivityKind = saved.gameActivityKind;
		peopleCounts = saved.peopleCounts;
		await tick();
		calendarMonthsExpanded = saved.calendarMonthsExpanded ?? false;
		await tick();
		// Ranked charts reset their expansion when sorting changes.
		categoryCounts = saved.categoryCounts;
		gameCounts = saved.gameCounts;
		await tick();
		root.querySelectorAll("details").forEach((el) => {
			el.open = saved.openDetails[detailsKey(el)] ?? false;
		});
		window.scrollTo({
			left: saved.scrollX,
			top: saved.scrollY,
			behavior: "instant",
		});
		dialogState = saved.dialog;
		selection = saved.selection;
	}

	function explore(label: string, keys: string[], personId?: number) {
		dialogState = undefined;
		const membership = new Set(keys);
		selection = {
			label,
			personId,
			description: isGame ? "played" : "watched",
			items: [...data.posters, ...(data.episodes ?? [])]
				.filter((c) =>
					membership.has(
						`${c.type}:${c.id}${c.episodeNumber !== undefined ? `:${c.seasonNumber ?? 0}:${c.episodeNumber}` : ""}`,
					),
				)
				.sort((a, b) => a.title.localeCompare(b.title)),
		};
	}
	function exploreSelection(value: StatsSelection) {
		dialogState = undefined;
		selection = value;
	}
	function explorePoint(point: ChartPoint) {
		dialogState = undefined;
		if (point.items)
			selection = {
				label: point.tooltipLabel ?? point.label,
				items: point.items,
				description: isGame ? "played" : "watched",
			};
		else explore(point.label, point.titleKeys ?? []);
	}
	function historyItems(
		point: StatsResponse["history"][number],
		metric: string,
	) {
		const reviewed = new Set(point.reviewedTitleKeys ?? []);
		return (point.items ?? []).filter((item) => {
			if (metric === "movies") return item.type === "movie";
			if (metric === "shows") return item.type === "tv";
			if (metric === "games") return item.type === "game";
			if (metric === "completed")
				return (point.completedTitleKeys ?? []).includes(
					`${item.type}:${item.id}`,
				);
			if (metric === "averageRating") return (item.rating ?? 0) > 0;
			return reviewed.has(`${item.type}:${item.id}`);
		});
	}
	function weekRange(start: string) {
		const first = new Date(`${start}T00:00:00Z`);
		const last = new Date(first.getTime() + 6 * 24 * 60 * 60 * 1000);
		const formatter = new Intl.DateTimeFormat(undefined, {
			month: "short",
			day: "numeric",
			timeZone: "UTC",
		});
		return `${formatter.format(first)} – ${formatter.format(last)}`;
	}

	const settings: RatingSettings = $derived({
		ratingSystem: data.owner.ratingSystem,
		ratingStep: data.owner.ratingStep,
	});
	const lifetime = $derived(data.scope === "lifetime");
	const statsView = $derived(`${data.scope}:${data.year}:${data.media}`);
	const period = $derived(lifetime ? "Lifetime" : String(data.year));
	const isGame = $derived(data.media === "game");
	const source = $derived(isGame ? "IGDB" : "TMDB");
	const mediaLabel = $derived(
		isGame ? "Games" : data.media === "movie" ? "Films" : "Shows",
	);
	const yearValue = $derived(lifetime ? "all" : String(data.year));
	const selectedYear = $derived(requestedYear ?? yearValue);
	const selectedMedia = $derived(requestedMedia ?? data.media);
	const primarySummaryCount = $derived(
		data.media === "movie"
			? data.summary.plays
			: data.media === "tv"
				? data.summary.shows
				: (data.summary.games ?? 0),
	);
	const newReleaseCount = $derived(
		lifetime
			? undefined
			: (data.breakdown.release.find((item) => item.label === "Selected year")
					?.count ?? 0),
	);
	const hoursPlayed = $derived(data.summary.hours);
	const hoursDescription = $derived(
		isGame
			? lifetime
				? "Saved cumulative playtime for games in this view."
				: "Saved playtime for games first completed in the selected year."
			: "Estimated from available runtimes, recorded watches, finished shows and finished episodes.",
	);
	// Summary tile targets and their tooltips.
	// Search data-stats-jump in src/lib/stats to find or add section targets.
	const summaryDestinations = $derived({
		titles: {
			target: "titles",
			label: isGame
				? "Jump to all games"
				: data.media === "tv"
					? "Jump to all shows"
					: "Jump to all movies",
		},
		secondary: {
			target: isGame
				? "completion"
				: data.media === "movie"
					? lifetime
						? "breakdown"
						: "release"
					: lifetime
						? data.library
							? "calendar"
							: "history"
						: "activity",
			label: isGame
				? "Jump to game completion chart"
				: data.media === "movie"
					? lifetime
						? "Jump to breakdown charts"
						: "Jump to release breakdown chart"
					: lifetime
						? data.library
							? "Jump to viewing calendar"
							: "Jump to watches through the years"
						: "Jump to viewing activity",
		},
		hours: {
			target: "calendar",
			label: isGame ? "Jump to gaming calendar" : "Jump to viewing calendar",
		},
		rating: {
			target: "rated-higher",
			label: "Jump to rated higher than average",
		},
	});
	async function jumpToSummary(metric: keyof typeof summaryDestinations) {
		if (loading || error) return;
		// Select a chart tab only when the summary tile targets that chart.
		if (isGame && metric === "hours" && !lifetime)
			gameCalendarKind = "completions";
		else if (
			isGame &&
			metric === "secondary" &&
			summaryDestinations.secondary.target === "activity"
		)
			gameActivityKind = "completions";
		else if (
			!isGame &&
			metric === "secondary" &&
			summaryDestinations.secondary.target === "activity"
		)
			activityMetric = "count";
		await tick();
		const target = root.querySelector<HTMLElement>(
			`[data-stats-jump="${summaryDestinations[metric].target}"]`,
		);
		if (!target) return;
		const navHeight =
			document.querySelector("nav")?.getBoundingClientRect().height ?? 0;
		target.style.scrollMarginTop = `${navHeight + 16}px`;
		target.focus({ preventScroll: true });
		target.scrollIntoView({
			block: "start",
			behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
				? "instant"
				: "smooth",
		});
	}

	const years = $derived(
		[
			...new Set([
				new Date().getUTCFullYear(),
				...(/^\d{4}$/.test(selectedYear) ? [Number(selectedYear)] : []),
				...data.availableYears,
				...(!lifetime && data.year ? [data.year] : []),
			]),
		].sort((a, b) => b - a),
	);
	const higher = $derived(data.ratingDifferences.higher ?? []);
	const lower = $derived(data.ratingDifferences.lower ?? []);
	const watchlistTitles = $derived(data.breakdown.watchlistTitles ?? []);
	const yearlyHighlights = $derived.by(() => {
		const year = data.year;
		if (!year) return [];
		const ranked = data.posters
			.filter(
				(card) =>
					(highestTab === "current"
						? card.releaseYear === year
						: (card.releaseYear ?? 0) > 0 && (card.releaseYear ?? 0) < year) &&
					(card.rating ?? 0) > 0,
			)
			.sort(
				(a, b) =>
					(b.rating ?? 0) - (a.rating ?? 0) || a.title.localeCompare(b.title),
			);
		const aboveThreshold = ranked.filter(
			(card) => (card.rating ?? 0) >= 8,
		).length;
		return ranked.slice(0, Math.max(5, aboveThreshold));
	});
	$effect(() => {
		if (statsView) {
			calendarYear = undefined;
			gameCalendarKind = "progress";
			waitingCount = 5;
			episodeCount = 5;
			episodeTab =
				!lifetime &&
				data.highestRatedEpisodes?.unknown?.length &&
				!data.highestRatedEpisodes.current.length &&
				!data.highestRatedEpisodes.older.length
					? "unknown"
					: "current";
			yearlyFavoriteCount = 5;
			calendarMonthsExpanded = false;
			favoriteCount = 5;
			highestTab = "current";
			categorySort = "count";
			higherCount = 5;
			lowerCount = 5;
			mostWatchedCount = 5;
			decadeCounts = {};
			crewCounts = {};
			peopleCounts = { cast: 5, directors: 5, studios: 5 };
			categoryCounts = { genres: 5, countries: 5, languages: 5 };
			peopleMode = "most";
			activityMode = "week";
			selection = undefined;
			dialogState = undefined;
		}
	});
	$effect(() => {
		if (loading) {
			selection = undefined;
			dialogState = undefined;
		}
	});
	const ratingPoints = $derived.by(() => {
		const distribution = data.breakdown.ratingDistribution;
		const rated = distribution.filter((b) => b.rating > 0 && b.count > 0);
		const step = rated.some(
			(b) => Math.abs(b.rating * 2 - Math.round(b.rating * 2)) > 0.001,
		)
			? 0.1
			: rated.some((b) => b.rating % 1 !== 0)
				? 0.5
				: 1;
		const points: ChartPoint[] = [
			{
				label: "Unrated",
				value: distribution.find((b) => b.rating === 0)?.count ?? 0,
				titleCount: distribution.find((b) => b.rating === 0)?.count ?? 0,
				averageRating: 0,
			},
		];
		for (let i = 1; i <= Math.round(10 / step); i++) {
			const rating = Number((i * step).toFixed(1));
			const titleCount =
				distribution.find((b) => Math.abs(b.rating - rating) < 0.001)?.count ??
				0;
			points.push({
				label: String(rating),
				value: titleCount,
				titleCount,
				averageRating: rating,
			});
		}
		return points;
	});
	const ratingsByTitle = $derived(
		new Map<string, number>(
			data.posters.map(
				(card) => [`${card.type}:${card.id}`, card.rating ?? 0] as const,
			),
		),
	);
	function averageForTitles(titleKeys: string[]) {
		return meanRating(
			[...new Set(titleKeys)].map((key) => ratingsByTitle.get(key) ?? 0),
		);
	}
	function pies(items: StatsPie[]): ChartPoint[] {
		const total = items.reduce((a, b) => a + b.count, 0);
		return items.map((p) => ({
			label: p.label,
			titleKeys: p.titleKeys ?? [],
			value: p.count,
			titleCount: new Set(p.titleKeys ?? []).size,
			averageRating: averageForTitles(p.titleKeys ?? []),
			detail: total
				? `${((p.count / total) * 100).toFixed(1)}%`
				: isGame
					? "No games"
					: "No watches",
		}));
	}
	const statusBreakdownTitle = $derived(
		lifetime ? "Current library" : "Status activity",
	);
	const statusBreakdownPoints: ChartPoint[] = $derived(
		(data.library?.statuses ?? [])
			.filter((group) => group.count > 0)
			.map((group) => ({
				label: group.label,
				value: group.count,
				titleCount: group.count,
				items: group.items,
				averageRating: meanRating(group.items.map((item) => item.rating ?? 0)),
			})),
	);
	function exploreStatusPoint(point: ChartPoint) {
		exploreSelection({
			label: point.label,
			items: point.items ?? [],
			description: lifetime
				? `currently ${point.label.toLowerCase()}`
				: `recorded as ${point.label.toLowerCase()}`,
			period: lifetime ? "Current library" : String(data.year),
		});
	}
	function runtimeDetail(card: StatsMediaCard): string {
		if (data.media !== "tv" || !lifetime) {
			return `${card.runtime} min${data.media === "tv" ? " per episode" : ""}`;
		}
		const runtime = card.runtime ?? 0;
		const hours = Math.floor(runtime / 60);
		const minutes = runtime % 60;
		if (hours === 0) return `${minutes} min total`;
		return `${hours} hr${hours === 1 ? "" : "s"}${
			minutes ? ` ${minutes} min` : ""
		} total`;
	}
	const highCards = $derived([
		{
			label: `Highest ${source} rating`,
			card: isGame
				? data.highsLows.highestCommunityRated
				: data.highsLows.highestTMDBRated,
			detail: (c: StatsMediaCard) =>
				`${(c.communityRating ?? c.tmdbRating)?.toFixed(1)}/10 on ${source}`,
		},
		{
			label: `Lowest ${source} rating`,
			card: data.highsLows.lowestRated,
			detail: (c: StatsMediaCard) =>
				`${(c.communityRating ?? c.tmdbRating)?.toFixed(1)}/10 on ${source}`,
		},
		{
			label: "Most popular",
			card: data.highsLows.mostVoted,
			detail: (c: StatsMediaCard) => `${c.voteCount?.toLocaleString()} votes`,
		},
		{
			label: "Most obscure",
			card: data.highsLows.leastVoted,
			detail: (c: StatsMediaCard) => `${c.voteCount?.toLocaleString()} votes`,
		},
		{
			label: "Newest",
			card: data.highsLows.newest,
			detail: (c: StatsMediaCard) => c.date ?? "",
		},
		{
			label: "Oldest",
			card: data.highsLows.oldest,
			detail: (c: StatsMediaCard) => c.date ?? "",
		},
		...(isGame
			? lifetime
				? [
						{
							label: "Most recorded playtime",
							card: data.highsLows.mostPlaytime,
							detail: (c: StatsMediaCard) =>
								`${c.playtimeHours?.toLocaleString()} hours`,
						},
						{
							label: "Least recorded playtime",
							card: data.highsLows.leastPlaytime,
							detail: (c: StatsMediaCard) =>
								`${c.playtimeHours?.toLocaleString()} hours`,
						},
					]
				: []
			: [
					{
						label: data.media === "tv" && lifetime ? "Longest show" : "Longest",
						card: data.highsLows.longest,
						detail: runtimeDetail,
					},
					{
						label:
							data.media === "tv" && lifetime ? "Shortest show" : "Shortest",
						card: data.highsLows.shortest,
						detail: runtimeDetail,
					},
				]),
	]);
</script>

{#snippet gameSection(
	section: "activity" | "categories" | "companies" | "playtime",
)}
	<StatsGameSections
		{data}
		{section}
		owner={publicOwner}
		{settings}
		onSelect={explorePoint}
		onExplore={explore}
		bind:counts={gameCounts}
		bind:sortBy={categorySort}
		bind:interval={activityMode}
		bind:activityKind={gameActivityKind}
	/>
{/snippet}

{#snippet episodeSection()}
	{#if data.media === "tv"}
		{@const episodes = data.highestRatedEpisodes?.[episodeTab] ?? []}
		<section data-stats-section="highest-rated-episodes">
			<div class="section-heading">
				<h2 class="norm">Highest rated episodes</h2>
				{#if lifetime}<span>Your favorites rated 9/10 or above</span>
				{:else}<StatsSegmentedControl label="Episode release year" wrap>
						<button
							class:active={episodeTab === "current"}
							aria-pressed={episodeTab === "current"}
							onclick={() => {
								episodeTab = "current";
								episodeCount = 5;
							}}>{data.year} releases</button
						>
						<button
							class:active={episodeTab === "older"}
							aria-pressed={episodeTab === "older"}
							onclick={() => {
								episodeTab = "older";
								episodeCount = 5;
							}}>Older</button
						>
						{#if data.highestRatedEpisodes?.unknown?.length}
							<button
								class:active={episodeTab === "unknown"}
								aria-pressed={episodeTab === "unknown"}
								onclick={() => {
									episodeTab = "unknown";
									episodeCount = 5;
								}}>Unknown release</button
							>
						{/if}
					</StatsSegmentedControl>{/if}
			</div>
			<StatsPosters
				items={episodes.slice(0, episodeCount)}
				owner={publicOwner}
				{settings}
			/>
			<StatsExpansion
				count={episodeCount}
				total={episodes.length}
				onChange={(value) => (episodeCount = value)}
			/>
		</section>
	{/if}
{/snippet}

{#snippet library_momentumSection()}
	{#if data.library}<StatsLibrarySections
			{data}
			section="library-momentum"
			owner={publicOwner}
			{settings}
			onSelect={exploreSelection}
			bind:waitingCount
		/>{/if}
{/snippet}

{#snippet library_waitingSection()}
	{#if data.library}<StatsLibrarySections
			{data}
			section="library-waiting"
			owner={publicOwner}
			{settings}
			onSelect={exploreSelection}
			bind:waitingCount
		/>{/if}
{/snippet}

{#snippet historySection()}
	{#if lifetime}<section data-stats-section="history">
			<div class="section-heading">
				<h2 class="norm" data-stats-jump="history" tabindex="-1">
					Through the years
				</h2>
				<span>{isGame ? "Games" : "Movies & TV"} · unique titles each year</span
				>
			</div>
			<div class="timeline-grid">
				{#each [...(isGame ? [{ key: "games", label: "Games played", color: "#29acf4" }, { key: "completed", label: "Games completed", color: "#51ad79" }] : [{ key: "movies", label: "Films watched", color: "#29acf4" }, { key: "shows", label: "Shows watched", color: "#51ad79" }]), { key: "averageRating", label: "Average rating", color: "#f5b85a" }, ...(data.reviewsVisible ? [{ key: "reviewed", label: "Reviewed", color: "#f47983" }] : [])] as metric (metric.key)}<div
					>
						<h3 class="norm">{metric.label}</h3>
						<StatsChart
							titleUnit={isGame ? "games" : "titles"}
							title={metric.label}
							kind={metric.key === "averageRating" ? "line" : "bar"}
							showData
							valueUnit={metric.key === "averageRating" && isGame
								? "games"
								: "titles"}
							color={metric.color}
							points={data.history.map((p) => ({
								label: String(p.year),
								tooltipLabel: `${metric.label} · ${p.year}`,
								items: historyItems(p, metric.key),
								browseValue:
									metric.key === "averageRating"
										? historyItems(p, metric.key).length
										: undefined,
								value:
									metric.key === "averageRating" && !p.averageRating
										? null
										: Number(p[metric.key as keyof typeof p] ?? 0),
								titleCount:
									metric.key === "averageRating"
										? historyItems(p, metric.key).length
										: Number(p[metric.key as keyof typeof p] ?? 0),
								averageRating: meanRating(
									historyItems(p, metric.key).map((item) => item.rating),
								),
							}))}
							{settings}
							onSelect={explorePoint}
						/>
					</div>{/each}
			</div>
		</section>{/if}
{/snippet}

{#snippet decadesSection()}
	{#if lifetime}<section data-stats-section="decades">
			<div class="section-heading">
				<h2 class="norm">Highest-rated decades</h2>
				<span>At least two rated titles</span>
			</div>
			<div class="decades">
				{#each data.decades as decade (decade.decade)}<div>
						<h3 class="norm">
							{decade.decade}s
							<span>{averageRating(decade.averageRating, settings)}</span>
						</h3>
						<p>{decade.titles} titles {isGame ? "played" : "watched"}</p>
						<StatsPosters
							items={decade.items.slice(0, decadeCounts[decade.decade] ?? 5)}
							owner={publicOwner}
							{settings}
							tiny
							fivePerRow
						/>
						{#if !decade.items.length}<p class="muted">
								No titles rated above 8/10 in this decade.
							</p>{/if}
						<StatsExpansion
							count={decadeCounts[decade.decade] ?? 5}
							total={decade.items.length}
							onChange={(value) => (decadeCounts[decade.decade] = value)}
						/>
					</div>{:else}<p class="muted">
						Watch and rate a few more titles to discover your favorite decades.
					</p>{/each}
			</div>
		</section>{/if}
{/snippet}

{#snippet highest_ratedSection()}
	<section data-stats-section="highest-rated">
		<div class="section-heading">
			<h2 class="norm">Highest rated {mediaLabel.toLowerCase()}</h2>
			{#if !lifetime}<StatsSegmentedControl>
					<button
						class:active={highestTab === "current"}
						aria-pressed={highestTab === "current"}
						onclick={() => {
							highestTab = "current";
							yearlyFavoriteCount = 5;
						}}>{data.year} releases</button
					><button
						class:active={highestTab === "older"}
						aria-pressed={highestTab === "older"}
						onclick={() => {
							highestTab = "older";
							yearlyFavoriteCount = 5;
						}}>Older</button
					>
				</StatsSegmentedControl>{:else}<span
					>Your favorites rated above 8/10</span
				>{/if}
		</div>
		<StatsPosters
			items={lifetime
				? data.highestRated.current.slice(0, favoriteCount)
				: yearlyHighlights.slice(0, yearlyFavoriteCount)}
			owner={publicOwner}
			{settings}
		/>
		{#if lifetime}<StatsExpansion
				count={favoriteCount}
				total={data.highestRated.current.length}
				onChange={(value) => (favoriteCount = value)}
			/>{/if}
		{#if !lifetime}<StatsExpansion
				count={yearlyFavoriteCount}
				total={yearlyHighlights.length}
				onChange={(value) => (yearlyFavoriteCount = value)}
			/>{/if}
	</section>
{/snippet}

{#snippet playtimeSection()}
	{#if isGame}{@render gameSection("playtime")}{/if}
{/snippet}

{#snippet activitySection()}
	{#if isGame}{@render gameSection("activity")}{/if}
	{#if !lifetime && !isGame}<section data-stats-section="activity">
			<div class="section-heading activity-heading">
				<h2 class="norm" data-stats-jump="activity" tabindex="-1">Activity</h2>
				<StatsSegmentedControl label="Activity metric">
					<button
						class:active={activityMetric === "count"}
						aria-pressed={activityMetric === "count"}
						onclick={() => (activityMetric = "count")}>Count</button
					>
					<button
						class:active={activityMetric === "rating"}
						aria-pressed={activityMetric === "rating"}
						onclick={() => (activityMetric = "rating")}>Rating</button
					>
				</StatsSegmentedControl>
				<StatsSegmentedControl label="Activity interval">
					<button
						class:active={activityMode === "week"}
						aria-pressed={activityMode === "week"}
						onclick={() => (activityMode = "week")}>Week</button
					>
					<button
						class:active={activityMode === "month"}
						aria-pressed={activityMode === "month"}
						onclick={() => (activityMode = "month")}>Month</button
					>
				</StatsSegmentedControl>
			</div>
			<div class="activity-averages">
				<span
					><strong>{data.activity.total.toLocaleString()}</strong>
					{data.media === "movie" ? "films" : "episodes"} watched</span
				>
				<span
					><strong>{decimal(data.activity.averagePerWeek)}</strong> / week</span
				><span
					><strong>{decimal(data.activity.averagePerMonth)}</strong> / month</span
				>
			</div>
			{#key `${activityMode}:${activityMetric}`}<StatsChart
					titleUnit={isGame ? "games" : "titles"}
					kind={activityMetric === "rating" ? "line" : "bar"}
					color={activityMetric === "rating" ? "#f5b85a" : "#29acf4"}
					title={activityMetric === "rating"
						? `Average rating by ${activityMode}`
						: activityMode === "week"
							? `${data.media === "tv" ? "Episodes" : "Watches"} by week`
							: `${data.media === "tv" ? "Episodes" : "Watches"} by month`}
					points={activityMode === "week"
						? data.activity.weeks.map((w) => ({
								label: w.start.slice(5),
								tooltipLabel: weekRange(w.start),
								value:
									activityMetric === "rating"
										? w.averageRating || null
										: w.plays,
								titleCount: w.uniqueTitles,
								browseValue: w.plays,
								averageRating: w.averageRating,
								items: w.items ?? [],
								detail: [
									`${w.plays} ${data.media === "tv" ? "episodes" : "watches"}`,
									...(data.media === "tv"
										? (w.items ?? []).map(
												(item) =>
													item.episodeName || "Episode name unavailable",
											)
										: w.titles),
								]
									.filter(Boolean)
									.join(" · "),
							}))
						: data.activity.months.map((m) => ({
								label: new Date(`${m.month}-01T00:00:00Z`).toLocaleDateString(
									undefined,
									{
										month: "short",
										timeZone: "UTC",
									},
								),
								tooltipLabel: new Date(
									`${m.month}-01T00:00:00Z`,
								).toLocaleDateString(undefined, {
									month: "long",
									year: "numeric",
									timeZone: "UTC",
								}),
								value:
									activityMetric === "rating"
										? m.averageRating || null
										: m.plays,
								titleCount: m.items?.length ?? 0,
								browseValue: m.plays,
								averageRating: m.averageRating,
								items: m.items ?? [],
								detail: `${m.plays} ${data.media === "tv" ? "episodes" : "films"}`,
							}))}
					{settings}
					alignFirstXAxisTick={true}
					valueUnit={data.media === "tv" ? "episodes" : "watches"}
					showData
					onSelect={explorePoint}
				/>{/key}
		</section>{/if}
{/snippet}

{#snippet calendarSection()}
	{#if data.library}<StatsCalendar
			{data}
			{settings}
			onSelect={exploreSelection}
			bind:calendarYear
			bind:activityKind={gameCalendarKind}
			bind:monthsExpanded={calendarMonthsExpanded}
		/>{/if}
{/snippet}

{#snippet milestonesSection()}
	{@const reviews = data.reviewsVisible ? data.reviewLengths : undefined}
	{@const hasReviews = !!(reviews?.shortest || reviews?.longest)}
	<section data-stats-section="milestones">
		<div class="section-heading">
			<h2 class="norm">Milestones</h2>
			<span>{lifetime ? "From the beginning" : `Bookends of ${data.year}`}</span
			>
		</div>
		<div class="milestones" class:with-reviews={hasReviews}>
			{#each [{ label: isGame ? "First recorded progress" : "First watch", card: data.milestones.first, wordCount: undefined }, { label: isGame ? "Last recorded progress" : "Last watch", card: data.milestones.last, wordCount: undefined }, ...(hasReviews ? [{ label: "Shortest review", card: reviews?.shortest?.item, wordCount: reviews?.shortest?.wordCount }, { label: "Longest review", card: reviews?.longest?.item, wordCount: reviews?.longest?.wordCount }] : [])] as milestone (milestone.label)}<div
				>
					<h3 class="norm">{milestone.label}</h3>
					<StatsPosters
						items={milestone.card ? [milestone.card] : []}
						owner={publicOwner}
						{settings}
						detail={(c) =>
							milestone.wordCount === undefined
								? (c.date ?? "")
								: `${milestone.wordCount.toLocaleString()} ${milestone.wordCount === 1 ? "word" : "words"}`}
					/>
				</div>{/each}
		</div>
		{#if reviews && !hasReviews}
			<p class="muted">No written reviews for this view.</p>
		{/if}
		{#if data.milestones.mostWatched.length}<h3 class="subheading norm">
				{isGame ? "Most replayed" : "Most watched"}
			</h3>
			<StatsPosters
				items={data.milestones.mostWatched.slice(0, mostWatchedCount)}
				owner={publicOwner}
				{settings}
				detail={(c) => `${c.plays} ${isGame ? "completions" : "watches"}`}
			/>
			<StatsExpansion
				count={mostWatchedCount}
				total={data.milestones.mostWatched.length}
				onChange={(value) => (mostWatchedCount = value)}
			/>
		{/if}
	</section>
{/snippet}

{#snippet categoriesSection()}
	{#if isGame}{@render gameSection("categories")}{:else}
		<section data-stats-section="categories">
			<div class="section-heading">
				<h2 class="norm">Genres, countries & languages</h2>
				<div class="category-controls">
					<StatsSegmentedControl role="group" label="Sort categories by">
						<button
							class:active={categorySort === "count"}
							aria-pressed={categorySort === "count"}
							onclick={() => (categorySort = "count")}>Most watched</button
						><button
							class:active={categorySort === "rating"}
							aria-pressed={categorySort === "rating"}
							onclick={() => (categorySort = "rating")}>Highest rated</button
						>
					</StatsSegmentedControl>
				</div>
			</div>
			<div class="categories">
				<StatsRankedChart
					title="Genres"
					bind:count={categoryCounts.genres}
					items={data.genres}
					{settings}
					sortBy={categorySort}
					onSelect={(item) => explore(item.label, item.titleKeys)}
				/>
				<StatsRankedChart
					title="Countries"
					bind:count={categoryCounts.countries}
					items={data.countries}
					color="#51ad79"
					{settings}
					sortBy={categorySort}
					onSelect={(item) => explore(item.label, item.titleKeys)}
				/>
				<StatsRankedChart
					title="Languages"
					bind:count={categoryCounts.languages}
					items={data.languages}
					color="#d9aa64"
					{settings}
					sortBy={categorySort}
					onSelect={(item) => explore(item.label, item.titleKeys)}
				/>
			</div>
		</section>
	{/if}
{/snippet}

{#snippet breakdownSection()}
	<section data-stats-section="breakdown">
		<div class="section-heading">
			<h2 class="norm" data-stats-jump="breakdown" tabindex="-1">Breakdown</h2>
			<span>Patterns in your {isGame ? "gaming" : "viewing"}</span>
		</div>
		<div class="pies">
			{#each [...(!lifetime ? [{ title: "Release years", points: pies(data.breakdown.release) }] : []), { title: isGame ? "Completions & replays" : "Watches & rewatches", points: pies(data.breakdown.plays) }, ...(data.library ? [{ title: statusBreakdownTitle, points: statusBreakdownPoints }] : []), ...(data.games ? [{ title: "Completion", points: pies(data.games.completion) }] : []), ...(data.reviewsVisible ? [{ title: "Reviews", points: pies(data.breakdown.reviews ?? []) }] : [])] as group (group.title)}<div
					class="pie"
				>
					<h3
						class="norm"
						data-stats-jump={group.title === "Completions & replays" ||
						group.title === "Watches & rewatches"
							? "plays"
							: group.title === "Release years"
								? "release"
								: group.title === "Completion"
									? "completion"
									: undefined}
						tabindex="-1"
					>
						{group.title}
					</h3>
					<StatsChart
						titleUnit={isGame ? "games" : "titles"}
						title={group.title}
						kind="pie"
						height={150}
						points={group.points}
						{settings}
						onSelect={group.title === statusBreakdownTitle
							? exploreStatusPoint
							: explorePoint}
					/>
					<div class="pie-legend">
						{#each group.points as item, i (item.label)}<button
								class="plain"
								onclick={() =>
									group.title === statusBreakdownTitle
										? exploreStatusPoint(item)
										: explorePoint(item)}
								><i
									style={`background:${["#29acf4", "#f5b85a", "#f47983", "#51ad79", "#b19bea"][i % 5]}`}
								></i>{item.label}<b>{item.value}</b></button
							>{/each}
					</div>
				</div>{/each}
		</div>
		<div class="rating-heading">
			<h3 class="norm" data-stats-jump="ratings" tabindex="-1">
				Rating distribution
			</h3>
		</div>
		<StatsChart
			titleUnit={isGame ? "games" : "titles"}
			title="Rating distribution"
			points={ratingPoints.map((p) => ({
				...p,
				tooltipLabel: p.label === "Unrated" ? "Unrated" : `Rated ${p.label}/10`,
				items: data.posters.filter(
					(c) =>
						(c.rating ?? 0) === (p.label === "Unrated" ? 0 : Number(p.label)),
				),
			}))}
			{settings}
			xAxisTicks={[
				"Unrated",
				"1",
				"2",
				"3",
				"4",
				"5",
				"6",
				"7",
				"8",
				"9",
				"10",
			]}
			onSelect={explorePoint}
			color="#f5b85a"
		/>
		<button
			class="plain watchlist-summary"
			disabled={!watchlistTitles.length}
			aria-haspopup="dialog"
			aria-label={`Show ${data.breakdown.watchlistAdditions} titles added to your ${isGame ? "backlog" : "watchlist"}`}
			onclick={() =>
				(selection = {
					label: isGame ? "Added to backlog" : "Added to watchlist",
					items: watchlistTitles,
					description: isGame ? "added to backlog" : "added to watchlist",
				})}
		>
			<strong>{data.breakdown.watchlistAdditions.toLocaleString()}</strong>
			<span class="watchlist-copy">
				<span class="watchlist-title"
					>Added to {isGame ? "backlog" : "watchlist"}</span
				>
				<span class="watchlist-period">
					{lifetime ? "Across your recorded history" : `In ${data.year}`} · distinct
					titles
				</span>
			</span>
		</button>
	</section>
{/snippet}

{#snippet highs_lowsSection()}
	<section data-stats-section="highs-lows">
		<div class="section-heading">
			<h2 class="norm" data-stats-jump="runtimes" tabindex="-1">
				Highs and lows
			</h2>
			<span
				>By {source} ratings, votes, release details{isGame
					? ""
					: " & runtime"}</span
			>
		</div>
		<div class="highs-lows">
			{#each highCards as item (item.label)}<div>
					<h3 class="norm">{item.label}</h3>
					<StatsPosters
						items={item.card ? [item.card] : []}
						owner={publicOwner}
						{settings}
						detail={item.detail}
					/>
				</div>{/each}
		</div>
	</section>
{/snippet}

{#snippet rated_higherSection()}
	<section data-stats-section="rated-higher">
		<div class="section-heading">
			<h2 class="norm" data-stats-jump="rated-higher" tabindex="-1">
				Rated higher than average
			</h2>
			<span>You vs {source} · /10 · At least +1 point</span>
		</div>
		<StatsPosters
			items={higher.slice(0, higherCount)}
			owner={publicOwner}
			{settings}
			comparison
		/><StatsExpansion
			count={higherCount}
			total={higher.length}
			onChange={(value) => (higherCount = value)}
		/>
	</section>
{/snippet}

{#snippet rated_lowerSection()}
	<section data-stats-section="rated-lower">
		<div class="section-heading">
			<h2 class="norm">Rated lower than average</h2>
			<span>You vs {source} · /10 · At least −1 point</span>
		</div>
		<StatsPosters
			items={lower.slice(0, lowerCount)}
			owner={publicOwner}
			{settings}
			comparison
		/><StatsExpansion
			count={lowerCount}
			total={lower.length}
			onChange={(value) => (lowerCount = value)}
		/>
	</section>
{/snippet}

{#snippet titlesSection()}
	<section data-stats-section="titles">
		<div class="section-heading">
			<h2 class="norm" data-stats-jump="titles" tabindex="-1">
				{mediaLabel}
				{isGame ? "played" : "watched"}
				{lifetime ? "so far" : `in ${data.year}`}
			</h2>
			<span>{data.posters.length} distinct titles</span>
		</div>
		<StatsPosters
			items={data.posters}
			owner={publicOwner}
			{settings}
			tiny
			wall
		/>
	</section>
{/snippet}

{#snippet watchlistSection()}
	<section data-stats-section="watchlist">
		<div class="section-heading">
			<h2 class="norm">
				Highly rated, {isGame ? "yet to play" : "yet to see"}
			</h2>
			<span
				>From {publicOwner ? `${data.owner.username}'s` : "your"}
				{isGame ? "backlog" : "watchlist"}</span
			>
		</div>
		<StatsPosters
			items={data.watchlist}
			owner={publicOwner}
			{settings}
			detail={(c) =>
				`${(c.communityRating ?? c.tmdbRating)?.toFixed(1) ?? "—"}/10 on ${source}`}
		/>
	</section>
{/snippet}

{#snippet peopleSection()}
	{#if !isGame}<section data-stats-section="people">
			<div class="section-heading">
				<h2 class="norm">
					{data.media === "tv"
						? "People behind the shows"
						: "People behind the films"}
				</h2>
				<StatsSegmentedControl>
					<button
						class:active={peopleMode === "most"}
						aria-pressed={peopleMode === "most"}
						onclick={() => (peopleMode = "most")}>Most watched</button
					><button
						class:active={peopleMode === "rating"}
						aria-pressed={peopleMode === "rating"}
						onclick={() => (peopleMode = "rating")}>Highest rated</button
					>
				</StatsSegmentedControl>
			</div>
			<div class="people-sections">
				<StatsPeople
					title="Cast"
					bind:count={peopleCounts.cast}
					people={data.people.cast}
					expansionRows={peopleExpansionRows}
					unit={data.media === "tv" ? "episodes" : "titles"}
					mode={peopleMode}
					onSelect={(p) => explore(p.name, p.titleKeys, p.id)}
					{settings}
				/><StatsPeople
					title="Directors & creators"
					bind:count={peopleCounts.directors}
					people={data.people.directors}
					expansionRows={peopleExpansionRows}
					mode={peopleMode}
					onSelect={(p) => explore(p.name, p.titleKeys, p.id)}
					{settings}
				/><StatsPeople
					title="Studios"
					bind:count={peopleCounts.studios}
					people={data.studios}
					mode={peopleMode}
					onSelect={(p) => explore(p.name, p.titleKeys)}
					{settings}
					studios
				/>
			</div>
			<h3 class="subheading norm">Crew by department</h3>
			<div class="crew">
				{#each data.crew as department (department.department)}<details>
						<summary
							>{department.department}<span>{department.jobs.length} roles</span
							></summary
						>{#each department.jobs as job (job.job)}{@const remaining =
								job.people.filter(
									(p) => peopleMode !== "rating" || p.averageRating > 0,
								).length -
								(crewCounts[`${department.department}:${job.job}`] ?? 5)}
							<details class="job">
								<summary>{job.job}</summary>
								<ul>
									{#each [...job.people]
										.filter((p) => peopleMode !== "rating" || p.averageRating > 0)
										.sort( (a, b) => (peopleMode === "rating" ? b.averageRating - a.averageRating : b.titles - a.titles) )
										.slice(0, crewCounts[`${department.department}:${job.job}`] ?? 5) as person (person.id)}<li
										>
											<button
												class="plain crew-person"
												onclick={() =>
													explore(person.name, person.titleKeys, person.id)}
												>{person.name}</button
											><span
												>{person.titles} titles · {averageRating(
													person.averageRating,
													settings,
												)}</span
											>
										</li>{/each}
								</ul>

								<StatsExpansion
									count={crewCounts[`${department.department}:${job.job}`] ?? 5}
									total={remaining +
										(crewCounts[`${department.department}:${job.job}`] ?? 5)}
									onChange={(value) =>
										(crewCounts[`${department.department}:${job.job}`] = value)}
								/>
							</details>{/each}
					</details>{:else}<p class="muted">
						No recurring crew credits yet.
					</p>{/each}
			</div>
		</section>{/if}
{/snippet}

{#snippet companiesSection()}
	{#if isGame}{@render gameSection("companies")}{/if}
{/snippet}

<svelte:head
	><title
		>{data.owner.username ? `${data.owner.username} · ` : ""}{period} stats · Watcharr</title
	></svelte:head
>

<svelte:window bind:scrollY bind:innerHeight={viewportHeight} />

<div
	bind:this={root}
	class="stats-page"
	class:background-off={!statsBackground.enabled}
>
	{#if scrollY > Math.max(600, viewportHeight) && !selection}
		<button
			class="back-to-top"
			type="button"
			aria-label="Back to top"
			title="Back to top"
			onclick={backToTop}
		>
			<span aria-hidden="true"><Icon i="arrow" facing="up" wh={20} /></span>
		</button>
	{/if}
	<header>
		<div class="heading">
			<div class="heading-top">
				<div class="heading-copy">
					<a
						class="back"
						aria-label={data.owner.username
							? `Back to ${data.owner.username}'s library`
							: "Back to your library"}
						href={publicOwner
							? resolve("/(public)/lists/[id]/[username]", publicOwner)
							: resolve("/")}
					>
						<span class="back-arrow" aria-hidden="true">←</span>
						<span class="library-label"
							>{data.owner.username
								? `${data.owner.username}'s list`
								: "Your list"}</span
						></a
					>
				</div>
				<h1 class="norm" class:lifetime-heading={selectedYear === "all"}>
					{selectedYear === "all" ? "Lifetime" : selectedYear}
					<span
						>{selectedMedia === "game"
							? "Games"
							: selectedMedia === "tv"
								? "Television"
								: "Film"}</span
					>
				</h1>
				<div class="controls">
					<StatsSegmentedControl label="Media" role="group">
						<button
							data-stats-control="movie"
							disabled={editingLayout}
							class:active={selectedMedia === "movie"}
							aria-pressed={selectedMedia === "movie"}
							onclick={() => onSelectionChange(selectedYear, "movie")}
							>Movies</button
						><button
							data-stats-control="tv"
							disabled={editingLayout}
							class:active={selectedMedia === "tv"}
							aria-pressed={selectedMedia === "tv"}
							onclick={() => onSelectionChange(selectedYear, "tv")}>TV</button
						><button
							data-stats-control="game"
							disabled={editingLayout}
							class:active={selectedMedia === "game"}
							aria-pressed={selectedMedia === "game"}
							onclick={() => onSelectionChange(selectedYear, "game")}
							>Games</button
						>
					</StatsSegmentedControl>
					<label
						><span class="control-label">Period</span><select
							data-stats-control="period"
							disabled={editingLayout}
							value={selectedYear}
							onchange={(e) =>
								onSelectionChange(e.currentTarget.value, selectedMedia)}
							><option value="all">Lifetime</option
							>{#each years as y (y)}<option value={String(y)}>{y}</option
								>{/each}</select
						></label
					>
				</div>
			</div>
			<div
				class="header-summary"
				role="group"
				aria-label="Media summary"
				aria-busy={loading}
			>
				<button
					type="button"
					class="plain summary-tile"
					title={summaryDestinations.titles.label}
					disabled={loading || !!error}
					onclick={() => jumpToSummary("titles")}
				>
					<span class="summary-label">
						{selectedMedia === "game"
							? "Games played"
							: selectedMedia === "tv"
								? "Shows watched"
								: "Movies watched"}
					</span>
					<span class="summary-value"
						>{loading || error
							? "—"
							: primarySummaryCount.toLocaleString()}</span
					>
				</button>
				<button
					type="button"
					class="plain summary-tile"
					title={selectedMedia === "movie" && selectedYear === "all"
						? "Select a year to see new releases."
						: summaryDestinations.secondary.label}
					disabled={loading ||
						!!error ||
						(selectedMedia === "movie" && selectedYear === "all")}
					onclick={() => jumpToSummary("secondary")}
				>
					<span class="summary-label">
						{selectedMedia === "game"
							? "Games completed"
							: selectedMedia === "tv"
								? "Episodes watched"
								: "New releases"}
					</span>
					<span class="summary-value">
						{loading || error
							? "—"
							: selectedMedia === "movie"
								? (newReleaseCount?.toLocaleString() ?? "—")
								: (isGame
										? (data.summary.completed ?? 0)
										: data.media === "tv"
											? data.activity.total
											: data.summary.plays
									).toLocaleString()}
					</span>
				</button>
				<button
					type="button"
					class="plain summary-tile"
					title={`${hoursDescription} ${summaryDestinations.hours.label}`}
					disabled={loading || !!error}
					onclick={() => jumpToSummary("hours")}
				>
					<span class="summary-label"
						>{isGame ? "Recorded hours" : "Estimated hours"}</span
					>
					<span class="summary-value">
						{loading || error || hoursPlayed === undefined
							? "—"
							: decimal(hoursPlayed)}
					</span>
				</button>
				<button
					type="button"
					class="plain summary-tile"
					title={summaryDestinations.rating.label}
					disabled={loading || !!error}
					onclick={() => jumpToSummary("rating")}
				>
					<span class="summary-label">Average rating</span>
					<span class="summary-value">
						{loading || error
							? "—"
							: averageRating(data.summary.averageRating, settings)}
					</span>
				</button>
			</div>
		</div>
	</header>
	{#if !publicOwner}
		<div class="layout-controls">
			<StatsLayoutEditor
				{data}
				disabled={loading || !!error}
				bind:editing={editingLayout}
				onSave={onSaveLayout}
			/>
		</div>
	{/if}
	{#if loading}<div class="load-status" role="status">
			<span>{initialLoading ? "Loading stats…" : "Updating stats…"}</span>
		</div>
	{:else if error}<div class="load-status">
			<Error {error} pretty="Unable to load these stats." />
		</div>{/if}
	{#key statsView}<div
			class="stats-content"
			class:updating={loading}
			class:failed={!!error}
			inert={loading || !!error}
			aria-busy={loading}
		>
			{#if data.metadata.partial}<details class="coverage">
					<summary
						>Some details are unavailable for {data.metadata.failedTitles
							.length} titles</summary
					>
					<p>
						Your recorded {isGame ? "progress" : "watches"} and ratings are included.
						Metadata charts may be incomplete.
					</p>
					<ul>
						{#each data.metadata.failedTitles as title (title)}<li>
								{title}
							</li>{/each}
					</ul>
				</details>{/if}

			{#if !initialLoading && !data.summary.titles && !data.activity.total}<div
					class="empty-year"
				>
					<h2 class="norm">A fresh page in your journal</h2>
					<p>
						No recorded {isGame
							? "game progress"
							: data.media === "movie"
								? "movie watches"
								: "episode watches"} for
						{lifetime ? "this lifetime view" : data.year}. Choose another year
						or explore your {isGame ? "backlog" : "watchlist"} below.
					</p>
				</div>{/if}

			{#each orderedSections as sectionId (sectionId)}
				{#if sectionId === "library-momentum"}
					{@render library_momentumSection()}
				{:else if sectionId === "library-waiting"}
					{@render library_waitingSection()}
				{:else if sectionId === "history"}
					{@render historySection()}
				{:else if sectionId === "decades"}
					{@render decadesSection()}
				{:else if sectionId === "highest-rated"}
					{@render highest_ratedSection()}
				{:else if sectionId === "highest-rated-episodes"}
					{@render episodeSection()}
				{:else if sectionId === "playtime"}
					{@render playtimeSection()}
				{:else if sectionId === "activity"}
					{@render activitySection()}
				{:else if sectionId === "calendar"}
					{@render calendarSection()}
				{:else if sectionId === "milestones"}
					{@render milestonesSection()}
				{:else if sectionId === "categories"}
					{@render categoriesSection()}
				{:else if sectionId === "breakdown"}
					{@render breakdownSection()}
				{:else if sectionId === "companies"}
					{@render companiesSection()}
				{:else if sectionId === "people"}
					{@render peopleSection()}
				{:else if sectionId === "highs-lows"}
					{@render highs_lowsSection()}
				{:else if sectionId === "rated-higher"}
					{@render rated_higherSection()}
				{:else if sectionId === "rated-lower"}
					{@render rated_lowerSection()}
				{:else if sectionId === "titles"}
					{@render titlesSection()}
				{:else if sectionId === "watchlist"}
					{@render watchlistSection()}
				{/if}
			{/each}
			{#if selection}{#key selection}<StatsTitlesDialog
						bind:this={titlesDialog}
						initialState={dialogState}
						{...selection}
						period={selection.period ?? period}
						owner={publicOwner}
						{settings}
						expansionRows={titleDialogExpansionRows}
						onClose={() => {
							selection = undefined;
							dialogState = undefined;
						}}
					/>{/key}{/if}
			<footer>
				{isGame
					? "Based on recorded starts, non-planned status changes and completions. Legacy games without progress history use their saved creation date."
					: data.media === "tv"
						? "TV activity counts episode finishes; show statistics use recorded whole-show watches."
						: "Based on recorded whole-title watches."} Past years use current saved
				ratings{data.reviewsVisible ? " and reviews" : ""}. Dates use UTC. {isGame
					? lifetime
						? "Game metadata from IGDB. Playtime shows current saved totals."
						: "Game metadata from IGDB. Yearly recorded hours are current saved totals for games first completed in that year."
					: "Movie and TV metadata from TMDB."}
			</footer>
		</div>{/key}
</div>

<style>
	.back-to-top {
		position: fixed;
		right: calc(20px + env(safe-area-inset-right, 0px));
		bottom: calc(24px + env(safe-area-inset-bottom, 0px));
		z-index: 20;
		display: grid;
		place-items: center;
		width: 44px;
		height: 44px;
		padding: 0;
		border: 1px solid var(--stats-border);
		border-radius: 50%;
		background: var(--stats-surface);
		color: var(--stats-muted);
		box-shadow: 0 2px 8px #0002;
		cursor: pointer;
		touch-action: manipulation;
	}
	.back-to-top span {
		display: flex;
	}
	@media (hover: hover) {
		.back-to-top:hover {
			color: var(--stats-accent);
			border-color: var(--stats-accent);
		}
	}
	:global(:root.theme-dark) .stats-page {
		--stats-accent: #29acf4;
	}
	.stats-page {
		--stats-accent: #086fa8;
		max-width: none;
		width: 100%;
		margin: 0;
		padding: 24px 32px 60px;
		box-sizing: border-box;
		min-width: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--stats-text);
		--stats-text: color-mix(in srgb, var(--text-color) 90%, var(--bg-color));
		--stats-muted: color-mix(in srgb, var(--text-color) 67%, var(--bg-color));
		--stats-border: color-mix(in srgb, var(--text-color) 14%, var(--bg-color));
		--stats-surface: color-mix(in srgb, var(--bg-color) 96%, var(--text-color));
		background-image:
			radial-gradient(
				circle,
				color-mix(in srgb, var(--text-color) 14%, transparent) 1px,
				transparent 1.2px
			),
			linear-gradient(
				90deg,
				color-mix(in srgb, var(--stats-accent) 5%, var(--bg-color)) 0%,
				color-mix(in srgb, var(--stats-accent) 1.5%, var(--bg-color)) 18%,
				var(--bg-color) 42%,
				var(--bg-color) 58%,
				color-mix(in srgb, var(--stats-accent) 1.5%, var(--bg-color)) 82%,
				color-mix(in srgb, var(--stats-accent) 5%, var(--bg-color)) 100%
			);
		background-size:
			24px 24px,
			100% 100%;
		background-repeat: repeat, no-repeat;
	}
	.stats-page.background-off {
		background-image: none;
	}
	.stats-page > :is(header, .stats-content, .load-status, .layout-controls) {
		max-width: 1050px;
		width: 100%;
		margin-right: auto;
		margin-left: auto;
	}
	.stats-page .stats-content > :global(section) {
		padding: 22px 32px 30px;
		margin-inline: -32px;
		margin-top: 14px;
		border-top: 0;
		border-radius: 10px;
		background: color-mix(in srgb, var(--text-color) 2%, var(--bg-color));
		box-shadow: inset 0 0 0 1px
			color-mix(in srgb, var(--text-color) 8%, var(--bg-color));
	}
	header {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		align-items: center;
		padding-bottom: 16px;
	}
	.heading {
		min-width: 0;
		flex: 1;
	}
	.heading-top {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, auto) minmax(0, 1fr);
		align-items: center;
		gap: 16px;
	}
	.heading-copy {
		max-width: 180px;
		min-width: 0;
	}
	.header-summary {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 10px;
		margin: 16px 0 0;
		width: 100%;
		max-width: none;
	}
	.header-summary > button {
		display: flex;
		flex-direction: column-reverse;
		justify-content: end;
		gap: 4px;
		min-width: 0;
		padding: 12px 16px;
		border: 1px solid var(--stats-border);
		border-radius: 10px;
		background: var(--stats-surface);
		text-align: left;
		font: inherit;
		width: 100%;
		box-shadow: none;
	}
	.header-summary > button:not(:disabled):hover {
		border-color: var(--stats-accent);
		background: color-mix(
			in srgb,
			var(--stats-accent) 6%,
			var(--stats-surface)
		);
	}
	.header-summary > button:disabled {
		cursor: default;
	}
	.header-summary .summary-label {
		color: var(--stats-muted);
		font-size: 13px;
		line-height: 1.3;
		font-weight: 500;
	}
	.header-summary .summary-value {
		margin: 0;
		color: var(--stats-accent);
		font-size: clamp(22px, 3vw, 30px);
		line-height: 1.1;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.load-status {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 0;
		font-size: 14px;
		color: var(--stats-muted);
	}
	.stats-content.updating {
		opacity: 0.4;
	}
	.stats-content.failed {
		display: none;
	}
	.back {
		display: inline-flex;
		align-items: start;
		gap: 6px;
		font-size: 16px;
		line-height: 1.35;
		font-weight: 600;
		color: var(--stats-text);
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	:global(:root.theme-dark) .back {
		color: #fff;
	}
	.back-arrow {
		flex-shrink: 0;
	}
	.library-label {
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	h1 {
		font-size: clamp(32px, 4vw, 44px);
		text-align: center;
		line-height: 1.2;
		margin: 0;
		font-weight: 700;
		letter-spacing: -0.035em;
		overflow-wrap: anywhere;
	}
	h1 span {
		color: var(--stats-muted);
		font-weight: 500;
		letter-spacing: -0.02em;
	}
	.controls {
		display: contents;
	}
	.controls label {
		width: 116px;
		grid-column: 3;
		grid-row: 1;
		justify-self: end;
	}
	.control-label {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	.controls > :global(.segmented) {
		min-width: 0;
		grid-column: 1 / -1;
		grid-row: 2;
		width: 100%;
	}
	.controls :global(button) {
		flex: 1 1 auto;
		min-width: 0;
		min-height: 44px;
		padding: 8px 12px;
	}
	select {
		min-width: 0;
		min-height: 44px;
		padding: 9px 36px 9px 12px;
		font-size: 14px;
		width: 100%;
		appearance: none;
		color-scheme: dark;
		color: #f3f7fa;
		background-color: #17232d;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'%3E%3Cpath fill='none' stroke='%2329acf4' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.8' d='m3.5 6 4.5 4 4.5-4'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
		background-size: 14px;
		border: 1px solid color-mix(in srgb, var(--stats-accent) 45%, #344450);
		border-radius: 7px;
		font-weight: 600;
		cursor: pointer;
		transition:
			border-color 140ms ease,
			background-color 140ms ease;
	}
	select:hover {
		border-color: var(--stats-accent);
		background-color: #1d2d39;
	}
	select:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 2px;
	}
	select option {
		color: #f3f7fa;
		background: #17232d;
	}
	section,
	:global(section.stats-game-section) {
		padding: 26px 0 30px;
		border-top: 1px solid #8883;
	}
	.category-controls {
		display: flex;
		align-items: center;
		justify-content: end;
		gap: 10px;
		flex-wrap: wrap;
	}
	.category-controls > span {
		font-size: 13px;
		color: var(--stats-muted);
	}
	h3 {
		font-size: 16px;
		font-weight: 600;
		margin: 0 0 14px;
	}
	.subheading {
		margin-top: 26px;
	}
	.timeline-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 24px;
	}
	.decades {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 24px;
	}
	.decades h3 {
		display: flex;
		justify-content: space-between;
		font-size: 24px;
		align-items: center;
	}
	.decades h3 span {
		color: var(--stats-accent, #29acf4);
		font-size: 18px;
	}
	.decades p {
		font-size: 13px;
		color: var(--stats-muted);
		margin: 0 0 12px;
	}
	summary {
		cursor: pointer;
		font-size: 14px;
	}
	.milestones {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 12px;
	}
	.milestones > :last-child {
		grid-column: -2 / -1;
	}
	.milestones.with-reviews {
		grid-template-columns: repeat(4, minmax(0, 1fr));
	}
	.milestones :global(.posters:not(.tiny):not(.wall):not(.episodes)) {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
	}
	.milestones :global(.posters:not(.tiny):not(.wall):not(.episodes) a) {
		width: 100%;
	}
	.categories {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 24px;
	}
	.pies {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 24px;
	}
	.pie h3 {
		text-align: center;
	}
	.pie-legend {
		display: grid;
		gap: 6px;
		font-size: 13px;
		max-width: 220px;
		margin: 12px auto 0;
	}
	.pie-legend button {
		color: inherit;
		text-align: left;
		display: flex;
		align-items: center;
		gap: 7px;
	}
	.pie-legend b {
		margin-left: auto;
	}
	.pie-legend i {
		height: 7px;
		width: 7px;
		border-radius: 50%;
	}
	.rating-heading {
		display: flex;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 10px;
		align-items: center;
		margin: 28px 0 8px;
	}
	.rating-heading h3 {
		margin: 0;
	}
	.fine-print {
		font-size: 13px;
		color: var(--stats-muted);
		margin-top: 10px;
	}
	.people-sections {
		display: grid;
		gap: 32px;
	}
	.crew {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 8px 18px;
	}
	.crew > details {
		border: 1px solid #8882;
		border-radius: 6px;
		padding: 12px;
		align-self: start;
		min-width: 0;
	}
	.crew > details > summary {
		font-size: 13px;
	}
	.crew summary span {
		float: right;
		font-size: 13px;
		color: var(--stats-muted);
	}
	.job {
		margin: 14px 0 0;
		padding-left: 8px;
	}
	.crew ul {
		list-style: none;
		padding: 0;
		margin: 12px 0;
		display: grid;
		gap: 9px;
	}
	.crew li {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 4px;
		font-size: 14px;
	}
	.crew-person {
		text-align: left;
		color: inherit;
		overflow-wrap: anywhere;
	}
	.crew li span {
		font-size: 13px;
		color: var(--stats-muted);
	}
	.highs-lows {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 28px 20px;
	}
	.highs-lows :global(.posters:not(.tiny):not(.wall):not(.episodes)) {
		display: grid;
		grid-template-columns: minmax(0, 1fr);
	}
	.highs-lows :global(.posters:not(.tiny):not(.wall):not(.episodes) a) {
		width: 100%;
	}
	.coverage {
		padding: 14px;
		background: #f5b85a15;
		border-radius: 8px;
		margin-bottom: 24px;
	}
	.coverage p,
	.coverage li {
		font-size: 14px;
	}
	.coverage ul {
		max-height: 160px;
		overflow: auto;
	}
	.empty-year {
		background: #51ad7912;
		border-radius: 10px;
		padding: 24px;
		margin-bottom: 20px;
	}
	.empty-year p,
	.muted {
		font-size: 13px;
		color: var(--stats-muted);
	}
	footer {
		border-top: 1px solid #8883;
		padding-top: 20px;
		font-size: 13px;
		color: var(--stats-muted);
		line-height: 1.7;
	}
	.watchlist-summary {
		display: flex;
		width: 100%;
		box-sizing: border-box;
		align-items: center;
		gap: 20px;
		padding: 20px 24px;
		margin-top: 28px;
		border: 1px solid var(--stats-border);
		border-radius: 12px;
		background: color-mix(in srgb, #29acf4 7%, var(--stats-surface));
		color: inherit;
		font: inherit;
		text-align: left;
		box-shadow: none;
		cursor: pointer;
		transition:
			border-color 0.15s,
			background-color 0.15s;
	}
	.watchlist-summary:not(:disabled):hover {
		border-color: var(--stats-accent);
		background: color-mix(in srgb, #29acf4 12%, var(--stats-surface));
	}
	.watchlist-summary:disabled {
		cursor: default;
		opacity: 1;
	}
	.watchlist-summary > strong {
		font-size: 40px;
		line-height: 1;
		color: var(--stats-accent);
	}
	.watchlist-title {
		display: block;
		font-size: 18px;
		font-weight: 600;
		margin-bottom: 4px;
	}
	.watchlist-period {
		display: block;
		font-size: 13px;
		color: var(--stats-muted);
	}
	.back:hover,
	.crew-person:hover {
		color: var(--stats-accent);
		text-decoration: underline;
	}
	button:focus-visible,
	summary:focus-visible,
	a:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.crew summary:hover {
		color: var(--stats-accent);
	}
	@media (min-width: 521px) and (max-width: 900px) {
		.milestones {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
	}
	@media (max-width: 700px) {
		.stats-page .stats-content > :global(section) {
			margin-inline: -24px;
			padding-inline: 24px;
		}
		header {
			flex-direction: column;
			align-items: stretch;
			gap: 20px;
		}
		.heading-top {
			grid-template-columns: 80px minmax(0, 1fr) 104px;
			gap: 12px 6px;
		}
		.library-label {
			white-space: normal;
			overflow: visible;
		}
		h1 {
			font-size: clamp(32px, 8.7vw, 34px);
			text-align: center;
		}
		h1.lifetime-heading {
			font-size: clamp(20px, 6vw, 28px);
		}
		h1 span {
			display: block;
			font-size: 18px;
			letter-spacing: 0;
		}
		.controls label {
			width: 104px;
		}
		select {
			padding-inline: 10px 28px;
			background-position: right 8px center;
		}
		.controls :global(button) {
			padding-inline: 8px;
			font-size: 13px;
		}
		.header-summary {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 8px;
			margin-top: 12px;
		}
		.header-summary > button {
			padding: 10px 12px;
		}
		.header-summary .summary-label {
			font-size: 12px;
		}
		.category-controls {
			justify-content: start;
		}
		.categories {
			grid-template-columns: 1fr;
			gap: 28px;
		}
		.decades {
			gap: 14px;
		}
		.highs-lows {
			gap: 24px 12px;
		}
	}
	@media (max-width: 520px) {
		.stats-page {
			padding: 20px 16px 40px;
		}
		header {
			flex-direction: column;
			align-items: stretch;
			gap: 18px;
		}
		.header-summary {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		select {
			min-width: 0;
		}
		.stats-page .stats-content > :global(section) {
			margin-inline: -12px;
			padding-inline: 12px;
		}
		.timeline-grid,
		.crew {
			grid-template-columns: 1fr;
		}
		.decades {
			grid-template-columns: 1fr;
			gap: 24px;
		}
		.highs-lows {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.pies {
			grid-template-columns: 1fr;
		}
		.category-controls {
			justify-content: start;
		}
		.milestones,
		.milestones.with-reviews {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 8px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.watchlist-summary {
			transition: none;
		}
	}
</style>
