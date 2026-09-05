<script lang="ts">
	import Error from "@/lib/Error.svelte";
	import Poster from "@/lib/poster/Poster.svelte";
	import PosterList from "@/lib/poster/PosterList.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import DropDown from "@/lib/DropDown.svelte";
	import type {
		DropDownItem,
		Media,
		PersonCreditsResponse,
		PersonDetailsResponse,
		PublicUser,
	} from "@/types";
	import { noAuthReq, req } from "@/lib/util/api.js";
	import Icon from "@/lib/Icon.svelte";
	import PageBackdrop from "@/lib/generic/PageBackdrop.svelte";
	import PosterImage from "@/lib/content/PosterImage.svelte";
	import ExpandableText from "@/lib/content/ExpandableText.svelte";
	import { resolve } from "$app/paths";
	import {
		readPersonPageState,
		savePersonPageState,
		type PersonCreditFilter,
		type PersonSortOption,
	} from "./personPageState";

	interface Props {
		personId: number;
		publicOwner?: { id: string; username: string };
	}

	let { personId, publicOwner }: Props = $props();

	let person: PersonDetailsResponse | undefined = $state();
	let owner: PublicUser | undefined = $state();
	let pageError: unknown | undefined = $state();
	let sortOption: PersonSortOption = $state("voteCount");
	let creditsType = $state("Acting");
	let credits: PersonCreditsResponse | undefined = $state();
	let creditFilter: PersonCreditFilter = $state("all");
	let stateReady = $state(false);

	let isPublic = $derived(Boolean(publicOwner));
	let ownerName = $derived(owner?.username ?? publicOwner?.username ?? "");
	let ratingSortLabel = $derived(
		isPublic ? `${ownerName || "Owner"}'s rating` : "My rating",
	);
	let sortOptions: DropDownItem[] = $derived([
		{ id: "voteCount", value: "Popularity" },
		{ id: "newest", value: "Release date (newest)" },
		{ id: "oldest", value: "Release date (oldest)" },
		{ id: "ownerRating", value: ratingSortLabel },
	]);
	let ratingSettings = $derived({
		ratingSystem: owner?.ratingSystem,
		ratingStep: owner?.ratingStep,
	});

	function getCreditsTypeOptions() {
		const options: string[] = [];
		const hasActing = credits?.hasActing ?? true;
		const hasDirecting = credits?.hasDirecting ?? false;
		if (hasActing) options.push("Acting");
		if (hasDirecting) options.push("Directing");
		if (options.length === 0) options.push("Acting");
		return options;
	}

	$effect(() => {
		if (personId) fetchPersonData();
	});

	$effect(() => {
		if (sortOption && credits) sortCredits(sortOption);
	});

	$effect(() => {
		if (personId && person && stateReady) {
			savePersonPageState(
				personId,
				{
					sortOption,
					creditsType,
					creditFilter,
				},
				publicOwner?.id,
			);
		}
	});

	$effect(() => {
		if (creditsType && personId && person) updatePersonCredits();
	});

	async function fetchPersonData() {
		try {
			stateReady = false;
			person = undefined;
			pageError = undefined;
			credits = undefined;
			if (!personId) return;

			if (publicOwner) {
				[person, owner] = await Promise.all([
					noAuthReq.get<PersonDetailsResponse>(
						`/public/users/${publicOwner.id}/${publicOwner.username}/content/person/${personId}`,
					),
					noAuthReq.get<PublicUser>(
						`/public/users/${publicOwner.id}/${publicOwner.username}`,
					),
				]);
			} else {
				person = await req.get<PersonDetailsResponse>(
					`/content/person/${personId}`,
				);
				owner = undefined;
			}

			const savedState = readPersonPageState(personId, publicOwner?.id);
			const defaultCreditsType =
				person?.knownForDepartment?.toLowerCase() === "directing"
					? "Directing"
					: "Acting";
			sortOption =
				savedState?.sortOption ?? (isPublic ? "ownerRating" : "voteCount");
			creditFilter = savedState?.creditFilter ?? "all";
			creditsType = savedState?.creditsType ?? defaultCreditsType;
			await updatePersonCredits();
			sortCredits(sortOption);
			stateReady = true;
		} catch (err) {
			person = undefined;
			pageError = err;
		}
	}

	async function updatePersonCredits() {
		const endpoint = publicOwner
			? `/public/users/${publicOwner.id}/${publicOwner.username}/content/person/${personId}/credits`
			: `/content/person/${personId}/credits`;
		credits = publicOwner
			? await noAuthReq.get<PersonCreditsResponse>(endpoint, {
					params: { creditsType },
				})
			: await req.get<PersonCreditsResponse>(endpoint, {
					params: { creditsType },
				});
		const options = getCreditsTypeOptions();
		if (!options.includes(creditsType)) {
			creditsType = options[0];
			return;
		}
		credits.credits = credits.credits?.filter(
			(value, index, all) =>
				all.findIndex((item) => item.ids.tmdb === value.ids.tmdb) === index,
		);
	}

	function newestOldestSort(a: Media, b: Media, newestFirst: boolean) {
		if (!a.releaseDate && !b.releaseDate) return 0;
		if (!a.releaseDate) return newestFirst ? -1 : 1;
		if (!b.releaseDate) return newestFirst ? 1 : -1;
		const difference =
			new Date(b.releaseDate).valueOf() - new Date(a.releaseDate).valueOf();
		return newestFirst ? difference : -difference;
	}

	function sortCredits(option: PersonSortOption) {
		if (!credits?.credits) return;
		switch (option) {
			case "voteCount":
				credits.credits.sort(
					(a, b) => (b.ratingCount ?? 0) - (a.ratingCount ?? 0),
				);
				break;
			case "newest":
				credits.credits.sort((a, b) => newestOldestSort(a, b, true));
				break;
			case "oldest":
				credits.credits.sort((a, b) => newestOldestSort(a, b, false));
				break;
			case "ownerRating":
				credits.credits.sort(
					(a, b) => (b.watched?.rating ?? -1) - (a.watched?.rating ?? -1),
				);
				break;
		}
		credits.credits = credits.credits;
	}

	function isWatchedCredit(credit: Media) {
		return (
			credit.watched?.status === "FINISHED" ||
			credit.watched?.status === "WATCHING" ||
			credit.watched?.status === "HOLD"
		);
	}

	function shouldShowCredit(credit: Media) {
		switch (creditFilter) {
			case "watched":
				return isWatchedCredit(credit);
			case "planned":
				return credit.watched?.status === "PLANNED";
			case "missing":
				return !credit.watched;
			default:
				return true;
		}
	}

	let creditCounts = $derived.by(() => {
		const allCredits = credits?.credits ?? [];
		return {
			all: allCredits.length,
			watched: allCredits.filter(isWatchedCredit).length,
			planned: allCredits.filter(
				(credit) => credit.watched?.status === "PLANNED",
			).length,
			missing: allCredits.filter((credit) => !credit.watched).length,
		};
	});
	let visibleCredits = $derived.by(() =>
		(credits?.credits ?? []).filter(shouldShowCredit),
	);
	let backToListHref = $derived.by(() => {
		if (!publicOwner) return;
		return resolve(`/lists/${publicOwner.id}/${publicOwner.username}`);
	});
