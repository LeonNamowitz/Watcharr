<script lang="ts">
	import Icon from "../Icon.svelte";
	import { store } from "@/store.svelte";
	import { SearchType } from "@/types";
	import type { SelectableSearchType } from "./searchTypes";

	interface Props {
		active?: string | string[];
		disabled?: boolean;
		showGames?: boolean;
		showAll?: boolean;
		onClear?: () => void;
		onChange: (nowActive: SelectableSearchType) => void;
	}

	let {
		active,
		disabled,
		showGames,
		showAll = false,
		onClear,
		onChange,
	}: Props = $props();
	let gamesVisible = $derived(showGames ?? store.serverFeatures?.games);
	let allActive = $derived(
		Array.isArray(active) ? active.length === 0 : !active,
	);
	let isActive = $derived((type: SelectableSearchType) =>
		Array.isArray(active) ? active.includes(type) : active === type,
	);
</script>

<div class:disabled>
	{#if showAll}
		<button class="plain" data-active={allActive} onclick={() => onClear?.()}>
			All
		</button>
	{/if}
	<button
		class="plain"
		data-active={isActive(SearchType.movie)}
		onclick={() => onChange(SearchType.movie)}
	>
		<Icon i="film" wh={20} /> Movies
	</button>
	<button
		class="plain"
		data-active={isActive(SearchType.show)}
		onclick={() => onChange(SearchType.show)}
	>
		<Icon i="tv" wh={20} /> TV Shows
	</button>
	{#if gamesVisible}
		<button
			class="plain"
			data-active={isActive(SearchType.game)}
			onclick={() => onChange(SearchType.game)}
		>
			<Icon i="gamepad" wh={20} /> Games
		</button>
	{/if}
	<button
		class="plain"
		data-active={isActive(SearchType.person)}
		onclick={() => onChange(SearchType.person)}
	>
		<Icon i="people-nocircle" wh={20} /> People
	</button>
</div>

<style lang="scss">
	div {
		display: flex;
		flex-flow: row;
		flex-wrap: wrap;
		gap: 10px;
		padding: 0 5px;

		button {
			display: flex;
			flex-flow: row;
			flex-wrap: wrap;
			gap: 8px;
			align-items: center;
			height: fit-content;
			padding: 8px 12px;
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

			@media screen and (max-width: 500px) {
				flex-flow: column;
				/* Keep the buttons all the same
				width so they look less weird. */
				width: 90px;
			}
		}

		&.disabled {
			button {
				opacity: 0.8;
				pointer-events: none;
			}
		}

		@media screen and (max-width: 500px) {
			width: 100%;
			justify-content: center;
		}

		/* Adjusting the gap to let the buttons stay on one line for
		longer. Makes it so it fits on one line for my phone, it's as
		far as I will take it for now. */

		@media screen and (max-width: 430px) {
			gap: 8px;
		}

		@media screen and (max-width: 424px) {
			gap: 6px;
		}

		@media screen and (max-width: 418px) {
			gap: 3px;
		}
	}
</style>
