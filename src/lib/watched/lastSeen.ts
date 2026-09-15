import type { Activity, Watched } from "@/types";

const finishedStatusActivityTypes = new Set([
	"STATUS_CHANGED",
	"STATUS_CHANGED_AUTO",
	"SEASON_ADDED",
	"SEASON_ADDED_AUTO",
	"SEASON_ADDED_JF",
	"SEASON_ADDED_PLEX",
	"SEASON_STATUS_CHANGED",
	"SEASON_STATUS_CHANGED_AUTO",
	"EPISODE_ADDED",
	"EPISODE_ADDED_JF",
	"EPISODE_ADDED_PLEX",
	"EPISODE_STATUS_CHANGED",
]);

function hasFinishedStatus(activity: Activity) {
	if (!finishedStatusActivityTypes.has(activity.type)) return false;
	if (activity.data.trim().replace(/^"|"$/g, "") === "FINISHED") return true;
	try {
		return JSON.parse(activity.data)?.status === "FINISHED";
	} catch {
		return false;
	}
}

export function getLastSeen(watched?: Watched) {
	// Detail responses include activity, which is the live source of truth as
	// activities are edited or removed locally. Poster/list responses use the
	// server-derived field instead.
	let lastSeen = watched?.activity ? undefined : watched?.lastSeen;
	let latest = lastSeen ? Date.parse(lastSeen) : Number.NaN;
	for (const activity of watched?.activity ?? []) {
		if (!activity.countAsPlay && !hasFinishedStatus(activity)) continue;
		const effectiveDate = activity.customDate ?? activity.createdAt;
		const timestamp = Date.parse(effectiveDate);
		if (
			!Number.isNaN(timestamp) &&
			(Number.isNaN(latest) || timestamp > latest)
		) {
			lastSeen = effectiveDate;
			latest = timestamp;
		}
	}
	return lastSeen;
}

export function formatLastSeen(value?: string) {
	if (!value) return;
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return;
	return `${String(date.getDate()).padStart(2, "0")}.${String(
		date.getMonth() + 1,
	).padStart(2, "0")}.${date.getFullYear()}`;
}
