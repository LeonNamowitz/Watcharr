import { SearchType } from "@/types";

export type SelectableSearchType =
	SearchType.movie | SearchType.show | SearchType.game | SearchType.person;

const mediaSearchTypes: SelectableSearchType[] = [
	SearchType.movie,
	SearchType.show,
	SearchType.game,
];

export function parseSearchTypes(value: string | null): SelectableSearchType[] {
	if (!value || value === SearchType.multi) return [];
	const selected = value
		.split(",")
		.map((type) => (type === "tv" ? SearchType.show : type))
		.filter((type): type is SelectableSearchType =>
			[
				SearchType.movie,
				SearchType.show,
				SearchType.game,
				SearchType.person,
			].includes(type as SearchType),
		);
	if (selected.includes(SearchType.person)) return [SearchType.person];
	return mediaSearchTypes.filter((type) => selected.includes(type));
}

export function toggleSearchType(
	selected: SelectableSearchType[],
	type: SelectableSearchType,
): SelectableSearchType[] {
	if (type === SearchType.person) {
		return selected.length === 1 && selected[0] === SearchType.person
			? []
			: [SearchType.person];
	}
	const mediaSelected = selected.includes(SearchType.person) ? [] : selected;
	const next = mediaSelected.includes(type)
		? mediaSelected.filter((selectedType) => selectedType !== type)
		: [...mediaSelected, type];
	return mediaSearchTypes.filter((searchType) => next.includes(searchType));
}

export function searchTypesParam(selected: SelectableSearchType[]) {
	return selected.length > 0 ? selected.join(",") : undefined;
}

export function setSearchTypesOnUrl(
	url: URL,
	selected: SelectableSearchType[],
) {
	const value = searchTypesParam(selected);
	if (value) {
		url.searchParams.set("type", value);
	} else {
		url.searchParams.delete("type");
	}
	if (hasPeopleSearch(selected)) {
		url.searchParams.set("scope", "all");
	}
}

export function hasPeopleSearch(selected: SelectableSearchType[]) {
	return selected.length === 1 && selected[0] === SearchType.person;
}
