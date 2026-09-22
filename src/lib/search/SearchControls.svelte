<script lang="ts">
	import MediaTypeFilter from "./MediaTypeFilter.svelte";
	import {
		hasPeopleSearch,
		toggleSearchType,
		type SelectableSearchType,
	} from "./searchTypes";

	interface Props {
		activeTypes: SelectableSearchType[];
		global: boolean;
		localLabel: string;
		disabled?: boolean;
		showGames?: boolean;
		onTypesChange: (types: SelectableSearchType[]) => void;
		onScopeChange: (global: boolean) => void;
	}

	let {
		activeTypes,
		global,
		localLabel,
		disabled = false,
		showGames = false,
		onTypesChange,
		onScopeChange,
	}: Props = $props();
	let peopleSelected = $derived(hasPeopleSearch(activeTypes));
</script>

<div class="search-controls">
	<div class="type-row">
		<MediaTypeFilter
			active={activeTypes}
			{disabled}
			{showGames}
			onChange={(type) => {
				onTypesChange(toggleSearchType(activeTypes, type));
			}}
		/>
	</div>
	<div class="source-control">
		<span class="control-label">Show results from:</span>
		<button
			class="plain source-toggle"
			class:global
			class:locked={peopleSelected}
			type="button"
			role="switch"
			aria-checked={global}
			aria-label="Show results from"
			disabled={peopleSelected}
			onclick={() => onScopeChange(!global)}
		>
			<span class="source-option local">{localLabel}</span>
			<span class="source-option global">Global</span>
		</button>
	</div>
</div>

<style lang="scss">
	.search-controls {
		flex: 1 1 100%;
		min-width: 0;
		display: flex;
		flex-flow: column;
		gap: 12px;
	}

	.type-row {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		width: 100%;
		min-width: 0;
	}

	.source-control {
		display: flex;
		flex-flow: column;
		gap: 8px;
		width: 100%;
		min-width: 0;
		padding: 8px;
		border: 1px solid rgba($color: $accent-color, $alpha: 0.45);
		border-radius: 10px;
		background-color: rgba($color: $accent-color, $alpha: 0.08);
		box-shadow: 0 3px 12px rgba(0, 0, 0, 0.14);
	}

	.control-label {
		display: block;
		font-size: 14px;
		font-weight: 600;
		line-height: 1.2;
		text-align: left;
		color: $text-color;
	}

	.source-toggle {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
		min-height: 42px;
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
			transition: transform 150ms ease;
		}

		&.global::before {
			transform: translateX(calc(100% + 1px));
		}

		&:disabled {
			cursor: not-allowed;
			opacity: 1;
		}

		.source-option {
			position: relative;
			z-index: 1;
			flex: 1 1 0;
			padding: 7px 8px;
			border-radius: 5px;
			text-align: center;
			cursor: pointer;
			transition:
				background-color 150ms ease,
				color 150ms ease;

			&:hover {
				color: $bg-color;
				background-color: $accent-color-hover;
			}
		}

		&:not(.global) .source-option.local,
		&.global .source-option.global {
			color: $bg-color;
			font-weight: 600;
		}

		&.locked .source-option.local {
			opacity: 0.4;
		}
	}

	@media screen and (max-width: 600px) {
		.type-row {
			flex-flow: column;
			align-items: stretch;
		}
	}
</style>
