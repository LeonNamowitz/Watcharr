<script lang="ts">
	import { afterNavigate, goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import Icon from "@/lib/Icon.svelte";
	import tooltip from "@/lib/actions/tooltip";
	import DetailedMenu from "@/lib/nav/DetailedMenu.svelte";
	import FilterMenu from "@/lib/nav/FilterMenu.svelte";
	import NavShell from "@/lib/nav/NavShell.svelte";
	import SortMenu from "@/lib/nav/SortMenu.svelte";
	import { optionalAuthReq } from "@/lib/util/api";
	import { ReqerError } from "@/lib/util/fetch";
	import { isTouch } from "@/lib/util/helpers";
	import {
		beginTemporaryWatchedListState,
		defaultSort,
		store,
	} from "@/store.svelte";
	import type { Follow, PrivateUser } from "@/types";
	import { onMount } from "svelte";

	interface Props {
		children?: import("svelte").Snippet;
	}

	let { children }: Props = $props();
	const restoreWatchedListState = beginTemporaryWatchedListState();

	let navEl: HTMLElement | undefined = $state();
	let mainSearchEl: HTMLInputElement | undefined = $state();
	let searchTimeout: number;
	let scroll = 0;
	let isAuthenticated = $derived(Boolean(store.userInfo));
	let detailedMenuShown = $state(false);
	let sortMenuShown = $state(false);
	let filterMenuShown = $state(false);
	let isListPage = $derived(
		/^\/lists\/[^/]+\/[^/]+\/?$/.test(page.url.pathname),
	);
	let isPersonPage = $derived(
		/^\/lists\/[^/]+\/[^/]+\/person\/[^/]+\/?$/.test(page.url.pathname),
	);
	let homeHref = $derived(
		`/lists/${page.params.id}/${page.params.username}` as `/lists/${string}/${string}`,
	);

	function closeMenus(except?: "detailed" | "sort" | "filter") {
		if (except !== "detailed") detailedMenuShown = false;
		if (except !== "sort") sortMenuShown = false;
		if (except !== "filter") filterMenuShown = false;
	}

	function shouldIgnoreSearchKey(ev: KeyboardEvent) {
		return [
			"ContextMenu",
			"Home",
			"End",
			"PageDown",
			"PageUp",
			"NumLock",
			"Escape",
			"Tab",
			"CapsLock",
			"OS",
			"ArrowLeft",
			"ArrowRight",
			"ArrowUp",
			"ArrowDown",
			"Control",
			"Alt",
			"AltGraph",
			"Shift",
			"Meta",
		].includes(ev.key);
	}

	function handleSearch(ev: KeyboardEvent) {
		if (shouldIgnoreSearchKey(ev)) return;
		clearTimeout(searchTimeout);
		searchTimeout = window.setTimeout(
			() => {
				const target = ev.target as HTMLInputElement;
				const query = target.value.trim();
				const location = new URL(page.url);
				if (query) {
					location.searchParams.set("query", query);
				} else {
					location.searchParams.delete("query");
				}
				const searchParams = location.searchParams.toString();
				const listLocation = searchParams
					? resolve(
							`/lists/${page.params.id}/${page.params.username}?${searchParams}`,
						)
					: resolve(`/lists/${page.params.id}/${page.params.username}`);
				if (listLocation === `${page.url.pathname}${page.url.search}`) return;

				target.autofocus = true;
				goto(listLocation).then(() => {
					if (!document.body.classList.contains("split-nav")) {
						mainSearchEl?.focus();
						if (mainSearchEl) mainSearchEl.autofocus = false;
					} else {
						target.focus();
					}
					target.autofocus = false;
				});
			},
			isTouch() ? 800 : 400,
		);
	}

	function decideOnNavSplit() {
		// Detail pages only have the logo and login/profile action in the top
		// row, so keep their search inline even on mobile.
		if (!isListPage) {
			document.body.classList.remove("split-nav");
			return;
		}
		// Give the search its own row on narrow list pages so the logo and
		// list controls remain comfortably usable, including at high zoom levels.
		if (window.innerWidth <= 520) {
			document.body.classList.add("split-nav");
			return;
		}
		const bigInput = navEl?.querySelector("input:not(.small)");
		if (!bigInput) return;
		if (bigInput.getBoundingClientRect().width <= 45) {
			document.body.classList.add("split-nav");
		} else {
			document.body.classList.remove("split-nav");
		}
	}

	function docOnScroll() {
		if (scroll > window.scrollY) {
			navEl?.classList.remove("scrolled-down");
			document.body.classList.add("nav-shown");
		} else {
			navEl?.classList.add("scrolled-down");
			document.body.classList.remove("nav-shown");
			closeMenus();
		}
		scroll = window.scrollY;
	}

	function handleGlobalKeybind(ev: KeyboardEvent) {
		if (ev.key.toLowerCase() === "s" && ev.ctrlKey) {
			ev.preventDefault();
			mainSearchEl?.focus();
		}
	}

	async function restoreSignedInViewer() {
		const [userResult, followsResult] = await Promise.allSettled([
			optionalAuthReq.get<PrivateUser>("/user"),
			optionalAuthReq.get<Follow[]>("/follow"),
		]);

		if (userResult.status === "fulfilled") {
			store.userInfo = userResult.value;
		} else {
			console.warn(
				"Public layout could not restore signed-in viewer",
				userResult.reason,
			);
			if (
				ReqerError.isReqerError(userResult.reason) &&
				userResult.reason.response?.status === 401
			) {
				localStorage.removeItem("token");
			}
			store.userInfo = undefined;
		}

		if (followsResult.status === "fulfilled") {
			store.follows = followsResult.value;
		} else {
			console.warn(
				"Public layout could not load follows",
				followsResult.reason,
			);
			store.follows = [];
		}
	}

	onMount(() => {
		const token = localStorage.getItem("token");
		if (token) {
			restoreSignedInViewer();
		}

		scroll = window.scrollY;
		decideOnNavSplit();
		window.addEventListener("resize", decideOnNavSplit);
		window.document.addEventListener("scroll", docOnScroll);
		window.document.addEventListener("keydown", handleGlobalKeybind);

		return () => {
			restoreWatchedListState();
			clearTimeout(searchTimeout);
			window.removeEventListener("resize", decideOnNavSplit);
			window.document.removeEventListener("scroll", docOnScroll);
			window.document.removeEventListener("keydown", handleGlobalKeybind);
			document.body.classList.remove("split-nav", "nav-shown");
		};
	});

	afterNavigate(() => {
		decideOnNavSplit();
		closeMenus();
	});
</script>

{#snippet navActions()}
	{#if isListPage || isPersonPage}
		<div class="control">
			<button
				class="plain other detailedView"
				onclick={() => {
					closeMenus("detailed");
					detailedMenuShown = !detailedMenuShown;
				}}
				use:tooltip={{
					text: "Detailed View",
					pos: "bot",
					condition: !detailedMenuShown,
				}}
			>
				<Icon i="eye" />
			</button>
			{#if detailedMenuShown}
				<DetailedMenu
					conf={{
						width: "200px",
						top: "49px",
						right: "0",
						arrowRight: "2px",
					}}
				/>
			{/if}
		</div>
	{/if}
	{#if isListPage}
		<div class="control">
			<button
				class="plain other sort"
				onclick={() => {
					closeMenus("sort");
					sortMenuShown = !sortMenuShown;
				}}
				use:tooltip={{
					text: "Sort",
					pos: "bot",
					condition: !sortMenuShown,
				}}
			>
				<Icon i="sort" />
				{#if store.activeSort?.length === 2 && store.activeSort[1] && JSON.stringify(store.activeSort) !== JSON.stringify(defaultSort)}
					<span class="indicator"></span>
				{/if}
			</button>
			{#if sortMenuShown}
				<SortMenu
					conf={{
						width: "180px",
						top: "49px",
						right: "0",
						arrowRight: "2px",
					}}
				/>
			{/if}
		</div>
		<div class="control">
			<button
				class="plain other filter"
				onclick={() => {
					closeMenus("filter");
					filterMenuShown = !filterMenuShown;
				}}
				use:tooltip={{
					text: "Filter",
					pos: "bot",
					condition: !filterMenuShown,
				}}
			>
				<Icon i="filter" />
				{#if store.hasActiveFilters}
					<span class="indicator"></span>
				{/if}
			</button>
			{#if filterMenuShown}
				<FilterMenu
					showGames={true}
					conf={{
						width: "200px",
						top: "49px",
						right: "0",
						arrowRight: "2px",
					}}
				/>
			{/if}
		</div>
	{/if}
	<a
		class="plain-btn other session"
		href={resolve(isAuthenticated ? "/" : "/login")}
		use:tooltip={{
			text: isAuthenticated ? "My List" : "Log In",
			pos: "bot",
		}}
	>
		<Icon i="person" wh={24} />
	</a>
{/snippet}

<NavShell
	bind:navEl
	bind:mainSearchEl
	{homeHref}
	searchPlaceholder="Search this list"
	bind:searchValue={store.searchQuery}
	onSearch={handleSearch}
	actions={navActions}
	variant="public"
/>

{@render children?.()}
