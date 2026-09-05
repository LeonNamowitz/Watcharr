import type {
	Filters,
	Follow,
	ImportedList,
	PrivateUser,
	ServerFeatures,
	Tag,
	Theme,
	UserSettings,
	WLDetailedViewOption,
} from "./types";
import type { Notification } from "./lib/util/notify";
import { browser } from "$app/environment";
import { toggleTheme } from "./lib/util/theme";

export const defaultSort = ["DATEADDED", "DOWN"];
export const defaultWLDetailedView: WLDetailedViewOption[] = [
	"statusRating",
	"lastWatched",
];

export type WatchedListPresetId = "watchlist" | "recentlyWatched";

type WatchedListPreset = {
	filters: Filters;
	sort: string[];
};

const cloneFilters = (filters: Filters): Filters => ({
	type: [...(filters.type ?? [])],
	status: [...(filters.status ?? [])],
});

const defaultWatchedListPresets: Record<
	WatchedListPresetId,
	WatchedListPreset
> = {
	watchlist: {
		filters: { type: ["tv", "movie"], status: ["planned", "watching"] },
		sort: defaultSort,
	},
	recentlyWatched: {
		filters: { type: ["tv", "movie"], status: ["finished"] },
		sort: ["LASTFIN", "DOWN"],
	},
};

interface Store {
	userInfo: PrivateUser | undefined;
	userSettings: UserSettings | undefined;
	notifications: Notification[];
	activeSort: string[];
	activeFilters: Filters;
	activeWatchedListPreset: WatchedListPresetId | undefined;
	sortAndFiltersForQueryParams: Record<string, string>;
	appTheme: Theme;
	importedList:
		| {
				data: string;
				type:
					| "text-list"
					| "tmdb"
					| "movary"
					| "watcharr"
					| "myanimelist"
					| "ryot"
					| "todomovies"
					| "imdb";
		  }
		| undefined;
	parsedImportedList: ImportedList[] | undefined;
	searchQuery: string;
	serverFeatures: ServerFeatures | undefined;
	follows: Follow[];
	wlDetailedView: WLDetailedViewOption[];
	tags: Tag[];
}

/**
 * This is our actual (private) store.
 */
const _store: Store = $state({
	notifications: [],
	activeSort: defaultSort,
	activeFilters: { type: [], status: [] },
	activeWatchedListPreset: undefined,
	appTheme: "system",
	sortAndFiltersForQueryParams: {},
	importedList: undefined,
	parsedImportedList: undefined,
	searchQuery: "",
	userInfo: undefined,
	userSettings: undefined,
	serverFeatures: undefined,
	follows: [],
	wlDetailedView: [...defaultWLDetailedView],
	tags: [],
});

type WatchedListStateSnapshot = {
	sort: string[];
	filters: Filters;
	preset: WatchedListPresetId | undefined;
};

let temporaryWatchedListState: WatchedListStateSnapshot | undefined;

const updateSortAndFiltersForQueryParams = () => {
	try {
		const qp: Record<string, string> = {};
		if (store.activeSort?.length === 2) {
			qp.sort = store.activeSort[0];
			qp.sortDir = store.activeSort[1] === "UP" ? "asc" : "desc";
		}
		if (store.activeFilters) {
			const t = store.activeFilters?.type?.join(",");
			if (t) {
				qp["type"] = t;
			}
			const s = store.activeFilters?.status?.join(",");
			if (s) {
				qp["status"] = s;
			}
		}
		_store.sortAndFiltersForQueryParams = qp;
	} catch (err) {
		console.error("updateSortAndFiltersForQueryParams: Failed!", err);
		_store.sortAndFiltersForQueryParams = {};
	}
};

/**
 * Lets a route reuse the watched-list controls without persisting its changes
 * over the user's own list preferences. The returned cleanup restores the
 * original in-memory state when that route group is left.
 */
