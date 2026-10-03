<script lang="ts">
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
	}: {
		title: string;
		people: StatsPerson[];
		mode: "most" | "rating";
		onSelect: (person: StatsPerson) => void;
		settings?: RatingSettings;
		studios?: boolean;
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
					aria-label={`Explore ${p.name}: ${p.titles} watched titles`}
					>{@render portrait()}</button
				>
				<span
					>{p.titles} titles · {averageRating(p.averageRating, settings)}</span
				>
			</div>
		{:else}<p class="empty">
				No contributors with two watched titles yet.
			</p>{/each}
	</div>
	{#if count < sorted.length}<button
			class="plain more"
			onclick={() => (count += 5)}
			>Show more <span>({sorted.length - count} remaining)</span></button
		>{/if}
</div>

<style>
	h3 {
		font-size: 18px;
		margin-bottom: 18px;
	}
	.people {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 18px 12px;
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
		width: min(108px, 100%);
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
		color: #6495ed;
	}
	.portrait img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		object-position: 50% 30%;
	}
	.portrait.studio {
		border-radius: 8px;
	}
	.studio img {
		object-fit: contain;
		padding: 8px;
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
	.more {
		margin-top: 20px;
		font-size: 14px;
		color: var(--stats-accent, #29acf4);
	}
	.more span {
		opacity: 0.7;
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
	.more:hover {
		text-decoration: underline;
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
