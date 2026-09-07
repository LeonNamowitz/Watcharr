<script lang="ts">
	import { resolve } from "$app/paths";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import WatchedDeleteModal from "@/lib/watched/WatchedDeleteModal.svelte";
	import { removeWatched, req, updateWatched } from "@/lib/util/api";
	import { notify } from "@/lib/util/notify";
	import type {
		Media,
		PaginationResponse,
		SearchResponseMeta,
		SupportedMedia,
		Watched,
		WatchedUpdateRequest,
		WatchedStatus,
	} from "@/types";
	import { SearchType } from "@/types";

	type GameRow = {
		media: Media;
		watched: Watched;
		value: string;
		status: WatchedStatus;
		originalStatus: WatchedStatus;
		originalValue?: number;
		error?: string;
	};

	type StatusFilter = "ALL" | WatchedStatus;
	type SortMode = "ALPHA" | "RATING_DESC" | "RATING_ASC";

	let rows = $state<GameRow[]>([]);
	let search = $state("");
	let statusFilter = $state<StatusFilter>("ALL");
	let sortMode = $state<SortMode>("ALPHA");
	let sortOrder = $state<number[]>([]);
	let loading = $state(true);
	let loadingPage = $state(1);
	let loadError = $state<unknown>();
	let saving = $state(false);
	let savedCount = $state(0);
	let saveTotal = $state(0);
	let saveErrorCount = $state(0);
	let backdateRatingActivities = $state(true);
	let addSearch = $state("");
	let addStatus = $state<WatchedStatus>("PLANNED");
	let addResults = $state<Media[]>([]);
	let addLoading = $state(false);
	let addError = $state<unknown>();
	let addingGameId = $state<number>();
	let deleteTarget = $state<GameRow>();
	let deletingGameId = $state<number>();

	const statusOptions: { value: StatusFilter; label: string }[] = [
		{ value: "ALL", label: "All statuses" },
		{ value: "FINISHED", label: "Played (finished)" },
		{ value: "WATCHING", label: "Playing" },
		{ value: "PLANNED", label: "Planned" },
		{ value: "HOLD", label: "On hold" },
		{ value: "DROPPED", label: "Dropped" },
	];

	const sortOptions: { value: SortMode; label: string }[] = [
		{ value: "ALPHA", label: "Alphabetical" },
		{ value: "RATING_DESC", label: "Rating: high to low" },
		{ value: "RATING_ASC", label: "Rating: low to high" },
	];
	const editableStatusOptions = statusOptions.filter(
		(option): option is { value: WatchedStatus; label: string } =>
			option.value !== "ALL",
	);

	const visibleRows = $derived.by(() => {
		const query = search.trim().toLocaleLowerCase();
		const positionById = new Map(
			sortOrder.map((watchedId, position) => [watchedId, position]),
		);
		return rows
			.filter(
				(row) =>
					(statusFilter === "ALL" || row.originalStatus === statusFilter) &&
					(!query || gameTitle(row.media).toLocaleLowerCase().includes(query)),
			)
			.sort((a, b) => {
				return (
					(positionById.get(a.watched.id) ?? Number.MAX_SAFE_INTEGER) -
						(positionById.get(b.watched.id) ?? Number.MAX_SAFE_INTEGER) ||
					gameTitle(a.media).localeCompare(gameTitle(b.media))
				);
			});
	});

	const changedRows = $derived(rows.filter(isChanged));
	const invalidRows = $derived(
		changedRows.filter((row) => ratingChanged(row) && !validRating(row.value)),
	);
	const ratedCount = $derived(
		rows.filter((row) => row.originalValue !== undefined).length,
	);

	function updateStatusFilter(event: Event) {
		statusFilter = (event.currentTarget as HTMLSelectElement)
			.value as StatusFilter;
	}

	function updateSortMode(event: Event) {
		const nextSortMode = (event.currentTarget as HTMLSelectElement)
			.value as SortMode;
		sortMode = nextSortMode;
		sortOrder = sortedRowIds(rows, nextSortMode);
	}

	function resortRows() {
		sortOrder = sortedRowIds(rows, sortMode);
	}

	function ratingValue(row: GameRow) {
		const rating = parseRating(row.value);
		return row.value.trim() && Number.isFinite(rating) ? rating : -1;
	}

	function gameTitle(media: Media) {
		return media.name?.trim() || "Untitled game";
	}

	function gameYear(media: Media) {
		if (!media.releaseDate) return "";
		const year = new Date(media.releaseDate).getFullYear();
		return Number.isNaN(year) ? "" : String(year);
	}

	function gameHref(media: Media): `/${SupportedMedia}/${string}` | undefined {
		return media.ids.igdb ? `/game/${media.ids.igdb}` : undefined;
	}

	function validRating(value: string) {
		if (!value.trim()) return false;
		const rating = parseRating(value);
		return Number.isFinite(rating) && rating >= 0.1 && rating <= 10;
	}

	function parseRating(value: string) {
		return Number(value.trim().replace(",", "."));
	}

	function ratingChanged(row: GameRow) {
		if (!row.value.trim()) return row.originalValue !== undefined;
		return (
			!validRating(row.value) || parseRating(row.value) !== row.originalValue
		);
	}

	function ratingError(row: GameRow) {
		return !row.value.trim() && row.originalValue !== undefined
			? "Restore the current rating; clearing is not supported."
			: "Enter a number from 0.1 to 10.";
	}

	function isChanged(row: GameRow) {
		return ratingChanged(row) || row.status !== row.originalStatus;
	}

	function updateValue(row: GameRow, event: Event) {
		row.value = (event.currentTarget as HTMLInputElement).value;
		row.error = undefined;
	}

	function updateRowStatus(row: GameRow, event: Event) {
		row.status = (event.currentTarget as HTMLSelectElement)
			.value as WatchedStatus;
		row.error = undefined;
	}

	function compareRows(a: GameRow, b: GameRow, mode: SortMode) {
		if (mode === "RATING_DESC") {
			return ratingValue(b) - ratingValue(a);
		}
		if (mode === "RATING_ASC") {
			return ratingValue(a) - ratingValue(b);
		}
		return 0;
	}

	function sortedRowIds(source: GameRow[], mode: SortMode) {
		return source
			.slice()
			.sort(
				(a, b) =>
					compareRows(a, b, mode) ||
					gameTitle(a.media).localeCompare(gameTitle(b.media)),
			)
			.map((row) => row.watched.id);
	}

	function formatRating(rating?: number) {
		return rating && rating > 0 ? String(rating) : "";
	}

	async function loadGames() {
		loading = true;
		loadError = undefined;
		rows = [];
		sortOrder = [];

		try {
			const games: Media[] = [];
			let page = 1;
			let totalPages = 1;

			do {
				loadingPage = page;
				const response = await req.get<PaginationResponse<Media, undefined>>(
					"/watched",
					{ params: { type: "game", page, limit: 100 } },
				);
				games.push(...(response.results ?? []));
				totalPages = response.totalPages || 1;
				page += 1;
			} while (page <= totalPages);

			rows = games
				.filter((media) => media.watched?.id)
				.map((media) => {
					const rating = media.watched?.rating;
					const status = (media.watched as Watched).status;
					return {
						media,
						watched: media.watched as Watched,
						value: formatRating(rating),
						status,
						originalStatus: status,
						originalValue: rating && rating > 0 ? rating : undefined,
					};
				});
			sortOrder = sortedRowIds(rows, sortMode);
		} catch (error) {
			console.error("game-ratings: Failed to load games", error);
			loadError = error;
		} finally {
			loading = false;
		}
	}

	async function saveRatings() {
		const pendingRows = rows.filter(isChanged);
		if (saving || pendingRows.length === 0) return;

		for (const row of pendingRows) {
			row.error =
				!ratingChanged(row) || validRating(row.value)
					? undefined
					: ratingError(row);
		}
		if (
			pendingRows.some((row) => ratingChanged(row) && !validRating(row.value))
		) {
			return;
		}

		saving = true;
		savedCount = 0;
		saveTotal = pendingRows.length;
		saveErrorCount = 0;
		const savingNotice = notify({
			text: `Saving 0 of ${pendingRows.length} changes…`,
			type: "loading",
		});

		for (const row of pendingRows) {
			try {
				const update: WatchedUpdateRequest = {};
				if (row.status !== row.originalStatus) {
					update.status = row.status;
				}
				if (ratingChanged(row)) {
					update.rating = parseRating(row.value);
					update.backdateRatingActivity = backdateRatingActivities;
				}
				await req.put(`/watched/${row.watched.id}`, update);
				if (ratingChanged(row)) {
					row.originalValue = parseRating(row.value);
					row.watched.rating = parseRating(row.value);
				}
				row.originalStatus = row.status;
				row.watched.status = row.status;
				savedCount += 1;
				notify({
					id: savingNotice,
					text: `Saving ${savedCount} of ${pendingRows.length} changes…`,
					type: "loading",
				});
			} catch (error) {
				console.error(
					`game-ratings: Failed to save ${gameTitle(row.media)}`,
					error,
				);
				row.error = "Could not save";
				saveErrorCount += 1;
			}
		}

		saving = false;
		notify({
			id: savingNotice,
			text:
				saveErrorCount > 0
					? `Saved ${savedCount}; ${saveErrorCount} failed.`
					: `Saved ${savedCount} change${savedCount === 1 ? "" : "s"}.`,
			type: saveErrorCount > 0 ? "error" : "success",
			time: 7000,
		});
	}

	async function searchGames() {
		const query = addSearch.trim();
		if (!query) {
			addResults = [];
			return;
		}
		addLoading = true;
		addError = undefined;
		try {
			const response = await req.get<
				PaginationResponse<Media, SearchResponseMeta>
			>("/search", {
				params: {
					query,
					type: SearchType.game,
					page: 1,
					limit: 20,
				},
			});
			const existingIds = new Set(
				rows
					.map((row) => row.media.ids.igdb)
					.filter((id): id is number => typeof id === "number"),
			);
			addResults = (response.results ?? []).filter(
				(media) =>
					typeof media.ids.igdb === "number" &&
					!existingIds.has(media.ids.igdb),
			);
		} catch (error) {
			console.error("game-ratings: Failed to search games", error);
			addError = error;
			addResults = [];
		} finally {
			addLoading = false;
		}
	}

	async function addGame(media: Media) {
		const igdbId = media.ids.igdb;
		if (
			typeof igdbId !== "number" ||
			rows.some((row) => row.media.ids.igdb === igdbId) ||
			addingGameId !== undefined
		) {
			return;
		}
		addingGameId = igdbId;
		try {
			const watched = await updateWatched(undefined, {
				contentId: igdbId,
				contentType: "game",
				status: addStatus,
			});
			if (!watched) return;
			const row: GameRow = {
				media,
				watched,
				value: formatRating(watched.rating),
				status: watched.status,
				originalStatus: watched.status,
				originalValue:
					watched.rating && watched.rating > 0 ? watched.rating : undefined,
			};
			rows = [...rows, row];
			sortOrder = [...sortOrder, watched.id];
			addResults = addResults.filter((result) => result !== media);
		} catch (error) {
			console.error(`game-ratings: Failed to add ${gameTitle(media)}`, error);
		} finally {
			addingGameId = undefined;
		}
	}

	function requestDelete(row: GameRow) {
		if (!saving && deletingGameId === undefined) {
			deleteTarget = row;
		}
	}

	async function closeDeleteModal(confirmed: boolean) {
		if (!confirmed) {
			deleteTarget = undefined;
			return;
		}
		const row = deleteTarget;
		if (!row) return;
		deletingGameId = row.watched.id;
		const removed = await removeWatched(row.watched.id);
		if (removed) {
			rows = rows.filter((candidate) => candidate !== row);
			sortOrder = sortOrder.filter((id) => id !== row.watched.id);
		} else {
			row.error = "Could not remove";
		}
		deletingGameId = undefined;
		deleteTarget = undefined;
	}

	loadGames();
