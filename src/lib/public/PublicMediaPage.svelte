<script lang="ts">
	import { resolve } from "$app/paths";
	import Activity from "@/lib/Activity.svelte";
	import Error from "@/lib/Error.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import ExpandableText from "@/lib/content/ExpandableText.svelte";
	import Genres from "@/lib/content/Genres.svelte";
	import PageBackdrop from "@/lib/generic/PageBackdrop.svelte";
	import PosterImage from "@/lib/content/PosterImage.svelte";
	import ProvidersList from "@/lib/content/ProvidersList.svelte";
	import SimilarContent from "@/lib/content/SimilarContent.svelte";
	import Title from "@/lib/content/Title.svelte";
	import TopCrewList from "@/lib/content/TopCrewList.svelte";
	import ViewTrailerButton from "@/lib/content/ViewTrailerButton.svelte";
	import PersonPoster from "@/lib/poster/PersonPoster.svelte";
	import { MediaStatusShow } from "@/lib/types/mediaStatus";
	import { noAuthReq } from "@/lib/util/api";
	import { getTopCrew } from "@/lib/util/helpers";
	import type {
		PublicMediaDetails,
		PublicUser,
		SupportedMedia,
		TMDBContentCredits,
		TMDBContentCreditsCrew,
	} from "@/types";
	import PublicReview from "./PublicReview.svelte";

	interface Props {
		ownerId: string;
		ownerName: string;
		mediaId: string;
		mediaType: string;
	}

	let { ownerId, ownerName, mediaId, mediaType }: Props = $props();
	let details: PublicMediaDetails | undefined = $state();
	let owner: PublicUser | undefined = $state();
	let pageError: unknown | undefined = $state();

	let supportedType = $derived(
		(["movie", "tv", "game"] as string[]).includes(mediaType)
			? (mediaType as SupportedMedia)
			: undefined,
	);
	let media = $derived(details?.media);
	let ratingSettings = $derived({
		ratingSystem: owner?.ratingSystem,
		ratingStep: owner?.ratingStep,
	});
	let publicListOwner = $derived({ id: ownerId, username: ownerName });
	let similarOnList = $derived(
		media?.similar?.filter((item) => Boolean(item.watched)) ?? [],
	);
	let posterSrc = $derived.by(() => {
		if (!media?.extPosterPath || !supportedType) return;
		return supportedType === "game"
			? `https://images.igdb.com/igdb/image/upload/t_cover_big/${media.extPosterPath}.jpg`
			: `https://image.tmdb.org/t/p/w500${media.extPosterPath}`;
	});
	let backdropSrc = $derived.by(() => {
		if (!media || !supportedType) return;
		if (supportedType === "game") {
			const path = media.extBackdropPath || media.extPosterPath;
			return path
				? `https://images.igdb.com/igdb/image/upload/t_1080p/${path}.jpg`
				: undefined;
		}
		return media.extBackdropPath
			? `https://www.themoviedb.org/t/p/w1920_and_h800_multi_faces${media.extBackdropPath}`
			: undefined;
	});

	$effect(() => {
		(async () => {
			details = undefined;
			pageError = undefined;
			owner = undefined;
			if (!ownerId || !ownerName || !mediaId || !supportedType) {
				pageError = new globalThis.Error("Unsupported public media page");
				return;
			}
			try {
				[details, owner] = await Promise.all([
					noAuthReq.get<PublicMediaDetails>(
						`/public/users/${ownerId}/${ownerName}/content/${supportedType}/${mediaId}`,
					),
					noAuthReq.get<PublicUser>(`/public/users/${ownerId}/${ownerName}`),
				]);
			} catch (err) {
				pageError = err;
			}
		})();
	});

	async function getCredits() {
		if (supportedType !== "movie" && supportedType !== "tv") return;
		const credits = await noAuthReq.get<
			TMDBContentCredits & { topCrew: TMDBContentCreditsCrew[] }
		>(
			`/public/users/${ownerId}/${ownerName}/content/${supportedType}/${mediaId}/credits`,
		);
		if (credits.crew?.length > 0) {
			credits.topCrew = getTopCrew(credits.crew);
		}
		return credits;
	}
</script>

