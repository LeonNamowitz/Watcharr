import { browser } from "$app/environment";

export type PersonPageState = {
	sortOption: string;
	creditsType: string;
	onMyListFilter: boolean;
};

const storageKeyPrefix = "person-page-state";

function getStorageKey(personId: number) {
	return `${storageKeyPrefix}:${personId}`;
}

export function readPersonPageState(
	personId: number,
): PersonPageState | undefined {
	if (!browser) {
		return undefined;
	}

	const rawState = sessionStorage.getItem(getStorageKey(personId));
	if (!rawState) {
		return undefined;
	}

	try {
		return JSON.parse(rawState) as PersonPageState;
	} catch {
		return undefined;
	}
}

export function savePersonPageState(personId: number, state: PersonPageState) {
	if (!browser) {
		return;
	}

	sessionStorage.setItem(getStorageKey(personId), JSON.stringify(state));
}
