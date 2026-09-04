<script lang="ts">
	import { store } from "@/store.svelte";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import { toRatingLabel } from "@/lib/rating/helpers";
	import {
		getOrdinalSuffix,
		monthsShort,
		watchedStatuses,
	} from "@/lib/util/helpers";
	import type { SupportedMedia, Watched } from "@/types";
	import Icon from "@/lib/Icon.svelte";
	import { toPublicStatusLabel } from "./helpers";

	interface Props {
		watched: Watched;
		mediaType: SupportedMedia;
		ratingSettings?: RatingSettings;
		active?: boolean;
	}

	let { watched, mediaType, ratingSettings, active = false }: Props = $props();

	let statusLabel = $derived(toPublicStatusLabel(watched.status, mediaType));
	let ratingLabel = $derived(toRatingLabel(watched.rating, ratingSettings));
	let showDateAdded = $derived(store.wlDetailedView.includes("dateAdded"));
	let showDateModified = $derived(
		store.wlDetailedView.includes("dateModified"),
	);
	let showLastWatched = $derived(
		Boolean(watched.watchingSeason) &&
			store.wlDetailedView.includes("lastWatched"),
	);
	let showOptional = $derived(
		!active && (showDateAdded || showDateModified || showLastWatched),
	);
	let showStatusRating = $derived(
		store.wlDetailedView.includes("statusRating"),
	);

	function formatDate(value: string) {
		const date = new Date(value);
		if (Number.isNaN(date.getTime())) return "Unknown";
		return `${date.getDate()}${getOrdinalSuffix(date.getDate())} ${
			monthsShort[date.getMonth()]
		} '${String(date.getFullYear()).substring(2, 4)}`;
	}
</script>

{#if showOptional || showStatusRating}
	<div class:active class="public-details">
		{#if showOptional}
			<div class="optional">
				{#if showDateAdded}
					<span title="Date added to watch list">
						<i><Icon i="calendar" /></i>
						<span>{formatDate(watched.createdAt)}</span>
					</span>
				{/if}
				{#if showDateModified}
					<span title="Date last modified">
						<i><Icon i="pencil" wh={15} /></i>
						<span>{formatDate(watched.updatedAt)}</span>
					</span>
				{/if}
				{#if showLastWatched}
					<span title="Latest season watched">
						<i><Icon i="play" wh={15} /></i>
						<span>{watched.watchingSeason}</span>
					</span>
				{/if}
			</div>
		{/if}

		{#if showStatusRating}
			<div class="status-rating">
				<span class="status {watched.status.toLowerCase()}">
					<i><Icon i={watchedStatuses[watched.status]} wh={15} /></i>
					<span>{statusLabel}</span>
				</span>
				<span class:unrated={!watched.rating} class="rating">{ratingLabel}</span
				>
			</div>
		{/if}
	</div>
{/if}

<style lang="scss">
	.public-details {
		position: absolute;
		bottom: 5px;
		left: 50%;
		z-index: 4;
		transform: translateX(-50%);
		display: flex;
		flex-flow: column;
		width: 160px;
		overflow: hidden;
		border-radius: 10px;
		color: white;
		background-color: rgba(28, 28, 28, 0.88);
		font-size: 14px;
		font-weight: bold;
		pointer-events: none;

		.optional {
			display: flex;
			flex-flow: column;
			gap: 5px;
			padding: 8px 7px 6px;

			& > span {
				display: flex;
				align-items: center;
				gap: 8px;
				height: 15px;
			}

			i {
				display: flex;
				width: 15px;
				fill: white;
			}
		}

		.status-rating {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 7px;
			min-height: 31px;
			padding: 7px 8px;
			background-color: rgba(0, 0, 0, 0.34);
		}

		.optional + .status-rating {
			border-top: 1px solid rgba(255, 255, 255, 0.2);
		}

		.status {
			display: flex;
			align-items: center;
			gap: 5px;
			min-width: 0;

			i {
				display: flex;
				flex: 0 0 15px;
				fill: currentColor;
			}

			span {
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}

			&.planned {
				color: #8dc8ff;
			}

			&.watching {
				color: #65e3ff;
			}

			&.finished {
				color: #7ee29a;
			}

			&.hold {
				color: #ffbc70;
			}

			&.dropped {
				color: #ff8b8f;
			}
		}

		.rating {
			flex: 0 0 auto;
			color: gold;
			font-variant-numeric: tabular-nums;

			&.unrated {
				color: rgba(255, 255, 255, 0.72);
				font-weight: normal;
			}
		}

		&.active {
			background-color: rgba(20, 20, 20, 0.94);

			.status-rating {
				border-top: 0;
			}
		}
	}
</style>
