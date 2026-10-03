import { RatingSystem } from "@/types";
import type { RatingSettings } from "@/lib/rating/helpers";

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
