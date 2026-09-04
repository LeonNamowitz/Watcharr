<script lang="ts">
	import type { Media } from "@/types";
	import HorizontalList from "../HorizontalList.svelte";
	import Poster from "../poster/Poster.svelte";
	import type { RatingSettings } from "../rating/helpers";

	interface Props {
		similar: Media[];
		publicListOwner?: { id: string | number; username: string };
		ratingSettings?: RatingSettings;
	}

	let { similar, publicListOwner, ratingSettings }: Props = $props();
</script>

{#if similar?.length > 0}
	<HorizontalList title="Similar">
		{#each similar as content, i (content.ids)}
			<Poster
				media={content}
				small={true}
				bind:watched={similar[i].watched}
				hideButtons={Boolean(publicListOwner)}
				publicView={Boolean(publicListOwner)}
				{publicListOwner}
				publicRatingSettings={ratingSettings}
			/>
		{/each}
	</HorizontalList>
{/if}
