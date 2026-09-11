import { goto } from "$app/navigation";
import type { ResolvedPathname } from "$app/types";
import type { PaginationResponse } from "@/types";
import type { Snapshot } from "@sveltejs/kit";
import { SvelteMap } from "svelte/reactivity";

interface PaginatedState<T, M> {
	data: T[];
	meta?: M;
	page: number;
	pageMax: number;
	reqLoading: boolean;
	reqLoadError: unknown;
}

interface PaginatedLoader<T, M> {
	state: PaginatedState<T, M>;
	abortReq: (reason: string) => void;
}

interface SavedListState {
	data: unknown[];
	meta?: unknown;
	page: number;
	pageMax: number;
}

interface ListSnapshot {
	token?: string;
	scrollY: number;
}

export interface PublicListNavigation {
	id: string | number;
	username: string;
	listDepth?: number;
}

interface ListSnapshotOptions<T, M> {
	key?: () => string;
	onRevalidated?: () => void;
	revalidatePage?: (
		page: number,
		signal: AbortSignal,
	) => Promise<PaginationResponse<T, M>>;
}

const MAX_SAVED_LISTS = 10;
const savedLists = new SvelteMap<string, SavedListState>();
let nextToken = 0;

export function createListSnapshot<T, M>(
	loader: PaginatedLoader<T, M>,
	options: ListSnapshotOptions<T, M> = {},
): Snapshot<ListSnapshot> {
	let revalidation: AbortController | undefined;

	async function revalidate(savedPage: number, key: string | undefined) {
		if (!options.revalidatePage) return;
		revalidation?.abort();
		const controller = new AbortController();
		revalidation = controller;
		loader.state.reqLoading = true;

		try {
			const data: T[] = [];
			let meta: M | undefined;
			let page = 0;
			let pageMax = 1;

			for (let currentPage = 1; currentPage <= savedPage; currentPage++) {
				const response = await options.revalidatePage(
					currentPage,
					controller.signal,
				);
				if (controller.signal.aborted) return;
				data.push(...(response.results ?? []));
				meta = response.meta;
				page = response.page;
				pageMax = response.totalPages;
				if (currentPage >= pageMax) break;
			}

			if (key !== options.key?.() || controller.signal.aborted) return;
			loader.state.data = data;
			loader.state.meta = meta;
			loader.state.page = page;
			loader.state.pageMax = pageMax;
			loader.state.reqLoading = false;
			loader.state.reqLoadError = undefined;
			options.onRevalidated?.();
		} catch (error) {
			if (controller.signal.aborted || key !== options.key?.()) return;
			loader.state.reqLoading = false;
			console.warn("List snapshot revalidation failed", error);
		}
	}

	return {
		capture: () => {
			revalidation?.abort();
			const snapshot: ListSnapshot = { scrollY: window.scrollY };
			if (loader.state.page <= 0) return snapshot;

			const token = `${Date.now()}-${nextToken++}`;
			savedLists.set(token, {
				data: structuredClone($state.snapshot(loader.state.data)),
				meta: structuredClone($state.snapshot(loader.state.meta)),
				page: loader.state.page,
				pageMax: loader.state.pageMax,
			});
			while (savedLists.size > MAX_SAVED_LISTS) {
				const oldest = savedLists.keys().next().value;
				if (!oldest) break;
				savedLists.delete(oldest);
			}
			return { token, scrollY: snapshot.scrollY };
		},
		restore: ({ token, scrollY }) => {
			if (!token) return;
			const saved = savedLists.get(token);
			if (!saved) return;

			loader.abortReq("restoring list navigation");
			loader.state.data = structuredClone(saved.data) as T[];
			loader.state.meta = structuredClone(saved.meta) as M | undefined;
			loader.state.page = saved.page;
			loader.state.pageMax = saved.pageMax;
			loader.state.reqLoading = false;
			loader.state.reqLoadError = undefined;
			requestAnimationFrame(() => window.scrollTo(0, scrollY));
			void revalidate(saved.page, options.key?.());
		},
	};
}

function validListDepth(depth: number | undefined) {
	return typeof depth === "number" && Number.isInteger(depth) && depth > 0
		? depth
		: undefined;
}

function isPublicList(url: URL) {
	return /^\/lists\/[^/]+\/[^/]+\/?$/.test(url.pathname);
}

export function publicListHistoryState(
	next: URL,
	current: URL,
	currentState: App.PageState,
): App.PageState {
	const state = { ...currentState };
	delete state.publicListDepth;
	if (!next.searchParams.get("query")?.trim()) {
		return state;
	}

	const currentDepth = validListDepth(currentState.publicListDepth);
	if (currentDepth) {
		state.publicListDepth = currentDepth + 1;
	} else if (
		isPublicList(current) &&
		!current.searchParams.get("query")?.trim()
	) {
		state.publicListDepth = 1;
	}
	return state;
}

export function publicListDetailDepth(url: URL, currentDepth?: number) {
	if (!url.searchParams.get("query")?.trim()) return 1;
	const depth = validListDepth(currentDepth);
	return depth ? depth + 1 : undefined;
}

export function publicListChild(
	owner: Omit<PublicListNavigation, "listDepth">,
	currentDepth: number | undefined,
): PublicListNavigation {
	const depth = validListDepth(currentDepth);
	return depth ? { ...owner, listDepth: depth + 1 } : owner;
}

export function backToPublicList(event: MouseEvent, listDepth?: number) {
	const depth = validListDepth(listDepth);
	if (
		!depth ||
		window.history.length <= depth ||
		event.button !== 0 ||
		event.metaKey ||
		event.ctrlKey ||
		event.shiftKey ||
		event.altKey
	) {
		return;
	}
	event.preventDefault();
	window.history.go(-depth);
}

export function gotoResolved(
	path: ResolvedPathname,
	event?: MouseEvent,
	publicListOwner?: PublicListNavigation,
) {
	if (
		event &&
		(event.button !== 0 ||
			event.metaKey ||
			event.ctrlKey ||
			event.shiftKey ||
			event.altKey)
	) {
		return;
	}
	event?.preventDefault();
	const depth = validListDepth(publicListOwner?.listDepth);
	return goto(path, {
		state: depth ? { publicListDepth: depth } : {},
	});
}
