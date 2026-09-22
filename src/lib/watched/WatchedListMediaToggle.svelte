<script lang="ts">
	import Icon from "@/lib/Icon.svelte";
	import {
		setWatchedListMediaGroup,
		store,
		type WatchedListMediaGroup,
	} from "@/store.svelte";

	interface Props {
		showGames?: boolean;
		placement?: "nav" | "list";
	}

	let {
		showGames = Boolean(store.serverFeatures?.games),
		placement = "list",
	}: Props = $props();

	let activeMediaGroup = $derived.by(() => {
		const types = store.activeFilters.type;
		if (types.length === 1 && types[0] === "game") return "games";
		if (types.length === 2 && types.includes("movie") && types.includes("tv")) {
			return "moviesTv";
		}
		return undefined;
	});

	function selectMediaGroup(mediaGroup: WatchedListMediaGroup) {
		if (activeMediaGroup === mediaGroup) return;
		setWatchedListMediaGroup(mediaGroup);
		window.scrollTo({ top: 0 });
	}
</script>

{#if showGames}
	<div
		class="media-toggle"
		class:nav-media={placement === "nav"}
		class:list-media={placement === "list"}
		class:games={activeMediaGroup === "games"}
		class:has-selection={activeMediaGroup !== undefined}
		role="group"
		aria-label="Media type"
	>
		<button
			class="plain media-option"
			class:active={activeMediaGroup === "moviesTv"}
			aria-pressed={activeMediaGroup === "moviesTv"}
			onclick={() => selectMediaGroup("moviesTv")}
		>
			<Icon i="film" wh={18} /> Movies &amp; TV
		</button>
		<button
			class="plain media-option"
			class:active={activeMediaGroup === "games"}
			aria-pressed={activeMediaGroup === "games"}
			onclick={() => selectMediaGroup("games")}
		>
			<Icon i="gamepad" wh={18} /> Games
		</button>
	</div>
{/if}

<style lang="scss">
	.media-toggle {
		position: relative;
		display: flex;
		align-items: center;
		width: 256px;
		min-width: 256px;
		flex: 0 0 auto;
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
			opacity: 0;
			pointer-events: none;
			transition:
				transform 150ms ease,
				opacity 150ms ease;
		}

		&.has-selection::before {
			opacity: 1;
		}

		&.games::before {
			transform: translateX(calc(100% + 1px));
		}

		.media-option {
			position: relative;
			z-index: 1;
			display: flex;
			flex: 1 1 0;
			align-items: center;
			justify-content: center;
			gap: 8px;
			min-height: 32px;
			padding: 7px 8px;
			border: 0;
			border-radius: 5px;
			color: $text-color;
			fill: $text-color;
			text-align: center;
			white-space: nowrap;
			cursor: pointer;
			transition:
				background-color 150ms ease,
				color 150ms ease;

			&.active {
				color: $bg-color;
				fill: $bg-color;
				font-weight: 600;
			}
		}
	}

	.nav-media {
		margin-right: 12px;

		@media screen and (max-width: 900px) {
			display: none;
		}
	}

	.list-media {
		@media screen and (min-width: 901px) {
			display: none;
		}
	}

	@media (hover: hover) {
		.media-option:hover {
			color: $bg-color;
			fill: $bg-color;
			background-color: $accent-color-hover;
		}
	}
</style>
