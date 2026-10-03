<script lang="ts">
	import { tick } from "svelte";
	import Error from "@/lib/Error.svelte";
	import StatsExpansion from "./StatsExpansion.svelte";
	import { resolve } from "$app/paths";
	import StatsChart from "./StatsChart.svelte";
	import StatsRankedChart from "./StatsRankedChart.svelte";
	import StatsTitlesDialog from "./StatsTitlesDialog.svelte";
	import { averageRating, decimal } from "./format";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsPeople from "./StatsPeople.svelte";
	import StatsGameSections from "./StatsGameSections.svelte";
	import StatsLibrarySections from "./StatsLibrarySections.svelte";
	import StatsCalendar from "./StatsCalendar.svelte";
	import { type RatingSettings } from "@/lib/rating/helpers";
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
		loading = false,
		error,
		requestedYear,
		requestedMedia,
	}: {
		loading?: boolean;
		error?: unknown;
		requestedYear?: string;
		requestedMedia?: StatsMedia;
		data: StatsResponse;
		publicOwner?: { id: string; username: string };
		onSelectionChange: (year: string, media: StatsMedia) => void;
	} = $props();
	let episodeCount = $state(5);
	let episodeTab = $state<"current" | "older">("current");
	let highestTab = $state<"current" | "older">("current");
	let yearlyFavoriteCount = $state(5);
	// Rows added per click for Cast and Directors & creators only.
	const peopleExpansionRows = 3;
	// Rows added per click in the title list popup.
	const titleDialogExpansionRows = 2;
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
	let backgroundPreview = $state<
		"panels" | "panels-edge-grid" | "tonal-wash" | "edge-grid"
	>("panels");
	let calendarYear = $state<number>();
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
			calendarYear,
			waitingCount,
			gameCounts,
			gameActivityKind,
			backgroundPreview,
			peopleCounts,
			categoryCounts,
			selection,
			dialog,
			openDetails: Array.from(
				root.querySelectorAll("details"),
				(el) => el.open,
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
		waitingCount = saved.waitingCount ?? 5;
		activityMode = saved.activityMode;
		activityMetric = saved.activityMetric ?? "count";
		gameActivityKind = saved.gameActivityKind;
		backgroundPreview = saved.backgroundPreview;
		peopleCounts = saved.peopleCounts;
		await tick();
		// Ranked charts reset their expansion when sorting changes.
		categoryCounts = saved.categoryCounts;
		gameCounts = saved.gameCounts;
		await tick();
		root.querySelectorAll("details").forEach((el, index) => {
			el.open = saved.openDetails[index] ?? false;
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
	const distinctTitles = $derived(
		data.media === "tv"
			? new Set(
					[
						...data.posters,
						...(data.episodes ?? []).filter((item) => (item.plays ?? 0) > 0),
					].map((item) => item.id),
				).size
			: data.summary.titles,
	);
	const hoursPlayed = $derived(data.summary.hours);
	const hoursDescription = $derived(
		isGame
			? lifetime
				? "Saved cumulative playtime for games in this view."
				: "Saved playtime for games first completed in the selected year."
			: "Estimated from available runtimes, recorded watches, finished shows and finished episodes.",
	);
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
			waitingCount = 5;
			episodeCount = 5;
			episodeTab = "current";
			yearlyFavoriteCount = 5;
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
		const ratings = [...new Set(titleKeys)]
			.map((key) => ratingsByTitle.get(key) ?? 0)
			.filter((rating) => rating > 0);
		return ratings.length
			? ratings.reduce((total, rating) => total + rating, 0) / ratings.length
			: 0;
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
		<section>
			<div class="section-heading">
				<h2 class="norm">Highest rated episodes</h2>
				{#if lifetime}<span>Your favorites rated 9/10 or above</span>
				{:else}<div class="segmented">
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
					</div>{/if}
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

<svelte:head
	><title>{data.owner.username} · {period} stats · Watcharr</title></svelte:head
>

<div
	bind:this={root}
	class="stats-page"
	data-background-style={backgroundPreview}
>
	<header>
		<div class="heading">
			<a
				class="back"
				href={publicOwner
					? resolve("/(public)/lists/[id]/[username]", publicOwner)
					: resolve("/")}
				>← {publicOwner ? "Back to library" : "Back to library"}</a
			>
			<p class="eyebrow">
				{data.owner.username +
					"'s " +
					(selectedYear === "all"
						? "all-time stats"
						: selectedMedia === "tv"
							? "year in television"
							: selectedMedia === "game"
								? "year in games"
								: "year in film")}
			</p>
			<h1 class="norm">
				{selectedYear === "all"
					? "A Life in " +
						(selectedMedia === "game"
							? "Games"
							: selectedMedia === "tv"
								? "Shows"
								: "Film")
					: selectedYear}
			</h1>
			<p class="intro">
				{selectedYear === "all"
					? "Across your recorded history"
					: selectedMedia === "game"
						? "Your gaming journal"
						: `Your ${selectedMedia === "tv" ? "television" : "film"} viewing journal`}
			</p>
			<dl class="header-summary" aria-label="Media summary" aria-busy={loading}>
				<div>
					<dt>
						Distinct {selectedMedia === "game"
							? "games"
							: selectedMedia === "tv"
								? "shows"
								: "films"}
					</dt>
					<dd>{loading || error ? "—" : distinctTitles.toLocaleString()}</dd>
				</div>
				<div>
					<dt>
						{selectedMedia === "game"
							? "Games completed"
							: selectedMedia === "tv"
								? "Episodes watched"
								: "Watches"}
					</dt>
					<dd>
						{loading || error
							? "—"
							: (isGame
									? (data.summary.completed ?? 0)
									: data.media === "tv"
										? data.activity.total
										: data.summary.plays
								).toLocaleString()}
					</dd>
				</div>
				<div title={hoursDescription}>
					<dt>{isGame ? "Recorded hours" : "Estimated hours"}</dt>
					<dd>
						{loading || error || hoursPlayed === undefined
							? "—"
							: decimal(hoursPlayed)}
					</dd>
				</div>
				<div>
					<dt>
						Average {selectedMedia === "game"
							? "game"
							: selectedMedia === "tv"
								? "show"
								: "film"} rating
					</dt>
					<dd>
						{loading || error
							? "—"
							: averageRating(data.summary.averageRating, settings)}
					</dd>
				</div>
			</dl>
		</div>
		<div class="controls">
			<label
				>Period<select
					data-stats-control="period"
					value={selectedYear}
					onchange={(e) =>
						onSelectionChange(e.currentTarget.value, selectedMedia)}
					><option value="all">Lifetime</option>{#each years as y (y)}<option
							value={String(y)}>{y}</option
						>{/each}</select
				></label
			>
			<div class="segmented" aria-label="Media">
				<button
					data-stats-control="movie"
					class:active={selectedMedia === "movie"}
					aria-pressed={selectedMedia === "movie"}
					onclick={() => onSelectionChange(selectedYear, "movie")}
					>Movies</button
				><button
					data-stats-control="tv"
					class:active={selectedMedia === "tv"}
					aria-pressed={selectedMedia === "tv"}
					onclick={() => onSelectionChange(selectedYear, "tv")}>TV</button
				><button
					data-stats-control="game"
					class:active={selectedMedia === "game"}
					aria-pressed={selectedMedia === "game"}
					onclick={() => onSelectionChange(selectedYear, "game")}>Games</button
				>
			</div>
			<label class="preview-control">
				Background preview
				<select bind:value={backgroundPreview}>
					<option value="panels">Section panels</option>
					<option value="panels-edge-grid">Panels + edge grid</option>
					<option value="tonal-wash">Soft full-page wash</option>
					<option value="edge-grid">Edge vignette + grid dots</option>
				</select>
			</label>
		</div>
	</header>
	{#if loading}<div class="load-status" role="status">
			<span>Updating stats…</span>
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
			{#if !isGame && data.library}<StatsLibrarySections
					{data}
					owner={publicOwner}
					{settings}
					onSelect={exploreSelection}
					bind:waitingCount
				/>{/if}
			{#if !data.summary.titles && !data.activity.total}<div class="empty-year">
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

			{#if lifetime}
				<section>
					<div class="section-heading">
						<h2 class="norm">Through the years</h2>
						<span
							>{isGame ? "Games" : "Movies & TV"} · unique titles each year</span
						>
					</div>
					<div class="timeline-grid">
						{#each [...(isGame ? [{ key: "games", label: "Games played", color: "#29acf4" }, { key: "completed", label: "Games completed", color: "#51ad79" }] : [{ key: "movies", label: "Films watched", color: "#29acf4" }, { key: "shows", label: "Shows watched", color: "#51ad79" }]), { key: "averageRating", label: "Average rating", color: "#f5b85a" }, ...(data.reviewsVisible ? [{ key: "reviewed", label: "Reviewed", color: "#f47983" }] : [])] as metric (metric.key)}<div
							>
								<h3 class="norm">{metric.label}</h3>
								<StatsChart
									titleUnit={isGame ? "games" : "titles"}
									title={metric.label}
									dataLabel={`Browse ${metric.label.toLowerCase()} by year`}
									kind={metric.key === "averageRating" ? "line" : "bar"}
									color={metric.color}
									points={data.history.map((p) => ({
										label: String(p.year),
										tooltipLabel: `${metric.label} · ${p.year}`,
										items: historyItems(p, metric.key),
										value:
											metric.key === "averageRating" && !p.averageRating
												? null
												: Number(p[metric.key as keyof typeof p] ?? 0),
										titleCount:
											metric.key === "averageRating"
												? historyItems(p, metric.key).length
												: Number(p[metric.key as keyof typeof p] ?? 0),
										averageRating: p.averageRating,
									}))}
									{settings}
									onSelect={explorePoint}
								/>
							</div>{/each}
					</div>
				</section>
				<section>
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
									items={decade.items.slice(
										0,
										decadeCounts[decade.decade] ?? 5,
									)}
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
								Watch and rate a few more titles to discover your favorite
								decades.
							</p>{/each}
					</div>
				</section>
			{/if}

			<section>
				<div class="section-heading">
					<h2 class="norm">Highest rated {mediaLabel.toLowerCase()}</h2>
					{#if !lifetime}<div class="segmented">
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
						</div>{:else}<span>Your favorites rated above 8/10</span>{/if}
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

			{@render episodeSection()}
			{#if isGame}{@render gameSection("playtime")}{@render gameSection(
					"activity",
				)}{/if}

			{#if !lifetime && !isGame}<section>
					<div class="section-heading activity-heading">
						<h2 class="norm">Activity</h2>
						<div class="segmented" aria-label="Activity metric">
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
						</div>
						<div class="segmented" aria-label="Activity interval">
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
						</div>
					</div>
					<div class="averages activity-averages">
						<span
							><strong
								>{(data.media === "tv"
									? data.activity.total
									: data.summary.titles
								).toLocaleString()}</strong
							>
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
										label: new Date(
											`${m.month}-01T00:00:00Z`,
										).toLocaleDateString(undefined, {
											month: "short",
											timeZone: "UTC",
										}),
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
										averageRating: m.averageRating,
										items: m.items ?? [],
										detail: `${m.plays} ${data.media === "tv" ? "episodes" : "films"}`,
									}))}
							{settings}
							valueUnit={data.media === "tv" ? "episodes" : "watches"}
							showData
							dataLabel={`Browse ${activityMode === "week" ? "weekly" : "monthly"} ${data.media === "tv" ? "episodes" : "films"}`}
							onSelect={explorePoint}
						/>{/key}
				</section>{/if}

			{#if !isGame && data.library}<StatsCalendar
					{data}
					onSelect={exploreSelection}
					bind:calendarYear
				/>{/if}

			<section>
				<div class="section-heading">
					<h2 class="norm">Milestones</h2>
					<span
						>{lifetime
							? "From the beginning"
							: `Bookends of ${data.year}`}</span
					>
				</div>
				<div class="milestones">
					{#each [{ label: isGame ? "First recorded progress" : "First watch", card: data.milestones.first }, { label: isGame ? "Last recorded progress" : "Last watch", card: data.milestones.last }] as milestone (milestone.label)}<div
						>
							<h3 class="norm">{milestone.label}</h3>
							<StatsPosters
								items={milestone.card ? [milestone.card] : []}
								owner={publicOwner}
								{settings}
								detail={(c) => c.date ?? ""}
							/>
						</div>{/each}
				</div>
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

			{#if isGame}{@render gameSection("categories")}{:else}
				<section>
					<div class="section-heading">
						<h2 class="norm">Genres, countries & languages</h2>
						<div class="category-controls">
							<div
								class="segmented"
								role="group"
								aria-label="Sort categories by"
							>
								<button
									class:active={categorySort === "count"}
									aria-pressed={categorySort === "count"}
									onclick={() => (categorySort = "count")}>Most watched</button
								><button
									class:active={categorySort === "rating"}
									aria-pressed={categorySort === "rating"}
									onclick={() => (categorySort = "rating")}
									>Highest rated</button
								>
							</div>
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

			<section>
				<div class="section-heading">
					<h2 class="norm">Breakdown</h2>
					<span>Patterns in your {isGame ? "gaming" : "viewing"}</span>
				</div>
				<div class="pies">
					{#each [...(!lifetime ? [{ title: "Release years", items: data.breakdown.release }] : []), { title: isGame ? "Completions & replays" : "Watches & rewatches", items: data.breakdown.plays }, ...(data.games ? [{ title: "Current statuses", items: data.games.statuses }, { title: "Completion", items: data.games.completion }] : []), ...(data.reviewsVisible ? [{ title: "Reviews", items: data.breakdown.reviews ?? [] }] : [])] as group (group.title)}<div
							class="pie"
						>
							<h3 class="norm">{group.title}</h3>
							<StatsChart
								titleUnit={isGame ? "games" : "titles"}
								title={group.title}
								kind="pie"
								height={150}
								points={pies(group.items)}
								{settings}
								onSelect={explorePoint}
							/>
							<div class="pie-legend">
								{#each pies(group.items) as item, i (item.label)}<button
										class="plain"
										onclick={() => explorePoint(item)}
										><i
											style={`background:${["#29acf4", "#f5b85a", "#f47983", "#51ad79", "#b19bea"][i % 5]}`}
										></i>{item.label}<b>{item.value}</b></button
									>{/each}
							</div>
						</div>{/each}
				</div>
				<div class="rating-heading">
					<h3 class="norm">Rating distribution</h3>
				</div>
				<StatsChart
					titleUnit={isGame ? "games" : "titles"}
					title="Rating distribution"
					dataLabel="Browse titles by rating"
					points={ratingPoints.map((p) => ({
						...p,
						tooltipLabel:
							p.label === "Unrated" ? "Unrated" : `Rated ${p.label}/10`,
						items: data.posters.filter(
							(c) =>
								(c.rating ?? 0) ===
								(p.label === "Unrated" ? 0 : Number(p.label)),
						),
					}))}
					{settings}
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

			{#if isGame}{@render gameSection("companies")}{:else}
				<section>
					<div class="section-heading">
						<h2 class="norm">
							{data.media === "tv"
								? "People behind the shows"
								: "People behind the films"}
						</h2>
						<div class="segmented">
							<button
								class:active={peopleMode === "most"}
								aria-pressed={peopleMode === "most"}
								onclick={() => (peopleMode = "most")}>Most watched</button
							><button
								class:active={peopleMode === "rating"}
								aria-pressed={peopleMode === "rating"}
								onclick={() => (peopleMode = "rating")}>Highest rated</button
							>
						</div>
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
									>{department.department}<span
										>{department.jobs.length} roles</span
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
											count={crewCounts[
												`${department.department}:${job.job}`
											] ?? 5}
											total={remaining +
												(crewCounts[`${department.department}:${job.job}`] ??
													5)}
											onChange={(value) =>
												(crewCounts[`${department.department}:${job.job}`] =
													value)}
										/>
									</details>{/each}
							</details>{:else}<p class="muted">
								No recurring crew credits yet.
							</p>{/each}
					</div>
				</section>
			{/if}

			<section>
				<div class="section-heading">
					<h2 class="norm">Highs and lows</h2>
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
			<section>
				<div class="section-heading">
					<h2 class="norm">Rated higher than average</h2>
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
			<section>
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
			<section>
				<div class="section-heading">
					<h2 class="norm">
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
			<section>
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
					: "Based on recorded whole-title watches."} Past years use current saved
				ratings{data.reviewsVisible ? " and reviews" : ""}. Dates use UTC. {isGame
					? "Game metadata from IGDB. Playtime shows current saved totals in Lifetime only."
					: "Movie and TV metadata from TMDB."}
			</footer>
		</div>{/key}
</div>

<style>
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
	}
	.stats-page > :is(header, .stats-content, .load-status) {
		max-width: 1050px;
		width: 100%;
		margin-right: auto;
		margin-left: auto;
	}
	.stats-page[data-background-style="tonal-wash"] {
		background-image: linear-gradient(
			175deg,
			color-mix(in srgb, var(--stats-accent) 5%, var(--bg-color)) 0%,
			var(--bg-color) 48%,
			color-mix(in srgb, #51ad79 2.5%, var(--bg-color)) 100%
		);
	}
	.stats-page[data-background-style="edge-grid"],
	.stats-page[data-background-style="panels-edge-grid"] {
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
	.stats-page[data-background-style="panels"] .stats-content > :global(section),
	.stats-page[data-background-style="panels-edge-grid"]
		.stats-content
		> :global(section) {
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
		padding-bottom: 28px;
	}
	.heading {
		min-width: 0;
		flex: 1;
	}
	.header-summary {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 16px;
		margin-top: 22px;
		max-width: 610px;
		padding-top: 18px;
		border-top: 1px solid var(--stats-border);
	}
	.header-summary > div {
		display: grid;
		grid-template-rows: 38px auto;
		gap: 4px;
		min-width: 0;
	}
	.header-summary dt {
		grid-row: 2;
		font-size: 12px;
		color: var(--stats-muted);
	}
	.header-summary dd {
		grid-row: 1;
		align-self: end;
		font-size: clamp(23px, 3vw, 32px);
		line-height: 1.2;
		font-weight: 600;
		letter-spacing: -0.035em;
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
		font-size: 14px;
		opacity: 0.7;
		color: inherit;
		text-decoration: none;
	}
	.eyebrow {
		text-transform: uppercase;
		letter-spacing: 0.15em;
		font-size: 13px;
		color: var(--stats-accent, #29acf4);
		margin: 22px 0 10px;
		overflow-wrap: anywhere;
	}
	h1 {
		font-size: clamp(28px, 5vw, 42px);
		line-height: 1.15;
		margin: 0;
		font-weight: 700;
		letter-spacing: -0.035em;
	}
	.intro {
		margin: 12px 0 0;
		font-size: 13px;
		color: var(--stats-muted);
	}
	.controls {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.controls label {
		font-size: 13px;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		display: grid;
		gap: 6px;
	}
	.preview-control {
		min-width: 200px;
	}
	select {
		min-width: 160px;
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
	.segmented {
		display: flex;
		background: #8881;
		border: 1px solid #8883;
		border-radius: 7px;
		padding: 3px;
		max-width: 100%;
	}
	.segmented button {
		flex: 1 0 auto;
		width: auto;
		white-space: nowrap;
		min-width: 80px;
		min-height: 40px;
		line-height: 20px;
		padding: 9px 13px;
		border: 0;
		background: transparent;
		color: inherit;
		font-size: 14px;
		border-radius: 4px;
		box-shadow: none;
	}
	.segmented button.active {
		background: #29acf425;
		color: var(--stats-accent, #29acf4);
	}
	section,
	:global(section.stats-game-section) {
		padding: 26px 0 30px;
		border-top: 1px solid #8883;
	}
	.section-heading {
		display: flex;
		justify-content: space-between;
		gap: 12px;
		align-items: center;
		flex-wrap: wrap;
		margin-bottom: 22px;
	}
	h2 {
		font-size: 19px;
		font-weight: 600;
		margin: 0;
		letter-spacing: -0.02em;
	}
	.section-heading > span {
		font-size: 13px;
		color: var(--stats-muted);
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
	.averages {
		display: flex;
		gap: 16px;
		font-size: 13px;
		opacity: 0.8;
	}
	.averages strong {
		color: var(--stats-accent, #29acf4);
		font-size: 18px;
		font-weight: 500;
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
	.activity-averages {
		align-items: baseline;
		flex-wrap: wrap;
		row-gap: 8px;
		margin: -4px 0 18px;
	}
	.segmented button:hover {
		background: color-mix(in srgb, #29acf4 12%, transparent);
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
		.stats-page[data-background-style="panels"]
			.stats-content
			> :global(section),
		.stats-page[data-background-style="panels-edge-grid"]
			.stats-content
			> :global(section) {
			margin-inline: -24px;
			padding-inline: 24px;
		}
		header {
			flex-direction: column;
			align-items: stretch;
			gap: 20px;
		}
		.controls {
			flex-direction: row;
			flex-wrap: wrap;
			align-items: end;
		}
		.controls label {
			flex: 1;
			min-width: 0;
		}
		.controls .preview-control {
			flex: 0 0 100%;
		}
		.header-summary {
			max-width: none;
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
		.controls {
			flex-direction: row;
			flex-wrap: wrap;
			align-items: end;
			gap: 12px;
		}
		.controls label {
			flex: 1;
			min-width: 0;
		}
		.controls .preview-control {
			flex: 0 0 100%;
		}
		select {
			min-width: 0;
		}
		.stats-page[data-background-style="panels"]
			.stats-content
			> :global(section),
		.stats-page[data-background-style="panels-edge-grid"]
			.stats-content
			> :global(section) {
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
		.section-heading {
			align-items: flex-start;
		}
		.category-controls {
			justify-content: start;
		}
		h2 {
			font-size: 18px;
		}
		.milestones {
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
