<script lang="ts">
	import Icon from "@/lib/Icon.svelte";
	import {
		clearActiveFilters,
		defaultSort,
		setWatchedListPreset,
		store,
	} from "@/store.svelte";
	import WatchedListMediaToggle from "./WatchedListMediaToggle.svelte";

	interface Props {
		showGames?: boolean;
	}

	let { showGames = Boolean(store.serverFeatures?.games) }: Props = $props();

	function showAllItems() {
		clearActiveFilters();
		store.activeSort = [...defaultSort];
	}
</script>

<div class="list-shortcuts">
	<div class="preset-toggle" role="group" aria-label="List shortcuts">
		<button
			class="plain"
			data-active={store.activeWatchedListPreset === "recentlyWatched"}
			onclick={() => setWatchedListPreset("recentlyWatched")}
		>
			<Icon i="film" wh={18} /> Finished
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

	{#if showGames}
		<WatchedListMediaToggle {showGames} placement="list" />
	{/if}
</div>

<style lang="scss">
	.list-shortcuts {
		display: flex;
		flex-flow: row wrap;
		gap: 10px 18px;
		align-items: center;
		justify-content: center;
		margin: 0 auto 15px auto;
	}

	.preset-toggle {
		display: flex;
		flex-flow: row wrap;
		gap: 10px;
		justify-content: center;
	}

	button {
		display: flex;
		flex-flow: row;
		align-items: center;
		justify-content: center;
		gap: 8px;
		min-height: 38px;
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

		&[data-active="true"] {
			color: $bg-color;
			fill: $bg-color;
			background-color: $accent-color-hover;
			outline: 3px solid $accent-color;
		}
	}

	@media (hover: hover) {
		button:hover {
			color: $bg-color;
			fill: $bg-color;
			background-color: $accent-color-hover;
		}
	}

	@media screen and (max-width: 520px) {
		.list-shortcuts {
			flex-direction: column;
			gap: 10px;
		}

		.preset-toggle {
			width: 100%;
		}
	}
</style>
