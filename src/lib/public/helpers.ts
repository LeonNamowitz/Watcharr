import type { SupportedMedia, WatchedStatus } from "@/types";

export function toPublicStatusLabel(
	status: WatchedStatus,
	mediaType: SupportedMedia,
) {
	switch (status) {
		case "PLANNED":
			return "Planned";
		case "WATCHING":
			return mediaType === "game" ? "Playing" : "Watching";
		case "FINISHED":
			return mediaType === "game" ? "Played" : "Watched";
		case "HOLD":
			return "On hold";
		case "DROPPED":
			return "Dropped";
	}
}

export function toPublicLastSeenLabel(progress?: string) {
	if (!progress) return;

	const episode = /^S(\d+)E(\d+)$/i.exec(progress);
	if (episode) {
		return `Season ${Number(episode[1])}, Episode ${Number(episode[2])}`;
	}
	return progress;
}

export function toPublicLastSeenSeason(progress?: string) {
	if (!progress) return;

	const compact = /^S(\d+)(?:E\d+)?$/i.exec(progress);
	if (compact) return Number(compact[1]);

	const season = /^Season\s+(\d+)$/i.exec(progress);
	return season ? Number(season[1]) : undefined;
}
