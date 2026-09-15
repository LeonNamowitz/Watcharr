<script lang="ts">
	import { store, clearActiveFilters } from "@/store.svelte";
	import type { Filters } from "@/types";
	import Icon from "../Icon.svelte";
	import tooltip from "../actions/tooltip";
	import Menu, { type MenuConfig } from "../Menu.svelte";
	import { SearchType } from "@/types";
	import {
		toggleSearchType,
		type SelectableSearchType,
	} from "@/lib/search/searchTypes";

	interface Props {
		conf?: MenuConfig;
		anchor?: HTMLElement;
		showGames?: boolean;
		showTypes?: boolean;
		searchTypes?: SelectableSearchType[];
		onSearchTypesChange?: (types: SelectableSearchType[]) => void;
	}

	let {
		conf,
		anchor,
		showGames = store.serverFeatures?.games,
		showTypes = true,
		searchTypes,
		onSearchTypesChange,
	}: Props = $props();
	let controlledTypes = $derived(searchTypes !== undefined);
	let hasFilters = $derived(
		store.activeFilters.status.length > 0 ||
			(controlledTypes
				? (searchTypes?.length ?? 0) > 0
				: store.activeFilters.type.length > 0),
	);

	function typeIsActive(searchType: SelectableSearchType, watchedType: string) {
		return controlledTypes
			? searchTypes?.includes(searchType)
			: store.activeFilters.type.includes(watchedType);
	}

	function typeClicked(searchType: SelectableSearchType, watchedType: string) {
		if (controlledTypes && searchTypes) {
			onSearchTypesChange?.(toggleSearchType(searchTypes, searchType));
			return;
		}
		filterClicked("type", watchedType);
	}

	function filterClicked(type: keyof Filters, f: string) {
		if (store.activeFilters[type]?.includes(f)) {
			store.activeFilters[type] = store.activeFilters[type]?.filter(
				(a) => a !== f,
			);
		} else {
			store.activeFilters[type] = [...store.activeFilters[type], f];
		}
		store.activeFilters = store.activeFilters;
		window.scrollTo({ top: 0 });
	}
</script>

<Menu
	{anchor}
	conf={conf ?? { width: "200px", right: "47px", arrowLeft: "38px" }}
>
	<div class="title">
		{#if showTypes}
			<h4 class="norm sm-caps">type</h4>
		{/if}
		{#if hasFilters}
			<button
				class="plain"
				use:tooltip={{ text: "Clear", pos: "left" }}
				onclick={() => {
					clearActiveFilters();
					onSearchTypesChange?.([]);
					window.scrollTo({ top: 0 });
				}}
			>
				<Icon i="close-circle" wh={18} />
			</button>
		{/if}
	</div>
	{#if showTypes}
		<div class="type-filter">
			<button
				class:active={typeIsActive(SearchType.show, "tv")}
				onclick={() => typeClicked(SearchType.show, "tv")}
			>
				SHOW
			</button>
			<button
				class:active={typeIsActive(SearchType.movie, "movie")}
				onclick={() => typeClicked(SearchType.movie, "movie")}
			>
				MOVIE
			</button>
			{#if showGames}
				<button
					class:active={typeIsActive(SearchType.game, "game")}
					onclick={() => typeClicked(SearchType.game, "game")}
				>
					GAME
				</button>
			{/if}
		</div>
	{/if}
	<h4 class="norm sm-caps">status</h4>
	<button
		class={`plain ${store.activeFilters.status.includes("planned") ? "on" : ""}`}
		onclick={() => filterClicked("status", "planned")}
	>
		planned
	</button>
	<button
		class={`plain ${store.activeFilters.status.includes("watching") ? "on" : ""}`}
		onclick={() => filterClicked("status", "watching")}
	>
		watching
		{#if showGames}
			(playing)
		{/if}
	</button>
	<button
		class={`plain ${store.activeFilters.status.includes("finished") ? "on" : ""}`}
		onclick={() => filterClicked("status", "finished")}
	>
		finished
		{#if showGames}
			(played)
		{/if}
	</button>
	<button
		class={`plain ${store.activeFilters.status.includes("hold") ? "on" : ""}`}
		onclick={() => filterClicked("status", "hold")}
	>
		on hold
	</button>
	<button
		class={`plain ${store.activeFilters.status.includes("dropped") ? "on" : ""}`}
		onclick={() => filterClicked("status", "dropped")}
	>
		dropped
	</button>
</Menu>

<style lang="scss">
	h4:not(:first-child) {
		margin-top: 8px;
		margin-bottom: 8px;
	}

	.title {
		display: flex;
		flex-flow: row;
		align-items: center;
		margin-bottom: 8px;
		gap: 5px;
		/* Always height of when clear filters btn is shown so there is no jump */
		min-height: 26px;

		button.plain {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 28px;
			height: 26px;
			padding: 2px 3px;
			border-radius: 8px;

			&:first-of-type {
				margin-left: auto;
			}
		}
	}

	button.plain {
		text-transform: capitalize;
		position: relative;

		&.on::before {
			content: "\2713";
		}

		&::before {
			position: absolute;
			top: 4px;
			left: 7.5px;
			font-family:
				system-ui,
				-apple-system,
				BlinkMacSystemFont;
			font-size: 18px;
		}
	}

	.type-filter {
		display: flex;
		flex-flow: row;
		flex-wrap: wrap;
		gap: 3px;
		width: 100%;

		button {
			flex: 1 1 45%;
			padding: 8px 0;
			border-radius: 10px;

			&:hover,
			&.active {
				color: $bg-color;
				background-color: $accent-color-hover;
			}
		}
	}
</style>
