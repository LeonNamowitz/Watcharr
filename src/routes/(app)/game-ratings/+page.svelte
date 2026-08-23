<script lang="ts">
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import Spinner from "@/lib/Spinner.svelte";
	import { req } from "@/lib/util/api";
	import { notify } from "@/lib/util/notify";
	import type {
		Media,
		PaginationResponse,
		Watched,
		WatchedStatus,
	} from "@/types";

	type GameRow = {
		media: Media;
		watched: Watched;
		value: string;
		originalValue?: number;
		error?: string;
	};

	type StatusFilter = "ALL" | WatchedStatus;
	type SortMode = "ALPHA" | "RATING_DESC" | "RATING_ASC";

	let rows = $state<GameRow[]>([]);
	let search = $state("");
	let statusFilter = $state<StatusFilter>("ALL");
	let sortMode = $state<SortMode>("ALPHA");
	let loading = $state(true);
	let loadingPage = $state(1);
	let loadError = $state<unknown>();
	let saving = $state(false);
	let savedCount = $state(0);
	let saveTotal = $state(0);
	let saveErrorCount = $state(0);

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

	const visibleRows = $derived.by(() => {
		const query = search.trim().toLocaleLowerCase();
		return rows
			.filter(
				(row) =>
					(statusFilter === "ALL" || row.watched.status === statusFilter) &&
					(!query || gameTitle(row.media).toLocaleLowerCase().includes(query)),
			)
			.sort((a, b) => {
				if (sortMode === "RATING_DESC") {
					return (
						ratingValue(b) - ratingValue(a) ||
						gameTitle(a.media).localeCompare(gameTitle(b.media))
					);
				}
				if (sortMode === "RATING_ASC") {
					return (
						ratingValue(a) - ratingValue(b) ||
						gameTitle(a.media).localeCompare(gameTitle(b.media))
					);
				}
				return gameTitle(a.media).localeCompare(gameTitle(b.media));
			});
	});

	const changedRows = $derived(rows.filter(isChanged));
	const invalidRows = $derived(
		changedRows.filter((row) => !validRating(row.value)),
	);
	const ratedCount = $derived(
		rows.filter((row) => row.originalValue !== undefined).length,
	);

	function updateStatusFilter(event: Event) {
		statusFilter = (event.currentTarget as HTMLSelectElement)
			.value as StatusFilter;
	}

	function updateSortMode(event: Event) {
		sortMode = (event.currentTarget as HTMLSelectElement).value as SortMode;
	}

	function ratingValue(row: GameRow) {
		const rating = Number(row.value);
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

	function gameHref(media: Media) {
		return media.ids.igdb ? `/game/${media.ids.igdb}` : undefined;
	}

	function validRating(value: string) {
		if (!value.trim()) return false;
		const rating = Number(value);
		return Number.isFinite(rating) && rating >= 0.1 && rating <= 10;
	}

	function isChanged(row: GameRow) {
		if (!validRating(row.value)) return row.value.trim() !== "";
		return Number(row.value) !== row.originalValue;
	}

	function updateValue(row: GameRow, event: Event) {
		row.value = (event.currentTarget as HTMLInputElement).value;
		row.error = undefined;
	}

	function formatRating(rating?: number) {
		return rating && rating > 0 ? String(rating) : "";
	}

	async function loadGames() {
		loading = true;
		loadError = undefined;
		rows = [];

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
					return {
						media,
						watched: media.watched as Watched,
						value: formatRating(rating),
						originalValue: rating && rating > 0 ? rating : undefined,
					};
				});
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
			row.error = validRating(row.value)
				? undefined
				: "Enter a number from 0.1 to 10.";
		}
		if (pendingRows.some((row) => !validRating(row.value))) return;

		saving = true;
		savedCount = 0;
		saveTotal = pendingRows.length;
		saveErrorCount = 0;
		const savingNotice = notify({
			text: `Saving 0 of ${pendingRows.length} ratings…`,
			type: "loading",
		});

		for (const row of pendingRows) {
			try {
				const rating = Number(row.value);
				await req.put(`/watched/${row.watched.id}`, { rating });
				row.originalValue = rating;
				row.watched.rating = rating;
				savedCount += 1;
				notify({
					id: savingNotice,
					text: `Saving ${savedCount} of ${pendingRows.length} ratings…`,
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
					: `Saved ${savedCount} rating${savedCount === 1 ? "" : "s"}.`,
			type: saveErrorCount > 0 ? "error" : "success",
			time: 7000,
		});
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
				Update every game on your list in one place. Ratings are stored on a
				0–10 scale. Leave a field unchanged to keep its current value.
			</p>
		</div>
		<a class="back-link" href="/">
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
		<div class="toolbar">
			<div class="filter-controls">
				<label class="search-box">
					<Icon i="search" wh={17} />
					<input bind:value={search} type="search" placeholder="Find a game…" />
				</label>
				<label class="status-filter">
					<span>Status</span>
					<select value={statusFilter} onchange={updateStatusFilter}>
						{#each statusOptions as option}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</label>
				<label class="sort-filter">
					<span>Sort</span>
					<select value={sortMode} onchange={updateSortMode}>
						{#each sortOptions as option}
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
					<span>Your rating</span>
				</div>
				<div class="game-list">
					{#each visibleRows as row (row.watched.id)}
						<div class:changed={isChanged(row)} class="game-row">
							<div class="game-info">
								{#if gameHref(row.media)}
									<a class="game-title" href={gameHref(row.media)}
										>{gameTitle(row.media)}</a
									>
								{:else}
									<span class="game-title">{gameTitle(row.media)}</span>
								{/if}
								{#if gameYear(row.media)}
									<span class="game-year">{gameYear(row.media)}</span>
								{/if}
							</div>
							<div class="rating-input-wrap">
								<input
									class:invalid={row.error}
									aria-label={`Rating for ${gameTitle(row.media)}`}
									type="number"
									min="0.1"
									max="10"
									step="0.1"
									placeholder="—"
									value={row.value}
									oninput={(event) => updateValue(row, event)}
								/>
								{#if row.error}
									<span class="row-error">{row.error}</span>
								{/if}
							</div>
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
						Some ratings could not be saved. You can retry them.
					{:else}
						{changedRows.length === 0
							? "Everything is up to date."
							: `${changedRows.length} rating${changedRows.length === 1 ? "" : "s"} ready to save.`}
					{/if}
				</p>
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

	.list-card {
		overflow: hidden;
		border: 2px solid $text-color;
		border-radius: 8px;
	}

	.list-header,
	.game-row {
		display: grid;
		grid-template-columns: minmax(0, 1fr) 130px;
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

		.list-header,
		.game-row {
			grid-template-columns: minmax(0, 1fr) 95px;
			gap: 10px;
		}

		.list-header,
		.game-row {
			padding-left: 11px;
			padding-right: 11px;
		}

		.save-bar button {
			width: 100%;
		}
	}
</style>
