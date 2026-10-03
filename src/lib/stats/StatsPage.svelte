<script lang="ts">
	import StatsExpansion from "./StatsExpansion.svelte";
	import { resolve } from "$app/paths";
	import StatsChart from "./StatsChart.svelte";
	import StatsRankedChart from "./StatsRankedChart.svelte";
	import StatsTitlesDialog from "./StatsTitlesDialog.svelte";
	import { averageRating, decimal } from "./format";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsPeople from "./StatsPeople.svelte";
	import { type RatingSettings } from "@/lib/rating/helpers";
	import type {
		StatsResponse,
		StatsMediaCard,
		StatsPie,
		ChartPoint,
	} from "./types";
	let {
		data,
		publicOwner,
		onSelectionChange,
	}: {
		data: StatsResponse;
		publicOwner?: { id: string; username: string };
		onSelectionChange: (year: string, media: "movie" | "tv") => void;
	} = $props();
	let episodeCount = $state(5);
	let episodeTab = $state<"current" | "older">("current");
	let highestTab = $state<"current" | "older">("current");
	let yearlyFavoriteCount = $state(5);
	// Rows added per click for Cast and Directors & creators only.
	const peopleExpansionRows = 3;
	let peopleMode = $state<"most" | "rating">("most");
	let categorySort = $state<"count" | "rating">("count");
	let higherCount = $state(5);
	let lowerCount = $state(5);
	let mostWatchedCount = $state(5);
	let favoriteCount = $state(5);
	let decadeCounts = $state<Record<number, number>>({});
	let crewCounts = $state<Record<string, number>>({});
	let activityMode = $state<"week" | "month">("week");
	let selection = $state<{
		label: string;
		items: StatsMediaCard[];
		personId?: number;
		description?: string;
	}>();
	function explore(label: string, keys: string[], personId?: number) {
		const membership = new Set(keys);
		selection = {
			label,
			personId,
			items: [...data.posters, ...(data.episodes ?? [])]
				.filter((c) =>
					membership.has(
						`${c.type}:${c.id}${c.episodeNumber !== undefined ? `:${c.seasonNumber ?? 0}:${c.episodeNumber}` : ""}`,
					),
				)
				.sort((a, b) => a.title.localeCompare(b.title)),
		};
	}
	function explorePoint(point: ChartPoint) {
		if (point.items)
			selection = {
				label: point.tooltipLabel ?? point.label,
				items: point.items,
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
	const mediaLabel = $derived(data.media === "movie" ? "Films" : "Shows");
	const yearValue = $derived(lifetime ? "all" : String(data.year));
	const years = $derived(
		[
			...new Set([
				new Date().getUTCFullYear(),
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
			episodeCount = 5;
			episodeTab = "current";
			yearlyFavoriteCount = 5;
			favoriteCount = 5;
			highestTab = "current";
			categorySort = "count";
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
			detail: total ? `${((p.count / total) * 100).toFixed(1)}%` : "No watches",
		}));
	}
	const highCards = $derived([
		{
			label: "Highest TMDB rating",
			card: data.highsLows.highestTMDBRated,
			detail: (c: StatsMediaCard) => `${c.tmdbRating?.toFixed(1)}/10 on TMDB`,
		},
		{
			label: "Lowest TMDB rating",
			card: data.highsLows.lowestRated,
			detail: (c: StatsMediaCard) => `${c.tmdbRating?.toFixed(1)}/10 on TMDB`,
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
		{
			label: "Longest",
			card: data.highsLows.longest,
			detail: (c: StatsMediaCard) =>
				`${c.runtime} min${data.media === "tv" ? " per episode" : ""}`,
		},
		{
			label: "Shortest",
			card: data.highsLows.shortest,
			detail: (c: StatsMediaCard) =>
				`${c.runtime} min${data.media === "tv" ? " per episode" : ""}`,
		},
	]);
</script>

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

<div class="stats-page">
	<header>
		<div class="heading">
			<a
				class="back"
				href={publicOwner
					? resolve("/(public)/lists/[id]/[username]", publicOwner)
					: resolve("/profile")}
				>← {publicOwner ? "Back to library" : "Profile"}</a
			>
			<p class="eyebrow">{data.owner.username + "'s " + (data.scope === "lifetime" ? "all-time stats" : "year in film")}</p>
			<h1 class="norm">
				{lifetime ? "A Life in " + (data.media === "tv" ? "Shows" : "Film") : `${data.year}`}
			</h1>
			<p class="intro">
				{data.summary.titles.toLocaleString()}
				{data.media === "movie" ? "films" : "shows"} · {data.summary
					.averageRating
					? `${averageRating(data.summary.averageRating, settings)} average rating`
					: "Your story, one watch at a time"}
			</p>
		</div>
		<div class="controls">
			<label
				>Period<select
					data-stats-control="period"
					value={yearValue}
					onchange={(e) => onSelectionChange(e.currentTarget.value, data.media)}
					><option value="all">Lifetime</option>{#each years as y (y)}<option
							value={String(y)}>{y}</option
						>{/each}</select
				></label
			>
			<div class="segmented" aria-label="Media">
				<button
					data-stats-control="movie"
					class:active={data.media === "movie"}
					aria-pressed={data.media === "movie"}
					onclick={() => onSelectionChange(yearValue, "movie")}>Movies</button
				><button
					data-stats-control="tv"
					class:active={data.media === "tv"}
					aria-pressed={data.media === "tv"}
					onclick={() => onSelectionChange(yearValue, "tv")}>TV</button
				>
			</div>
		</div>
	</header>
	{#if data.metadata.partial}<details class="coverage">
			<summary
				>Some details are unavailable for {data.metadata.failedTitles.length} titles</summary
			>
			<p>
				Your recorded watches and ratings are included. Metadata charts may be
				incomplete.
			</p>
			<ul>
				{#each data.metadata.failedTitles as title (title)}<li>
						{title}
					</li>{/each}
			</ul>
		</details>{/if}
	{#if !data.summary.titles && !data.activity.total}<div class="empty-year">
			<h2 class="norm">A fresh page in your journal</h2>
			<p>
				No recorded {data.media === "movie" ? "movie" : "episode"} watches for
				{lifetime ? "this lifetime view" : data.year}. Choose another year or
				explore your watchlist below.
			</p>
		</div>{/if}

	{#if lifetime}
		<section>
			<div class="section-heading">
				<h2 class="norm">Through the years</h2>
				<span>Movies & TV · unique titles each year</span>
			</div>
			<div class="timeline-grid">
				{#each [{ key: "movies", label: "Films watched", color: "#29acf4" }, { key: "shows", label: "Shows watched", color: "#51ad79" }, { key: "averageRating", label: "Average rating", color: "#f5b85a" }, ...(data.reviewsVisible ? [{ key: "reviewed", label: "Reviewed", color: "#f47983" }] : [])] as metric (metric.key)}<div
					>
						<h3 class="norm">{metric.label}</h3>
						<StatsChart
							title={metric.label}
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
						<p>{decade.titles} titles watched</p>
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

	{#if !lifetime}<section>
			<div class="section-heading activity-heading">
				<h2 class="norm">Activity</h2>
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
			{#key activityMode}<StatsChart
					title={activityMode === "week"
						? `${data.media === "tv" ? "Episodes" : "Watches"} by week`
						: `${data.media === "tv" ? "Episodes" : "Watches"} by month`}
					points={activityMode === "week"
						? data.activity.weeks.map((w) => ({
								label: w.start.slice(5),
								tooltipLabel: weekRange(w.start),
								value: w.plays,
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
								label: new Date(`${m.month}-01T00:00:00Z`).toLocaleDateString(
									undefined,
									{ month: "short", timeZone: "UTC" },
								),
								tooltipLabel: m.month,
								value: m.plays,
								titleCount: m.items?.length ?? 0,
								averageRating: m.averageRating,
								items: m.items ?? [],
								detail: `${m.plays} ${data.media === "tv" ? "episodes" : "films"}`,
							}))}
					{settings}
					showData={activityMode === "week"}
					dataLabel={data.media === "tv"
						? "Browse weekly episodes"
						: "Browse weekly titles"}
					onSelect={explorePoint}
				/>{/key}
		</section>{/if}

	<section>
		<div class="section-heading">
			<h2 class="norm">Milestones</h2>
			<span>{lifetime ? "From the beginning" : `Bookends of ${data.year}`}</span
			>
		</div>
		<div class="milestones">
			{#each [{ label: "First watch", card: data.milestones.first }, { label: "Last watch", card: data.milestones.last }] as milestone (milestone.label)}<div
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
				Most watched
			</h3>
			<StatsPosters
				items={data.milestones.mostWatched.slice(0, mostWatchedCount)}
				owner={publicOwner}
				{settings}
				detail={(c) => `${c.plays} watches`}
			/>
			<StatsExpansion
				count={mostWatchedCount}
				total={data.milestones.mostWatched.length}
				onChange={(value) => (mostWatchedCount = value)}
			/>
		{/if}
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">Genres, countries & languages</h2>
			<div class="category-controls">
				<div class="segmented" role="group" aria-label="Sort categories by">
					<button
						class:active={categorySort === "count"}
						aria-pressed={categorySort === "count"}
						onclick={() => (categorySort = "count")}>Most watched</button
					><button
						class:active={categorySort === "rating"}
						aria-pressed={categorySort === "rating"}
						onclick={() => (categorySort = "rating")}>Highest rated</button
					>
				</div>
			</div>
		</div>
		<div class="categories">
			<StatsRankedChart
				title="Genres"
				items={data.genres}
				{settings}
				sortBy={categorySort}
				onSelect={(item) => explore(item.label, item.titleKeys)}
			/>
			<StatsRankedChart
				title="Countries"
				items={data.countries}
				color="#51ad79"
				{settings}
				sortBy={categorySort}
				onSelect={(item) => explore(item.label, item.titleKeys)}
			/>
			<StatsRankedChart
				title="Languages"
				items={data.languages}
				color="#d9aa64"
				{settings}
				sortBy={categorySort}
				onSelect={(item) => explore(item.label, item.titleKeys)}
			/>
		</div>
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">Breakdown</h2>
			<span>Patterns in your viewing</span>
		</div>
		<button
			class="plain watchlist-summary"
			disabled={!watchlistTitles.length}
			aria-haspopup="dialog"
			aria-label={`Show ${data.breakdown.watchlistAdditions} titles added to your watchlist`}
			onclick={() =>
				(selection = {
					label: "Added to watchlist",
					items: watchlistTitles,
					description: "added to watchlist",
				})}
		>
			<strong>{data.breakdown.watchlistAdditions.toLocaleString()}</strong>
			<span class="watchlist-copy">
				<span class="watchlist-title">Added to watchlist</span>
				<span class="watchlist-period">
					{lifetime ? "Across your recorded history" : `In ${data.year}`} · distinct
					titles
				</span>
			</span>
		</button>
		<div class="pies">
			{#each [...(!lifetime ? [{ title: "Release years", items: data.breakdown.release }] : []), { title: "Watches & rewatches", items: data.breakdown.plays }, ...(data.reviewsVisible ? [{ title: "Reviews", items: data.breakdown.reviews ?? [] }] : [])] as group (group.title)}<div
					class="pie"
				>
					<h3 class="norm">{group.title}</h3>
					<StatsChart
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
								><i style={`background:${i === 0 ? "#29acf4" : "#f5b85a"}`}
								></i>{item.label}<b>{item.value}</b></button
							>{/each}
					</div>
				</div>{/each}
		</div>
		<div class="rating-heading"><h3 class="norm">Rating distribution</h3></div>
		<StatsChart
			title="Rating distribution"
			points={ratingPoints.map((p) => ({
				...p,
				items: data.posters.filter(
					(c) =>
						(c.rating ?? 0) === (p.label === "Unrated" ? 0 : Number(p.label)),
				),
			}))}
			{settings}
			onSelect={explorePoint}
			color="#f5b85a"
		/>
		<p class="fine-print">
			Each title counts once. Rating bars show exact saved scores on the
			10-point scale.
		</p>
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">
				{data.media === "tv" ? "People behind the shows" : "People behind the films"}
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
				people={data.people.cast}
				expansionRows={peopleExpansionRows}
				unit={data.media === "tv" ? "episodes" : "titles"}
				mode={peopleMode}
				onSelect={(p) => explore(p.name, p.titleKeys, p.id)}
				{settings}
			/><StatsPeople
				title="Directors & creators"
				people={data.people.directors}
				expansionRows={peopleExpansionRows}
				mode={peopleMode}
				onSelect={(p) => explore(p.name, p.titleKeys, p.id)}
				{settings}
			/><StatsPeople
				title="Studios"
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
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">Highs and lows</h2>
			<span>By TMDB ratings, votes & release details</span>
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
			<span>You vs TMDB · /10 · At least +1 point</span>
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
			<span>You vs TMDB · /10 · At least −1 point</span>
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
				{mediaLabel} watched {lifetime ? "so far" : `in ${data.year}`}
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
			<h2 class="norm">Highly rated, yet to see</h2>
			<span
				>From {publicOwner ? `${data.owner.username}'s` : "your"} watchlist</span
			>
		</div>
		<StatsPosters
			items={data.watchlist}
			owner={publicOwner}
			{settings}
			detail={(c) => `${c.tmdbRating?.toFixed(1) ?? "—"}/10 on TMDB`}
		/>
	</section>
	{#if selection}{#key selection}<StatsTitlesDialog
				{...selection}
				{period}
				owner={publicOwner}
				{settings}
				onClose={() => (selection = undefined)}
			/>{/key}{/if}
	<footer>
		Based on recorded whole-title watches. Past years use current saved ratings{data.reviewsVisible
			? " and reviews"
			: ""}. Dates use UTC. Movie and TV metadata from TMDB.
	</footer>
</div>

<style>
	:global(:root.theme-dark) .stats-page {
		--stats-accent: #29acf4;
	}
	.stats-page {
		--stats-accent: #086fa8;
		max-width: 1050px;
		width: 100%;
		margin: 0 auto;
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
	header {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		align-items: flex-end;
		padding-bottom: 28px;
	}
	.heading {
		min-width: 0;
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
	select {
		min-width: 160px;
		padding: 8px 12px;
		font-size: 14px;
		width: 100%;
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
	section {
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
		color: #f5b85a;
		font-size: 14px;
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
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 24px;
		max-width: 430px;
	}
	.milestones :global(.posters) {
		grid-template-columns: minmax(0, 130px);
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
	.highs-lows :global(.posters) {
		grid-template-columns: minmax(0, 1fr);
		max-width: 135px;
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
		margin-bottom: 28px;
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
	@media (max-width: 700px) {
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
		.controls {
			flex-direction: row;
			align-items: end;
			gap: 12px;
		}
		.controls label {
			flex: 1;
			min-width: 0;
		}
		select {
			min-width: 0;
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
			gap: 16px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.watchlist-summary {
			transition: none;
		}
	}
</style>
