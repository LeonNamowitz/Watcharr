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
	import Checkbox from "@/lib/Checkbox.svelte";
	import Icon from "@/lib/Icon.svelte";
	import PageBackdrop from "@/lib/generic/PageBackdrop.svelte";
	import PosterImage from "@/lib/content/PosterImage.svelte";
	import ExpandableText from "@/lib/content/ExpandableText.svelte";
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
	let onListFilter = $state(false);
	let creditFilter: PersonCreditFilter = $state("all");
	let stateReady = $state(false);

	let isPublic = $derived(Boolean(publicOwner));
	let ownerName = $derived(owner?.username ?? publicOwner?.username ?? "");
	let listFilterLabel = $derived(
		isPublic ? `On ${ownerName}'s list` : "On my list",
	);
	let ratingSortLabel = $derived(
		isPublic ? `${ownerName}'s Rating` : "My Rating",
	);
	let sortOptions: DropDownItem[] = $derived([
		{ id: "voteCount", value: "Vote Count" },
		{ id: "newest", value: "Newest" },
		{ id: "oldest", value: "Oldest" },
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
					onListFilter,
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
			onListFilter = savedState?.onListFilter ?? isPublic;
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

	function shouldHideCredit(credit: Media) {
		if (onListFilter && !credit.watched) return true;
		switch (creditFilter) {
			case "watched":
				return credit.watched?.status !== "FINISHED";
			case "planned":
				return credit.watched?.status !== "PLANNED";
			default:
				return false;
		}
	}
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
					<div class="filter-control">
						<span class="control-label">Credits:</span>
						<DropDown
							bind:active={creditsType}
							placeholder="Acting"
							options={getCreditsTypeOptions()}
							isDropDownItem={false}
							showActiveElementsInOptions={true}
						/>
					</div>
					<div class="credit-filter" role="group" aria-label="List status">
						<button
							class="plain"
							class:active={creditFilter === "watched"}
							aria-pressed={creditFilter === "watched"}
							onclick={() => (creditFilter = "watched")}
						>
							Watched
						</button>
						<button
							class="plain"
							class:active={creditFilter === "all"}
							aria-pressed={creditFilter === "all"}
							onclick={() => (creditFilter = "all")}
						>
							All
						</button>
						<button
							class="plain"
							class:active={creditFilter === "planned"}
							aria-pressed={creditFilter === "planned"}
							onclick={() => (creditFilter = "planned")}
						>
							Planned
						</button>
					</div>
					<div class="list-filter">
						<span>{listFilterLabel}</span>
						<Checkbox name={listFilterLabel} bind:value={onListFilter} />
					</div>
					<div class="filter-control sort-control">
						<span class="control-label">Sort:</span>
						<DropDown
							bind:active={sortOption}
							placeholder="Vote Count"
							options={sortOptions}
							isDropDownItem={true}
							showActiveElementsInOptions={true}
						/>
					</div>
				</div>
				<div class="page">
					<PosterList>
						{#each credits.credits as credit, index (`${index}-${credit.ids.tmdb}`)}
							<Poster
								media={credit}
								bind:watched={credits.credits[index].watched}
								fluidSize
								hidden={shouldHideCredit(credit)}
								hideButtons={isPublic}
								publicView={isPublic}
								publicListOwner={publicOwner}
								publicRatingSettings={isPublic ? ratingSettings : undefined}
							/>
						{/each}
					</PosterList>
				</div>
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
		align-items: end;
		display: grid;
		grid-template-columns: minmax(130px, auto) auto minmax(145px, 1fr) minmax(
				150px,
				auto
			);
		gap: 14px 22px;
		margin: 4px auto 8px;
		padding: 14px 18px;
		width: calc(100% - 40px);
		max-width: 1200px;
		border: 1px solid rgba($color: $text-color, $alpha: 0.16);
		border-radius: 12px;
		background-color: rgba($color: $bg-color, $alpha: 0.72);
		box-shadow: 0 6px 18px rgba(0, 0, 0, 0.16);

		.filter-control {
			display: flex;
			flex-flow: column;
			gap: 4px;
		}

		.control-label {
			font-size: 12px;
			color: rgba($color: $text-color, $alpha: 0.72);
		}

		.list-filter {
			display: flex;
			align-items: center;
			justify-content: center;
			gap: 8px;
			min-height: 38px;
		}

		.sort-control {
			justify-self: end;
		}

		.credit-filter {
			display: grid;
			grid-template-columns: repeat(3, 1fr);
			align-self: end;
			padding: 3px;
			border: 1px solid rgba($color: $text-color, $alpha: 0.22);
			border-radius: 9px;
			background-color: rgba(0, 0, 0, 0.18);

			button {
				padding: 7px 11px;
				border-radius: 6px;
				color: rgba($color: $text-color, $alpha: 0.76);
				font-size: 13px;
				transition:
					background-color 120ms ease,
					color 120ms ease;

				&:hover,
				&:focus-visible,
				&.active {
					color: $bg-color;
					background-color: $text-color;
				}
			}
		}

		@media screen and (max-width: 850px) {
			grid-template-columns: 1fr 1fr;

			.credit-filter {
				grid-column: 1 / -1;
				grid-row: 2;
			}

			.list-filter {
				grid-column: 1 / -1;
				grid-row: 3;
				justify-content: center;
			}

			.sort-control {
				grid-column: 2;
				grid-row: 1;
			}
		}

		@media screen and (max-width: 480px) {
			grid-template-columns: 1fr;
			width: calc(100% - 24px);
			padding: 12px;

			.filter-control,
			.sort-control,
			.credit-filter,
			.list-filter {
				grid-column: 1;
				justify-self: stretch;
			}

			.sort-control {
				grid-row: 3;
			}

			.credit-filter {
				grid-row: 2;
			}

			.list-filter {
				grid-row: 4;
				justify-content: space-between;
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
		}
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
