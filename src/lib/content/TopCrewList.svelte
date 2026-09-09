<script lang="ts">
	import { resolve } from "$app/paths";
	import {
		withPublicListNavigation,
		type PublicListNavigation,
	} from "@/lib/util/listNavigation.svelte";
	import type { TMDBContentCreditsCrew } from "@/types";

	interface Props {
		topCrew: TMDBContentCreditsCrew[];
		disableInteraction?: boolean;
		publicListOwner?: PublicListNavigation;
	}

	let {
		topCrew,
		disableInteraction = false,
		publicListOwner,
	}: Props = $props();

	function personLink(personId: number) {
		if (publicListOwner) {
			return withPublicListNavigation(
				resolve("/(public)/lists/[id]/[username]/person/[personId]", {
					id: String(publicListOwner.id),
					username: publicListOwner.username,
					personId: String(personId),
				}),
				publicListOwner,
			);
		}
		return resolve("/(app)/person/[id]", { id: String(personId) });
	}
</script>

<div class="creators">
	{#each topCrew as crew (crew.credit_id)}
		<div>
			{#if disableInteraction}
				<strong>{crew.name}</strong>
			{:else}
				<a href={personLink(crew.id)}>{crew.name}</a>
			{/if}
			<span>{crew.job}</span>
		</div>
	{/each}
</div>

<style lang="scss">
	.creators {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 35px;
		margin: 10px 60px;

		div {
			display: flex;
			flex-flow: column;
			min-width: 150px;

			a {
				font-weight: bold;
			}
		}
	}
</style>