</script>

<svelte:head>
	<title>Bulk Game Ratings</title>
</svelte:head>

<main class="ratings-page">
	<div class="intro">
		<div>
			<p class="eyebrow">Temporary tool</p>
			<h1>Bulk game ratings</h1>
			<p class="description">
				Edit ratings and statuses, add games, or remove them from your list.
				Ratings are stored on a 0–10 scale; for a 0–100 display, enter 8.7 to
				show 87 in the rest of the app.
			</p>
		</div>
		<a class="back-link" href={resolve("/")}>
			<Icon i="arrow" wh={16} />
			Back to list
		</a>
	</div>

	{#if loading}
		<div class="loading-state">
			<Spinner />
			<p>Loading games{loadingPage > 1 ? ` (page ${loadingPage})` : ""}…</p>
		</div>
	{:else if loadError}
		<Error
			pretty="Couldn't load your games."
			error={loadError}
			onRetry={() => {
				loadGames();
			}}
		/>
	{:else}
		<section class="add-panel">
			<div class="section-heading">
				<div>
					<h2>Add a game</h2>
					<p>Search IGDB and add a game directly to your list.</p>
				</div>
				<label class="add-status">
					<span>Add as</span>
					<select bind:value={addStatus}>
						{#each editableStatusOptions as option (option.value)}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</label>
			</div>
			<form
				class="add-search"
				onsubmit={(event) => {
					event.preventDefault();
					searchGames();
				}}
			>
				<input
					bind:value={addSearch}
					type="search"
					placeholder="Search for a game to add…"
					aria-label="Search for a game to add"
				/>
				<button disabled={addLoading || !addSearch.trim()} type="submit">
					{addLoading ? "Searching…" : "Search"}
				</button>
			</form>
			{#if addError}
				<p class="add-error">Couldn't search for games. Try again.</p>
			{:else if addResults.length > 0}
				<div class="add-results">
					{#each addResults as media (media.ids.igdb)}
						{@const igdbId = media.ids.igdb}
						<div class="add-result">
							<div>
								<strong>{gameTitle(media)}</strong>
								{#if gameYear(media)}
									<span>{gameYear(media)}</span>
								{/if}
							</div>
							<button
								disabled={addingGameId !== undefined}
								onclick={() => addGame(media)}
							>
								{addingGameId === igdbId ? "Adding…" : "Add"}
							</button>
						</div>
					{/each}
				</div>
			{:else if addSearch.trim() && !addLoading}
				<p class="add-empty">No new games found.</p>
			{/if}
		</section>

		<div class="toolbar">
			<div class="filter-controls">
				<label class="search-box">
					<Icon i="search" wh={17} />
					<input bind:value={search} type="search" placeholder="Find a game…" />
				</label>
				<label class="status-filter">
					<span>Status</span>
					<select value={statusFilter} onchange={updateStatusFilter}>
						{#each statusOptions as option (option.value)}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</label>
				<label class="sort-filter">
					<span>Sort</span>
					<select value={sortMode} onchange={updateSortMode}>
						{#each sortOptions as option (option.value)}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</label>
			</div>
			<div class="summary">
				<span
					>{visibleRows.length === rows.length
						? rows.length
						: visibleRows.length + " of " + rows.length}
					{rows.length === 1 ? "game" : "games"}</span
				>
				<span>{ratedCount} rated</span>
				{#if changedRows.length > 0}
					<strong>{changedRows.length} unsaved</strong>
				{/if}
			</div>
			<p class="sort-help">
				Changing a rating will not move it.
				<button class="plain resort-button" type="button" onclick={resortRows}
					>Re-sort now</button
				>
			</p>
		</div>

		{#if rows.length === 0}
			<div class="empty-state">
				<Icon i="gamepad" wh={56} />
				<h2>No games on your list</h2>
				<p>Add some games first, then return to this URL.</p>
			</div>
		{:else if visibleRows.length === 0}
			<div class="empty-state compact">
				<Icon i="search" wh={40} />
				<h2>No games match these filters</h2>
				<p>Try changing the search or status filter.</p>
			</div>
		{:else}
			<div class="list-card">
				<div class="list-header">
					<span>Game</span>
					<span>Status</span>
					<span>Your rating</span>
					<span aria-hidden="true"></span>
				</div>
				<div class="game-list">
					{#each visibleRows as row (row.watched.id)}
						{@const href = gameHref(row.media)}
						<div class:changed={isChanged(row)} class="game-row">
							<div class="game-info">
								{#if href}
									<a class="game-title" href={resolve(href)}
										>{gameTitle(row.media)}</a
									>
								{:else}
									<span class="game-title">{gameTitle(row.media)}</span>
								{/if}
								{#if gameYear(row.media)}
									<span class="game-year">{gameYear(row.media)}</span>
								{/if}
							</div>
							<select
								class="row-status"
								aria-label={`Status for ${gameTitle(row.media)}`}
								value={row.status}
								disabled={saving || deletingGameId === row.watched.id}
								onchange={(event) => updateRowStatus(row, event)}
							>
								{#each editableStatusOptions as option (option.value)}
									<option value={option.value}>{option.label}</option>
								{/each}
							</select>
							<div class="rating-input-wrap">
								<input
									class:invalid={row.error}
									aria-label={`Rating for ${gameTitle(row.media)}`}
									type="number"
									min="0.1"
									max="10"
									step="0.1"
									placeholder="—"
									disabled={saving || deletingGameId === row.watched.id}
									value={row.value}
									oninput={(event) => updateValue(row, event)}
								/>
								{#if row.error}
									<span class="row-error">{row.error}</span>
								{/if}
							</div>
							<button
								class="delete-row"
								type="button"
								disabled={saving || deletingGameId !== undefined}
								aria-label={`Remove ${gameTitle(row.media)} from your list`}
								onclick={() => requestDelete(row)}
							>
								<Icon i="trash" wh={18} />
							</button>
						</div>
					{/each}
				</div>
			</div>

			<div class="save-bar">
				<p>
					{#if saving}
						Saving {savedCount} of {saveTotal}…
					{:else if invalidRows.length > 0}
						Fix invalid ratings before saving.
					{:else if saveErrorCount > 0}
						Some changes could not be saved. You can retry them.
					{:else}
						{changedRows.length === 0
							? "Everything is up to date."
							: `${changedRows.length} change${changedRows.length === 1 ? "" : "s"} ready to save.`}
					{/if}
				</p>
				<label class="backdate-option">
					<input
						type="checkbox"
						bind:checked={backdateRatingActivities}
						disabled={saving}
					/>
					<span>Keep rating activity dates with existing history</span>
				</label>
				<button
					disabled={saving || changedRows.length === 0}
					onclick={saveRatings}
				>
					{saving ? "Saving…" : "Save changes"}
				</button>
			</div>
		{/if}
	{/if}
</main>

{#if deleteTarget}
	<WatchedDeleteModal
		mediaName={gameTitle(deleteTarget.media)}
		onClose={closeDeleteModal}
	/>
{/if}

<style lang="scss">
	.ratings-page {
		width: min(100% - 30px, 900px);
		margin: 0 auto 60px;
	}

	.intro {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 25px;
		margin: 0 0 25px;
	}

	.eyebrow {
		color: $text-color-accent;
		font-size: 12px;
		font-weight: bold;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	h1 {
		font-size: clamp(30px, 6vw, 48px);
		line-height: 1.1;
		margin: 3px 0 10px;
	}

	.description {
		max-width: 610px;
		color: $text-color-accent;
		line-height: 1.5;
	}

	.add-panel {
		margin-bottom: 18px;
		padding: 16px;
		border: 2px solid $text-color;
		border-radius: 8px;
	}

	.section-heading {
		display: flex;
		align-items: start;
		justify-content: space-between;
		gap: 15px;
		margin-bottom: 12px;

		h2 {
			font-size: 20px;
			margin-bottom: 3px;
		}

		p {
			color: $text-color-accent;
			font-size: 13px;
		}
	}

	.add-status {
		display: flex;
		align-items: center;
		gap: 8px;
		color: $text-color-accent;
		font-size: 13px;
		font-weight: bold;
		white-space: nowrap;

		select {
			min-width: 135px;
		}
	}

	.add-search {
		display: flex;
		gap: 8px;

		input {
			min-width: 0;
			flex: 1;
		}

		button {
			width: auto;
			min-width: 95px;
		}
	}

	.add-results {
		display: flex;
		flex-flow: column;
		gap: 7px;
		margin-top: 12px;
	}

	.add-result {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding-top: 7px;
		border-top: 1px solid $bg-color-accent;

		> div {
			display: flex;
			align-items: baseline;
			gap: 8px;
			min-width: 0;
		}

		strong {
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		span {
			color: $text-color-accent;
			font-size: 13px;
		}

		button {
			width: auto;
			padding: 5px 12px;
		}
	}

	.add-empty,
	.add-error {
		margin-top: 10px;
		color: $text-color-accent;
		font-size: 13px;
	}

	.add-error {
		color: $error;
	}

	.back-link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		border-bottom: 2px solid $text-color;
		padding-bottom: 3px;
		font-weight: bold;
		white-space: nowrap;

		:global(svg) {
			transform: rotate(180deg);
		}
	}

	.toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 15px;
		margin-bottom: 12px;
	}

	.filter-controls {
		display: flex;
		flex: 1;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
	}

	.status-filter,
	.sort-filter {
		display: flex;
		align-items: center;
		gap: 8px;
		color: $text-color-accent;
		font-size: 13px;
		font-weight: bold;
		white-space: nowrap;

		select {
			width: auto;
			min-width: 150px;
			padding: 7px 10px;
			border: 2px solid $text-color;
			border-radius: 5px;
			background: $bg-color;
			color: $text-color;
			font-weight: bold;
		}
	}

	.search-box {
		display: flex;
		align-items: center;
		gap: 8px;
		width: min(100%, 340px);
		padding: 0 10px;
		border: 2px solid $text-color;
		border-radius: 5px;

		:global(svg) {
			flex: 0 0 auto;
		}

		input {
			border: 0;
			box-shadow: none;
			padding-left: 0;

			&:focus {
				box-shadow: none;
			}
		}
	}

	.summary {
		display: flex;
		align-items: center;
		gap: 12px;
		color: $text-color-accent;
		font-size: 13px;
		white-space: nowrap;

		strong {
			color: $text-color;
		}
	}

	.sort-help {
		flex-basis: 100%;
		margin-top: 5px;
		color: $text-color-accent;
		font-size: 12px;
		text-align: right;

		.resort-button {
			color: $text-color;
			font-weight: bold;
			text-decoration: underline;
		}
	}

	.list-card {
		overflow: hidden;
		border: 2px solid $text-color;
		border-radius: 8px;
	}

	.list-header,
	.game-row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 145px 115px 34px;
		gap: 20px;
		align-items: center;
	}

	.list-header {
		padding: 11px 16px;
		background: $text-color;
		color: $bg-color;
		font-size: 12px;
		font-weight: bold;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.game-row {
		min-height: 64px;
		padding: 10px 16px;
		border-top: 1px solid $bg-color-accent;
		transition: background-color 120ms ease;

		&:first-child {
			border-top: 0;
		}

		&.changed {
			background: $accent-color;
		}
	}

	.game-info {
		display: flex;
		align-items: baseline;
		gap: 8px;
		min-width: 0;
	}

	.row-status {
		width: 100%;
		min-width: 0;
		padding: 7px 6px;
		font-size: 12px;
	}

	.game-title {
		overflow: hidden;
		font-weight: bold;
		text-overflow: ellipsis;
		white-space: nowrap;

		&:hover {
			text-decoration: underline;
		}
	}

	.game-year {
		color: $text-color-accent;
		font-size: 13px;
		white-space: nowrap;
	}

	.rating-input-wrap {
		position: relative;

		input {
			width: 100%;
			font-size: 16px;
			font-weight: bold;
			text-align: center;

			&.invalid {
				border-color: $error;
			}
		}
	}

	.row-error {
		position: absolute;
		top: calc(100% + 2px);
		right: 0;
		z-index: 1;
		color: $error;
		font-size: 11px;
		white-space: nowrap;
	}

	.delete-row {
		width: 30px;
		padding: 5px;

		&:hover {
			color: $error;
		}
	}

	.save-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 15px;
		margin-top: 15px;

		p {
			color: $text-color-accent;
			font-size: 13px;
		}

		button {
			width: auto;
			min-width: 135px;
		}
	}

	.backdate-option {
		display: flex;
		align-items: center;
		gap: 7px;
		margin-left: auto;
		color: $text-color-accent;
		font-size: 12px;

		input {
			width: auto;
		}
	}

	.loading-state,
	.empty-state {
		display: flex;
		flex-flow: column;
		align-items: center;
		justify-content: center;
		gap: 10px;
		min-height: 220px;
		text-align: center;

		:global(.container) {
			margin-bottom: 5px;
			transform: scale(0.6);
		}

		p {
			color: $text-color-accent;
		}
	}

	.empty-state {
		:global(svg) {
			fill: $text-color-accent;
		}

		h2 {
			font-size: 24px;
		}

		&.compact {
			min-height: 160px;
		}
	}

	@media screen and (max-width: 600px) {
		.intro {
			align-items: start;
			flex-flow: column;
			gap: 15px;
		}

		.back-link {
			order: -1;
		}

		.toolbar,
		.save-bar {
			align-items: stretch;
			flex-flow: column;
		}

		.filter-controls {
			align-items: stretch;
			flex-flow: column;
		}

		.search-box {
			width: 100%;
		}

		.status-filter,
		.sort-filter {
			justify-content: space-between;

			select {
				flex: 1;
			}
		}

		.summary {
			justify-content: space-between;
		}

		.section-heading {
			flex-flow: column;
		}

		.add-status {
			justify-content: space-between;
			width: 100%;

			select {
				flex: 1;
			}
		}

		.sort-help {
			text-align: left;
		}

		.list-header,
		.game-row {
			grid-template-columns: minmax(0, 1fr) 95px 72px 26px;
			gap: 7px;
		}

		.list-header,
		.game-row {
			padding-left: 11px;
			padding-right: 11px;
		}

		.save-bar button {
			width: 100%;
		}

		.backdate-option {
			margin-left: 0;
		}
	}
</style>
