<script lang="ts">
	import { resolve } from "$app/paths";
	import Activity from "@/lib/Activity.svelte";
	import Error from "@/lib/Error.svelte";
	import HorizontalList from "@/lib/HorizontalList.svelte";
	import Icon from "@/lib/Icon.svelte";
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
	import SeasonsList from "@/lib/season/SeasonsList.svelte";
	import { MediaStatusShow } from "@/lib/types/mediaStatus";
	import { noAuthReq } from "@/lib/util/api";
	import { getTopCrew } from "@/lib/util/helpers";
	import type {
		PublicMediaDetails,
		PublicUser,
		SupportedMedia,
		TMDBContentCredits,
		TMDBContentCreditsCrew,
		TMDBSeasonDetails,
	} from "@/types";
	import { toPublicLastSeenSeason } from "./helpers";
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
	let lastSeenSeason = $derived(
		toPublicLastSeenSeason(media?.watched?.watchingSeason),
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

	async function getSeasonDetails(seasonNumber: number) {
		return await noAuthReq.get<TMDBSeasonDetails>(
			`/public/users/${ownerId}/${ownerName}/content/tv/${mediaId}/season/${seasonNumber}`,
		);
	}

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
							ratingSource={supportedType === "game" ? "IGDB" : "TMDB"}
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
			{:else}
				<div class="not-on-list">
					<Icon i="film" wh={22} />
					<span>Not on {ownerName}'s list</span>
				</div>
			{/if}
		</div>

		<div class="page">
			{#if supportedType === "movie" || supportedType === "tv"}
				{#await getCredits()}
					<Spinner />
				{:then credits}
					{#if credits?.topCrew && credits.topCrew.length > 0}
						<TopCrewList topCrew={credits.topCrew} {publicListOwner} />
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
									{publicListOwner}
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
			{#if supportedType === "tv" && media.seasons}
				<SeasonsList
					tvId={Number(mediaId)}
					seasons={media.seasons}
					watchedItem={media.watched}
					lastViewedSeason={lastSeenSeason}
					lastViewedSeasonChanged={() => {}}
					readOnly={true}
					{ratingSettings}
					loadSeasonDetails={getSeasonDetails}
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

	.not-on-list {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 9px;
		width: calc(100% - 160px);
		max-width: 940px;
		margin: 22px auto 0;
		padding: 15px;
		border: 2px solid rgba(255, 255, 255, 0.22);
		border-radius: 12px;
		color: rgba(255, 255, 255, 0.9);
		fill: currentColor;
		background: linear-gradient(
			135deg,
			rgba(31, 31, 31, 0.96),
			rgba(14, 14, 14, 0.9)
		);
		font-weight: 600;

		@media screen and (max-width: 900px) {
			width: calc(100% - 80px);
		}

		@media screen and (max-width: 720px) {
			width: calc(100% - 40px);
		}
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
