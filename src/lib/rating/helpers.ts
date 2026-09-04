import { store } from "@/store.svelte";
import { RatingStep, RatingSystem, type UserSettings } from "@/types";

export type RatingSettings = Pick<UserSettings, "ratingSystem" | "ratingStep">;

/**
 * Used for scaling users 'actual' rating we store in db
 * into one we can show that takes into account their
 * settings for how they want stars displayed.
 * Only used for star ratings, not thumbs.
 */
export function toShowableRating(r?: number, settings?: RatingSettings) {
	if (!r) {
		return 0;
	}
	const ratingSettings = settings ?? store.userSettings;
	if (
		!ratingSettings ||
		(!ratingSettings.ratingSystem && !ratingSettings.ratingStep)
	) {
		return Math.round(r);
	}
	if (ratingSettings.ratingSystem === RatingSystem.OutOf100) {
		return r * 10;
	}
	if (ratingSettings.ratingSystem === RatingSystem.OutOf5) {
		if (ratingSettings.ratingStep === RatingStep.Point5) {
			return Math.ceil((r / 2) * 2) / 2;
		}
		if (ratingSettings.ratingStep === RatingStep.Point1) {
			return Math.round((r / 2) * 10) / 10;
		}
		return Math.round(r / 2);
	}
	if (ratingSettings.ratingSystem === RatingSystem.OutOf10) {
		if (ratingSettings.ratingStep === RatingStep.Point5) {
			return Math.ceil(r * 2) / 2;
		}
		if (ratingSettings.ratingStep === RatingStep.Point1) {
			return r;
		}
		return Math.round(r);
	}
	return Math.round(r);
}

export function toRatingLabel(r?: number, settings?: RatingSettings) {
	if (!r) return "Unrated";

	if (settings?.ratingSystem === RatingSystem.Thumbs) {
		const thumb = toWhichThumb(r);
		if (thumb === -1) return "Disliked";
		if (thumb === 0) return "Mixed";
		if (thumb === 1) return "Liked";
		return "Unrated";
	}

	const rating = toShowableRating(r, settings);
	if (settings?.ratingSystem === RatingSystem.OutOf100) return `${rating}/100`;
	if (settings?.ratingSystem === RatingSystem.OutOf5) return `${rating}/5`;
	return `${rating}/10`;
}

export function toWhichThumb(r?: number) {
	if (!r) {
		return;
	}
	const rr = Math.round(r);
	if (rr > 0 && rr <= 4) {
		return -1;
	} else if (r >= 4 && r <= 7) {
		return 0;
	} else if (r >= 8) {
		return 1;
	}
}
