import { MediaTypeE, RatingStep, RatingSystem, type Media } from "@/types";
import type { RatingSettings } from "./helpers";

export type RatingHelperMediaType = "movie" | "tv" | "game";
export type RatingHelperCategory = "content" | "technical" | "design" | "bias";
export type ComparisonDecision = "better" | "same" | "worse" | "skip";
export type ComparisonSide = "above" | "below";

export interface RatingRubricCategory {
	id: RatingHelperCategory;
	label: string;
	max: number;
	weight: number;
	description: string;
}

export interface RatingRubric {
	categories: RatingRubricCategory[];
	intro: string;
	guidance: string;
}

export interface RatingComparison {
	rating: number;
	decision: ComparisonDecision;
}

export interface ScoreEvaluation {
	baseScore100: number;
	score100: number;
	sameAnchor?: number;
	lowerBound100: number;
	upperBound100: number;
	comparisonCount: number;
}

const rubrics: Record<RatingHelperMediaType, RatingRubric> = {
	movie: {
		categories: [
			{
				id: "content",
				label: "Content",
				max: 10,
				weight: 4,
				description: "Story, characters, themes, screenplay, and ideas.",
			},
			{
				id: "technical",
				label: "Technical",
				max: 10,
				weight: 4,
				description:
					"Cinematography, editing, sound design, acting, music, and craft.",
			},
			{
				id: "bias",
				label: "Personal bias",
				max: 10,
				weight: 2,
				description: "Your personal taste, attachment, mood, and preferences.",
			},
		],
		intro:
			"Score each category from 0–10, then weighted as 40% content, 40% technical, and 20% personal bias.",
		guidance:
			"Around 70 is generally a good movie. 80+ usually means great elements and a broadly recommended, fantastic movie.",
	},
	tv: {
		categories: [
			{
				id: "content",
				label: "Content",
				max: 10,
				weight: 4,
				description: "Story, characters, themes, screenplay, and ideas.",
			},
			{
				id: "technical",
				label: "Technical",
				max: 10,
				weight: 4,
				description:
					"Cinematography, editing, sound design, acting, music, and craft.",
			},
			{
				id: "bias",
				label: "Personal bias",
				max: 10,
				weight: 2,
				description: "Your personal taste, attachment, mood, and preferences.",
			},
		],
		intro:
			"Score each category from 0–10, then weighted as 40% content, 40% design, and 20% personal bias.",
		guidance:
			"Around 70 is generally a good show. 80+ usually means great elements and a broadly recommended, fantastic show.",
	},
	game: {
		categories: [
			{
				id: "content",
				label: "Content",
				max: 10,
				weight: 4,
				description:
					"Story, characters, world, art direction, music, and sound.",
			},
			{
				id: "design",
				label: "Design",
				max: 10,
				weight: 4,
				description:
					"Gameplay, features, flow, level design, controls, and usability.",
			},
			{
				id: "bias",
				label: "Personal bias",
				max: 10,
				weight: 2,
				description: "Your personal taste, attachment, mood, and preferences.",
			},
		],
		intro:
			"Score each category from 0–10, then weighted as 40% content, 40% design, and 20% personal bias.",
		guidance:
			"Around 70 is generally a good game. 80+ usually means great elements and a broadly recommended, fantastic game.",
	},
};

export function getRatingRubric(type: RatingHelperMediaType): RatingRubric {
	return rubrics[type];
}

export function clampScore100(score: number): number {
	return Math.max(0, Math.min(100, score));
}

/** Convert Watcharr's stored 0–10 value to the helper's 0–100 scale. */
export function toCanonicalScore100(rating?: number): number | undefined {
	if (typeof rating !== "number" || !Number.isFinite(rating) || rating <= 0) {
		return undefined;
	}
	return clampScore100(rating * 10);
}

/** Convert the helper's 0–100 score to Watcharr's one-decimal 0–10 value. */
export function fromCanonicalScore100(score: number): number {
	return Math.round(clampScore100(score)) / 10;
}

export function getDefaultCategoryValues(
	type: RatingHelperMediaType,
	initialRating10: number,
): Record<RatingHelperCategory, number> {
	const rubric = getRatingRubric(type);
	const normalizedRating = Math.max(0, Math.min(10, initialRating10));
	return Object.fromEntries(
		rubric.categories.map((category) => [
			category.id,
			(category.max / 10) * normalizedRating,
		]),
	) as Record<RatingHelperCategory, number>;
}

export function calculateRubricScore(
	type: RatingHelperMediaType,
	values: Partial<Record<RatingHelperCategory, number>>,
): number {
	const rubric = getRatingRubric(type);
	return clampScore100(
		rubric.categories.reduce(
			(total, category) =>
				total +
				Math.max(0, Math.min(category.max, values[category.id] ?? 0)) *
					category.weight,
			0,
		),
	);
}

