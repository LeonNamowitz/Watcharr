<script lang="ts">
	import { resolve } from "$app/paths";
	import type { Snippet } from "svelte";
	import Icon from "../Icon.svelte";

	interface Props {
		navEl?: HTMLElement;
		mainSearchEl?: HTMLInputElement;
		homeHref: "/" | `/lists/${string}/${string}`;
		searchPlaceholder: string;
		searchValue: string;
		onSearch: (event: KeyboardEvent) => void;
		actions: Snippet;
		variant: "app" | "public";
	}

	let {
		navEl = $bindable(),
		mainSearchEl = $bindable(),
		homeHref,
		searchPlaceholder,
		searchValue = $bindable(),
		onSearch,
		actions,
		variant,
	}: Props = $props();
</script>

<nav bind:this={navEl} class={variant}>
	<div class="wrapper">
		<div class="left-side">
			<a href={resolve(homeHref)}>
				<span class="large">Watcharr</span>
				<span class="small">W</span>
			</a>
		</div>
		<div class="search">
			<input
				bind:this={mainSearchEl}
				type="text"
				placeholder={searchPlaceholder}
				bind:value={searchValue}
				onkeydown={onSearch}
			/>
			<Icon i="search" wh={19} />
		</div>
		<div class="btns">
			{@render actions()}
		</div>
	</div>
	<input
		class="small"
		type="text"
		placeholder={searchPlaceholder}
		bind:value={searchValue}
		onkeydown={onSearch}
	/>
</nav>

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
				input:not(.small) {
					width: 100%;
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

		@media screen and (max-width: 666px) {
			&.app .search:focus-within + .btns :global(button:not(.face)) {
				display: none;
			}

			&.public .search:focus-within + .btns :global(.control) {
				display: none;
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

			:global(.other) {
				padding-top: 2px;
				width: 28px;
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

			:global(.filter) {
				&:hover,
				&:focus-visible {
					:global(path) {
						stroke-width: 15px;
					}
				}
			}

			:global(.filter),
			:global(.sort) {
				position: relative;

				:global(.indicator) {
					position: absolute;
					top: 1px;
					right: -6px;
					width: 6px;
					height: 6px;
					background-color: $text-color;
					border-radius: 50%;
				}
			}
		}

		&.public .btns {
			align-items: center;
			min-width: max-content;

			:global(.control) {
				position: relative;
				margin-right: 12px;
			}

			:global(.other) {
				display: flex;
				align-items: center;
				justify-content: center;
				height: 32px;
			}

			:global(.session) {
				flex: 0 0 28px;
			}
		}

		&.app .btns {
			> :global(button:not(.face)) {
				margin-right: 12px;
			}

			:global(.discover) {
				transition:
					fill 150ms ease,
					stroke 150ms ease,
					stroke-width 150ms ease,
					transform 150ms ease;

				&:hover,
				&:focus-visible {
					transform: rotate(60deg);
				}
			}

			:global(.following) {
				margin-right: 17px;
			}

			:global(.face) {
				font-family:
					"Shrikhand",
					system-ui,
					-apple-system,
					BlinkMacSystemFont;
				font-size: 25px;
				transform: rotate(90deg);
				cursor: pointer;
				margin-left: 3px;
				transition:
					-webkit-text-stroke 150ms ease,
					color 150ms ease;

				&:hover,
				&:focus-visible {
					color: $bg-color;
					-webkit-text-stroke: 1.5px $text-color;
				}
			}
		}
	}

	:global(body.split-nav) nav {
		.search {
			opacity: 0;
			visibility: hidden;
		}

		input.small {
			display: block;
		}

		.wrapper {
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
</style>
