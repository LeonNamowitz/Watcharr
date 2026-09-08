<script lang="ts">
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import PageBackdrop from "@/lib/generic/PageBackdrop.svelte";
	import Modal from "@/lib/Modal.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import { baseURL, req, updateWatched } from "@/lib/util/api";
	import {
		fromCanonicalScore100,
		calculateRubricScore,
		evaluateComparisonScore,
		getAdaptiveComparisonProbe,
		getCanonicalRatingStep,
		getDefaultCategoryValues,
		getMediaKey,
		getRatedSimilarCandidates,
		getRatingRubric,
		pickComparisonCandidate,
		pickComparisonCandidateNearScore,
		rankRecentCandidates,
		toCanonicalScore100,
		type ComparisonDecision,
		type ComparisonSide,
		type RatingComparison,
		type RatingHelperCategory,
		type RatingHelperMediaType,
	} from "@/lib/rating/ratingHelper";
	import { toRatingLabel, type RatingSettings } from "@/lib/rating/helpers";
	import { store } from "@/store.svelte";
	import { SvelteSet } from "svelte/reactivity";
	import { RatingSystem, type Media, type PaginationResponse } from "@/types";

	type HelperPhase = "initial" | "comparison" | "review";
	type ComparisonSource = "similar" | "recent";

	interface ActiveComparison {
		media: Media;
		source: ComparisonSource;
	}

	interface CompletedComparison extends ActiveComparison {
		decision: ComparisonDecision;
	}

	let { data } = $props();

	const batchSize = 5;
	const libraryPageSize = 100;
	const decisionOptions: {
		value: ComparisonDecision;
		label: string;
		description: string;
	}[] = [
		{
			value: "better",
			label: "Target is better",
			description: "The target deserves a higher rating",
		},
		{
			value: "same",
			label: "About the same",
			description: "These deserve roughly the same rating",
		},
		{
			value: "worse",
			label: "Target is worse",
			description: "The target deserves a lower rating",
		},
		{
			value: "skip",
			label: "Skip",
			description: "Do not use this comparison",
		},
	];

	let target: Media | undefined = $state();
	let recentItems: Media[] = $state([]);
	let phase = $state<HelperPhase>("initial");
	let activeComparison = $state<ActiveComparison>();
	let comparisonHistory = $state<CompletedComparison[]>([]);
	let batchSimilarCount = $state(0);
	let batchRecentCount = $state(0);
	let batchComparisonNumber = $state(0);
	let comparisonStartSide = $state<ComparisonSide>("above");
	let comparisonSearchExhausted = $state(false);
	let precisionMode = $state(false);
	let precisionBatchStartIndex = $state(0);
	let pageLoading = $state(true);
	let pageError: unknown | undefined = $state();
	let recentLoading = $state(false);
	let recentError: unknown | undefined = $state();
	let recentPage = $state(0);
	let recentHasMore = $state(true);
	let rubricOpen = $state(false);
	let initialFeeling = $state(7);
	let categoryValues = $state<Record<RatingHelperCategory, number>>({
		content: 7,
		technical: 7,
		design: 7,
		bias: 7,
	});
	let saving = $state(false);
	let saveError: unknown | undefined = $state();
	let confirmOpen = $state(false);

	const helperType = $derived(data.helperType as RatingHelperMediaType);
	const rubric = $derived(getRatingRubric(helperType));
	const ratingSettings = $derived<RatingSettings | undefined>(
		store.userSettings
			? {
					ratingSystem: store.userSettings.ratingSystem,
					ratingStep: store.userSettings.ratingStep,
				}
			: undefined,
	);
	const targetKey = $derived(target ? getMediaKey(target) : "");
	const initialScore100 = $derived(
		Math.max(0, Math.min(100, initialFeeling * 10)),
	);
	const baseScore100 = $derived(initialScore100);
	const similarCandidates = $derived.by(() => {
		if (!target) return [];
		return getRatedSimilarCandidates(
			target,
			helperType,
			new SvelteSet([targetKey]),
		);
	});
	const recentExcludedKeys = $derived.by(() => {
		const keys = new SvelteSet<string>([targetKey]);
		for (const candidate of similarCandidates) {
			keys.add(getMediaKey(candidate));
		}
		return keys;
	});
	const recentCandidates = $derived.by(() =>
		rankRecentCandidates(recentItems, helperType, recentExcludedKeys),
	);
	const comparisonInputs = $derived(getComparisonInputs(comparisonHistory));
	const evaluation = $derived(
		evaluateComparisonScore(
			baseScore100,
			comparisonInputs,
			getCanonicalRatingStep(ratingSettings),
		),
	);
	const usedComparisonKeys = $derived.by(() => {
		const keys = new SvelteSet<string>(
			comparisonHistory.map(({ media }) => getMediaKey(media)),
		);
		if (activeComparison) keys.add(getMediaKey(activeComparison.media));
		return keys;
	});
	const canDoMore = $derived(
		!comparisonSearchExhausted &&
			(similarCandidates.some(
				(media) => !usedComparisonKeys.has(getMediaKey(media)),
			) ||
				recentCandidates.some(
					(media) => !usedComparisonKeys.has(getMediaKey(media)),
				) ||
				recentHasMore),
	);
	const cannotConfirm = $derived(evaluation.score100 <= 0);
	const canGoBack = $derived(
		comparisonHistory.length > (precisionMode ? precisionBatchStartIndex : 0),
	);
	const detailPath = $derived(
		helperType === "movie"
			? resolve("/(app)/movie/[id]", { id: String(data.contentId) })
			: helperType === "tv"
				? resolve("/(app)/tv/[id]", { id: String(data.contentId) })
				: resolve("/(app)/game/[id]", { id: String(data.contentId) }),
	);
	const backdropSrc = $derived.by(() => {
		if (!target?.extBackdropPath && !target?.extPosterPath) return undefined;
		if (helperType === "game") {
			const path = target.extBackdropPath ?? target.extPosterPath;
			return `https://images.igdb.com/igdb/image/upload/t_1080p/${path}.jpg`;
		}
		if (!target.extBackdropPath) return undefined;
		return `https://www.themoviedb.org/t/p/w1920_and_h800_multi_faces${target.extBackdropPath}`;
	});

	function mediaTypePlural(type: RatingHelperMediaType): string {
		if (type === "movie") return "movies";
		if (type === "tv") return "shows";
		return "games";
	}

	function mediaPoster(media: Media): string | undefined {
		if (media.poster?.path) return `${baseURL}/${media.poster.path}`;
		if (!media.extPosterPath) return undefined;
		if (media.type === "tmdb_movie" || media.type === "tmdb_tv") {
			return `https://image.tmdb.org/t/p/w342${media.extPosterPath}`;
		}
		if (media.type === "igdb_game") {
			return `https://images.igdb.com/igdb/image/upload/t_cover_big/${media.extPosterPath}.jpg`;
		}
		return undefined;
	}

	function formatCanonicalScore(score100: number): string {
		const rating10 = fromCanonicalScore100(score100);
		if (ratingSettings?.ratingSystem === RatingSystem.Thumbs) {
			return `${rating10.toFixed(1)}/10`;
		}
		if (!ratingSettings?.ratingSystem && !ratingSettings?.ratingStep) {
			return `${rating10.toFixed(1)}/10`;
		}
		return toRatingLabel(rating10, ratingSettings);
	}

	function formatCanonicalExact(score100: number): string {
		return `${fromCanonicalScore100(score100).toFixed(1)}/10 internal`;
	}

	function setInitialFeeling(value: number) {
		initialFeeling = Math.max(0, Math.min(10, value));
		categoryValues = getDefaultCategoryValues(helperType, initialFeeling);
	}

	function setCategoryValue(id: RatingHelperCategory, value: number) {
		const nextValues = {
			...categoryValues,
			[id]: value,
		};
		categoryValues = nextValues;
		initialFeeling = calculateRubricScore(helperType, nextValues) / 10;
	}

	function getAvailableCandidates(
		items: Media[],
		history: CompletedComparison[],
	): Media[] {
		const used = new SvelteSet(history.map(({ media }) => getMediaKey(media)));
		return items.filter((media) => !used.has(getMediaKey(media)));
	}

	function getAvailableLibraryCandidates(
		history: CompletedComparison[],
	): Media[] {
		return getAvailableCandidates(
			rankRecentCandidates(recentItems, helperType, recentExcludedKeys),
			history,
		);
	}

	async function loadRemainingLibrary(): Promise<void> {
		while (recentHasMore && !recentError) {
			const previousPage = recentPage;
			await loadRecentPage();
			if (recentPage === previousPage) break;
		}
	}

	function getComparisonInputs(
		history: CompletedComparison[],
	): RatingComparison[] {
		return history.flatMap(({ media, decision }) => {
			const rating = media.watched?.rating;
			return typeof rating === "number" ? [{ rating, decision }] : [];
		});
	}

	function getLastPrecisionComparison(
		history: CompletedComparison[],
	): RatingComparison | undefined {
		for (
			let index = history.length - 1;
			index >= precisionBatchStartIndex;
			index -= 1
		) {
			const comparison = history[index];
			const rating = comparison?.media.watched?.rating;
			if (comparison?.decision !== "skip" && typeof rating === "number") {
				return { rating, decision: comparison.decision };
			}
		}
		return undefined;
	}

	async function chooseNextComparison(
		history: CompletedComparison[] = comparisonHistory,
	): Promise<void> {
		if (!target || phase !== "comparison") return;
		let availableSimilar = getAvailableCandidates(similarCandidates, history);
		let availableRecent = getAvailableLibraryCandidates(history);
		let source: ComparisonSource | undefined;
		let pool: Media[] = [];
		const ratingStep100 = getCanonicalRatingStep(ratingSettings);
		const currentEvaluation = evaluateComparisonScore(
			initialScore100,
			getComparisonInputs(history),
			ratingStep100,
		);

		if (precisionMode) {
			if (batchSimilarCount + batchRecentCount >= batchSize) {
				activeComparison = undefined;
				phase = "review";
				return;
			}

			if (recentHasMore) {
				// Rating distance is the primary signal. Load older pages too so an
				// exact library match is not hidden behind the first recent page.
				await loadRemainingLibrary();
				if (phase !== "comparison" || recentError) return;
				availableSimilar = getAvailableCandidates(similarCandidates, history);
				availableRecent = getAvailableLibraryCandidates(history);
			}

			const lastComparison = getLastPrecisionComparison(history);
			const precisionComparisonNumber = Math.max(
				0,
				history.length - precisionBatchStartIndex,
			);
			const probeScore100 = getAdaptiveComparisonProbe(
				currentEvaluation,
				lastComparison,
				ratingStep100,
				precisionComparisonNumber % 2 === 0 ? "above" : "below",
			);
			const lastRating100 = toCanonicalScore100(lastComparison?.rating);
			const candidateSide: ComparisonSide | undefined =
				lastComparison?.decision === "better"
					? "above"
					: lastComparison?.decision === "worse"
						? "below"
						: undefined;
			let media = pickComparisonCandidateNearScore(
				[...availableSimilar, ...availableRecent],
				probeScore100,
				{
					maximumDistance100: 100,
					side: candidateSide,
					sidePivot100: lastRating100,
					tieSide:
						candidateSide === "above"
							? "below"
							: candidateSide === "below"
								? "above"
								: undefined,
				},
			);
			// At the edge of the library there may be nothing left on the ideal
			// side. Keep the session useful with the closest unused item instead
			// of ending the batch early.
			if (!media && candidateSide) {
				media = pickComparisonCandidateNearScore(
					[...availableSimilar, ...availableRecent],
					probeScore100,
					{ maximumDistance100: 100 },
				);
			}
			if (!media) {
				comparisonSearchExhausted = true;
				activeComparison = undefined;
				phase = "review";
				return;
			}

			const mediaKey = getMediaKey(media);
			source = availableSimilar.some(
				(candidate) => getMediaKey(candidate) === mediaKey,
			)
				? "similar"
				: "recent";
			activeComparison = { media, source };
			return;
		}

		if (batchSimilarCount < batchSize && availableSimilar.length > 0) {
			source = "similar";
			pool = availableSimilar;
		} else if (batchRecentCount < batchSize) {
			if (recentHasMore) {
				await loadRemainingLibrary();
				if (phase !== "comparison" || recentError) return;
			}
			availableRecent = getAvailableLibraryCandidates(history);
			if (availableRecent.length > 0) {
				source = "recent";
				pool = availableRecent;
			}
		}

		if (!source) {
			activeComparison = undefined;
			phase = "review";
			return;
		}

		let media = pickComparisonCandidate(
			pool,
			currentEvaluation.score100,
			batchComparisonNumber,
			batchComparisonNumber % 2 === 0
				? comparisonStartSide
				: comparisonStartSide === "above"
					? "below"
					: "above",
		);
		if (!media && source === "recent") {
			media = pickComparisonCandidateNearScore(
				pool,
				currentEvaluation.score100,
				{ maximumDistance100: 100 },
			);
		}
		if (!media) {
			if (source === "similar") {
				// No suitably close similar item remains. Continue with the recent
				// pool instead of showing a distant comparison just for alternation.
				batchSimilarCount = batchSize;
				await chooseNextComparison(history);
				return;
			}
			comparisonSearchExhausted = true;
			activeComparison = undefined;
			phase = "review";
			return;
		}
		activeComparison = { media, source };
	}

	function startComparisons() {
		phase = "comparison";
		activeComparison = undefined;
		batchSimilarCount = 0;
		batchRecentCount = 0;
		batchComparisonNumber = 0;
		comparisonStartSide = Math.random() < 0.5 ? "above" : "below";
		comparisonSearchExhausted = false;
		precisionMode = false;
		precisionBatchStartIndex = 0;
		void chooseNextComparison([]);
	}

	function setDecision(decision: ComparisonDecision) {
		if (!activeComparison) return;
		const completed = { ...activeComparison, decision };
		const nextHistory = [...comparisonHistory, completed];
		comparisonHistory = nextHistory;
		if (completed.source === "similar") batchSimilarCount += 1;
		else batchRecentCount += 1;
		batchComparisonNumber += 1;
		activeComparison = undefined;
		void chooseNextComparison(nextHistory);
	}

	function previousComparison() {
		if (!canGoBack) return;
		const previous = comparisonHistory[comparisonHistory.length - 1];
		if (!previous) return;
		comparisonHistory = comparisonHistory.slice(0, -1);
		if (previous.source === "similar") {
			batchSimilarCount = Math.max(0, batchSimilarCount - 1);
		} else {
			batchRecentCount = Math.max(0, batchRecentCount - 1);
		}
		batchComparisonNumber = Math.max(0, batchComparisonNumber - 1);
		phase = "comparison";
		activeComparison = { media: previous.media, source: previous.source };
	}

	function finishComparisons() {
		if (activeComparison) {
			const skipped = { ...activeComparison, decision: "skip" as const };
			comparisonHistory = [...comparisonHistory, skipped];
			if (skipped.source === "similar") batchSimilarCount += 1;
			else batchRecentCount += 1;
			batchComparisonNumber += 1;
		}
		activeComparison = undefined;
		phase = "review";
	}

	function requestConfirmation() {
		if (cannotConfirm || saving) return;
		saveError = undefined;
		confirmOpen = true;
	}

	function closeConfirmation() {
		if (!saving) confirmOpen = false;
	}

	function doMoreComparisons() {
		phase = "comparison";
		activeComparison = undefined;
		batchSimilarCount = 0;
		batchRecentCount = 0;
		// Keep the global comparison number so an extended session continues
		// narrowing instead of restarting at the widest offset.
		batchComparisonNumber = comparisonHistory.length;
		comparisonSearchExhausted = false;
		precisionMode = true;
		precisionBatchStartIndex = comparisonHistory.length;
		void chooseNextComparison(comparisonHistory);
	}

	async function loadRecentPage(reset = false): Promise<void> {
		if (recentLoading || (!recentHasMore && !reset)) return;
		recentLoading = true;
		recentError = undefined;
		const nextPage = reset ? 1 : recentPage + 1;
		try {
			const response = await req.get<PaginationResponse<Media, undefined>>(
				"/watched",
				{
					params: {
						page: nextPage,
						limit: libraryPageSize,
						type: helperType,
						sort: "LASTCHANGED",
						sortDir: "desc",
					},
				},
			);
			const results = response?.results ?? [];
			recentItems = reset ? results : [...recentItems, ...results];
			recentPage = nextPage;
			recentHasMore = nextPage < (response?.totalPages ?? nextPage);
		} catch (err) {
			recentError = err;
		} finally {
			recentLoading = false;
		}
	}

	async function retryRecentLoad(): Promise<void> {
		await loadRecentPage(recentPage === 0);
		if (phase === "comparison" && !activeComparison && !recentError) {
			await chooseNextComparison(comparisonHistory);
		}
	}

	async function loadPage(): Promise<void> {
		pageLoading = true;
		pageError = undefined;
		recentError = undefined;
		saveError = undefined;
		confirmOpen = false;
		recentItems = [];
		recentPage = 0;
		recentHasMore = true;
		phase = "initial";
		activeComparison = undefined;
		comparisonHistory = [];
		batchSimilarCount = 0;
		batchRecentCount = 0;
		batchComparisonNumber = 0;
		comparisonStartSide = "above";
		comparisonSearchExhausted = false;
		precisionMode = false;
		precisionBatchStartIndex = 0;
		try {
			const response = await req.get<Media>(
				helperType === "movie"
					? `/content/movie/${data.contentId}`
					: helperType === "tv"
						? `/content/tv/${data.contentId}`
						: `/game/${data.contentId}`,
				helperType === "movie" || helperType === "tv"
					? { params: { region: store.userSettings?.country } }
					: undefined,
			);
			target = response;
			const existingRating = toCanonicalScore100(response?.watched?.rating);
			initialFeeling = existingRating === undefined ? 7 : existingRating / 10;
			categoryValues = getDefaultCategoryValues(helperType, initialFeeling);
		} catch (err) {
			target = undefined;
			pageError = err;
		} finally {
			pageLoading = false;
		}

		// Library comparisons are fetched only when the dynamic picker reaches
		// them, so the initial screen does not preload a fixed comparison queue.
	}

	async function confirmRating(): Promise<void> {
		if (!target || phase !== "review" || cannotConfirm || saving) return;
		saving = true;
		saveError = undefined;
		try {
			const rating = fromCanonicalScore100(evaluation.score100);
			target.watched = await updateWatched(target.watched, {
				contentId: data.contentId,
				contentType: helperType,
				rating,
			});
			confirmOpen = false;
			await goto(detailPath);
		} catch (err) {
			saveError = err;
		} finally {
			saving = false;
		}
	}

	$effect(() => {
		void loadPage();
	});
