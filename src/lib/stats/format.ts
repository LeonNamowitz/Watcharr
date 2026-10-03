import { RatingSystem } from "@/types";
import type { RatingSettings } from "@/lib/rating/helpers";

export type StatsUnit =
	| "titles"
	| "watches"
	| "episodes"
	| "games"
	| "progress events"
	| "completions"
	| "hours";

const singularUnits: Record<StatsUnit, string> = {
	titles: "title",
	watches: "watch",
	episodes: "episode",
	games: "game",
	"progress events": "progress event",
	completions: "completion",
	hours: "hour",
};

export function statsUnitLabel(unit: StatsUnit, count: number) {
	return count === 1 ? singularUnits[unit] : unit;
}

// Saved ratings use a common /10 scale; missing and unrated values are excluded.
export function meanRating(values: readonly (number | null | undefined)[]) {
	const ratings = values.filter((value): value is number => (value ?? 0) > 0);
	return ratings.length
		? ratings.reduce((sum, rating) => sum + rating, 0) / ratings.length
		: 0;
}

export function decimal(value: number) {
	return value.toLocaleString(undefined, {
		minimumFractionDigits: 1,
		maximumFractionDigits: 1,
	});
}

// An average is continuous; individual saved-rating step preferences do not apply.
export function averageRating(value: number, settings?: RatingSettings) {
	if (!value) return "Unrated";
	if (settings?.ratingSystem === RatingSystem.OutOf100)
		return `${decimal(value * 10)}/100`;
	if (settings?.ratingSystem === RatingSystem.OutOf5)
		return `${decimal(value / 2)}/5`;
	return `${decimal(value)}/10`;
}
