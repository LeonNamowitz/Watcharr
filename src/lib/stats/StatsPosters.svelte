<script lang="ts">
	import { resolve } from "$app/paths";
	import PosterImage from "@/lib/content/PosterImage.svelte";
	import { baseURL } from "@/lib/util/api";
	import { toRatingLabel, type RatingSettings } from "@/lib/rating/helpers";
	import type { StatsMediaCard } from "./types";
	let {
		items,
		owner,
		settings,
		tiny = false,
		detail,
	}: {
		items: StatsMediaCard[];
		owner?: { id: string; username: string };
		settings?: RatingSettings;
		tiny?: boolean;
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

<div class="posters" class:tiny>
	{#each items as c (`${c.type}:${c.id}`)}
		<a
			href={href(c)}
			title={`${c.title}${c.releaseYear ? ` (${c.releaseYear})` : ""} · ${toRatingLabel(c.rating, settings)}${detail ? ` · ${detail(c)}` : ""}`}
		>
			<div class="image">
				{#if c.posterPath}<PosterImage
						fluid
						src={`${baseURL}/img${c.posterPath}`}
						alt={tiny ? c.title : ""}
						loading="lazy"
					/>{:else}<span>{c.title}</span>{/if}
			</div>
			{#if !tiny}<strong>{c.title}</strong><span class="meta"
					>{detail ? detail(c) : toRatingLabel(c.rating, settings)}</span
				>{/if}
		</a>
	{:else}<p class="empty">No titles to show yet.</p>{/each}
</div>

<style>
	.posters {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
		gap: 16px 12px;
		min-width: 0;
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
		font-size: 12px;
		padding: 8px;
		text-align: center;
	}
	a:hover :global(img) {
		filter: brightness(1.12);
	}
	a:focus-visible {
		outline: 2px solid #39cfa2;
		outline-offset: 4px;
		border-radius: 6px;
	}
	strong {
		font-size: 12px;
		line-height: 1.4;
		overflow-wrap: anywhere;
	}
	.meta {
		font-size: 11px;
		color: var(--stats-accent, #39cfa2);
		overflow-wrap: anywhere;
	}
	.tiny {
		grid-template-columns: repeat(auto-fill, minmax(42px, 1fr));
		gap: 5px;
	}
	.tiny .image {
		border-radius: 3px;
	}
	.empty {
		font-size: 13px;
		opacity: 0.6;
		grid-column: 1/-1;
	}
	@media (max-width: 520px) {
		.posters:not(.tiny) {
			grid-template-columns: repeat(3, minmax(0, 1fr));
			gap: 12px 8px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.image :global(img) {
			transition: none;
		}
	}
</style>