export const beginTemporaryWatchedListState = () => {
	if (!browser) {
		return () => {};
	}
	if (temporaryWatchedListState) {
		return () => {};
	}
	temporaryWatchedListState = {
		sort: [..._store.activeSort],
		filters: cloneFilters(_store.activeFilters),
		preset: _store.activeWatchedListPreset,
	};

	return () => {
		if (!temporaryWatchedListState) return;
		const original = temporaryWatchedListState;
		temporaryWatchedListState = undefined;
		_store.activeSort = original.sort;
		_store.activeFilters = original.filters;
		_store.activeWatchedListPreset = original.preset;
		updateSortAndFiltersForQueryParams();
	};
};

/**
 * Expose store to app through getters/setters
 * to control what can and can't be accessed.
 * With setters we can easily and more reliably
 * save certain properties to localStorage when
 * they are updated.
 */
export const store = {
	get notifications() {
		return _store.notifications;
	},
	set notifications(v) {
		_store.notifications = v;
	},
	get activeSort() {
		return _store.activeSort;
	},
	set activeSort(v) {
		_store.activeSort = v;
		if (!temporaryWatchedListState) {
			localStorage.setItem("activeFilter", JSON.stringify(v));
		}
		clearActiveWatchedListPreset();
		console.debug("Store: Saved activeSort:", v);
		updateSortAndFiltersForQueryParams();
	},
	get activeFilters() {
		return _store.activeFilters;
	},
	get hasActiveFilters(): boolean {
		return (
			this.activeFilters &&
			(this.activeFilters.status?.length > 0 ||
				this.activeFilters.type?.length > 0)
		);
	},
	set activeFilters(v) {
		_store.activeFilters = v;
		if (!temporaryWatchedListState) {
			localStorage.setItem("activeFilterReal", JSON.stringify(v));
		}
		clearActiveWatchedListPreset();
		console.debug("Store: Saved activeFilters:", v);
		updateSortAndFiltersForQueryParams();
	},
	get activeWatchedListPreset() {
		return _store.activeWatchedListPreset;
	},
	/**
	 * Return our `activeSort` and `activeFilters` in an object
	 * that is in the correct format for our get watched page
	 * requests (object that is given to axios for query params).
	 */
	get sortAndFiltersForQueryParams() {
		return _store.sortAndFiltersForQueryParams;
	},
	get appTheme() {
		return _store.appTheme;
	},
	/**
	 * Only set appTheme through toggleTheme() helper.
	 */
	set appTheme(v) {
		_store.appTheme = v;
		localStorage.setItem("theme", v);
		console.debug("Store: Saved appTheme:", v);
	},
	get importedList() {
		return _store.importedList;
	},
	set importedList(v) {
		_store.importedList = v;
	},
	get parsedImportedList() {
		return _store.parsedImportedList;
	},
	set parsedImportedList(v) {
		_store.parsedImportedList = v;
	},
	get searchQuery() {
		return _store.searchQuery;
	},
	set searchQuery(v) {
		_store.searchQuery = v;
	},
	get userInfo() {
		return _store.userInfo;
	},
	set userInfo(v) {
		_store.userInfo = v;
	},
	get userSettings() {
		return _store.userSettings;
	},
	set userSettings(v) {
		_store.userSettings = v;
	},
	get serverFeatures() {
		return _store.serverFeatures;
	},
	set serverFeatures(v) {
		_store.serverFeatures = v;
	},
	get follows() {
		return _store.follows;
	},
	set follows(v) {
		_store.follows = v;
	},
	get wlDetailedView() {
		return _store.wlDetailedView;
	},
	set wlDetailedView(v) {
		_store.wlDetailedView = v;
		if (v) {
			localStorage.setItem(
				"wlDetailedView",
				JSON.stringify(store.wlDetailedView),
			);
			console.debug("Store: Saved wlDetailedView:", v);
		} else {
			localStorage.removeItem("wlDetailedView");
			console.debug("Store: Removed wlDetailedView");
		}
	},
	get tags() {
		return _store.tags;
	},
	set tags(v) {
		_store.tags = v;
	},
};

