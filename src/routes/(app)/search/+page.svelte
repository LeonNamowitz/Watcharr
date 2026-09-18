<script lang="ts">
	import { afterNavigate, goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import UsersList from "@/lib/UsersList.svelte";
	import PageTitle from "@/lib/generic/PageTitle.svelte";
	import PersonPoster from "@/lib/poster/PersonPoster.svelte";
	import Poster from "@/lib/poster/Poster.svelte";
	import PosterList from "@/lib/poster/PosterList.svelte";
	import SearchControls from "@/lib/search/SearchControls.svelte";
	import SearchAllButton from "@/lib/search/SearchAllButton.svelte";
	import {
		hasPeopleSearch,
		parseSearchTypes,
		searchTypesParam,
		setSearchTypesOnUrl,
		type SelectableSearchType,
	} from "@/lib/search/searchTypes";
	import { req } from "@/lib/util/api";
	import infScroll from "@/lib/util/infScroll";
	import { createListSnapshot } from "@/lib/util/listNavigation.svelte";
	import paginatedLoader, {
		PaginatedLoaderRunFnAction,
	} from "@/lib/util/paginatedLoader.svelte";
	import {
		applyWatchedListState,
		beginTemporaryWatchedListState,
		defaultWLDetailedView,
		store,
		type WatchedListStateSnapshot,
	} from "@/store.svelte";
	import {
		MediaTypeE,
		type Media,
		type PaginationResponse,
		type PublicUser,
		type SearchResponseMeta,
	} from "@/types";
	import { onDestroy, onMount, untrack } from "svelte";
	import Filters from "./components/Filters.svelte";

	let { data } = $props();
	const restoreWatchedListState = beginTemporaryWatchedListState();
	const allSearchStatuses = [
		"planned",
		"watching",
		"finished",
		"hold",
		"dropped",
	];
	let searchReady = $state(false);
	let stateQuery = $state("");
	let allowEmptyLocalResults = $state(false);
	let searchQuery = $derived(data?.query ? decodeURIComponent(data.query) : "");
	let searchTypes = $derived(
		parseSearchTypes(page.url.searchParams.get("type")),
	);
	let isPersonSearch = $derived(hasPeopleSearch(searchTypes));
	let isGlobalSearch = $derived(
		page.url.searchParams.get("scope") === "all" || isPersonSearch,
	);

	const scroll = infScroll({ callback: onScrollToBottom });
	const dataLoader = paginatedLoader<Media, SearchResponseMeta>(load);
	export const snapshot = createListSnapshot(dataLoader, {
		key: searchKey,
		onRevalidated: scroll.dataLoaded,
		revalidatePage: loadPage,
	});

	let requestParams: Record<string, string> = $derived.by(() => {
		const type = searchTypesParam(searchTypes);
		if (isGlobalSearch) {
			return {
				query: searchQuery,
				scope: "all",
				...(type ? { type } : {}),
			};
		}
		const params = { ...store.sortAndFiltersForQueryParams };
		delete params.type;
		return {
			...params,
			query: searchQuery,
			scope: "list",
			...(type ? { type } : {}),
		};
	});
	let requestKey = $derived(JSON.stringify(requestParams));
	let nextLoadParams: Record<string, string | number> = $derived({
		page: dataLoader.state.page + 1,
		...requestParams,
	});

	function defaultSearchState(): WatchedListStateSnapshot {
		return {
			sort: ["LASTFIN", "DOWN"],
			filters: {
				type: [],
				status: [...allSearchStatuses],
			},
			preset: undefined,
			detailedView: [...defaultWLDetailedView],
		};
	}

	function searchKey() {
		return requestKey;
	}

	async function loadPage(pageNumber: number, signal: AbortSignal) {
		return req.get<PaginationResponse<Media, SearchResponseMeta>>("/search", {
			params: { ...requestParams, page: pageNumber },
			signal,
		});
	}

	async function load(signal: AbortSignal) {
		if (nextLoadParams.page === dataLoader.state.page || !searchQuery) return;
		const searchedPage = nextLoadParams.page;
		const searchedScope = requestParams.scope;
		const response = await req.get<
			PaginationResponse<Media, SearchResponseMeta>
		>("/search", { params: nextLoadParams, signal });
		scroll.dataLoaded();
		if (
			searchedScope === "list" &&
			searchedPage === 1 &&
			!allowEmptyLocalResults &&
			(response.results?.length ?? 0) === 0
		) {
			setSearchScope(true, true);
		}
		return response;
	}

	async function onScrollToBottom() {
		if (!dataLoader.state.reqLoadError) await dataLoader.runFn();
	}

	async function searchUsers(query: string) {
		return req.get<PublicUser[]>("/user/search", { params: { q: query } });
	}

	function setActiveSearchTypes(types: SelectableSearchType[]) {
		const location = new URL(page.url);
		setSearchTypesOnUrl(location, types);
		window.scrollTo({ top: 0 });
		goto(resolve(`/search?${location.searchParams.toString()}`));
	}

	function selectAllSearchTypes() {
		applyWatchedListState(defaultSearchState());
		setActiveSearchTypes([]);
	}

	function setSearchScope(global: boolean, replaceState = false) {
		const location = new URL(page.url);
		if (global) {
			allowEmptyLocalResults = false;
			location.searchParams.set("scope", "all");
		} else if (!isPersonSearch) {
			allowEmptyLocalResults = true;
			location.searchParams.delete("scope");
		}
		window.scrollTo({ top: 0 });
		goto(resolve(`/search?${location.searchParams.toString()}`), {
			replaceState,
		});
	}

	$effect(() => {
		if (!searchReady || stateQuery !== searchQuery || !requestKey) return;
		untrack(() => {
			dataLoader.reset();
			dataLoader.runFn();
		});
	});

	onMount(() => {
		store.searchQuery = searchQuery;
		applyWatchedListState(defaultSearchState());
		stateQuery = searchQuery;
		searchReady = true;
	});

	afterNavigate((event) => {
		if (!searchReady || !event.from?.route?.id?.includes("/search")) return;
		store.searchQuery = searchQuery;
		if (stateQuery !== searchQuery) {
			allowEmptyLocalResults = false;
			applyWatchedListState(defaultSearchState());
			stateQuery = searchQuery;
		}
	});

	onDestroy(() => {
		searchReady = false;
		store.searchQuery = "";
		scroll.destroy();
		dataLoader.abortReq("page destroyed");
		restoreWatchedListState();
	});
</script>

<svelte:head>
	<title>Search Results{searchQuery ? ` for '${searchQuery}'` : ""}</title>
</svelte:head>

{#snippet filterHelp()}
	<SearchAllButton
		active={searchTypes.length === 0}
		disabled={dataLoader.state.reqLoading}
		onclick={selectAllSearchTypes}
	/>
	<Filters />
{/snippet}

<div class="content">
	<div class="inner">
		{#if searchQuery}
			{#await searchUsers(searchQuery) then results}
				{#if results?.length > 0}
					<UsersList users={results} />
				{/if}
			{:catch err}
				<Error pretty="Failed to load users!" error={err} />
			{/await}

			<PageTitle title="Results" actions={filterHelp}>
				<SearchControls
					activeTypes={searchTypes}
					global={isGlobalSearch}
					localLabel="Your list"
					disabled={dataLoader.state.reqLoading}
					showGames={Boolean(store.serverFeatures?.games)}
					onTypesChange={setActiveSearchTypes}
					onScopeChange={setSearchScope}
				/>
			</PageTitle>

			<PosterList>
				{#if dataLoader.state.data?.length > 0}
					{#each dataLoader.state.data as w, i (`${i}-${w.type}`)}
						{#if w.type === MediaTypeE.tmdbPerson}
							<PersonPoster
								id={w.ids.tmdb}
								name={w.name}
								path={w.extPosterPath}
							/>
						{:else if w.type === MediaTypeE.tmdbMovie || w.type === MediaTypeE.tmdbShow || w.type === MediaTypeE.igdbGame}
							<Poster
								media={w}
								bind:watched={dataLoader.state.data[i].watched}
								fluidSize
							/>
						{/if}
					{/each}
				{:else if !dataLoader.state.reqLoading && !dataLoader.state.reqLoadError}
					<div class="empty-results">
						<Icon i="search" wh={80} />
						<h2 class="norm">No Results!</h2>
						<h4 class="norm">
							{isGlobalSearch
								? `Nothing global matches “${searchQuery}”.`
								: `Nothing on your list matches “${searchQuery}”.`}
						</h4>
						{#if !isGlobalSearch}
							<button
								class="search-global"
								onclick={() => setSearchScope(true)}
							>
								Search Global
							</button>
						{/if}
					</div>
				{/if}
			</PosterList>

			{#if dataLoader.state.reqLoading}
				<div class="loader"><Spinner /></div>
			{/if}

			{#if dataLoader.state.reqLoadError}
				<div class="loader">
					<Error
						pretty="Failed to load results!"
						error={dataLoader.state.reqLoadError}
						onRetry={() => {
							dataLoader.state.reqLoadError = undefined;
							dataLoader.runFn(
								PaginatedLoaderRunFnAction.ResetIfOnFirstOrNoPage,
							);
						}}
					/>
				</div>
			{/if}
		{:else}
			<h2>No Search Query!</h2>
		{/if}
	</div>
</div>

<style lang="scss">
	.content {
		display: flex;
		width: 100%;
		justify-content: center;

		.inner {
			width: 100%;
			max-width: 1200px;
		}
	}

	.empty-results {
		display: flex;
		flex-flow: column;
		gap: 5px;
		align-items: center;
		max-width: 400px;

		h4 {
			font-weight: normal;
			text-align: center;
		}

		.search-global {
			width: max-content;
			padding: 7px 12px;
			margin-top: 10px;
		}
	}

	.loader {
		margin-bottom: 60px;
	}
</style>
