<script lang="ts">
	import type { RatingSettings } from "@/lib/rating/helpers";
	import { toRatingLabel } from "@/lib/rating/helpers";
	import { watchedStatuses } from "@/lib/util/helpers";
	import { toPublicLastSeenLabel, toPublicStatusLabel } from "./helpers";
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
	let reviewRevealed = $state(false);

	let statusLabel = $derived(toPublicStatusLabel(watched.status, mediaType));
	let lastSeenLabel = $derived(
		mediaType === "tv" &&
			["PLANNED", "WATCHING", "HOLD", "DROPPED"].includes(watched.status)
			? toPublicLastSeenLabel(watched.watchingSeason)
			: undefined,
	);
	let reviewThoughts = $derived(watched.thoughts ?? "");
	let reviewIsSpoiler = $derived(
		thoughtsPublic && reviewThoughts.trim().length > 0,
	);
	let thoughts = $derived(
		thoughtsPublic
			? reviewThoughts || "No review thoughts yet."
			: "Review thoughts are private.",
	);
	let hasMore = $derived(thoughtsPublic && reviewThoughts.length > 800);
	let visibleThoughts = $derived(
		hasMore && !expanded
			? `${reviewThoughts.slice(0, 600).trimEnd()}…`
			: thoughts,
	);
</script>

<div class="review">
	<div class="review-card">
		<h2>{ownerName}'s review</h2>
		<div class="summary">
			<span class="status {watched.status.toLowerCase()}">
				<i><Icon i={watchedStatuses[watched.status]} wh={18} /></i>
				<span>{statusLabel}</span>
				{#if lastSeenLabel}
					<span class="last-seen">— Last seen: {lastSeenLabel}</span>
				{/if}
			</span>
			<span class:unrated={!watched.rating} class="rating">
				{toRatingLabel(watched.rating, ratingSettings)}
			</span>
		</div>
		<div
			class:private={!thoughtsPublic}
			class:spoiler-hidden={reviewIsSpoiler && !reviewRevealed}
			class="thoughts"
		>
			<i><Icon i={thoughtsPublic ? "document" : "lock-closed"} wh={22} /></i>
			<p>{visibleThoughts}</p>
			{#if hasMore && (!reviewIsSpoiler || reviewRevealed)}
				<button class="plain read-more" onclick={() => (expanded = !expanded)}>
					{expanded ? "Show less" : "Read more"}
				</button>
			{/if}
			{#if reviewIsSpoiler && !reviewRevealed}
				<button
					class="plain review-spoiler"
					onclick={() => (reviewRevealed = true)}
				>
					<Icon i="eye-closed" wh={34} />
					<span>Show review - potentially very spoilery!!</span>
				</button>
			{/if}
		</div>
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

	.review-card {
		padding: 15px;
		border: 2px solid rgba(255, 215, 0, 0.55);
		border-radius: 12px;
		background: linear-gradient(
			135deg,
			rgba(31, 31, 31, 0.96),
			rgba(14, 14, 14, 0.9)
		);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.38);
	}

	h2 {
		margin-bottom: 10px;
		color: rgba(255, 255, 255, 0.95);
		font-size: 22px;
		font-weight: 600;
		text-shadow: 0 1px 5px rgba(0, 0, 0, 0.95);
	}

	.summary {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		min-height: 52px;
		padding: 10px 14px;
		border: 1px solid rgba(255, 255, 255, 0.16);
		border-radius: 8px;
		color: white;
		background-color: rgba(255, 255, 255, 0.07);
		font-weight: bold;

		.status {
			display: flex;
			align-items: center;
			flex-wrap: wrap;
			gap: 7px;

			.last-seen {
				color: rgba(255, 255, 255, 0.82);
				font-size: 13px;
				font-weight: normal;
			}

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

	.review-spoiler {
		display: flex;
		position: absolute;
		inset: 0;
		align-items: center;
		justify-content: center;
		flex-flow: column;
		gap: 8px;
		width: 100%;
		height: 100%;
		padding: 12px 14px;
		border: 0;
		border-radius: 8px;
		background-color: transparent;
		color: white;
		fill: white;
		font-size: 20px;
		font-weight: bolder;
		transition:
			background-color 150ms ease,
			transform 150ms ease;
		cursor: pointer;

		span {
			text-shadow: 0 0 6px $bg-color;
		}

		:global(svg) {
			filter: drop-shadow(0 0 8px $bg-color);
		}

		&:hover,
		&:focus-visible {
			background-color: transparent;
			transform: translateY(-1px);
		}
	}

	.thoughts {
		position: relative;
		width: 100%;
		margin-top: 14px;
		padding: 15px 30px 0 0;
		border-top: 1px solid rgba(255, 255, 255, 0.24);

		&.private {
			opacity: 0.65;
		}

		&.spoiler-hidden {
			p {
				filter: blur(4px);
				user-select: none;
			}
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