/**
 * Return the smallest meaningful increment on the canonical scale for the
 * user's configured rating display. The stored value remains one decimal on
 * the 0–10 scale, but the helper keeps the comparison behavior consistent
 * with the visible rating controls.
 */
export function getCanonicalRatingStep(settings?: RatingSettings): number {
	const system = settings?.ratingSystem ?? RatingSystem.OutOf10;
	const step = settings?.ratingStep ?? RatingStep.One;

	switch (system) {
		case RatingSystem.OutOf100:
			return 1;
		case RatingSystem.OutOf5:
			if (step === RatingStep.Point1) return 2;
			if (step === RatingStep.Point5) return 10;
			return 20;
		case RatingSystem.Thumbs:
			return 1;
		case RatingSystem.OutOf10:
		default:
			if (step === RatingStep.Point1) return 1;
			if (step === RatingStep.Point5) return 5;
			return 10;
	}
}

/**
 * Walk the comparisons in decision order. Better can only keep or raise the
 * current score; worse can only keep or lower it; same resets it to the known
 * rating. When a new decision crosses an older bound, the newer decision wins
 * and the stale opposing bound is discarded.
 */
export function evaluateComparisonScore(
	baseScore100: number,
	comparisons: RatingComparison[],
	step100: number,
): ScoreEvaluation {
	const safeBase = clampScore100(baseScore100);
	const safeStep = Math.max(1, step100);
	let score100 = safeBase;
	let sameAnchor: number | undefined;
	let comparisonCount = 0;
	let lowerBound = 0;
	let upperBound = 100;

	for (const comparison of comparisons) {
		const rating100 = toCanonicalScore100(comparison.rating);
		if (rating100 === undefined || comparison.decision === "skip") continue;
		comparisonCount += 1;

		switch (comparison.decision) {
			case "better": {
				const nextLowerBound = clampScore100(rating100 + safeStep);
				if (nextLowerBound > upperBound) {
					// The newest answer supersedes an incompatible older upper bound.
					upperBound = 100;
					lowerBound = nextLowerBound;
				} else {
					lowerBound = Math.max(lowerBound, nextLowerBound);
				}
				score100 = Math.max(score100, lowerBound);
				break;
			}
			case "same":
				sameAnchor = rating100;
				score100 = rating100;
				lowerBound = 0;
				upperBound = 100;
				break;
			case "worse": {
				const nextUpperBound = clampScore100(rating100 - safeStep);
				if (nextUpperBound < lowerBound) {
					// The newest answer supersedes an incompatible older lower bound.
					lowerBound = 0;
					upperBound = nextUpperBound;
				} else {
					upperBound = Math.min(upperBound, nextUpperBound);
				}
				score100 = Math.min(score100, upperBound);
				break;
			}
		}
	}

	return {
		baseScore100: safeBase,
		score100: clampScore100(score100),
		sameAnchor,
		lowerBound100: lowerBound,
		upperBound100: upperBound,
		comparisonCount,
	};
}

/**
 * Choose the rating to probe next during precision rounds. A one-sided result
 * explores beyond the compared item; established bounds use their midpoint.
 */
export function getAdaptiveComparisonProbe(
	evaluation: ScoreEvaluation,
	lastComparison: RatingComparison | undefined,
	step100: number,
	sameSide: ComparisonSide = "above",
): number {
	if (!lastComparison || lastComparison.decision === "skip") {
		return evaluation.score100;
	}

	const rating100 = toCanonicalScore100(lastComparison.rating);
	if (rating100 === undefined) return evaluation.score100;

	const explorationStep = Math.max(1, step100, 3);
	const hasLowerBound = evaluation.lowerBound100 > 0;
	const hasUpperBound = evaluation.upperBound100 < 100;
	if (hasLowerBound && hasUpperBound) {
		return clampScore100(
			(evaluation.lowerBound100 + evaluation.upperBound100) / 2,
		);
	}

	switch (lastComparison.decision) {
		case "better":
			return clampScore100(
				Math.max(evaluation.score100, rating100 + explorationStep),
			);
		case "worse":
			return clampScore100(
				Math.min(evaluation.score100, rating100 - explorationStep),
			);
		case "same":
			return clampScore100(
				rating100 + (sameSide === "above" ? explorationStep : -explorationStep),
			);
	}
}

export interface NearScorePickOptions {
	maximumDistance100: number;
	side?: ComparisonSide;
	sidePivot100?: number;
	tieSide?: ComparisonSide;
	random?: () => number;
}

