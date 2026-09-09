import { goto } from "$app/navigation";
import type { ResolvedPathname } from "$app/types";
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

const MAX_SAVED_LISTS = 10;
const savedLists = new SvelteMap<string, SavedListState>();
let nextToken = 0;

export function createListSnapshot<T, M>(
	loader: PaginatedLoader<T, M>,
): Snapshot<ListSnapshot> {
	return {
		capture: () => {
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
		},
	};
}

function validListDepth(depth: number | undefined) {
	return typeof depth === "number" && Number.isInteger(depth) && depth > 0
		? depth
		: undefined;
}

export function withPublicListNavigation(
	path: ResolvedPathname,
	owner: PublicListNavigation,
): ResolvedPathname {
	const depth = validListDepth(owner.listDepth);
	return depth ? (`${path}?listDepth=${depth}` as ResolvedPathname) : path;
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
	if (!depth || window.history.length <= depth) return;
	event.preventDefault();
	window.history.go(-depth);
}

export function gotoResolved(path: ResolvedPathname) {
	return goto(path);
}
