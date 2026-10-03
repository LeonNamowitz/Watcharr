import type { PublicUser } from "@/types";

export interface StatsMediaCard {
	id: number;
	type: "movie" | "tv";
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
	label: string;
	count: number;
}
export interface StatsResponse {
	scope: "year" | "lifetime";
	year?: number;
	media: "movie" | "tv";
	owner: PublicUser;
	reviewsVisible: boolean;
	availableYears: number[];
	summary: {
		titles: number;
		movies: number;
		shows: number;
		plays: number;
		averageRating: number;
	};
	history: {
		year: number;
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
	highestRated: { current: StatsMediaCard[]; older: StatsMediaCard[] };
	activity: {
		weeks: {
			start: string;
			plays: number;
			uniqueTitles: number;
			averageRating: number;
			titles: string[];
		}[];
		months: { month: string; plays: number; averageRating: number }[];
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
	tooltipLabel?: string;
	label: string;
	value: number | null;
	detail?: string;
}
