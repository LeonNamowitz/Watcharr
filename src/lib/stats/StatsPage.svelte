<script lang="ts">
	import { resolve } from "$app/paths";
	import StatsChart from "./StatsChart.svelte";
	import StatsPosters from "./StatsPosters.svelte";
	import StatsPeople from "./StatsPeople.svelte";
	import { toRatingLabel, type RatingSettings } from "@/lib/rating/helpers";
	import type {
		StatsResponse,
		StatsMediaCard,
		StatsBar,
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
	let highestTab = $state<"current" | "older">("current");
	let peopleMode = $state<"most" | "rating">("most");
	let rankingCounts = $state<Record<string, number>>({
		Genres: 10,
		Countries: 10,
		Languages: 10,
	});
	let higherCount = $state(6);
	let lowerCount = $state(6);
	let mostWatchedCount = $state(6);
	const settings: RatingSettings = $derived({
		ratingSystem: data.owner.ratingSystem,
		ratingStep: data.owner.ratingStep,
	});
	const lifetime = $derived(data.scope === "lifetime");
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
		const points = [
			{
				label: "Unrated",
				value: distribution.find((b) => b.rating === 0)?.count ?? 0,
			},
		];
		for (let i = 1; i <= Math.round(10 / step); i++) {
			const rating = Number((i * step).toFixed(1));
			points.push({
				label: String(rating),
				value:
					distribution.find((b) => Math.abs(b.rating - rating) < 0.001)
						?.count ?? 0,
			});
		}
		return points;
	});
	function bars(items: StatsBar[]): ChartPoint[] {
		return items.map((p) => ({
			label: p.label,
			value: p.count,
			detail: `${p.count} distinct titles · Average personal rating ${toRatingLabel(p.averageRating, settings)}`,
		}));
	}
	function pies(items: StatsPie[]): ChartPoint[] {
		const total = items.reduce((a, b) => a + b.count, 0);
		return items.map((p) => ({
			label: p.label,
			value: p.count,
			detail: total ? `${((p.count / total) * 100).toFixed(1)}%` : "No watches",
		}));
	}
	function delta(c: StatsMediaCard) {
		const n = (c.rating ?? 0) - (c.tmdbRating ?? 0);
		return `${n > 0 ? "+" : ""}${n.toFixed(1)} · You ${(c.rating ?? 0).toFixed(1)} / TMDB ${(c.tmdbRating ?? 0).toFixed(1)}`;
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
	function personHref(id: number) {
		return publicOwner
			? resolve("/(public)/lists/[id]/[username]/person/[personId]", {
					...publicOwner,
					personId: String(id),
				})
			: resolve("/(app)/person/[id]", { id: String(id) });
	}
</script>

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
			<p class="eyebrow">{data.owner.username}'s viewing journal</p>
			<h1 class="norm">
				{lifetime ? "A lifetime in stories" : `${data.year} in review`}
			</h1>
			<p class="intro">
				{data.summary.titles.toLocaleString()}
				{data.media === "movie" ? "films" : "shows"} · {data.summary
					.averageRating
					? `${toRatingLabel(data.summary.averageRating, settings)} average rating`
					: "Your story, one watch at a time"}
			</p>
		</div>
		<div class="controls">
			<label
				>Period<select
					value={yearValue}
					onchange={(e) => onSelectionChange(e.currentTarget.value, data.media)}
					><option value="all">Lifetime</option>{#each years as y (y)}<option
							value={String(y)}>{y}</option
						>{/each}</select
				></label
			>
			<div class="segmented" aria-label="Media">
				<button
					class:active={data.media === "movie"}
					aria-pressed={data.media === "movie"}
					onclick={() => onSelectionChange(yearValue, "movie")}>Movies</button
				><button
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
	{#if !data.summary.titles}<div class="empty-year">
			<h2 class="norm">A fresh page in your journal</h2>
			<p>
				No recorded {data.media === "movie" ? "movie" : "whole-show"} watches for
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
				{#each [{ key: "movies", label: "Films watched", color: "#39cfa2" }, { key: "shows", label: "Shows watched", color: "#6495ed" }, { key: "averageRating", label: "Average rating", color: "#f5b85a" }, ...(data.reviewsVisible ? [{ key: "reviewed", label: "Reviewed", color: "#f47983" }] : [])] as metric (metric.key)}<div
					>
						<h3 class="norm">{metric.label}</h3>
						<StatsChart
							title={metric.label}
							kind={metric.key === "averageRating" ? "line" : "bar"}
							color={metric.color}
							points={data.history.map((p) => ({
								label: String(p.year),
								value:
									metric.key === "averageRating" && !p.averageRating
										? null
										: Number(p[metric.key as keyof typeof p] ?? 0),
								detail:
									metric.key === "averageRating"
										? "Personal average on a 10-point scale; unrated titles excluded"
										: "Distinct titles watched this year",
							}))}
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
							<span>{toRatingLabel(decade.averageRating, settings)}</span>
						</h3>
						<p>{decade.titles} titles watched</p>
						<StatsPosters
							items={decade.items}
							owner={publicOwner}
							{settings}
							tiny
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
			{#if !lifetime}<div class="segmented small">
					<button
						class:active={highestTab === "current"}
						aria-pressed={highestTab === "current"}
						onclick={() => (highestTab = "current")}
						>{data.year} releases</button
					><button
						class:active={highestTab === "older"}
						aria-pressed={highestTab === "older"}
						onclick={() => (highestTab = "older")}>Older</button
					>
				</div>{:else}<span>Your personal favorites</span>{/if}
		</div>
		<StatsPosters
			items={lifetime || highestTab === "current"
				? data.highestRated.current
				: data.highestRated.older}
			owner={publicOwner}
			{settings}
		/>
	</section>

	{#if !lifetime}<section>
			<div class="section-heading">
				<h2 class="norm">By week</h2>
				<div class="averages">
					<span
						><strong>{data.activity.averagePerWeek.toFixed(1)}</strong> / week</span
					><span
						><strong>{data.activity.averagePerMonth.toFixed(1)}</strong> / month</span
					>
				</div>
			</div>
			<StatsChart
				title="Watches by week"
				points={data.activity.weeks.map((w) => ({
					label: w.start.slice(5),
					value: w.plays,
					detail: `Week of ${w.start} · ${w.uniqueTitles} distinct titles${w.averageRating ? ` · ${toRatingLabel(w.averageRating, settings)} average` : ""}${w.titles.length ? ` · ${w.titles.join(", ")}` : ""}`,
				}))}
			/>
			<details class="monthly">
				<summary>Monthly overview</summary><StatsChart
					title="Watches by month"
					color="#6495ed"
					points={data.activity.months.map((m) => ({
						label: new Date(`${m.month}-01T00:00:00Z`).toLocaleDateString(
							undefined,
							{ month: "short", timeZone: "UTC" },
						),
						value: m.plays,
						detail: `${m.month} · ${toRatingLabel(m.averageRating, settings)} average`,
					}))}
				/>
			</details>
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
			{#if data.milestones.mostWatched.length > mostWatchedCount}
				<button class="plain more" onclick={() => (mostWatchedCount += 6)}>
					Show more
				</button>
			{/if}
		{/if}
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">Genres, countries & languages</h2>
			<span>The worlds you explored</span>
		</div>
		<div class="categories">
			{#each [{ title: "Genres", items: data.genres, color: "#39cfa2" }, { title: "Countries", items: data.countries, color: "#6495ed" }, { title: "Languages", items: data.languages, color: "#f5b85a" }] as group (group.title)}<div
				>
					<h3 class="norm">{group.title}</h3>
					<div class="horizontal-bars">
						<div class="bar-labels">
							{#each group.items.slice(0, rankingCounts[group.title]) as item (item.label)}<span
									title={item.label}>{item.label}<b>{item.count}</b></span
								>{/each}
						</div>
						<StatsChart
							title={group.title}
							kind="horizontal"
							color={group.color}
							height={Math.max(
								120,
								Math.min(group.items.length, rankingCounts[group.title]) * 26 +
									30,
							)}
							points={bars(group.items.slice(0, rankingCounts[group.title]))}
						/>
					</div>
					{#if rankingCounts[group.title] < group.items.length}<button
							class="plain more"
							onclick={() => (rankingCounts[group.title] += 10)}
							>Show more ({group.items.length - rankingCounts[group.title]} remaining)</button
						>{/if}
				</div>{/each}
		</div>
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">Breakdown</h2>
			<span>Patterns in your viewing</span>
		</div>
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
					/>
					<div class="pie-legend">
						{#each pies(group.items) as item, i (item.label)}<span
								><i style={`background:${i === 0 ? "#39cfa2" : "#6495ed"}`}
								></i>{item.label}<b>{item.value}</b></span
							>{/each}
					</div>
				</div>{/each}
		</div>
		<div class="rating-heading">
			<h3 class="norm">Your ratings</h3>
			<span class="watchlist-count"
				><strong>{data.breakdown.watchlistAdditions}</strong> added to watchlist</span
			>
		</div>
		<StatsChart
			title="Personal rating totals"
			points={ratingPoints}
			color="#f5b85a"
		/>
		<p class="fine-print">
			Each title counts once. Rating bars show exact saved scores on the
			10-point scale.
		</p>
	</section>

	<section>
		<div class="section-heading">
			<h2 class="norm">People behind the stories</h2>
			<div class="segmented small">
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
				mode={peopleMode}
				owner={publicOwner}
				{settings}
			/><StatsPeople
				title="Directors & creators"
				people={data.people.directors}
				mode={peopleMode}
				owner={publicOwner}
				{settings}
			/><StatsPeople
				title="Studios"
				people={data.studios}
				mode={peopleMode}
				owner={publicOwner}
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
					>{#each department.jobs as job (job.job)}<details class="job">
							<summary>{job.job}</summary>
							<ul>
								{#each [...job.people]
									.filter((p) => peopleMode !== "rating" || p.averageRating > 0)
									.sort( (a, b) => (peopleMode === "rating" ? b.averageRating - a.averageRating : b.titles - a.titles) ) as person (person.id)}<li
									>
										<a href={personHref(person.id)}>{person.name}</a><span
											>{person.titles} titles · {toRatingLabel(
												person.averageRating,
												settings,
											)}</span
										>
									</li>{/each}
							</ul>
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
			<span>At least +1 point on a 10-point scale</span>
		</div>
		<StatsPosters
			items={higher.slice(0, higherCount)}
			owner={publicOwner}
			{settings}
			detail={delta}
		/>{#if higherCount < higher.length}<button
				class="plain more"
				onclick={() => (higherCount += 6)}
				>Show more ({higher.length - higherCount} remaining)</button
			>{/if}
	</section>
	<section>
		<div class="section-heading">
			<h2 class="norm">Rated lower than average</h2>
			<span>At least −1 point on a 10-point scale</span>
		</div>
		<StatsPosters
			items={lower.slice(0, lowerCount)}
			owner={publicOwner}
			{settings}
			detail={delta}
		/>{#if lowerCount < lower.length}<button
				class="plain more"
				onclick={() => (lowerCount += 6)}
				>Show more ({lower.length - lowerCount} remaining)</button
			>{/if}
	</section>
	<section>
		<div class="section-heading">
			<h2 class="norm">
				{mediaLabel} watched {lifetime ? "so far" : `in ${data.year}`}
			</h2>
			<span>{data.posters.length} distinct titles</span>
		</div>
		<StatsPosters items={data.posters} owner={publicOwner} {settings} tiny />
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
	<footer>
		Based on recorded whole-title watches. Past years use current saved ratings{data.reviewsVisible
			? " and reviews"
			: ""}. Dates use UTC. Movie and TV metadata from TMDB.
	</footer>
</div>

<style>
	:global(:root.theme-dark) .stats-page {
		--stats-accent: #39cfa2;
		--stats-blue: #6495ed;
	}
	.stats-page {
		--stats-accent: #087d61;
		--stats-blue: #3a67b6;
		max-width: 1050px;
		width: 100%;
		margin: 0 auto;
		padding: 24px 32px 60px;
		box-sizing: border-box;
		min-width: 0;
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
		font-size: 12px;
		opacity: 0.7;
		color: inherit;
		text-decoration: none;
	}
	.eyebrow {
		text-transform: uppercase;
		letter-spacing: 0.15em;
		font-size: 10px;
		color: var(--stats-accent, #39cfa2);
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
		opacity: 0.65;
	}
	.controls {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.controls label {
		font-size: 11px;
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
		flex: 1;
		min-width: 0;
		padding: 7px 13px;
		border: 0;
		background: transparent;
		color: inherit;
		font-size: 12px;
		border-radius: 4px;
		box-shadow: none;
	}
	.segmented button.active {
		background: #39cfa225;
		color: var(--stats-accent, #39cfa2);
	}
	.small button {
		font-size: 11px;
		padding: 6px 10px;
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
		font-size: 11px;
		opacity: 0.6;
	}
	h3 {
		font-size: 13px;
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
		font-size: 12px;
	}
	.decades p {
		font-size: 11px;
		opacity: 0.6;
		margin: 0 0 12px;
	}
	.averages {
		display: flex;
		gap: 16px;
		font-size: 11px;
		opacity: 0.8;
	}
	.averages strong {
		color: var(--stats-accent, #39cfa2);
		font-size: 18px;
		font-weight: 500;
	}
	.monthly {
		margin-top: 18px;
	}
	summary {
		cursor: pointer;
		font-size: 12px;
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
	.horizontal-bars {
		position: relative;
	}
	.bar-labels {
		position: absolute;
		inset: 4px 10px 26px 0;
		display: grid;
		z-index: 1;
		pointer-events: none;
	}
	.bar-labels span {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 10px;
		padding: 0 6px;
		text-shadow: 0 1px 3px #0008;
		gap: 10px;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.bar-labels b {
		flex: none;
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
		font-size: 11px;
		max-width: 220px;
		margin: 12px auto 0;
	}
	.pie-legend span {
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
	.watchlist-count {
		font-size: 12px;
		opacity: 0.8;
	}
	.watchlist-count strong {
		color: #6495ed;
	}
	.fine-print {
		font-size: 10px;
		opacity: 0.55;
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
		font-size: 10px;
		opacity: 0.5;
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
		font-size: 12px;
	}
	.crew a {
		color: inherit;
		overflow-wrap: anywhere;
	}
	.crew li span {
		font-size: 10px;
		opacity: 0.6;
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
	.more {
		color: var(--stats-accent, #39cfa2);
		font-size: 12px;
		margin-top: 18px;
	}
	.coverage {
		padding: 14px;
		background: #f5b85a15;
		border-radius: 8px;
		margin-bottom: 24px;
	}
	.coverage p,
	.coverage li {
		font-size: 12px;
	}
	.coverage ul {
		max-height: 160px;
		overflow: auto;
	}
	.empty-year {
		background: #6495ed12;
		border-radius: 10px;
		padding: 24px;
		margin-bottom: 20px;
	}
	.empty-year p,
	.muted {
		font-size: 13px;
		opacity: 0.65;
	}
	footer {
		border-top: 1px solid #8883;
		padding-top: 20px;
		font-size: 10px;
		opacity: 0.5;
		line-height: 1.7;
	}
	@media (max-width: 700px) {
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
		.decades :global(.posters) {
			max-width: 240px;
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
		h2 {
			font-size: 18px;
		}
		.milestones {
			gap: 16px;
		}
	}
</style>
