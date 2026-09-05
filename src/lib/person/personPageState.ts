import { browser } from "$app/environment";

export type PersonSortOption =
	"voteCount" | "newest" | "oldest" | "ownerRating";
export type PersonCreditFilter = "watched" | "all" | "planned";

export type PersonPageState = {
	sortOption: PersonSortOption;
	creditsType: string;
	onListFilter: boolean;
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
			onListFilter?: unknown;
			onMyListFilter?: unknown;
			creditFilter?: unknown;
		};
		const sortOption = normalizeSortOption(parsed.sortOption);
		if (!sortOption || typeof parsed.creditsType !== "string") return;
		const onListFilter =
			typeof parsed.onListFilter === "boolean"
				? parsed.onListFilter
				: typeof parsed.onMyListFilter === "boolean"
					? parsed.onMyListFilter
					: false;
		const creditFilter = ["watched", "all", "planned"].includes(
			String(parsed.creditFilter),
		)
			? (parsed.creditFilter as PersonCreditFilter)
			: "all";
		return {
			sortOption,
			creditsType: parsed.creditsType,
			onListFilter,
			creditFilter,
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
