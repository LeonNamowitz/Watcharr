import type { SupportedMedia, Watched, WatchedStatus } from "@/types";
import { getLastSeen } from "../watched/lastSeen";

export type PosterExtraDetails = {
	rating: number | undefined;
	status: WatchedStatus | undefined;
	dateAdded?: string;
	dateLastSeen?: string;
	dateModified?: string;
	progress?: string;
};

export function buildExtraDetails(
	t: SupportedMedia | undefined,
	w: Watched,
): PosterExtraDetails {
	let progress: string | undefined;
	if (t === "tv") {
		progress = w.watchingSeason;
	} else if (t === "game" && typeof w.playtimeHours === "number") {
		progress = `${w.playtimeHours} ${w.playtimeHours === 1 ? "hour" : "hours"}`;
	}

	const obj = {
		rating: w.rating,
		status: w.status,
		dateAdded: w.createdAt,
		dateLastSeen: getLastSeen(w),
		dateModified: w.updatedAt,
		progress,
	} as PosterExtraDetails;
	return obj;
}