</script>

<svelte:head>
	<title>{target?.name ? `Rate ${target.name}` : "Rating Helper"}</title>
</svelte:head>

{#if pageLoading}
	<Spinner />
{:else if pageError}
	<Error
		pretty="Failed to load rating helper"
		error={pageError}
		onRetry={() => void loadPage()}
	/>
{:else if target}
	{#if backdropSrc}
		<PageBackdrop src={backdropSrc} />
	{/if}
	<main class="helper-page">
		<header class="helper-header">
			<a class="back-link" href={detailPath} aria-label="Back to details">
				<Icon i="arrow" wh={18} /> Back
			</a>
			<div>
				<span class="eyebrow">Rating Helper</span>
				<h1>{target.name}</h1>
			</div>
		</header>

		{#if phase === "initial"}
			<section class="session-card initial-card">
				<span class="eyebrow">Step 1 · First impression</span>
				<h2>What is your initial feeling?</h2>
				<p>
					Choose one starting point. You can optionally use the rubric as a
					reminder; it will only shape this initial feeling.
				</p>
				<div class="initial-score">
					{initialFeeling.toFixed(1)}<small>/10</small>
				</div>
				<label class="range-label" for="initial-feeling">
					<span><strong>Initial rating</strong></span>
					<input
						id="initial-feeling"
						type="range"
						min="0"
						max="10"
						step="0.1"
						value={initialFeeling}
						oninput={(event) =>
							setInitialFeeling(
								Number((event.currentTarget as HTMLInputElement).value),
							)}
					/>
				</label>

				<div class="rubric-panel">
					<button
						class="rubric-toggle"
						type="button"
						aria-expanded={rubricOpen}
						onclick={() => (rubricOpen = !rubricOpen)}
					>
						<span><Icon i="document" wh={18} /> Categories guideline</span>
						<Icon i="chevron" wh={16} />
					</button>
					{#if rubricOpen}
						<p class="rubric-intro">{rubric.intro}</p>
						<p class="rubric-note">
							These sliders only help shape your initial feeling. The
							comparisons below make the final suggestion.
						</p>
						{#each rubric.categories as category (category.id)}
							<div class="category-control">
								<label for={`category-${category.id}`}>
									<span>
										<strong>{category.label}</strong>
										<small
											>{(categoryValues[category.id] ?? 0).toFixed(1)} / {category.max}</small
										>
									</span>
								</label>
								<p>{category.description}</p>
								<input
									id={`category-${category.id}`}
									type="range"
									min="0"
									max={category.max}
									step="0.5"
									value={categoryValues[category.id] ?? 0}
									oninput={(event) =>
										setCategoryValue(
											category.id,
											Number((event.currentTarget as HTMLInputElement).value),
										)}
								/>
							</div>
						{/each}
						<p class="rubric-guidance">{rubric.guidance}</p>
					{/if}
				</div>

				<div class="card-actions">
					<button type="button" onclick={startComparisons}>
						Start comparisons
					</button>
				</div>
			</section>
		{/if}

		{#if phase === "comparison"}
			<section class="session-card comparison-card">
				<span class="eyebrow">Step 2 · Compare</span>
				<h2>Which feels better?</h2>
				<p class="comparison-help">
					Your choice advances immediately. The next item is selected from the
					remaining library using the updated suggestion.
				</p>

				{#if recentError}
					<Error
						pretty="Failed to load rated library items"
						error={recentError}
						onRetry={() => void retryRecentLoad()}
					/>
				{/if}

				{#if activeComparison}
					{@const candidate = activeComparison.media}
					<article class="candidate-card">
						<div class="candidate-source">
							{activeComparison.source === "similar"
								? "Similar item"
								: "Library match"}
						</div>
						{#if mediaPoster(candidate)}
							<img src={mediaPoster(candidate)} alt={candidate.name ?? ""} />
						{:else}
							<div class="poster-placeholder">
								<Icon i={helperType === "game" ? "gamepad" : "film"} wh={36} />
							</div>
						{/if}
						<div class="candidate-info">
							<span class="comparison-position">
								Comparison {comparisonHistory.length + 1}
							</span>
							<h3>{candidate.name}</h3>
							<fieldset>
								<legend>Compare {target.name} with {candidate.name}</legend>
								<p class="compare-prompt">
									Is <strong>{target.name}</strong>
									<span>better, about the same, or worse</span>
									than <strong>{candidate.name}</strong>?
								</p>
								<div class="decision-buttons">
									{#each decisionOptions as option (option.value)}
										<button
											type="button"
											onclick={() => setDecision(option.value)}
										>
											<strong>{option.label}</strong>
											<small>{option.description}</small>
										</button>
									{/each}
								</div>
							</fieldset>
						</div>
					</article>
				{:else if recentLoading}
					<div class="loading-more"><Spinner /></div>
				{:else}
					<div class="empty-comparisons">
						<Icon i="people" wh={42} />
						<p>
							No rated {mediaTypePlural(helperType)} were found in your library for
							comparison.
						</p>
					</div>
				{/if}

				<div class="comparison-navigation">
					{#if canGoBack}
						<button
							class="secondary"
							type="button"
							onclick={previousComparison}
						>
							Back
						</button>
					{/if}
					<button class="secondary" type="button" onclick={finishComparisons}>
						Finish this batch
					</button>
				</div>
			</section>
		{:else if phase === "review"}
			<section class="session-card review-card">
				<span class="eyebrow">Comparison batch complete</span>
				<h2>Suggested rating</h2>
				<div class="suggested-score">
					{formatCanonicalScore(evaluation.score100)}
				</div>
				<p class="internal-score">
					{formatCanonicalExact(evaluation.score100)}
				</p>
				<p>
					This combines your initial feeling with {comparisonInputs.length} soft comparison{comparisonInputs.length ===
					1
						? ""
						: "s"}.
				</p>
				{#if evaluation.score100 <= 0}
					<div class="zero-warning" role="alert">
						<Icon i="star" wh={20} />
						Watcharr treats a 0 rating as unrated. Raise the suggestion above zero
						before confirming.
					</div>
				{/if}
				{#if saveError}
					<Error
						pretty="Failed to save rating"
						error={saveError}
						onRetry={() => void confirmRating()}
					/>
				{/if}
				<div class="card-actions review-actions">
					{#if canDoMore}
						<button class="secondary" type="button" onclick={doMoreComparisons}>
							Do more comparisons
						</button>
					{/if}
					<button
						type="button"
						disabled={saving || cannotConfirm}
						onclick={requestConfirmation}
					>
						Confirm {formatCanonicalScore(evaluation.score100)}
					</button>
				</div>
			</section>
		{/if}
	</main>

	{#if confirmOpen}
		<Modal
			title="Confirm this rating"
			desc="This is the final step. Your rating will be saved to your library."
			onClose={closeConfirmation}
			maxWidth="500px"
		>
			<div class="confirm-dialog">
				<div class="confirm-score">
					<span>Save as</span>
					<strong>{formatCanonicalScore(evaluation.score100)}</strong>
					<small>{formatCanonicalExact(evaluation.score100)}</small>
				</div>
				<p>
					Confirming will update <strong>{target.name}</strong> and return you to
					its detail page. Comparison items will not be changed.
				</p>
				{#if saveError}
					<Error
						pretty="Failed to save rating"
						error={saveError}
						onRetry={() => void confirmRating()}
					/>
				{/if}
				<div class="card-actions confirm-actions">
					<button
						class="secondary"
						type="button"
						disabled={saving}
						onclick={closeConfirmation}>Cancel</button
					>
					<button
						type="button"
						disabled={saving}
						onclick={() => void confirmRating()}
					>
						{saving ? "Saving..." : "Confirm and save"}
					</button>
				</div>
			</div>
		</Modal>
	{/if}
{/if}

<style lang="scss">
	.helper-page {
		display: flex;
		flex-flow: column;
		gap: 24px;
		width: 100%;
		max-width: 1250px;
		padding: 28px 50px 80px;
		margin: 0 auto;
		color: $text-color;

		@media screen and (max-width: 650px) {
			padding: 20px 16px 60px;
		}
	}

	.helper-header {
		display: flex;
		flex-flow: column;
		align-items: center;
		gap: 10px;
		width: 100%;
		max-width: 760px;
		padding: 16px 24px;
		margin: 0 auto;
		border: 1px solid color-mix(in srgb, $text-color 20%, transparent);
		border-radius: 12px;
		background: color-mix(in srgb, $bg-color 82%, transparent);
		box-shadow: 0 8px 24px rgb(0 0 0 / 18%);
		backdrop-filter: blur(5px);
		text-align: center;

		.back-link {
			display: inline-flex;
			align-items: center;
			gap: 6px;
			width: max-content;
			padding: 5px 10px;
			border: 1px solid color-mix(in srgb, $text-color 18%, transparent);
			border-radius: 6px;
			background: color-mix(in srgb, $bg-color 72%, transparent);
			color: $text-color;
			text-decoration: none;

			&:hover {
				border-color: $accent-color-hover;
				color: $accent-color-hover;
			}

			:global(svg) {
				transform: rotate(180deg);
			}
		}

		h1 {
			margin: 0;
			font-size: clamp(28px, 5vw, 44px);
		}
	}

	.eyebrow {
		display: block;
		margin-bottom: 4px;
		color: $accent-color-hover;
		font-size: 12px;
		font-weight: bold;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.session-card,
	.rubric-panel,
	.candidate-card {
		width: 100%;
		max-width: 760px;
		padding: 24px;
		align-self: center;
		border: 1px solid color-mix(in srgb, $text-color 20%, transparent);
		border-radius: 12px;
		background: color-mix(in srgb, $bg-color 88%, $accent-color);
	}

	.session-card {
		p {
			max-width: 650px;
			margin: 8px 0 0;
			color: $placeholder-color;
			line-height: 1.5;
		}

		h2 {
			margin: 0;
			font-size: clamp(24px, 5vw, 36px);
		}
	}

	.initial-card {
		max-width: 680px;
	}

	.confirm-dialog {
		display: flex;
		flex-flow: column;
		gap: 12px;

		p {
			margin: 0;
			line-height: 1.5;
		}
	}

	.confirm-score {
		display: flex;
		align-items: baseline;
		justify-content: center;
		gap: 10px;
		padding: 14px;
		border-radius: 8px;
		background: color-mix(in srgb, $accent-color 14%, $bg-color);

		span,
		small {
			color: $placeholder-color;
			font-size: 13px;
		}

		strong {
			color: $accent-color-hover;
			font-size: 30px;
		}
	}

	.confirm-actions {
		justify-content: flex-end;
		margin-top: 8px;

		button {
			min-width: 0;
		}
	}

	.initial-score,
	.suggested-score {
		margin: 28px 0 20px;
		font-size: clamp(56px, 12vw, 92px);
		font-weight: bold;
		line-height: 0.9;
		text-align: center;
	}

	.suggested-score {
		margin-bottom: 10px;
		color: $accent-color-hover;
	}

	.card-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		margin-top: 24px;

		button {
			width: max-content;
			min-width: 150px;
		}
	}

	.review-actions {
		justify-content: center;
		flex-wrap: wrap;
	}

	.internal-score {
		margin: 0;
		color: $placeholder-color;
		font-size: 13px;
		text-align: center;
	}

	.initial-score small {
		font-size: 24px;
		font-weight: normal;
	}

	.range-label > span,
	.category-control label > span {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
	}

	.category-control small {
		color: $placeholder-color;
		font-size: 13px;
	}

	.rubric-intro,
	.rubric-guidance,
	.category-control p {
		color: $placeholder-color;
		font-size: 14px;
		line-height: 1.45;
	}

	.range-label,
	.category-control label {
		display: block;
		font-size: 15px;
	}

	.range-label input,
	.category-control input {
		width: 100%;
		margin-top: 10px;
		accent-color: $accent-color-hover;
	}

	.rubric-toggle {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 0;
		color: $text-color;
		font-size: 17px;

		& > span {
			display: flex;
			align-items: center;
			gap: 8px;
		}
	}

	.rubric-intro {
		margin: 15px 0;
	}

	.rubric-note {
		margin: 0 0 8px;
		color: $accent-color-hover;
		font-size: 14px;
		line-height: 1.45;
	}

	.category-control {
		padding: 13px 0;
		border-top: 1px solid color-mix(in srgb, $text-color 12%, transparent);

		p {
			margin: 5px 0 0;
		}
	}

	.rubric-guidance {
		margin: 15px 0 0;
		font-weight: bold;
	}

	.candidate-card {
		position: relative;
		display: grid;
		grid-template-columns: 130px minmax(0, 1fr);
		gap: 20px;
		min-height: 230px;
		overflow: hidden;
		transition:
			border-color 150ms ease,
			background-color 150ms ease;

		& > img,
		.poster-placeholder {
			width: 130px;
			height: 195px;
			border-radius: 6px;
			object-fit: cover;
		}

		.poster-placeholder {
			display: flex;
			align-items: center;
			justify-content: center;
			background: color-mix(in srgb, $text-color 15%, $bg-color);
			color: $placeholder-color;
		}
	}

	.candidate-source {
		position: absolute;
		top: 8px;
		left: 8px;
		padding: 3px 6px;
		border-radius: 4px;
		background: $bg-color;
		color: $placeholder-color;
		font-size: 10px;
		font-weight: bold;
		text-transform: uppercase;
	}

	.candidate-info {
		min-width: 0;

		.comparison-position {
			display: block;
			margin: 2px 0 7px;
			color: $accent-color-hover;
			font-size: 12px;
			font-weight: bold;
			letter-spacing: 0.04em;
			text-transform: uppercase;
		}

		h3 {
			margin: 0 0 4px;
			font-size: 22px;
		}
	}

	.comparison-help {
		margin-bottom: 20px !important;
	}

	.compare-prompt {
		margin: 0 0 12px;
		font-size: 16px;
		line-height: 1.45;

		span {
			color: $placeholder-color;
		}
	}

	fieldset {
		padding: 0;
		border: 0;

		legend {
			position: absolute;
			width: 1px;
			height: 1px;
			overflow: hidden;
			clip: rect(0 0 0 0);
		}
	}

	.decision-buttons {
		display: flex;
		flex-flow: column;
		gap: 5px;

		button {
			display: flex;
			flex-flow: column;
			align-items: flex-start;
			justify-content: center;
			width: 100%;
			padding: 9px 11px;
			border: 1px solid color-mix(in srgb, $text-color 20%, transparent);
			border-radius: 6px;
			color: $text-color;
			font-size: 13px;
			text-align: left;

			small {
				margin-top: 2px;
				color: $placeholder-color;
				font-size: 11px;
			}

			&:hover {
				border-color: $accent-color-hover;
				background: $accent-color-hover;
				color: $bg-color;

				small {
					color: color-mix(in srgb, $bg-color 75%, transparent);
				}
			}
		}
	}

	.comparison-navigation {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 10px;
		min-height: 40px;

		button {
			width: max-content;
			min-width: 150px;
		}

		.secondary {
			border: 1px solid $text-color;
			color: $text-color;
		}
	}

	.empty-comparisons {
		display: flex;
		flex-flow: column;
		align-items: center;
		gap: 10px;
		padding: 40px 20px;
		border: 1px dashed color-mix(in srgb, $text-color 25%, transparent);
		border-radius: 10px;
		color: $placeholder-color;
		text-align: center;
	}

	.loading-more {
		display: flex;
		justify-content: center;
	}

	.zero-warning {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		border: 1px solid $warn;
		border-radius: 8px;
		background: color-mix(in srgb, $warn 12%, $bg-color);
		color: $text-color;
		font-size: 14px;
	}

	@media screen and (max-width: 560px) {
		.session-card {
			padding: 18px;
		}

		.candidate-card {
			grid-template-columns: 84px minmax(0, 1fr);
			gap: 14px;
			padding: 14px;

			& > img,
			.poster-placeholder {
				width: 84px;
				height: 126px;
			}
		}

		.comparison-navigation {
			button {
				min-width: 0;
				flex: 1;
			}
		}
	}
</style>
