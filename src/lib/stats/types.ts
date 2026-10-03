import type { PublicUser } from "@/types";

export type StatsMedia = "movie" | "tv" | "game";

export interface StatsMediaCard {
	communityRating?: number;
	coverId?: string;
	playtimeHours?: number;
	id: number;
	episodeName?: string;
	stillPath?: string;
	seasonNumber?: number;
	episodeNumber?: number;
	type: StatsMedia;
	title: string;
	posterPath?: string;
	releaseYear?: number;
	rating?: number;
	tmdbRating?: number;
	voteCount?: number;
	plays?: number;
	runtime?: number;
	date?: string;
}
export interface StatsPerson {
	titleKeys: string[];
	id: number;
	name: string;
	profilePath?: string;
	titles: number;
	averageRating: number;
}
export interface StatsBar {
	titleKeys: string[];
	label: string;
	count: number;
	averageRating: number;
}
export interface StatsPie {
	titleKeys?: string[] | null;
	label: string;
	count: number;
}
export interface StatsResponse {
	library?: StatsLibrary;
	calendar?: StatsDay[];
	games?: StatsGames;
	scope: "year" | "lifetime";
	year?: number;
	media: StatsMedia;
	owner: PublicUser;
	reviewsVisible: boolean;
	availableYears: number[];
	summary: {
		hours?: number;
		titles: number;
		games?: number;
		completed?: number;
		movies: number;
		shows: number;
		plays: number;
		averageRating: number;
	};
	history: {
		items: StatsMediaCard[];
		reviewedTitleKeys?: string[];
		completedTitleKeys?: string[];
		year: number;
		games?: number;
		completed?: number;
		movies: number;
		shows: number;
		titles: number;
		averageRating: number;
		reviewed?: number;
	}[];
	decades: {
		decade: number;
		titles: number;
		averageRating: number;
		items: StatsMediaCard[];
	}[];
	episodes: StatsMediaCard[];
	highestRatedEpisodes: { current: StatsMediaCard[]; older: StatsMediaCard[] };
	highestRated: { current: StatsMediaCard[]; older: StatsMediaCard[] };
	activity: {
		total: number;
		weeks: {
			items: StatsMediaCard[] | null;
			start: string;
			plays: number;
			uniqueTitles: number;
			averageRating: number;
			titles: string[];
		}[];
		months: {
			items: StatsMediaCard[] | null;
			month: string;
			plays: number;
			averageRating: number;
		}[];
		averagePerWeek: number;
		averagePerMonth: number;
	};
	milestones: {
		first?: StatsMediaCard;
		last?: StatsMediaCard;
		mostWatched: StatsMediaCard[];
	};
	genres: StatsBar[];
	countries: StatsBar[];
	languages: StatsBar[];
	breakdown: {
		release: StatsPie[];
		plays: StatsPie[];
		reviews: StatsPie[] | null;
		ratingDistribution: { rating: number; count: number }[];
		watchlistAdditions: number;
		watchlistTitles: StatsMediaCard[];
	};
	people: { cast: StatsPerson[]; directors: StatsPerson[] };
	studios: StatsPerson[];
	crew: {
		department: string;
		jobs: { job: string; people: StatsPerson[] }[];
	}[];
	highsLows: {
		highestTMDBRated?: StatsMediaCard;
		highestCommunityRated?: StatsMediaCard;
		mostPlaytime?: StatsMediaCard;
		leastPlaytime?: StatsMediaCard;
		lowestRated?: StatsMediaCard;
		mostVoted?: StatsMediaCard;
		leastVoted?: StatsMediaCard;
		newest?: StatsMediaCard;
		oldest?: StatsMediaCard;
		longest?: StatsMediaCard;
		shortest?: StatsMediaCard;
	};
	ratingDifferences: {
		higher: StatsMediaCard[] | null;
		lower: StatsMediaCard[] | null;
		average: number;
	};
	posters: StatsMediaCard[];
	watchlist: StatsMediaCard[];
	metadata: { partial: boolean; failedTitles: string[] };
}
export interface ChartPoint {
	titleKeys?: string[];
	items?: StatsMediaCard[];
	tooltipLabel?: string;
	label: string;
	value: number | null;
	titleCount: number;
	averageRating: number;
	detail?: string;
}

export interface StatsGames {
	platforms: StatsBar[];
	modes: StatsBar[];
	themes: StatsBar[];
	perspectives: StatsBar[];
	developers: StatsBar[];
	publishers: StatsBar[];
	statuses: StatsPie[];
	completion: StatsPie[];
	completionPercentage: number;
	completions: StatsResponse["activity"];
	playtime?: {
		totalHours: number;
		averageHours: number;
		medianHours: number;
		recordedGames: number;
		mostPlayed: StatsMediaCard[];
		distribution: StatsPie[];
		byRating: { rating: number; hours: number; items: StatsMediaCard[] }[];
	};
}

export interface StatsLibrary {
	statuses: {
		status: string;
		label: string;
		count: number;
		items: StatsMediaCard[];
	}[];
	momentum: {
		period: string;
		planned: StatsMediaCard[];
		watched: StatsMediaCard[];
	}[];
	planned: number;
	watched: number;
	waiting: {
		medianDays: number | null;
		excluded: number;
		buckets: { label: string; items: StatsMediaCard[] }[];
		longest: {
			item: StatsMediaCard;
			days: number;
			plannedDate: string;
			watchedDate: string;
		}[];
	};
}
export interface StatsDay {
	date: string;
	plays: number;
	items: StatsMediaCard[];
}
export interface StatsSelection {
	label: string;
	items: StatsMediaCard[];
	description?: string;
	period?: string;
}
