<script lang="ts">
	import type { RatingSettings } from "@/lib/rating/helpers";
	import { toRatingLabel } from "@/lib/rating/helpers";
	import { watchedStatuses } from "@/lib/util/helpers";
	import { toPublicStatusLabel } from "./helpers";
	import type { SupportedMedia, Watched } from "@/types";
	import Icon from "../Icon.svelte";

	interface Props {
		watched: Watched;
		ownerName: string;
		mediaType: SupportedMedia;
		thoughtsPublic: boolean;
		ratingSettings?: RatingSettings;
	}

	let { watched, ownerName, mediaType, thoughtsPublic, ratingSettings }: Props =
		$props();
	let expanded = $state(false);

	let statusLabel = $derived(toPublicStatusLabel(watched.status, mediaType));
	let thoughts = $derived(
		thoughtsPublic
			? watched.thoughts || "No review thoughts yet."
			: "Review thoughts are private.",
	);
	let hasMore = $derived(thoughtsPublic && watched.thoughts.length > 600);
	let visibleThoughts = $derived(
		hasMore && !expanded
			? `${watched.thoughts.slice(0, 600).trimEnd()}…`
			: thoughts,
	);
</script>

<div class="review">
	<h2>{ownerName}'s review</h2>
	<div class="summary">
		<span class="status {watched.status.toLowerCase()}">
			<i><Icon i={watchedStatuses[watched.status]} wh={18} /></i>
			{statusLabel}
		</span>
		<span class:unrated={!watched.rating} class="rating">
			{toRatingLabel(watched.rating, ratingSettings)}
		</span>
	</div>
	<div class:private={!thoughtsPublic} class="thoughts">
		<i><Icon i={thoughtsPublic ? "document" : "lock-closed"} wh={22} /></i>
		<p>{visibleThoughts}</p>
		{#if hasMore}
			<button class="plain read-more" onclick={() => (expanded = !expanded)}>
				{expanded ? "Show less" : "Read more"}
			</button>
		{/if}
	</div>
	{#if typeof watched.plays === "number" && watched.plays > 0}
		<div class="plays">
			{watched.plays}
			{watched.plays > 1 ? "Plays" : "Play"}
		</div>
	{/if}
</div>

<style lang="scss">
	.review {
		display: flex;
		flex-flow: column;
		gap: 10px;
		width: calc(100% - 160px);
		max-width: 940px;
		color: $text-color;
		margin: 22px auto 0;

		@media screen and (max-width: 900px) {
			width: calc(100% - 80px);
		}

		@media screen and (max-width: 720px) {
			width: calc(100% - 40px);
		}
	}

	h2 {
		color: rgba(255, 255, 255, 0.95);
		font-size: 20px;
		font-weight: normal;
		text-align: center;
		text-shadow: 0 1px 5px rgba(0, 0, 0, 0.95);
	}

	.summary {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		min-height: 48px;
		padding: 10px 14px;
		border-radius: 8px;
		color: white;
		background-color: rgba(20, 20, 20, 0.82);
		font-weight: bold;

		.status {
			display: flex;
			align-items: center;
			gap: 7px;

			i {
				display: flex;
				fill: currentColor;
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
			color: gold;
			font-size: 18px;
			font-variant-numeric: tabular-nums;

			&.unrated {
				color: rgba(255, 255, 255, 0.72);
				font-size: 14px;
				font-weight: normal;
			}
		}
	}

	.thoughts {
		position: relative;
		width: 100%;
		padding: 14px 44px 14px 14px;
		border: 2px solid $text-color;
		border-radius: 8px;
		background-color: $bg-color;

		&.private {
			opacity: 0.65;
		}

		& > i {
			display: flex;
			position: absolute;
			right: 12px;
			top: 13px;
			fill: $text-color;
		}

		p {
			white-space: pre-wrap;
			overflow-wrap: anywhere;
		}

		.read-more {
			display: block;
			width: max-content;
			margin: 10px 0 0 auto;
			padding: 3px 0;
			color: $text-color-accent;
			font-size: 13px;
			text-decoration: underline;
			text-underline-offset: 3px;

			&:hover,
			&:focus-visible {
				color: $text-color;
			}
		}
	}

	.plays {
		text-align: center;
	}
</style>
