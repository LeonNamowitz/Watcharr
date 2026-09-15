<script lang="ts">
	import type { Watched, WatchedStatus } from "@/types";
	import Rating from "../rating/Rating.svelte";
	import Status from "../Status.svelte";
	import MyThoughts from "./MyThoughts.svelte";
	import Playtime from "./Playtime.svelte";
	import { formatLastSeen, getLastSeen } from "../watched/lastSeen";

	interface Props {
		watched?: Watched;
		contentTitle?: string;
		onRatingChanged: (newRating: number) => Promise<boolean>;
		onStatusChanged: (newStatus: WatchedStatus) => Promise<boolean>;
		onThoughtsChanged: (newThoughts: string) => Promise<boolean>;
		onPlaytimeChanged?: (newHours?: number) => Promise<boolean>;
	}

	let {
		watched,
		contentTitle,
		onRatingChanged,
		onStatusChanged,
		onThoughtsChanged,
		onPlaytimeChanged,
	}: Props = $props();

	let lastSeen = $derived(formatLastSeen(getLastSeen(watched)));
</script>

<div class="review">
	<Rating rating={watched?.rating} onChange={onRatingChanged} />
	<Status status={watched?.status} onChange={onStatusChanged} />
	{#if watched}
		{#if onPlaytimeChanged}
			<Playtime hours={watched.playtimeHours} onChange={onPlaytimeChanged} />
		{/if}
		<MyThoughts
			{contentTitle}
			thoughts={watched?.thoughts}
			onChange={onThoughtsChanged}
		/>
		{#if (typeof watched.plays == "number" && watched.plays > 0) || lastSeen}
			<div class="watch-summary">
				{#if typeof watched.plays == "number" && watched.plays > 0}
					<span>
						{watched.plays}
						{watched.plays > 1 ? "Plays" : "Play"}
					</span>
				{/if}
				{#if lastSeen}<span class="last-seen">Last: {lastSeen}</span>{/if}
			</div>
		{/if}
	{/if}
</div>

<style lang="scss">
	.review {
		display: flex;
		flex-flow: column;
		gap: 10px;
		width: 100%;
		max-width: 380px;
		color: $text-color;
		margin-left: auto;
		margin-right: auto;
		margin-top: 22px;

		@media screen and (max-width: 420px) {
			max-width: 340px;
		}
	}

	.watch-summary {
		display: flex;
		justify-content: space-between;
		gap: 12px;

		.last-seen {
			margin-left: auto;
			text-align: right;
		}
	}
</style>
