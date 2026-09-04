<script lang="ts">
	import { dateValid } from "../util/date";

	interface Props {
		homepage?: string;
		title?: string;
		releaseDate?: Date;
		endDate?: Date;
		voteAverage?: number;
		voteCount?: number;
		ratingSource?: string;
	}

	let {
		homepage,
		title,
		releaseDate,
		endDate,
		voteAverage,
		voteCount,
		ratingSource,
	}: Props = $props();

	// if voteAvg bigger than 10, it is out of 100, so no need to * by 10
	const vote = $derived(
		voteAverage
			? Math.round(voteAverage > 10 ? voteAverage : voteAverage * 10) / 10
			: 0,
	);
	const titleSafe = $derived(title ? title : "Unknown Title");
	const releaseYear = $derived(
		dateValid(releaseDate) ? releaseDate.getFullYear() : undefined,
	);
	const endYear = $derived(
		dateValid(endDate) ? endDate.getFullYear() : undefined,
	);
</script>

<span class="title-container">
	<span class="title">
		{#if homepage}
			<a href={homepage} rel="external" target="_blank">{titleSafe}</a>
		{:else}
			<span class="t">{titleSafe}</span>
		{/if}
		{#if releaseYear}
			<span class="year">
				<!--First span ends on the line with the #if so there's no whitespace-->
				<span title={releaseDate?.toLocaleDateString()}>{releaseYear}</span
				>{#if endYear && endYear != releaseYear}
					<span title={endDate?.toLocaleDateString()}>-{endYear}</span>
				{/if}
			</span>
		{/if}
	</span>
	<span
		class:community={Boolean(ratingSource)}
		class="rating"
		title={`${ratingSource ? `${ratingSource} ` : ""}Rating: ${vote} out of 10 (based on ${voteCount ?? 0} votes)`}
	>
		{#if ratingSource}
			<small>{ratingSource}</small>
		{/if}
		<span class="star">*</span>
		{vote}
	</span>
</span>

<style lang="scss">
	.title-container {
		display: flex;
		gap: 10px;

		.title {
			a,
			span.t {
				color: white;
				text-decoration: none;
				font-size: 30px;
				font-weight: bold;
				padding-right: 3px;
			}

			span.year {
				font-size: 20px;
				color: rgba($color: #fff, $alpha: 0.7);
			}
		}

		.rating {
			display: flex;
			align-items: start;
			justify-content: center;
			gap: 5px;
			margin-left: auto;
			color: gold;
			font-size: 22px;
			font-weight: bolder;

			small {
				align-self: center;
				font-size: 10px;
				font-weight: normal;
				letter-spacing: 0.08em;
			}

			span.star {
				margin-top: 7px;
				font-family: "Rampart One";
				-webkit-text-stroke: 1px gold;
				font-size: 40px;
				line-height: 0.7;
			}

			&.community {
				color: white;

				span.star {
					-webkit-text-stroke-color: white;
				}
			}
		}
	}
</style>
