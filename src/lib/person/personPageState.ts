import { browser } from "$app/environment";

export type PersonSortOption =
	"voteCount" | "newest" | "oldest" | "ownerRating";
export type PersonCreditFilter = "all" | "watched" | "planned" | "missing";

export type PersonPageState = {
	sortOption: PersonSortOption;
	creditsType: string;
	creditFilter: PersonCreditFilter;
};

const storageKeyPrefix = "person-page-state";

function getStorageKey(personId: number, publicOwnerId?: string) {
	return publicOwnerId
		? `${storageKeyPrefix}:public:v2:${publicOwnerId}:${personId}`
		: `${storageKeyPrefix}:${personId}`;
}

function normalizeSortOption(value: unknown): PersonSortOption | undefined {
	switch (value) {
		case "voteCount":
		case "Vote count":
			return "voteCount";
		case "newest":
		case "Newest":
			return "newest";
		case "oldest":
		case "Oldest":
			return "oldest";
		case "ownerRating":
		case "My Rating":
			return "ownerRating";
	}
	return undefined;
}

function normalizeCreditFilter(value: unknown): PersonCreditFilter | undefined {
	switch (value) {
		case "all":
		case "onList":
			return "all";
		case "watched":
		case "planned":
		case "missing":
			return value;
	}
	return undefined;
}

export function readPersonPageState(
	personId: number,
	publicOwnerId?: string,
): PersonPageState | undefined {
	if (!browser) return undefined;

	const rawState = sessionStorage.getItem(
		getStorageKey(personId, publicOwnerId),
	);
	if (!rawState) return undefined;

	try {
		const parsed = JSON.parse(rawState) as {
			sortOption?: unknown;
			creditsType?: unknown;
			creditFilter?: unknown;
		};
		const sortOption = normalizeSortOption(parsed.sortOption);
		if (!sortOption || typeof parsed.creditsType !== "string") return;
		const parsedCreditFilter = normalizeCreditFilter(parsed.creditFilter);
		return {
			sortOption,
			creditsType: parsed.creditsType,
			creditFilter: parsedCreditFilter ?? "all",
		};
	} catch {
		return undefined;
	}
}

export function savePersonPageState(
	personId: number,
	state: PersonPageState,
	publicOwnerId?: string,
) {
	if (!browser) return;

	sessionStorage.setItem(
		getStorageKey(personId, publicOwnerId),
		JSON.stringify(state),
	);
}
