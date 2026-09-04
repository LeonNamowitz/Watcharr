<script lang="ts">
	import { afterNavigate, goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import Icon from "@/lib/Icon.svelte";
	import tooltip from "@/lib/actions/tooltip";
	import DetailedMenu from "@/lib/nav/DetailedMenu.svelte";
	import FilterMenu from "@/lib/nav/FilterMenu.svelte";
	import SortMenu from "@/lib/nav/SortMenu.svelte";
	import { optionalAuthReq } from "@/lib/util/api";
	import { ReqerError } from "@/lib/util/fetch";
	import { isTouch } from "@/lib/util/helpers";
	import { defaultSort, store } from "@/store.svelte";
	import type { Follow, PrivateUser } from "@/types";
	import { onMount } from "svelte";

	interface Props {
		children?: import("svelte").Snippet;
	}

	let { children }: Props = $props();

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

	onMount(() => {
		const token = localStorage.getItem("token");
		if (token) {
			Promise.all([
				optionalAuthReq.get<PrivateUser>("/user"),
				optionalAuthReq.get<Follow[]>("/follow"),
			])
				.then(([user, follows]) => {
					store.userInfo = user;
					store.follows = follows;
				})
				.catch((err) => {
					console.warn("Public layout could not restore signed-in viewer", err);
					if (ReqerError.isReqerError(err) && err.response?.status === 401) {
						localStorage.removeItem("token");
					}
					store.userInfo = undefined;
					store.follows = [];
				});
		}

		scroll = window.scrollY;
		decideOnNavSplit();
		window.addEventListener("resize", decideOnNavSplit);
		window.document.addEventListener("scroll", docOnScroll);
		window.document.addEventListener("keydown", handleGlobalKeybind);

		return () => {
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

<nav bind:this={navEl}>
	<div class="wrapper">
		<div class="left-side">
			<a href={resolve(`/lists/${page.params.id}/${page.params.username}`)}>
				<span class="large">Watcharr</span>
				<span class="small">W</span>
			</a>
		</div>
		<div class="search">
			<input
				bind:this={mainSearchEl}
				type="text"
				placeholder="Search this list"
				bind:value={store.searchQuery}
				onkeydown={handleSearch}
			/>
			<Icon i="search" wh={19} />
		</div>
		<div class="btns">
			{#if isListPage}
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
		</div>
	</div>
	<input
		class="small"
		type="text"
		placeholder="Search this list"
		bind:value={store.searchQuery}
		onkeydown={handleSearch}
	/>
</nav>

{@render children?.()}

<style lang="scss">
	nav {
		display: flex;
		flex-flow: column;
		margin-bottom: 20px;
		padding: 10px 20px;
		position: sticky;
		top: 0;
		gap: 3px;
		z-index: 99990;
		transition: top 200ms ease-in-out;
		@include nav-blur;

		&:global(.scrolled-down) {
			top: -110px;
		}

		.wrapper {
			display: flex;
			flex-flow: row;
			gap: 20px;
			justify-content: space-between;
			align-items: center;

			.left-side,
			.btns {
				flex: 1;
			}

			@media screen and (max-width: 435px) {
				gap: 15px;
			}

			body.split-nav & {
				@media screen and (max-width: 380px) {
					gap: 10px;
				}

				@media screen and (max-width: 375px) {
					gap: 8px;
				}

				@media screen and (max-width: 370px) {
					gap: 5px;
				}

				@media screen and (max-width: 350px) {
					gap: 0;
				}
			}
		}

		.left-side {
			a {
				display: inline-flex;
				text-decoration: none;
				font-family:
					"Shrikhand",
					system-ui,
					-apple-system,
					BlinkMacSystemFont;
				font-size: 35px;
				transition:
					-webkit-text-stroke 150ms ease,
					color 150ms ease,
					font-weight 150ms ease;

				&:hover,
				&:focus-visible {
					color: $bg-color;
					-webkit-text-stroke: 3px $text-color;
					font-weight: bold;
				}

				span.large {
					display: block;
					width: 185.2px;
				}

				span.small {
					display: none;
					width: 40px;
				}

				@media screen and (max-width: 620px) {
					span.large {
						display: none;
					}

					span.small {
						display: block;
					}
				}
			}
		}

		.search {
			width: 100%;
			position: relative;
			margin-bottom: 2px;

			:global(svg) {
				display: none;
				position: absolute;
				top: 50%;
				left: 50%;
				transform: translate(-50%, -50%);
				pointer-events: none;
				user-select: none;
			}

			input:focus-within + :global(svg),
			input:not(:placeholder-shown) + :global(svg) {
				display: none;
			}

			@media screen and (min-width: 666px) {
				max-width: 250px;
			}

			@media screen and (max-width: 666px) {
				& input:not(.small) {
					width: 100%;
				}

				&:focus-within + .btns .control {
					display: none;
				}
			}

			@media screen and (max-width: 460px) {
				:global(svg) {
					display: block;
				}

				input::placeholder {
					color: transparent;
				}
			}
		}

		:global(body.split-nav) & {
			.search {
				opacity: 0;
				visibility: hidden;
			}

			input.small {
				display: block;
			}
		}

		input {
			width: 100%;
			font-weight: bold;
			text-align: center;
			box-shadow: 4px 4px 0 0 $text-color;
			text-overflow: ellipsis;
			transition:
				width 150ms ease,
				box-shadow 150ms ease;

			&.small {
				display: none;
				margin-left: auto;
				margin-right: auto;
			}

			&:hover,
			&:focus {
				box-shadow: 2px 2px 0 0 $text-color;
			}

			@media screen and (max-width: 290px) {
				&.small {
					width: 100%;
				}
			}
		}

		.btns {
			display: flex;
			flex-flow: row;
			justify-content: end;
			align-items: center;
			min-width: max-content;

			.control {
				position: relative;
				margin-right: 12px;
			}

			.other {
				display: flex;
				align-items: center;
				justify-content: center;
				padding-top: 2px;
				width: 28px;
				height: 32px;
				transition:
					fill 150ms ease,
					stroke 150ms ease,
					stroke-width 150ms ease;
				fill: $text-color;

				&:hover,
				&:focus-visible {
					:global(path) {
						fill: none;
						stroke: $text-color;
						stroke-width: 30px;
						stroke-linejoin: round;
					}
				}
			}

			.filter {
				&:hover,
				&:focus-visible {
					:global(path) {
						stroke-width: 15px;
					}
				}
			}

			.filter,
			.sort {
				position: relative;

				.indicator {
					position: absolute;
					top: 1px;
					right: -6px;
					width: 6px;
					height: 6px;
					background-color: $text-color;
					border-radius: 50%;
				}
			}

			.session {
				flex: 0 0 28px;
			}
		}
	}
</style>
