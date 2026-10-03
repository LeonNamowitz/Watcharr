<script lang="ts">
	import { resolve } from "$app/paths";
	import PosterImage from "@/lib/content/PosterImage.svelte";
	import { baseURL } from "@/lib/util/api";
	import { toRatingLabel, type RatingSettings } from "@/lib/rating/helpers";
	import { decimal } from "./format";
	import type { StatsMediaCard } from "./types";
	let {
		items,
		owner,
		settings,
		tiny = false,
		wall = false,
		fivePerRow = false,
		comparison = false,
		detail,
	}: {
		items: StatsMediaCard[];
		owner?: { id: string; username: string };
		settings?: RatingSettings;
		tiny?: boolean;
		wall?: boolean;
		fivePerRow?: boolean;
		comparison?: boolean;
		detail?: (card: StatsMediaCard) => string;
	} = $props();
	function href(c: StatsMediaCard) {
		return owner
			? resolve("/(public)/lists/[id]/[username]/[type]/[mediaId]", {
					...owner,
					type: c.type,
					mediaId: String(c.id),
				})
			: c.type === "movie"
				? resolve("/(app)/movie/[id]", { id: String(c.id) })
				: resolve("/(app)/tv/[id]", { id: String(c.id) });
	}
</script>

<div
	class="posters"
	class:tiny
	class:wall
	class:five-per-row={fivePerRow}
	class:episodes={items.some((item) => item.episodeNumber !== undefined)}
	class:mixed={items.some((item) => item.episodeNumber !== undefined) &&
		items.some((item) => item.episodeNumber === undefined)}
>
	{#each items as c (`${c.type}:${c.id}:${c.seasonNumber ?? ""}:${c.episodeNumber ?? ""}`)}
		{@const imagePath =
			c.episodeNumber !== undefined ? c.stillPath : c.posterPath}
		<a
			class:episode={c.episodeNumber !== undefined}
			href={href(c)}
			title={`${c.title}${c.releaseYear ? ` (${c.releaseYear})` : ""}${tiny ? ` · ${toRatingLabel(c.rating, settings)}` : ""}`}
		>
			<div class="image" class:episode={c.episodeNumber !== undefined}>
				{#if imagePath}<PosterImage
						fluid
						src={c.episodeNumber !== undefined
							? `https://www.themoviedb.org/t/p/w227_and_h127_bestv2${imagePath}`
							: `${baseURL}/img${imagePath}`}
						alt={tiny ? c.title : ""}
						loading="lazy"
					/>{:else}<span>{c.title}</span>{/if}
			</div>
			{#if !tiny}<strong>{c.title}</strong>{#if comparison}<span
						class="comparison"
						><b>{decimal(c.rating ?? 0)}</b><span>vs</span><b
							>{decimal(c.tmdbRating ?? 0)}</b
						></span
					>{:else}<span class="meta"
						>{detail ? detail(c) : toRatingLabel(c.rating, settings)}</span
					>{/if}{/if}
		</a>
	{:else}<p class="empty">No titles to show yet.</p>{/each}
</div>

<style>
	.posters {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 16px 12px;
		min-width: 0;
	}
	.posters.episodes {
		grid-template-columns: repeat(3, minmax(0, 1fr));
	}
	.posters.mixed {
		grid-auto-flow: row dense;
		align-items: start;
	}
	.posters.mixed a:not(.episode) {
		grid-row: span 2;
	}
	.image.episode {
		aspect-ratio: 16 / 9;
	}
	.posters a {
		display: flex;
		flex-direction: column;
		min-width: 0;
		color: inherit;
		text-decoration: none;
		gap: 5px;
	}
	.image {
		aspect-ratio: 2/3;
		background: #8882;
		border-radius: 6px;
		overflow: hidden;
		border: 1px solid #8882;
		display: grid;
		place-items: center;
	}
	.image :global(img) {
		width: 100%;
		height: 100%;
		object-fit: cover;
		transition: filter 0.15s;
	}
	.image span {
		font-size: 14px;
		padding: 8px;
		text-align: center;
	}
	a:hover .image,
	a:focus-visible .image {
		border-color: var(--stats-accent);
	}
	a:hover strong,
	a:focus-visible strong {
		color: var(--stats-accent);
	}
	a:hover :global(img) {
		filter: brightness(1.12);
	}
	a:focus-visible {
		outline: 2px solid #29acf4;
		outline-offset: 4px;
		border-radius: 6px;
	}
	strong {
		font-size: 14px;
		line-height: 1.4;
		overflow-wrap: anywhere;
	}
	.meta {
		font-size: 14px;
		color: var(--stats-accent, #29acf4);
		overflow-wrap: anywhere;
	}
	.tiny {
		grid-template-columns: repeat(auto-fill, minmax(42px, 1fr));
		gap: 5px;
	}
	.tiny .image {
		border-radius: 3px;
	}
	.five-per-row {
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 8px;
	}
	.wall {
		grid-template-columns: repeat(auto-fill, minmax(min(70px, 100%), 72px));
		gap: 6px;
	}
	.empty {
		font-size: 13px;
		opacity: 0.6;
		grid-column: 1/-1;
	}
	.comparison {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 16px;
	}
	.comparison b {
		font-size: 17px;
	}
	.comparison b:first-child {
		color: var(--stats-accent);
	}
	.comparison b:last-child {
		color: var(--stats-muted);
	}
	.comparison > span {
		font-size: 12px;
		color: var(--stats-muted);
	}
	@media (min-width: 521px) and (max-width: 900px) {
		.posters:not(.tiny) {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.posters.episodes {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 520px) {
		.posters:not(.tiny) {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 12px 8px;
		}
		.posters.episodes {
			grid-template-columns: minmax(0, 1fr);
		}
		.posters.mixed a:not(.episode) {
			grid-row: auto;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.image :global(img) {
			transition: none;
		}
	}
</style>
