<script lang="ts">
	import Icon from "@/lib/Icon.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import tooltip from "@/lib/actions/tooltip.js";
	import UserAvatar from "@/lib/img/UserAvatar.svelte";
	import { followUser, noAuthReq, unfollowUser } from "@/lib/util/api.js";
	import {
		clearActiveFilters,
		setWatchedListPreset,
		store,
	} from "@/store.svelte.js";
	import type { Media, PaginationResponse, PublicUser } from "@/types.js";
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

	let meta = $derived.by(() => {
		return {
			id: page.params.id ?? "",
			username: page.params.username ?? "",
		};
	});

	let searchQuery = $derived(page.url.searchParams.get("query")?.trim() ?? "");
	let isSearching = $derived(searchQuery.length > 0);

	let followBtnDisabled = $state(false);
	let user: PublicUser | undefined = $state();
	let viewerUserId = parseTokenPayload()?.userId;
	let isAuthenticated = $derived(Boolean(store.userInfo));
	let canFollow = $derived(isAuthenticated && viewerUserId !== Number(meta.id));

	let isFollowing = $derived(
		!!store.follows?.find((f) => f.followedUser.id === Number(meta.id)),
	);

	const scroll = infScroll({ callback: onScrollToBottom });
	const dataLoader = paginatedLoader<Media, undefined>(load);

	let requestParams: Record<string, string> = $derived.by(() => {
		const params = store.sortAndFiltersForQueryParams;
		if (!isSearching) return params;

		return { ...params, query: searchQuery };
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
		const r = await noAuthReq.get<PaginationResponse<Media, undefined>>(
			endpoint,
			{
				params: nextLoadParams,
				signal,
			},
		);
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

	async function getPublicUser() {
		return await noAuthReq.get<PublicUser>(
			`/public/users/${meta.id}/${meta.username}`,
		);
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
		user = undefined;
		if (meta?.id && meta?.username) {
			getPublicUser()
				.then((u) => {
					user = u;
				})
				.catch((err) => {
					console.error("getPublicUser failed!", err);
				});
		}
	});

	$effect(() => {
		store.searchQuery = searchQuery;
	});

	function clearSearch() {
		const location = new URL(page.url);
		location.searchParams.delete("query");
		const searchParams = location.searchParams.toString();
		goto(
			searchParams
				? resolve(
						`/lists/${page.params.id}/${page.params.username}?${searchParams}`,
					)
				: resolve(`/lists/${page.params.id}/${page.params.username}`),
		);
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

<div class="content">
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
				{:else if !isAuthenticated}
					<a
						class="follow"
						href={resolve("/login")}
						use:tooltip={{ text: "Log in to follow" }}
					>
						<Icon i="person-add" />
					</a>
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
			<Icon i="reel" wh={18} /> Recently Finished
		</button>
		<button
			class="plain"
			data-active={store.activeWatchedListPreset === "watchlist"}
			onclick={() => setWatchedListPreset("watchlist")}
		>
			<Icon i="film" wh={18} /> Watchlist
		</button>
	</div>
{/if}

<PosterList>
	{#if dataLoader.state.data?.length > 0}
		{#each dataLoader.state.data as w, i (`${i}-${w.type}`)}
			{#if w}
				<Poster
					watched={dataLoader.state.data[i].watched}
					media={w}
					fluidSize={true}
					hideButtons={true}
					publicView={true}
					publicRatingSettings={{
						ratingSystem: user?.ratingSystem,
						ratingStep: user?.ratingStep,
					}}
					publicListOwner={{ id: meta.id, username: meta.username }}
				/>
			{/if}
		{/each}
	{:else if !dataLoader.state.reqLoading && !dataLoader.state.reqLoadError}
		<div class="empty-list">
			{#if isSearching}
				<Icon i="search" wh={80} />
				<h2 class="norm">No matching items!</h2>
				<h4 class="norm">
					Nothing on {meta.username}'s list matches “{searchQuery}”.
				</h4>
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
