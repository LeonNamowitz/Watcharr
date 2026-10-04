import type { StatsMedia, StatsResponse } from "./types";

export const statsSections = [
	{ id: "library-momentum", label: "Watchlist momentum" },
	{ id: "library-waiting", label: "Time on your watchlist" },
	{ id: "history", label: "Through the years", scope: "lifetime" },
	{ id: "decades", label: "Highest-rated decades", scope: "lifetime" },
	{ id: "highest-rated", label: "Highest rated titles" },
	{
		id: "highest-rated-episodes",
		label: "Highest rated episodes",
		media: "tv",
	},
	{
		id: "playtime",
		label: "Lifetime game stats",
		media: "game",
		scope: "lifetime",
	},
	{ id: "activity", label: "Activity" },
	{ id: "calendar", label: "Activity calendar" },
	{ id: "milestones", label: "Milestones" },
	{ id: "categories", label: "Genres, countries & languages" },
	{ id: "breakdown", label: "Breakdown" },
	{ id: "companies", label: "Developers & publishers", media: "game" },
	{ id: "people", label: "People behind the films & shows", media: "screen" },
	{ id: "highs-lows", label: "Highs and lows" },
	{ id: "rated-higher", label: "Rated higher than average" },
	{ id: "rated-lower", label: "Rated lower than average" },
	{ id: "titles", label: "Titles watched" },
	{ id: "watchlist", label: "Highly rated, yet to see" },
] as const;

export type StatsSectionId = (typeof statsSections)[number]["id"];

export function sectionsForMedia(media: StatsMedia) {
	return statsSections.filter(
		(section) =>
			!("media" in section) ||
			section.media === media ||
			(section.media === "screen" && media !== "game"),
	);
}

export function normalizeSectionOrder(
	media: StatsMedia,
	saved: readonly string[] = [],
): StatsSectionId[] {
	const remaining = new Set(
		sectionsForMedia(media).map((section) => section.id),
	);
	return [...saved, ...remaining].filter((id): id is StatsSectionId => {
		if (!remaining.has(id as StatsSectionId)) return false;
		remaining.delete(id as StatsSectionId);
		return true;
	});
}

export function sectionLabel(id: StatsSectionId, data: StatsResponse) {
	const game = data.media === "game";
	const labels: Partial<Record<StatsSectionId, string>> = {
		"library-momentum": `${game ? "Backlog" : "Watchlist"} momentum`,
		"library-waiting": `Time ${game ? "in your backlog" : "on your watchlist"}`,
		"highest-rated": `Highest rated ${game ? "games" : data.media === "tv" ? "shows" : "films"}`,
		categories: game ? "Genres & play styles" : "Genres, countries & languages",
		calendar: `${game ? "Gaming" : "Viewing"} calendar`,
		people: `People behind the ${data.media === "tv" ? "shows" : "films"}`,
		titles: game
			? "Games played"
			: `${data.media === "tv" ? "Shows" : "Films"} watched`,
		watchlist: `Highly rated, ${game ? "yet to play" : "yet to see"}`,
	};
	return (
		labels[id] ?? statsSections.find((section) => section.id === id)!.label
	);
}

export function sectionAvailability(id: StatsSectionId, data: StatsResponse) {
	const section = statsSections.find((section) => section.id === id)!;
	if ("scope" in section && section.scope !== data.scope)
		return "Lifetime only";
	if (id === "activity" && data.media !== "game" && data.scope === "lifetime")
		return "Year views only";
	if ((id.startsWith("library-") || id === "calendar") && !data.library)
		return "Unavailable in this view";
	if (
		data.media === "game" &&
		!data.games &&
		["activity", "categories", "companies", "playtime"].includes(id)
	)
		return "Unavailable in this view";
	if (id === "playtime" && !data.games?.playtime)
		return "Unavailable in this view";
	return undefined;
}
