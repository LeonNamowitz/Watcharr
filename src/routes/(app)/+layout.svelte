<script lang="ts">
	import { afterNavigate, goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import tooltip from "@/lib/actions/tooltip";
	import DetailedMenu from "@/lib/nav/DetailedMenu.svelte";
	import FaceMenu from "@/lib/nav/FaceMenu.svelte";
	import FilterMenu from "@/lib/nav/FilterMenu.svelte";
	import FollowingMenu from "@/lib/nav/FollowingMenu.svelte";
	import NavShell from "@/lib/nav/NavShell.svelte";
	import SortMenu from "@/lib/nav/SortMenu.svelte";
	import TagMenu from "@/lib/tag/TagMenu.svelte";
	import { req } from "@/lib/util/api";
	import { isTouch } from "@/lib/util/helpers";
	import { store, defaultSort } from "@/store.svelte";
	import type {
		ServerFeatures,
		Follow,
		PrivateUser,
		Tag,
		UserSettings,
	} from "@/types";
	import { onMount } from "svelte";
	import { SvelteURLSearchParams } from "svelte/reactivity";
	interface Props {
		children?: import("svelte").Snippet;
	}

	let { children }: Props = $props();

	let navEl: HTMLElement | undefined = $state();
	let mainSearchEl: HTMLInputElement | undefined = $state();
	let searchTimeout: number;
	let subMenuShown = $state(false);
	let filterMenuShown = $state(false);
	let sortMenuShown = $state(false);
	let followingMenuShown = $state(false);
	let detailedMenuShown = $state(false);
	let tagMenuShown = $state(false);
	let tagOrderEditMode = $state(false);
	let scroll = window.scrollY;

	function handleProfileClick() {
		if (!localStorage.getItem("token")) {
			goto(resolve("/login"));
		} else {
			closeAllSubMenus("sub");
			subMenuShown = !subMenuShown;
		}
	}

	function handleSearch(ev: KeyboardEvent) {
		if (
			ev.key === "ContextMenu" ||
			ev.key === "Home" ||
			ev.key === "End" ||
			ev.key === "PageDown" ||
			ev.key === "PageUp" ||
			ev.key === "NumLock" ||
			ev.key === "Escape" ||
			ev.key === "Tab" ||
			ev.key === "CapsLock" ||
			ev.key === "OS" ||
			ev.key === "ArrowLeft" ||
			ev.key === "ArrowRight" ||
			ev.key === "ArrowUp" ||
			ev.key === "ArrowDown" ||
			ev.key === "Control" ||
			ev.key === "Alt" ||
			ev.key === "AltGraph" ||
			ev.key === "Shift" ||
			ev.key === "Meta"
		)
			return;
		clearTimeout(searchTimeout);
		searchTimeout = window.setTimeout(
			() => {
				const target = ev.target as HTMLInputElement;
				const query = target?.value.trim();
				if (!query) return;
				const currentSearchType = page.url.searchParams.get("type");
				const searchParams = new SvelteURLSearchParams({
					query: encodeURIComponent(query),
					preferMyList: "true",
				});
				if (page.route?.id === "/(app)/search" && currentSearchType) {
					// If we are already on the search page, we can attempt
					// to keep any existing type filter on the next query.
					searchParams.set("type", currentSearchType);
				}
				// Enable autofocus before running `goto` because on chromium
				// the .focus() call won't work, even after a timeout.
				// Using autofocus seems to work. Disables after goto runs.
				// https://github.com/sbondCo/Watcharr/issues/169
				target.autofocus = true;
				goto(resolve(`/search?${searchParams.toString()}`)).then(() => {
					// Use mainSearchEl if nav not split, otherwise use ev target.
					if (!document.body.classList.contains("split-nav") && mainSearchEl) {
						mainSearchEl.focus();
						mainSearchEl.autofocus = false;
					} else {
						target?.focus();
					}
					target.autofocus = false;
				});
			},
			isTouch() ? 800 : 400,
		);
	}

	async function getInitialData() {
		if (!localStorage.getItem("token")) {
			console.warn("getInitialData: No token found, redirecting to login!");
			goto(resolve("/login?again=1"));
			return;
		}
		const [u, s, f, fo, ts] = await Promise.all([
			req.get<PrivateUser>("/user"),
			req.get<UserSettings>("/user/settings"),
			req.get<ServerFeatures>("/features"),
			req.get<Follow[]>("/follow"),
			req.get<Tag[]>("/tag"),
		]);
		if (u) {
			store.userInfo = u;
		}
		if (s) {
			store.userSettings = s;
		}
		if (f) {
			store.serverFeatures = f;
		}
		if (fo) {
			store.follows = fo;
		}
		if (ts) {
			store.tags = ts;
		}
	}

	function closeAllSubMenus(except?: string) {
		if (except !== "sub") subMenuShown = false;
		if (except !== "filter") filterMenuShown = false;
		if (except !== "sort") sortMenuShown = false;
		if (except !== "following") followingMenuShown = false;
		if (except !== "detailed") detailedMenuShown = false;
		if (except !== "tag") {
			tagMenuShown = false;
			tagOrderEditMode = false;
		}
	}

	/**
	 * Adds or removed `split-nav` tag to body depending
	 * on how big the main search bar is.
	 */
	function decideOnNavSplit() {
		if (window.innerWidth <= 305) {
			document.body.classList.add("split-nav");
			return;
		}
		const bigInput = navEl?.querySelector("input:not(.small)");
		if (bigInput) {
			const b = bigInput.getBoundingClientRect();
			console.debug("decideOnNavSplit: bigInput width:", b.width);
			if (b.width <= 45) {
				document.body.classList.add("split-nav");
				console.debug("decideOnNavSplit: Splitting nav.");
			} else {
				document.body.classList.remove("split-nav");
				console.debug("decideOnNavSplit: Unsplitting nav.");
			}
		} else {
			console.warn("decideOnNavSplit: bigInput not found!", bigInput);
		}
	}

	function docOnScroll() {
		if (scroll > window.scrollY) {
			navEl?.classList.remove("scrolled-down");
			document.body.classList.add("nav-shown");
		} else {
			navEl?.classList.add("scrolled-down");
			document.body.classList.remove("nav-shown");
			if (!tagOrderEditMode) closeAllSubMenus();
		}
		scroll = window.scrollY;
	}

	function focusSearch() {
		try {
			if (!mainSearchEl) {
				console.warn("focusSearch: mainSearchEl not defined!");
				return;
			}
			if (document.activeElement === mainSearchEl) {
				console.debug("focusSearch: mainSearchEl is already focused.");
				return;
			}
			mainSearchEl.focus();
		} catch (err) {
			console.error("focusSearch: Failed!", err);
		}
	}

	function handleGlobalKeybind(ev: KeyboardEvent) {
		switch (ev.key.toLowerCase()) {
			case "s":
				if (ev.ctrlKey) {
					ev.preventDefault();
					focusSearch();
				}
				break;
		}
	}

	afterNavigate(() => {
		decideOnNavSplit();
		closeAllSubMenus();
	});

	onMount(() => {
		if (navEl) {
			decideOnNavSplit();
			window.addEventListener("resize", decideOnNavSplit);
			window.document.addEventListener("scroll", docOnScroll);
			window.document.addEventListener("keydown", handleGlobalKeybind);

			return () => {
				window.removeEventListener("resize", decideOnNavSplit);
				window.document.removeEventListener("scroll", docOnScroll);
				window.document.removeEventListener("keydown", handleGlobalKeybind);
			};
		} else {
			console.error(
				"navEl doesn't exist, failed to initialize up/down listener",
			);
		}
	});
</script>

{#snippet navActions()}
	<!-- Detailed posters supported on watched lists, tags, search and people. -->
	{#if page.url?.pathname === "/" || page.url?.pathname.startsWith("/search") || page.url?.pathname.startsWith("/tag") || page.url?.pathname.startsWith("/person")}
		<button
			class="plain other detailedView"
			onclick={() => {
				closeAllSubMenus("detailed");
				detailedMenuShown = !detailedMenuShown;
			}}
			use:tooltip={{
				text: "Detailed View",
				pos: "bot",
				condition: !detailedMenuShown,
			}}
		>
			<Icon i="eye" />
			{#if store.activeFilters?.type?.length > 0 || store.activeFilters?.status?.length > 0}
				<div class="indicator"></div>
			{/if}
		</button>
		{#if detailedMenuShown}
			<DetailedMenu />
		{/if}
	{/if}
	<!-- Show on the watched list and tag lists. -->
	{#if page.url?.pathname === "/" || page.url?.pathname.includes("/tag/")}
		<button
			class="plain other sort"
			onclick={() => {
				closeAllSubMenus("sort");
				sortMenuShown = !sortMenuShown;
			}}
			use:tooltip={{ text: "Sort", pos: "bot", condition: !sortMenuShown }}
		>
			<Icon i="sort" />
			<!-- Show indicator if not equal to default and second item in array is not falsy -->
			{#if store.activeSort?.length === 2 && store.activeSort[1] && JSON.stringify(store.activeSort) !== JSON.stringify(defaultSort)}
				<div class="indicator"></div>
			{/if}
		</button>
		<button
			class="plain other filter"
			onclick={() => {
				closeAllSubMenus("filter");
				filterMenuShown = !filterMenuShown;
			}}
			use:tooltip={{
				text: "Filter",
				pos: "bot",
				condition: !filterMenuShown,
			}}
		>
			<Icon i="filter" />
			{#if store.activeFilters?.type?.length > 0 || store.activeFilters?.status?.length > 0}
				<div class="indicator"></div>
			{/if}
		</button>
		{#if filterMenuShown}
			<FilterMenu />
		{/if}
		{#if sortMenuShown}
			<SortMenu />
		{/if}
	{/if}
	<button
		class="plain other tag"
		onclick={() => {
			tagOrderEditMode = false;
			closeAllSubMenus("tag");
			tagMenuShown = !tagMenuShown;
		}}
		use:tooltip={{ text: "Tags", pos: "bot", condition: !tagMenuShown }}
	>
		<Icon i="tag" />
	</button>
	{#if tagMenuShown}
		<TagMenu
			onTagClick={(tag) => {
				goto(resolve(`/tag/${tag.id}`));
				tagMenuShown = false;
			}}
			showManageBtn={true}
			onOrderEditModeChange={(editing) => (tagOrderEditMode = editing)}
		/>
	{/if}
	<button
		class="plain other discover"
		onclick={() => goto(resolve("/discover"))}
		use:tooltip={{ text: "Discover", pos: "bot" }}
	>
		<Icon i="compass" wh={26} />
	</button>
	<button
		class="plain other following"
		onclick={() => {
			closeAllSubMenus("following");
			followingMenuShown = !followingMenuShown;
		}}
		use:tooltip={{
			text: "Following",
			pos: "bot",
			condition: !followingMenuShown,
		}}
	>
		<Icon i="people" wh={26} />
	</button>
	{#if followingMenuShown}
		<FollowingMenu close={() => (followingMenuShown = false)} />
	{/if}
	<button class="plain face" onclick={handleProfileClick}>:)</button>
	{#if subMenuShown}
		<FaceMenu />
	{/if}
{/snippet}

<NavShell
	bind:navEl
	bind:mainSearchEl
	homeHref="/"
	searchPlaceholder="Search"
	bind:searchValue={store.searchQuery}
	onSearch={handleSearch}
	actions={navActions}
	variant="app"
/>

{#await getInitialData()}
	<Spinner />
{:then}
	{@render children?.()}
{:catch err}
	<Error
		pretty="Couldn't fetch app data!"
		error={err}
		onRetry={() => {
			location.reload();
		}}
	/>
{/await}
