<script lang="ts">
	import { store, clearActiveFilters } from "@/store.svelte";
	import type { Filters } from "@/types";
	import Icon from "../Icon.svelte";
	import tooltip from "../actions/tooltip";
	import Menu, { type MenuConfig } from "../Menu.svelte";

	interface Props {
		conf?: MenuConfig;
		showGames?: boolean;
		showTypes?: boolean;
	}

	let {
		conf,
		showGames = store.serverFeatures?.games,
		showTypes = true,
	}: Props = $props();

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

<Menu conf={conf ?? { width: "200px", right: "47px", arrowLeft: "38px" }}>
	<div class="title">
		{#if showTypes}
			<h4 class="norm sm-caps">type</h4>
		{/if}
		{#if store.activeFilters?.type?.length > 0 || store.activeFilters?.status?.length > 0}
			<button
				class="plain"
				use:tooltip={{ text: "Clear", pos: "left" }}
				onclick={() => {
					clearActiveFilters();
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
				class:active={store.activeFilters.type.includes("tv")}
				onclick={() => filterClicked("type", "tv")}
			>
				SHOW
			</button>
			<button
				class:active={store.activeFilters.type.includes("movie")}
				onclick={() => filterClicked("type", "movie")}
			>
				MOVIE
			</button>
			{#if showGames}
				<button
					class:active={store.activeFilters.type.includes("game")}
					onclick={() => filterClicked("type", "game")}
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
		}
	}
</style>