/** Pick the remaining rated item closest to an adaptive probe score. */
export function pickComparisonCandidateNearScore(
	items: Media[],
	probeScore100: number,
	options: NearScorePickOptions,
): Media | undefined {
	const sidePivot100 = options.sidePivot100 ?? probeScore100;
	const ranked = items
		.flatMap((candidate, index) => {
			const rating100 = toCanonicalScore100(candidate.watched?.rating);
			if (rating100 === undefined) return [];
			return [
				{
					candidate,
					index,
					rating100,
					distance: Math.abs(rating100 - probeScore100),
				},
			];
		})
		.filter(
			({ distance, rating100 }) =>
				distance <= options.maximumDistance100 &&
				(!options.side ||
					(options.side === "above"
						? rating100 > sidePivot100
						: rating100 < sidePivot100)),
		)
		.sort(
			(a, b) =>
				a.distance - b.distance ||
				(options.tieSide === "above"
					? b.rating100 - a.rating100
					: options.tieSide === "below"
						? a.rating100 - b.rating100
						: a.index - b.index),
		);
	if (ranked.length === 0) return undefined;

	const bestDistance = ranked[0]?.distance ?? 0;
	const shortlist = ranked.filter(({ distance }) => distance === bestDistance);
	if (options.tieSide) return shortlist[0]?.candidate;
	const randomValue = Math.max(
		0,
		Math.min(0.999999, (options.random ?? Math.random)()),
	);
	return shortlist[Math.floor(randomValue * shortlist.length)]?.candidate;
}

const comparisonOffsetSchedule = [18, 12, 7, 4, 2, 1, 1, 1];

/**
 * Pick one candidate for the next comparison. The first pick is randomized
 * among the best few wide-offset matches; later picks use the current score
 * and the offset schedule to narrow the comparison.
 */
export function pickComparisonCandidate(
	items: Media[],
	baseScore100: number,
	comparisonNumber: number,
	side: ComparisonSide,
	random: () => number = Math.random,
): Media | undefined {
	if (items.length === 0) return undefined;

	const safeBase = clampScore100(baseScore100);
	const desiredOffset =
		comparisonOffsetSchedule[
			Math.min(comparisonNumber, comparisonOffsetSchedule.length - 1)
		] ?? 0;
	const ranked = items
		.map((candidate, index) => {
			const rating = toCanonicalScore100(candidate.watched?.rating) ?? safeBase;
			const offset = Math.abs(rating - safeBase);
			return {
				candidate,
				index,
				offset,
				distance: Math.abs(offset - desiredOffset),
				hasOffset: offset > 0,
				isPreferredSide:
					side === "above" ? rating > safeBase : rating < safeBase,
			};
		})
		.sort((a, b) => a.distance - b.distance || a.index - b.index);
	const maximumOffset =
		comparisonNumber === 0 ? 28 : Math.max(5, desiredOffset * 2);
	const nearby = ranked.filter(({ offset }) => offset <= maximumOffset);
	if (nearby.length === 0) return undefined;

	const bestDistance = nearby[0]?.distance ?? 0;
	const sideTolerance = Math.max(2, desiredOffset / 2);
	const withPreferredSide = nearby.filter(
		({ distance, isPreferredSide }) =>
			isPreferredSide && distance <= bestDistance + sideTolerance,
	);
	const sideCandidates =
		withPreferredSide.length > 0 ? withPreferredSide : nearby;
	const withOffset =
		comparisonNumber === 0
			? sideCandidates.filter(({ hasOffset }) => hasOffset)
			: sideCandidates;
	const candidates = withOffset.length > 0 ? withOffset : sideCandidates;
	const shortlist = candidates.slice(0, comparisonNumber === 0 ? 3 : 1);
	const randomValue = Math.max(0, Math.min(0.999999, random()));
	return shortlist[Math.floor(randomValue * shortlist.length)]?.candidate;
}

export function mediaMatchesHelperType(
	media: Media,
	type: RatingHelperMediaType,
): boolean {
	if (type === "movie") return media.type === MediaTypeE.tmdbMovie;
	if (type === "tv") return media.type === MediaTypeE.tmdbShow;
	return media.type === MediaTypeE.igdbGame;
}

export function getMediaKey(media: Media): string {
	const id = media.ids.tmdb ?? media.ids.igdb;
	return `${media.type ?? "unknown"}:${id ?? media.name ?? "unknown"}`;
}

export function isRatedMedia(media: Media): boolean {
	return typeof media.watched?.rating === "number" && media.watched.rating > 0;
}

export function getRatedSimilarCandidates(
	media: Media,
	type: RatingHelperMediaType,
	excludedKeys: Set<string>,
): Media[] {
	const seen = new Set(excludedKeys);
	return (media.similar ?? []).filter((candidate) => {
		if (!mediaMatchesHelperType(candidate, type) || !isRatedMedia(candidate)) {
			return false;
		}
		const key = getMediaKey(candidate);
		if (seen.has(key)) return false;
		seen.add(key);
		return true;
	});
}

export function rankRecentCandidates(
	items: Media[],
	type: RatingHelperMediaType,
	excludedKeys: Set<string>,
): Media[] {
	const seen = new Set(excludedKeys);
	const candidates = items.filter((candidate) => {
		if (!mediaMatchesHelperType(candidate, type) || !isRatedMedia(candidate)) {
			return false;
		}
		const key = getMediaKey(candidate);
		if (seen.has(key)) return false;
		seen.add(key);
		return true;
	});

	return candidates;
}