/**
 * Reset everything in `store` back to default values.
 */
export const clearAllStores = () => {
	store.notifications = [];
	store.activeSort = defaultSort;
	store.appTheme = "system";
	store.importedList = undefined;
	store.parsedImportedList = undefined;
	store.searchQuery = "";
	store.userInfo = undefined;
	store.userSettings = undefined;
	store.serverFeatures = undefined;
	store.follows = [];
	store.wlDetailedView = [...defaultWLDetailedView];
	store.tags = [];
	clearActiveFilters();
};

export const clearActiveFilters = () => {
	store.activeFilters = { type: [], status: [] };
};

const clearActiveWatchedListPreset = () => {
	_store.activeWatchedListPreset = undefined;
	if (!temporaryWatchedListState) {
		localStorage.removeItem("activeWatchedListPreset");
	}
};

export const setWatchedListPreset = (presetId: WatchedListPresetId) => {
	const preset = defaultWatchedListPresets[presetId];
	store.activeFilters = cloneFilters(preset.filters);
	store.activeSort = [...preset.sort];
	_store.activeWatchedListPreset = presetId;
	if (!temporaryWatchedListState) {
		localStorage.setItem("activeWatchedListPreset", presetId);
	}
};

if (browser) {
	rehydrateStore();
}

/**
 * Restore state from localStorage and apply values into
 * our `store`.
 * Rehydrates directly into `_store` (the real store)
 * to avoid the setters that would trigger a save right
 * after rehydrate.
 */
function rehydrateStore() {
	console.info("rehydrateStore: Running..");
	// Restore activeSort
	const raf = localStorage.getItem("activeFilter");
	if (raf) {
		_store.activeSort = JSON.parse(raf);
		console.debug(
			"rehydrateStore: Restored activeSort:",
			$state.snapshot(store.activeSort),
		);
	}
	// Restore activeFilters
	const filters = localStorage.getItem("activeFilterReal");
	if (filters) {
		_store.activeFilters = JSON.parse(filters);
		console.debug(
			"rehydrateStore: Restored activeFilters:",
			$state.snapshot(store.activeFilters),
		);
	}
	// Restore active watched list preset
	const activeWatchedListPreset = localStorage.getItem(
		"activeWatchedListPreset",
	) as WatchedListPresetId | null;
	if (activeWatchedListPreset) {
		_store.activeWatchedListPreset = activeWatchedListPreset;
		console.debug(
			"rehydrateStore: Restored activeWatchedListPreset:",
			$state.snapshot(store.activeWatchedListPreset),
		);
	}
	// Presets are fixed defaults now; discard preferences saved by older versions.
	localStorage.removeItem("watchedListPresets");
	// After restoring activeSort and activeFilter, set
	// an initial value for our related query param state.
	updateSortAndFiltersForQueryParams();
	// Restore appTheme
	const theme = localStorage.getItem("theme") as Theme;
	if (theme) {
		_store.appTheme = theme;
		toggleTheme(theme, false);
		console.debug(
			"rehydrateStore: Restored appTheme:",
			$state.snapshot(store.appTheme),
		);
	} else {
		const defTheme: Theme = "system";
		_store.appTheme = defTheme;
		toggleTheme(defTheme, false);
		console.debug(
			"rehydrateStore: appTheme hydrated from system default (wont save):",
			defTheme,
		);
	}
	// Restore wlDetailedView
	const wlDetailedViewR = localStorage.getItem("wlDetailedView");
	if (wlDetailedViewR) {
		_store.wlDetailedView = JSON.parse(wlDetailedViewR);
		console.debug(
			"rehydrateStore: Restored wlDetailedView:",
			$state.snapshot(store.wlDetailedView),
		);
	}
	console.info("rehydrateStore: Done.");
}
