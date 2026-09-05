<script lang="ts">
	import Icon from "@/lib/Icon.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import tooltip from "@/lib/actions/tooltip.js";
	import UserAvatar from "@/lib/img/UserAvatar.svelte";
	import { followUser, noAuthReq, unfollowUser } from "@/lib/util/api.js";
	import {
		clearActiveFilters,
		defaultSort,
		setWatchedListPreset,
		store,
	} from "@/store.svelte.js";
	import {
		MediaTypeE,
		SearchType,
		type Media,
		type PaginationResponse,
		type PublicUser,
		type SearchResponseMeta,
	} from "@/types.js";
	import { onDestroy, untrack } from "svelte";
	import paginatedLoader from "@/lib/util/paginatedLoader.svelte.js";
	import infScroll from "@/lib/util/infScroll.js";
	import { page } from "$app/state";
	import PosterList from "@/lib/poster/PosterList.svelte";
	import Poster from "@/lib/poster/Poster.svelte";
	import Error from "@/lib/Error.svelte";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { parseTokenPayload } from "@/lib/util/helpers";
	import MediaTypeFilter from "@/lib/search/MediaTypeFilter.svelte";
	import PageTitle from "@/lib/generic/PageTitle.svelte";
	import PersonPoster from "@/lib/poster/PersonPoster.svelte";

	let ownerId = $derived(page.params.id ?? "");
	let ownerUsername = $derived(page.params.username ?? "");
	let meta = $derived.by(() => ({ id: ownerId, username: ownerUsername }));

	let searchQuery = $derived(page.url.searchParams.get("query")?.trim() ?? "");
	let isSearching = $derived(searchQuery.length > 0);
	let searchType: SearchType | undefined = $derived.by(() => {
		const value = page.url.searchParams.get("type");
		switch (value) {
			case SearchType.movie:
			case SearchType.show:
			case SearchType.game:
			case SearchType.person:
				return value;
			default:
				return undefined;
		}
	});
	let searchScope = $derived(
		page.url.searchParams.get("scope") === "all" ||
			searchType === SearchType.person
			? "all"
			: "list",
	);
	let isGlobalSearch = $derived(isSearching && searchScope === "all");
	let globalSource = $derived.by(() => {
		if (searchType === SearchType.game) return "IGDB";
		if (!searchType) return "TMDB and IGDB";
		return "TMDB";
	});
	const allSearchStatuses = [
		"planned",
		"watching",
		"finished",
		"hold",
		"dropped",
	];
	let previousSearchQuery = $state("");

	let followBtnDisabled = $state(false);
	let user: PublicUser | undefined = $state();
	let loadedPublicUserKey = "";
	let viewerUserId = parseTokenPayload()?.userId;
	let isAuthenticated = $derived(Boolean(store.userInfo));
	let canFollow = $derived(isAuthenticated && viewerUserId !== Number(meta.id));

	let isFollowing = $derived(
		!!store.follows?.find((f) => f.followedUser.id === Number(meta.id)),
	);

	const scroll = infScroll({ callback: onScrollToBottom });
	const dataLoader = paginatedLoader<Media, SearchResponseMeta>(load);

	$effect(() => {
		const query = searchQuery;
		if (query && query !== previousSearchQuery) {
			store.activeFilters = {
				type: [],
				status: [...allSearchStatuses],
			};
			store.activeSort = ["LASTFIN", "DOWN"];
		} else if (!query && previousSearchQuery) {
			setWatchedListPreset("recentlyWatched");
		}
		previousSearchQuery = query;
	});

	let requestParams: Record<string, string> = $derived.by(() => {
		const params = { ...store.sortAndFiltersForQueryParams };
		if (!isSearching) return params;
		delete params.type;

		return {
			...params,
			query: searchQuery,
			scope: searchScope,
			...(searchType ? { type: searchType } : {}),
		};
	});
	let requestKey = $derived(
		JSON.stringify({
			id: meta.id,
			username: meta.username,
			params: requestParams,
		}),
	);
	let nextLoadParams: {
		page: number;
		[x: string]: unknown;
	} = $derived({
		page: dataLoader.state.page + 1,
		...requestParams,
	});

	async function load(signal: AbortSignal) {
		console.debug("load: loadParams:", nextLoadParams);
		if (nextLoadParams.page === dataLoader.state.page) {
			console.warn("load: Already on this page, not loading it again!");
			return;
		}
		if (!meta.id || !meta.username) {
			console.warn("load: Missing id or username!");
			return;
		}
		const endpoint = isSearching
			? `/public/users/${meta.id}/${meta.username}/search`
			: `/public/users/${meta.id}/${meta.username}/watched`;
		const r = await noAuthReq.get<
			PaginationResponse<Media, SearchResponseMeta>
		>(endpoint, {
			params: nextLoadParams,
			signal,
		});
		scroll.dataLoaded();
		return r;
	}

	async function onScrollToBottom() {
		// If an error is being shown, no more infinite scroll.
		if (dataLoader.state.reqLoadError) {
			return;
		}
		console.debug("onScrollToBottom");
		await dataLoader.runFn();
	}

	// Handles the initial load and reloads when the owner, query, sort, or
	// non-search filters change.
	$effect(() => {
		if (requestKey) {
			untrack(() => {
				// We don't want state inside these funcs to become dependencies.
				dataLoader.reset();
				dataLoader.runFn();
			});
		}
	});

	async function getPublicUser(id: string, username: string) {
		return await noAuthReq.get<PublicUser>(`/public/users/${id}/${username}`);
	}

	async function follow() {
		followBtnDisabled = true;
		console.log(isFollowing);
		if (isFollowing) {
			await unfollowUser(Number(meta.id));
		} else {
			await followUser(Number(meta.id));
		}
		followBtnDisabled = false;
	}

	$effect(() => {
		const id = ownerId;
		const username = ownerUsername;
		if (!id || !username) {
			user = undefined;
			loadedPublicUserKey = "";
			return;
		}

		const userKey = `${id}/${username}`;
		if (userKey === loadedPublicUserKey) return;
		loadedPublicUserKey = userKey;
		user = undefined;
		getPublicUser(id, username)
			.then((u) => {
				if (loadedPublicUserKey === userKey) user = u;
			})
			.catch((err) => {
				console.error("getPublicUser failed!", err);
			});
	});

	$effect(() => {
		store.searchQuery = searchQuery;
	});

	function clearSearch() {
		const location = new URL(page.url);
		location.searchParams.delete("query");
		location.searchParams.delete("scope");
		location.searchParams.delete("type");
		gotoListLocation(location);
	}

	function gotoListLocation(location: URL) {
		const searchParams = location.searchParams.toString();
		goto(
			searchParams
				? resolve(
						`/lists/${page.params.id}/${page.params.username}?${searchParams}`,
					)
				: resolve(`/lists/${page.params.id}/${page.params.username}`),
		);
	}

	function resetSearchListControls() {
		store.activeFilters = {
			type: [],
			status: [...allSearchStatuses],
		};
		store.activeSort = ["LASTFIN", "DOWN"];
	}

	function setActiveSearchFilter(to: SearchType) {
		const location = new URL(page.url);
		const nextType = searchType === to ? undefined : to;
		if (nextType) {
			location.searchParams.set("type", nextType);
		} else {
			location.searchParams.delete("type");
		}
		if (nextType === SearchType.person) {
			location.searchParams.set("scope", "all");
		} else if (
			to === SearchType.movie ||
			to === SearchType.show ||
			to === SearchType.game
		) {
			// A specific media search is always scoped back to the owner's list.
			location.searchParams.delete("scope");
		}
		resetSearchListControls();
		window.scrollTo({ top: 0 });
		gotoListLocation(location);
	}

	function setSearchScope(global: boolean) {
		const location = new URL(page.url);
		if (global) {
			if (isGlobalSearch) return;
			location.searchParams.set("scope", "all");
		} else {
			if (!isGlobalSearch || searchType === SearchType.person) return;
			location.searchParams.delete("scope");
		}
		window.scrollTo({ top: 0 });
		gotoListLocation(location);
	}

	function showAllItems() {
		clearActiveFilters();
		store.activeSort = [...defaultSort];
	}

	onDestroy(() => {
		console.debug("PAGE DESTROYED");
		store.searchQuery = "";
		scroll.destroy();
		dataLoader.abortReq("page destroyed");
	});
