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