<svelte:head>
	<title>{media?.name ? `${media.name} - ` : ""}{ownerName}'s List</title>
</svelte:head>

{#if pageError}
	<Error pretty="Failed to load this public list item!" error={pageError} />
{:else if !media || !supportedType}
	<Spinner />
{:else}
	{#if backdropSrc}
		<PageBackdrop src={backdropSrc} />
	{/if}
	<div>
		<div class="content">
			<div class="details-wrap">
				<div class="details-container">
					{#if posterSrc}
						<PosterImage src={posterSrc} />
					{/if}

					<div class="details">
						<Title
							title={media.name}
							homepage={media.homepage}
							releaseDate={media.releaseDate
								? new Date(media.releaseDate)
								: undefined}
							endDate={(media.status === MediaStatusShow.Ended ||
								media.status === MediaStatusShow.Canceled) &&
							media.releaseDateLast
								? new Date(media.releaseDateLast)
								: undefined}
							voteAverage={media.rating}
							voteCount={media.ratingCount}
						/>

						<span class="quick-info">
							{#if supportedType === "movie" && media.runtime}
								<span>{media.runtime} min</span>
							{/if}
							<Genres genres={media.genres} />
							{#if supportedType === "game"}
								<Genres genres={media.gameModes} />
							{/if}
						</span>

						<ExpandableText text={media.summary} style="margin-bottom: 18px;" />

						<div class="btns">
							<ViewTrailerButton videos={media.videos} />
							<a
								class="btn back-to-list"
								href={resolve(`/lists/${ownerId}/${ownerName}`)}
							>
								Back to {ownerName}'s list
							</a>
						</div>

						{#if media.providers}
							<ProvidersList
								providers={media.providers}
								fullListLink={media.providersFullListLink}
								fullListLinkText={supportedType === "game"
									? undefined
									: "JustWatch"}
							/>
						{/if}
					</div>
				</div>
			</div>

			{#if media.watched}
				<PublicReview
					watched={media.watched}
					{ownerName}
					mediaType={supportedType}
					thoughtsPublic={details?.thoughtsPublic ?? false}
					{ratingSettings}
				/>
			{/if}
		</div>

		<div class="page">
			{#if supportedType === "movie" || supportedType === "tv"}
				{#await getCredits()}
					<Spinner />
				{:then credits}
					{#if credits?.topCrew && credits.topCrew.length > 0}
						<TopCrewList topCrew={credits.topCrew} disableInteraction={true} />
					{/if}

					{#if credits?.cast && credits.cast.length > 0}
						<HorizontalList title="Cast">
							{#each credits.cast.slice(0, 50) as cast (cast.credit_id)}
								<PersonPoster
									id={cast.id}
									name={cast.name}
									path={cast.profile_path}
									role={cast.character}
									zoomOnHover={false}
									disableInteraction={true}
								/>
							{/each}
						</HorizontalList>
					{/if}
				{:catch err}
					<Error error={err} pretty="Failed to load cast!" />
				{/await}
			{/if}

			{#if similarOnList.length > 0}
				<SimilarContent
					similar={similarOnList}
					{publicListOwner}
					{ratingSettings}
				/>
			{/if}

			{#if media.watched}
				<Activity
					activity={media.watched.activity}
					readOnly={true}
					title={`${ownerName}'s activity`}
					emptyText={`${ownerName} has no activity for this item.`}
				/>
			{/if}
		</div>
	</div>
{/if}

<style lang="scss">
	@use "../content/page.scss";

	.content {
		position: relative;
		color: white;

		.details-container .details {
			.quick-info {
				display: flex;
				flex-wrap: wrap;
				gap: 10px;
				margin-bottom: 8px;
			}

			.btns {
				display: flex;
				flex-flow: row;
				flex-wrap: wrap;
				gap: 8px;
				margin-top: auto;
			}
		}
	}

	.back-to-list {
		width: max-content;
		font-size: 14px;
	}

	.page {
		display: flex;
		flex-flow: column;
		align-items: center;
		margin-left: auto;
		margin-right: auto;
		gap: 30px;
		padding: 20px 50px;
		max-width: 1200px;

		@media screen and (max-width: 500px) {
			padding: 20px;
		}
	}
</style>
