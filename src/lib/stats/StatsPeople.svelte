<script lang="ts">
	import StatsExpansion from "./StatsExpansion.svelte";
	import type { StatsPerson } from "./types";
	import { type RatingSettings } from "@/lib/rating/helpers";
	import { averageRating } from "./format";
	let {
		title,
		people,
		mode,
		onSelect,
		settings,
		studios = false,
		unit = "titles",
	}: {
		title: string;
		people: StatsPerson[];
		mode: "most" | "rating";
		onSelect: (person: StatsPerson) => void;
		settings?: RatingSettings;
		studios?: boolean;
		unit?: string;
	} = $props();
	let count = $state(5);
	const sorted = $derived(
		[...people]
			.filter(
				(p) => p.titles >= 2 && (mode !== "rating" || p.averageRating > 0),
			)
			.sort((a, b) =>
				mode === "rating"
					? b.averageRating - a.averageRating ||
						b.titles - a.titles ||
						a.name.localeCompare(b.name)
					: b.titles - a.titles || a.name.localeCompare(b.name),
			),
	);
</script>

<div class="people-block">
	<h3 class="norm">{title}</h3>
	<div class="people">
		{#each sorted.slice(0, count) as p (p.id)}
			<div class="person">
				{#snippet portrait()}<div class="portrait" class:studio={studios}>
						{#if p.profilePath}<img
								src={`https://image.tmdb.org/t/p/w185${p.profilePath}`}
								alt=""
								loading="lazy"
							/>{:else}<span>{p.name.slice(0, 1)}</span>{/if}
					</div>
					<strong>{p.name}</strong>{/snippet}
				<button
					class="plain person-button"
					onclick={() => onSelect(p)}
					aria-label={`Explore ${p.name}: ${p.titles} watched ${unit}`}
					>{@render portrait()}</button
				>
				<span
					>{p.titles} {unit} · {averageRating(p.averageRating, settings)}</span
				>
			</div>
		{:else}<p class="empty">
				No contributors with two watched {unit} yet.
			</p>{/each}
	</div>
	<StatsExpansion
		{count}
		total={sorted.length}
		onChange={(value) => (count = value)}
	/>
</div>

<style>
	h3 {
		font-size: 18px;
		margin-bottom: 18px;
	}
	.people {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 14px 8px;
	}
	.person {
		min-width: 0;
		text-align: center;
	}
	.person-button {
		width: 100%;
		text-align: center;
		color: inherit;
		text-decoration: none;
	}
	.portrait {
		position: relative;
		width: min(160px, 100%);
		aspect-ratio: 1;
		border-radius: 50%;
		overflow: hidden;
		margin: 0 auto 9px;
		border: 2px solid var(--stats-border);
		box-sizing: border-box;
		background: #29acf414;
		display: grid;
		place-items: center;
		font-size: 28px;
		color: #51ad79;
	}
	.portrait img {
		position: absolute;
		inset: 0;
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
		/* Vertical crop: 0% keeps the source top and moves the face lower.
           Increase the second percentage to move the photo up. */
		object-position: 50% 30%;
	}
	.portrait.studio {
		border-radius: 8px;
		aspect-ratio: 3 / 2;
		background: #fff;
		border-color: #fff;
	}
	.studio img {
		object-fit: contain;
		padding: 12px;
		filter: none;
		opacity: 1;
		mix-blend-mode: normal;
	}
	strong {
		font-size: 14px;
		display: block;
		overflow-wrap: anywhere;
		line-height: 1.4;
	}
	.person > span {
		display: block;
		font-size: 13px;
		color: var(--stats-muted);
		margin-top: 4px;
	}
	.empty {
		grid-column: 1/-1;
		opacity: 0.6;
		font-size: 13px;
	}
	.person-button:hover .portrait,
	.person-button:focus-visible .portrait {
		border-color: var(--stats-accent);
	}
	.person-button:hover strong,
	.person-button:focus-visible strong {
		color: var(--stats-accent);
	}
	.person-button:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 4px;
		border-radius: 8px;
	}
	@media (min-width: 521px) and (max-width: 900px) {
		.people {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
	}
	@media (max-width: 520px) {
		.people {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
</style>