</script>

<svelte:head>
	<title>{meta.username}'s Watched List</title>
</svelte:head>

<div class="content" class:logged-out={!isAuthenticated}>
	<div class="inner">
		<UserAvatar img={user?.avatar} />
		<div class="basic-ctr">
			<div class="name-row">
				<h2 title={user?.username}>
					{meta.username}
				</h2>
				{#if canFollow}
					<button
						class="plain follow"
						disabled={followBtnDisabled}
						onclick={follow}
						use:tooltip={{ text: isFollowing ? "Unfollow" : "Follow" }}
					>
						<Icon i={isFollowing ? "person-minus" : "person-add"} />
					</button>
				{/if}
			</div>
			{#if user?.bio}
				<span title={user?.bio}>{user?.bio}</span>
			{/if}
		</div>
	</div>
</div>

{#if !isSearching}
	<div class="type-toggle">
		<button
			class="plain"
			data-active={store.activeWatchedListPreset === "recentlyWatched"}
			onclick={() => setWatchedListPreset("recentlyWatched")}
		>
			<Icon i="film" wh={18} /> Recently Finished
		</button>
		<button
			class="plain"
			data-active={!store.hasActiveFilters}
			onclick={showAllItems}
		>
			All Items
		</button>
		<button
			class="plain"
			data-active={store.activeWatchedListPreset === "watchlist"}
			onclick={() => setWatchedListPreset("watchlist")}
		>
			<Icon i="calendar" wh={18} /> Watchlist
		</button>
	</div>
{/if}

{#if isSearching}
	<div class="search-results">
		<PageTitle title="Results">
			<div class="search-controls">
				<div class="search-type-filter">
					<MediaTypeFilter
						active={searchType}
						disabled={dataLoader.state.reqLoading}
						showGames={true}
						onChange={(nowActive) => {
							setActiveSearchFilter(nowActive as SearchType);
						}}
					/>
				</div>
				<div class="source-control">
					<span class="control-label">Show results from:</span>
					<button
						class="plain source-toggle"
						class:global={isGlobalSearch}
						class:locked={searchType === SearchType.person}
						type="button"
						role="switch"
						aria-checked={isGlobalSearch}
						aria-label="Show results from"
						disabled={searchType === SearchType.person}
						onclick={() => setSearchScope(!isGlobalSearch)}
					>
						<span class="source-option local">Leon's list</span>
						<span class="source-option global">Global</span>
					</button>
				</div>
			</div>
		</PageTitle>
	</div>
{/if}

<PosterList>
	{#if dataLoader.state.data?.length > 0}
		{#each dataLoader.state.data as w, i (`${i}-${w.type}`)}
			{#if w.type === MediaTypeE.tmdbPerson}
				<PersonPoster
					id={w.ids.tmdb}
					name={w.name}
					path={w.extPosterPath}
					publicListOwner={{ id: meta.id, username: meta.username }}
				/>
			{:else if w}
				<Poster
					watched={w.watched?.id ? w.watched : undefined}
					media={w}
					fluidSize={true}
					hideButtons={true}
					publicView={Boolean(w.watched?.id)}
					publicRatingSettings={w.watched?.id
						? {
								ratingSystem: user?.ratingSystem,
								ratingStep: user?.ratingStep,
							}
						: undefined}
					publicListOwner={{ id: meta.id, username: meta.username }}
				/>
			{/if}
		{/each}
	{:else if !dataLoader.state.reqLoading && !dataLoader.state.reqLoadError}
		<div class="empty-list">
			{#if isSearching}
				<Icon i="search" wh={80} />
				{#if isGlobalSearch}
					<h2 class="norm">No Results!</h2>
					<h4 class="norm">
						No {globalSource} results match “{searchQuery}”.
					</h4>
				{:else}
					<h2 class="norm">No matching items!</h2>
					<h4 class="norm">
						Nothing on {meta.username}'s list matches “{searchQuery}”.
					</h4>
				{/if}
				<button onclick={clearSearch}>Clear Search</button>
			{:else}
				<Icon i={store.hasActiveFilters ? "filter-circle" : "reel"} wh={80} />
				<h2 class="norm">This list is empty!</h2>
				<h4 class="norm">
					Come back later to see if they have added anything.
				</h4>
				{#if store.hasActiveFilters}
					<button onclick={() => clearActiveFilters()}>Clear Filters</button>
				{/if}
			{/if}
		</div>
	{/if}
</PosterList>

{#if dataLoader.state.reqLoading}
	<div style="margin-bottom: 60px;">
		<Spinner />
	</div>
{/if}

{#if dataLoader.state.reqLoadError}
	<div style="margin-bottom: 60px;">
		<Error
			pretty="Failed to load results!"
			error={dataLoader.state.reqLoadError}
			onRetry={() => {
				dataLoader.state.reqLoadError = undefined;
				dataLoader.runFn();
			}}
		/>
	</div>
{/if}

<style lang="scss">
	.content {
		display: flex;
		width: 100%;
		justify-content: center;

		.inner {
			display: flex;
			flex-flow: row;
			gap: 15px;
			justify-content: center;
			align-items: center;
			width: 100%;
			max-width: 1200px;
			margin: 20px 30px;
			margin-top: 0;
		}
	}

	.logged-out .inner {
		flex-flow: column;
	}

	.logged-out .basic-ctr {
		text-align: center;

		.name-row {
			justify-content: center;
		}
	}

	button {
		width: max-content;
	}

	.type-toggle {
		display: flex;
		flex-flow: row;
		flex-wrap: wrap;
		gap: 10px;
		justify-content: center;
		margin: 0 auto 15px auto;

		button {
			display: flex;
			flex-flow: row;
			align-items: center;
			gap: 8px;
			padding: 8px 14px;
			border: 2px solid $text-color;
			border-radius: 8px;
			font-size: 14px;
			color: $text-color;
			fill: $text-color;
			transition:
				background-color 150ms ease,
				color 150ms ease,
				outline 150ms ease;

			&:hover,
			&[data-active="true"] {
				color: $bg-color;
				fill: $bg-color;
				background-color: $accent-color-hover;
			}

			&[data-active="true"] {
				outline: 3px solid $accent-color;
			}
		}
	}

	.search-results {
		width: 100%;
		max-width: 1200px;
		margin: 0 auto;
	}

	.search-controls {
		flex: 1 1 100%;
		min-width: 0;
		display: flex;
		flex-flow: column;
		gap: 12px;
	}

	.search-type-filter {
		width: 100%;
	}

	.source-control {
		display: flex;
		flex-flow: column;
		gap: 8px;
		width: 100%;
		min-width: 0;
		padding: 8px;
		border: 1px solid rgba($color: $accent-color, $alpha: 0.45);
		border-radius: 10px;
		background-color: rgba($color: $accent-color, $alpha: 0.08);
		box-shadow: 0 3px 12px rgba(0, 0, 0, 0.14);
	}

	.control-label {
		display: block;
		font-size: 14px;
		font-weight: 600;
		line-height: 1.2;
		text-align: left;
		color: $text-color;
	}

	.source-toggle {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
		min-height: 42px;
		padding: 4px;
		border: 2px solid $text-color;
		border-radius: 8px;
		background-color: transparent;
		color: $text-color;
		fill: $text-color;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		overflow: hidden;
		transition:
			background-color 150ms ease,
			border-color 150ms ease,
			outline 150ms ease;

		&::before {
			position: absolute;
			top: 4px;
			bottom: 4px;
			left: 4px;
			width: calc(50% - 4px);
			border-radius: 5px;
			background-color: $accent-color-hover;
			content: "";
			transition: transform 150ms ease;
		}

		&.global::before {
			transform: translateX(100%);
		}

		&:disabled {
			cursor: not-allowed;
			opacity: 1;
		}

		.source-option {
			position: relative;
			z-index: 1;
			flex: 1 1 0;
			padding: 7px 8px;
			border-radius: 5px;
			text-align: center;
			cursor: pointer;
			transition:
				background-color 150ms ease,
				color 150ms ease;

			&:hover {
				color: $bg-color;
				background-color: $accent-color-hover;
			}
		}

		&:not(.global) .source-option.local,
		&.global .source-option.global {
			color: $bg-color;
			font-weight: 600;
		}

		&.locked .source-option.local {
			opacity: 0.4;
		}
	}

	.basic-ctr {
		min-width: 200px;
		max-width: 300px;
		overflow: hidden;

		.name-row {
			display: flex;
			flex-flow: row;
			gap: 15px;

			h2 {
				overflow: hidden;
				text-overflow: ellipsis;
			}

			.follow {
				margin-left: auto;
				display: flex;
				align-items: center;
				width: 28px;
				fill: $text-color;
			}
		}

		span {
			font-family: monospace;
			overflow: hidden;
			text-overflow: ellipsis;
			display: -webkit-box;
			line-clamp: 2;
			-webkit-line-clamp: 2;
			-webkit-box-orient: vertical;
		}
	}

	.empty-list {
		display: flex;
		flex-flow: column;
		gap: 5px;
		align-items: center;
		max-width: 400px;

		h2 {
			margin-top: 10px;
		}

		h4 {
			font-weight: normal;
			text-align: center;
		}

		button {
			width: max-content;
			padding-left: 20px;
			padding-right: 20px;
			margin-top: 15px;
		}
	}
</style>
