<script lang="ts">
	import { resolve } from "$app/paths";
	import type { StatsPerson } from "./types";
	import { toRatingLabel, type RatingSettings } from "@/lib/rating/helpers";
	let {
		title,
		people,
		mode,
		owner,
		settings,
		studios = false,
	}: {
		title: string;
		people: StatsPerson[];
		mode: "most" | "rating";
		owner?: { id: string; username: string };
		settings?: RatingSettings;
		studios?: boolean;
	} = $props();
	let count = $state(6);
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
	function href(p: StatsPerson) {
		return owner
			? resolve("/(public)/lists/[id]/[username]/person/[personId]", {
					...owner,
					personId: String(p.id),
				})
			: resolve("/(app)/person/[id]", { id: String(p.id) });
	}
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
				{#if studios}<div>{@render portrait()}</div>{:else}<a href={href(p)}
						>{@render portrait()}</a
					>{/if}
				<span
					>{p.titles} titles · {toRatingLabel(p.averageRating, settings)}</span
				>
			</div>
		{:else}<p class="empty">
				No contributors with two watched titles yet.
			</p>{/each}
	</div>
	{#if count < sorted.length}<button
			class="plain more"
			onclick={() => (count += 6)}
			>Show more <span>({sorted.length - count} remaining)</span></button
		>{/if}
</div>

<style>
	h3 {
		font-size: 15px;
		margin-bottom: 18px;
	}
	.people {
		display: grid;
		grid-template-columns: repeat(6, minmax(0, 1fr));
		gap: 18px 12px;
	}
	.person {
		min-width: 0;
		text-align: center;
	}
	a {
		color: inherit;
		text-decoration: none;
	}
	.portrait {
		width: min(86px, 100%);
		aspect-ratio: 1;
		border-radius: 50%;
		overflow: hidden;
		margin: 0 auto 9px;
		background: #6495ed20;
		display: grid;
		place-items: center;
		font-size: 28px;
		color: #6495ed;
	}
	.portrait img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.portrait.studio {
		border-radius: 8px;
	}
	.studio img {
		object-fit: contain;
		padding: 8px;
	}
	strong {
		font-size: 12px;
		display: block;
		overflow-wrap: anywhere;
		line-height: 1.4;
	}
	.person > span {
		display: block;
		font-size: 11px;
		opacity: 0.65;
		margin-top: 4px;
	}
	.more {
		margin-top: 20px;
		font-size: 12px;
		color: var(--stats-accent, #39cfa2);
	}
	.more span {
		opacity: 0.7;
	}
	.empty {
		grid-column: 1/-1;
		opacity: 0.6;
		font-size: 13px;
	}
	@media (max-width: 600px) {
		.people {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
	}
</style>
