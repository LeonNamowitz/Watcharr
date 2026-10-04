import type { PublicUser } from "@/types";
import type { StatsMedia, StatsResponse } from "./types";

// Render the usual loading UI before the first response, without another owner's
// data or invented totals. StatsPage keeps this content inert while loading.
export function loadingStats(
	owner: PublicUser,
	year: string,
	media: StatsMedia,
): StatsResponse {
	return {
		owner,
		media,
		scope: year === "all" ? "lifetime" : "year",
		year: year === "all" ? undefined : Number(year),
		reviewsVisible: false,
		availableYears: [],
		summary: {
			titles: 0,
			movies: 0,
			shows: 0,
			plays: 0,
			averageRating: 0,
		},
		history: [],
		decades: [],
		episodes: [],
		highestRatedEpisodes: { current: [], older: [] },
		highestRated: { current: [], older: [] },
		activity: {
			total: 0,
			weeks: [],
			months: [],
			averagePerWeek: 0,
			averagePerMonth: 0,
		},
		milestones: { mostWatched: [] },
		genres: [],
		countries: [],
		languages: [],
		breakdown: {
			release: [],
			plays: [],
			reviews: null,
			ratingDistribution: [],
			watchlistAdditions: 0,
			watchlistTitles: [],
		},
		people: { cast: [], directors: [] },
		studios: [],
		crew: [],
		highsLows: {},
		ratingDifferences: { higher: [], lower: [], average: 0 },
		posters: [],
		watchlist: [],
		metadata: { partial: false, failedTitles: [] },
	};
}