</script>

<svelte:head>
	<title
		>{person?.name ? `${person.name} - ` : ""}{isPublic
			? `${ownerName}'s List`
			: "Person"}</title
	>
</svelte:head>

<div>
	{#if pageError}
		<Error pretty="Failed to load person!" error={pageError} />
	{:else if !person}
		<Spinner />
	{:else if Object.keys(person).length > 0}
		{#if credits?.credits?.[0]?.extBackdropPath}
			<PageBackdrop
				src={`https://www.themoviedb.org/t/p/w1920_and_h800_multi_faces${credits.credits[0].extBackdropPath}`}
			/>
		{/if}
		<div class="content">
			<div class="details-wrap">
				<div class="details-container">
					{#if person.extPosterPath}
						<PosterImage
							src={`https://image.tmdb.org/t/p/w500${person.extPosterPath}`}
						/>
					{/if}
					<div class="details">
						<span class="title-container">
							<a href={person.homepage} rel="external" target="_blank">
								{person.name}
							</a>
							<span></span>
						</span>
						<ExpandableText title="Biography" text={person.biography} />
						<div class="detail-info">
							{#if person.knownForDepartment}
								<div>
									<span>Department</span><span>{person.knownForDepartment}</span
									>
								</div>
							{/if}
							{#if person.placeOfBirth}
								<div>
									<span>Born In</span><span>{person.placeOfBirth}</span>
								</div>
							{/if}
							{#if person.birthday}
								<div>
									<span>Born On</span>
									<span
										>{new Date(
											Date.parse(person.birthday),
										).toLocaleDateString()}</span
									>
								</div>
							{/if}
							{#if person.deathday}
								<div>
									<span>Died On</span>
									<span
										>{new Date(
											Date.parse(person.deathday),
										).toLocaleDateString()}</span
									>
								</div>
							{/if}
							{#if person.age}
								<div><span>Age</span><span>{person.age} Years</span></div>
							{/if}
						</div>
						{#if backToListHref}
							<div class="btns">
								<a class="btn back-to-list" href={backToListHref}>
									Back to {ownerName}'s list
								</a>
							</div>
						{/if}
					</div>
				</div>
			</div>
		</div>

		{#if credits}
			{#if credits.credits && credits.credits.length > 0}
				<div
					class="filters"
					role="group"
					aria-label="Credit filters and sorting"
				>
					<div class="filter-heading">
						<div>
							<span class="filter-summary">
								Showing {visibleCredits.length} of {creditCounts.all} credits.
							</span>
						</div>
					</div>
					<div class="controls-row">
						<div class="filter-control">
							<span class="control-label">Credit type</span>
							<div class="dropdown-wrap">
								<DropDown
									bind:active={creditsType}
									placeholder="Acting"
									options={getCreditsTypeOptions()}
									isDropDownItem={false}
									disabled={getCreditsTypeOptions().length < 2}
								/>
							</div>
						</div>
						<div class="filter-control sort-control">
							<span class="control-label">Sort by</span>
							<div class="dropdown-wrap">
								<DropDown
									bind:active={sortOption}
									placeholder="Popularity"
									options={sortOptions}
									isDropDownItem={true}
								/>
							</div>
						</div>
					</div>
					<div class="credit-filter" role="group" aria-label="Show credits">
						<button
							class="plain"
							class:active={creditFilter === "all"}
							aria-pressed={creditFilter === "all"}
							onclick={() => (creditFilter = "all")}
						>
							<Icon i="film" wh={17} />
							<span>All credits</span>
							<strong>{creditCounts.all}</strong>
						</button>
						<button
							class="plain"
							class:active={creditFilter === "watched"}
							aria-pressed={creditFilter === "watched"}
							onclick={() => (creditFilter = "watched")}
						>
							<Icon i="check" wh={17} />
							<span>Watched</span>
							<strong>{creditCounts.watched}</strong>
						</button>
						<button
							class="plain"
							class:active={creditFilter === "planned"}
							aria-pressed={creditFilter === "planned"}
							onclick={() => (creditFilter = "planned")}
						>
							<Icon i="calendar" wh={17} />
							<span>Planned</span>
							<strong>{creditCounts.planned}</strong>
						</button>
						<button
							class="plain"
							class:active={creditFilter === "missing"}
							aria-pressed={creditFilter === "missing"}
							onclick={() => (creditFilter = "missing")}
						>
							<Icon i="eye-closed" wh={17} />
							<span>Missing</span>
							<strong>{creditCounts.missing}</strong>
						</button>
					</div>
				</div>
				{#if visibleCredits.length > 0}
					<div class="page">
						<PosterList>
							{#each credits.credits as credit, index (`${index}-${credit.ids.tmdb}`)}
								<Poster
									media={credit}
									bind:watched={credits.credits[index].watched}
									fluidSize
									hidden={!shouldShowCredit(credit)}
									hideButtons={isPublic}
									publicView={isPublic}
									publicListOwner={publicOwner}
									publicRatingSettings={isPublic ? ratingSettings : undefined}
								/>
							{/each}
						</PosterList>
					</div>
				{:else}
					<div class="no-credits-message filtered-empty">
						<Icon i="filter-circle" wh={80} />
						<h2 class="norm">No credits in this view</h2>
						<h4 class="norm">
							Try another view to see more of {person.name}'s credits.
						</h4>
					</div>
				{/if}
			{:else}
				<div class="no-credits-message">
					<Icon i="star" wh={80} />
					<h2 class="norm">We found no credits!</h2>
					<h4 class="norm">It seems that this person has no credits.</h4>
				</div>
			{/if}
		{:else}
			<Spinner />
		{/if}
	{:else}
		<Error error="Person not found" pretty="Person not found" />
	{/if}
</div>

<style lang="scss">
	@use "../content/page.scss";

	.filters {
		display: flex;
		flex-flow: column;
		gap: 12px;
		margin: 4px auto 8px;
		padding: 14px 18px;
		width: calc(100% - 40px);
		max-width: 1200px;
		border: 1px solid rgba($color: $text-color, $alpha: 0.16);
		border-radius: 12px;
		background-color: rgba($color: $bg-color, $alpha: 0.72);
		box-shadow: 0 6px 18px rgba(0, 0, 0, 0.16);

		.filter-heading {
			display: flex;
			align-items: center;
			justify-content: space-between;
		}

		.filter-control {
			display: flex;
			flex-flow: column;
			gap: 4px;
			min-width: 0;
			width: 180px;
		}

		.control-label {
			display: block;
			font-size: 12px;
			color: rgba($color: $text-color, $alpha: 0.72);
		}

		.filter-summary {
			margin-left: 4px;
			font-size: 12px;
			color: rgba($color: $text-color, $alpha: 0.56);
		}

		.credit-filter {
			display: grid;
			grid-template-columns: repeat(4, minmax(0, 1fr));
			gap: 4px;
			padding: 4px;
			border: 1px solid rgba($color: $text-color, $alpha: 0.22);
			border-radius: 9px;
			background-color: rgba(0, 0, 0, 0.18);

			button {
				display: flex;
				align-items: center;
				gap: 7px;
				min-width: 0;
				padding: 8px 10px;
				border: 2px solid $text-color;
				border-radius: 6px;
				color: rgba($color: $text-color, $alpha: 0.76);
				fill: currentColor;
				font-size: 13px;
				text-align: start;
				transition:
					background-color 120ms ease,
					color 120ms ease;

				:global(svg) {
					flex: 0 0 auto;
				}

				span {
					overflow: hidden;
					text-overflow: ellipsis;
					white-space: nowrap;
				}

				strong {
					margin-left: auto;
					font-size: 12px;
					font-weight: normal;
					opacity: 0.7;
				}

				&:hover,
				&:focus-visible,
				&.active {
					color: $bg-color;
					background-color: $text-color;

					strong {
						opacity: 1;
					}
				}
			}
		}

		.controls-row {
			display: flex;
			align-items: flex-end;
			justify-content: space-between;
			gap: 14px;
			padding: 0 4px;

			.dropdown-wrap {
				width: 100%;

				:global(> div) {
					width: 100%;
				}

				:global(> div > button) {
					width: 100%;
					justify-content: space-between;
					text-align: start;
				}
			}
		}

		@media screen and (max-width: 600px) {
			width: calc(100% - 24px);
			padding: 12px;

			.credit-filter {
				grid-template-columns: repeat(2, minmax(0, 1fr));
			}

			.controls-row {
				display: grid;
				grid-template-columns: repeat(2, minmax(0, 1fr));
				gap: 8px;

				.filter-control {
					width: auto;
				}
			}

			.filter-summary {
				display: block;
				margin: 3px 0 0;
			}
		}
	}

	.content {
		position: relative;
		color: white;
		margin-bottom: 15px;

		.details-container .details {
			.title-container {
				a {
					color: white;
					text-decoration: none;
					font-size: 30px;
					font-weight: bold;
					padding-right: 3px;
				}

				span {
					font-size: 20px;
					color: rgba($color: #fff, $alpha: 0.7);
				}
			}

			.detail-info {
				display: flex;
				flex-wrap: wrap;
				gap: 15px 30px;
				margin-top: 10px;
				font-size: 14px;

				div {
					display: flex;
					flex-flow: column;

					span:first-child {
						font-weight: bold;
					}
				}
			}

			.btns {
				display: flex;
				flex-flow: row;
				flex-wrap: wrap;
				gap: 8px;
				margin-top: 18px;
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
		gap: 30px;
		padding: 10px 0;
	}

	.no-credits-message {
		display: flex;
		flex-flow: column;
		gap: 5px;
		align-items: center;
		margin-top: 20px;

		h2 {
			margin-top: 10px;
		}

		h4 {
			font-weight: normal;
		}
	}
</style>
