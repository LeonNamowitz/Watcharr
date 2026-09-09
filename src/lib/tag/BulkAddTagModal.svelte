<script lang="ts">
	import { onMount } from "svelte";
	import { SvelteSet } from "svelte/reactivity";
	import Modal from "../Modal.svelte";
	import Spinner from "../Spinner.svelte";
	import Icon from "../Icon.svelte";
	import { baseURL, req } from "../util/api";
	import { notify } from "../util/notify";
	import { bulkTagWatched } from "./api";
	import {
		MediaTypeE,
		type PaginationResponse,
		type Tag,
		type TagCandidate,
		type TagCandidateMeta,
		type TagSuggestionKind,
		type TagSuggestionOptionsResponse,
	} from "@/types";

	interface Props {
		tag: Tag;
		onClose: () => void;
		onAdded: (count: number) => void;
	}

	let { tag, onClose, onAdded }: Props = $props();

	let mode: "browse" | "suggestions" = $state("browse");
	let source: "genre" | "company" | "future" = $state("genre");
	let criterionId = $state("");
	let options: TagSuggestionOptionsResponse | undefined = $state();
	let candidates: TagCandidate[] = $state([]);
	const selectedIds = new SvelteSet<number>();
	let queryInput = $state("");
	let activeQuery = $state("");
	let page = $state(1);
	let totalPages = $state(0);
	let totalResults = $state(0);
	let candidateMeta: TagCandidateMeta | undefined = $state();
	let loadingOptions = $state(false);
	let loadingCandidates = $state(false);
	let saving = $state(false);
	let error = $state("");
	let requestVersion = 0;

	let activeOptions = $derived(
		source === "genre" ? options?.genres : options?.companies,
	);

	function posterSource(candidate: TagCandidate) {
		const media = candidate.media;
		if (media.poster?.path) return `${baseURL}/${media.poster.path}`;
		if (!media.extPosterPath) return;
		if (
			media.type === MediaTypeE.tmdbMovie ||
			media.type === MediaTypeE.tmdbShow
		) {
			return `${baseURL}/img${media.extPosterPath}`;
		}
		if (media.type === MediaTypeE.igdbGame) {
			return `https://images.igdb.com/igdb/image/upload/t_cover_big/${media.extPosterPath}.jpg`;
		}
	}

	function mediaTypeLabel(candidate: TagCandidate) {
		switch (candidate.media.type) {
			case MediaTypeE.tmdbMovie:
				return "Movie";
			case MediaTypeE.tmdbShow:
				return "TV";
			case MediaTypeE.igdbGame:
				return "Game";
			default:
				return "Media";
		}
	}

	function watchedId(candidate: TagCandidate) {
		return candidate.media.watched?.id ?? 0;
	}

	async function loadOptions() {
		if (options || loadingOptions) return;
		loadingOptions = true;
		error = "";
		try {
			options = await req.get<TagSuggestionOptionsResponse>(
				`/tag/${tag.id}/suggestion-options`,
			);
		} catch (err) {
			console.error("BulkAddTagModal: Failed loading suggestion options", err);
			error = "Failed to load suggestion options.";
		} finally {
			loadingOptions = false;
		}
	}

	function currentKind(): TagSuggestionKind {
		return mode === "browse" ? "all" : source;
	}

	async function loadCandidates(nextPage = 1) {
		const version = ++requestVersion;
		if (mode === "suggestions" && source !== "future" && !criterionId) {
			candidates = [];
			totalResults = 0;
			totalPages = 0;
			candidateMeta = undefined;
			loadingCandidates = false;
			return;
		}
		loadingCandidates = true;
		error = "";
		try {
			const response = await req.get<
				PaginationResponse<TagCandidate, TagCandidateMeta>
			>(`/tag/${tag.id}/candidates`, {
				params: {
					page: nextPage,
					limit: 30,
					kind: currentKind(),
					criterionId:
						mode === "suggestions" && source !== "future"
							? Number(criterionId)
							: undefined,
					q: activeQuery || undefined,
				},
			});
			if (version !== requestVersion) return;
			candidates = response.results ?? [];
			page = response.page;
			totalPages = response.totalPages;
			totalResults = response.totalResults;
			candidateMeta = response.meta;
		} catch (err) {
			if (version !== requestVersion) return;
			console.error("BulkAddTagModal: Failed loading candidates", err);
			candidates = [];
			error = "Failed to load tag candidates.";
		} finally {
			if (version === requestVersion) loadingCandidates = false;
		}
	}

	async function changeMode(nextMode: "browse" | "suggestions") {
		mode = nextMode;
		page = 1;
		candidateMeta = undefined;
		if (mode === "suggestions") {
			await loadOptions();
		}
		await loadCandidates();
	}

	async function changeSource(event: Event) {
		source = (event.currentTarget as HTMLSelectElement).value as typeof source;
		criterionId = "";
		page = 1;
		candidateMeta = undefined;
		await loadCandidates();
	}

	async function changeCriterion(event: Event) {
		criterionId = (event.currentTarget as HTMLSelectElement).value;
		await loadCandidates(1);
	}

	async function search(event: SubmitEvent) {
		event.preventDefault();
		activeQuery = queryInput.trim();
		await loadCandidates(1);
	}

	function toggleCandidate(watchedId: number) {
		if (selectedIds.has(watchedId)) selectedIds.delete(watchedId);
		else selectedIds.add(watchedId);
	}

	function selectVisible() {
		for (const candidate of candidates) {
			if (candidate.media.watched?.id) {
				selectedIds.add(candidate.media.watched.id);
			}
		}
	}

	async function addSelected() {
		if (selectedIds.size === 0 || saving) return;
		saving = true;
		error = "";
		const notificationId = notify({
			text: `Adding ${selectedIds.size} item${selectedIds.size === 1 ? "" : "s"}`,
			type: "loading",
		});
		try {
			const response = await bulkTagWatched(tag.id, [...selectedIds]);
			notify({
				id: notificationId,
				text: `${response.added} item${response.added === 1 ? "" : "s"} added!`,
				type: "success",
			});
			onAdded(response.added);
		} catch (err) {
			console.error("BulkAddTagModal: Failed adding selected items", err);
			notify({
				id: notificationId,
				text: "Failed to add selected items.",
				type: "error",
			});
			error = "Nothing was changed. Your selection has been kept.";
		} finally {
			saving = false;
		}
	}

	onMount(() => loadCandidates());
</script>

<Modal
	title={`Add multiple to ${tag.name}`}
	desc="Review candidates, then add only the items you select. Existing tag members are not changed."
	maxWidth="900px"
	{onClose}
	{error}
>
	<div class="bulk-editor">
		<div class="tabs" role="tablist" aria-label="Candidate source">
			<button
				class:active={mode === "browse"}
				role="tab"
				aria-selected={mode === "browse"}
				onclick={() => changeMode("browse")}
			>
				Browse my list
			</button>
			<button
				class:active={mode === "suggestions"}
				role="tab"
				aria-selected={mode === "suggestions"}
				onclick={() => changeMode("suggestions")}
			>
				Suggestions
			</button>
		</div>

		{#if mode === "suggestions"}
			<div class="suggestion-controls">
				<label>
					<span>Match using</span>
					<select value={source} onchange={changeSource}>
						<option value="genre">Genre</option>
						<option value="company">Studio</option>
						<option value="future">Future release</option>
					</select>
				</label>
				{#if source !== "future"}
					<label>
						<span>{source === "genre" ? "Genre" : "Studio"}</span>
						<select
							value={criterionId}
							onchange={changeCriterion}
							disabled={loadingOptions}
						>
							<option value="">Choose one...</option>
							{#each activeOptions ?? [] as option (option.id)}
								<option value={option.id}>
									{option.name} ({option.count})
								</option>
							{/each}
						</select>
					</label>
				{:else if options}
					<span class="future-count">
						{options.futureReleaseCount} future release{options.futureReleaseCount ===
						1
							? ""
							: "s"} available
					</span>
				{/if}
			</div>
			{#if loadingOptions}
				<div class="inline-loading"><Spinner /></div>
			{:else if options?.incomplete}
				<p class="warning">
					Some suggestion data could not be loaded ({options.skippedCount}
					{options.skippedCount === 1 ? "item" : "items"} skipped).
				</p>
			{/if}
		{/if}

		<form class="search" onsubmit={search}>
			<input
				type="search"
				placeholder="Filter candidates by title"
				aria-label="Filter candidates by title"
				bind:value={queryInput}
			/>
			<button type="submit" disabled={loadingCandidates}>Search</button>
		</form>

		<div class="selection-bar">
			<span>{selectedIds.size} selected</span>
			<div>
				<button
					class="plain"
					onclick={selectVisible}
					disabled={candidates.length === 0}>Select page</button
				>
				<button
					class="plain"
					onclick={() => selectedIds.clear()}
					disabled={selectedIds.size === 0}>Clear all</button
				>
			</div>
		</div>

		{#if candidateMeta?.incomplete}
			<p class="warning">
				Results are incomplete because metadata for {candidateMeta.skippedCount}
				{candidateMeta.skippedCount === 1 ? "item" : "items"} could not be loaded.
			</p>
		{/if}

		<div class="candidate-list" aria-busy={loadingCandidates}>
			{#if loadingCandidates}
				<div class="inline-loading"><Spinner /></div>
			{:else if candidates.length > 0}
				{#each candidates as candidate (watchedId(candidate))}
					{@const candidateWatchedId = watchedId(candidate)}
					<label class:selected={selectedIds.has(candidateWatchedId)}>
						<input
							type="checkbox"
							checked={selectedIds.has(candidateWatchedId)}
							disabled={!candidateWatchedId}
							onchange={() => toggleCandidate(candidateWatchedId)}
						/>
						{#if posterSource(candidate)}
							<img src={posterSource(candidate)} alt="" />
						{:else}
							<span class="poster-placeholder"><Icon i="ticket" wh={26} /></span
							>
						{/if}
						<span class="candidate-info">
							<strong>{candidate.media.name}</strong>
							<small>
								{mediaTypeLabel(candidate)}
								{#if candidate.media.releaseDate}
									· {new Date(candidate.media.releaseDate).getUTCFullYear()}
								{/if}
							</small>
							{#if candidate.reason}<span class="reason"
									>{candidate.reason}</span
								>{/if}
						</span>
					</label>
				{/each}
			{:else if mode === "suggestions" && source !== "future" && !criterionId}
				<p class="empty">Choose an exact {source} to see suggestions.</p>
			{:else}
				<p class="empty">No untagged candidates found.</p>
			{/if}
		</div>

		<div class="footer">
			<div class="pagination">
				<button
					class="secondary"
					disabled={loadingCandidates || page <= 1}
					onclick={() => loadCandidates(page - 1)}>Previous</button
				>
				<span>{totalResults} result{totalResults === 1 ? "" : "s"}</span>
				<button
					class="secondary"
					disabled={loadingCandidates || page >= totalPages}
					onclick={() => loadCandidates(page + 1)}>Next</button
				>
			</div>
			<button
				class="add-selected"
				disabled={selectedIds.size === 0 || saving}
				onclick={addSelected}
			>
				{saving ? "Adding..." : `Add selected (${selectedIds.size})`}
			</button>
		</div>
	</div>
</Modal>

<style lang="scss">
	.bulk-editor {
		display: flex;
		flex-flow: column;
		gap: 12px;
		min-height: min(620px, 70dvh);
	}

	.tabs {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
	}

	.suggestion-controls {
		display: flex;
		align-items: end;
		gap: 10px;
		flex-wrap: wrap;

		label {
			display: flex;
			flex-flow: column;
			gap: 4px;
			min-width: 180px;
			flex: 1;
		}

		span {
			font-size: 13px;
			font-weight: bold;
		}

		select {
			width: 100%;
			padding: 8px;
			border: 2px solid $text-color;
			border-radius: 5px;
			background: $bg-color;
			color: $text-color;
		}

		.future-count {
			padding: 10px 0;
			font-weight: normal;
		}
	}

	.search {
		display: flex;
		gap: 8px;

		input {
			min-width: 0;
			flex: 1;
		}

		button {
			width: max-content;
		}
	}

	.selection-bar,
	.footer,
	.pagination {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.selection-bar {
		justify-content: space-between;
		font-size: 13px;

		div {
			display: flex;
			gap: 12px;
		}

		button {
			color: $text-color;
			text-decoration: underline;
		}
	}

	.warning {
		padding: 8px 10px;
		border-radius: 5px;
		background: color-mix(in srgb, #e9a23b 24%, transparent);
		font-size: 13px;
	}

	.candidate-list {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		align-content: start;
		gap: 7px;
		min-height: 250px;
		overflow-y: auto;

		& > label {
			display: grid;
			grid-template-columns: auto 42px minmax(0, 1fr);
			align-items: center;
			gap: 9px;
			padding: 7px;
			border: 1px solid color-mix(in srgb, $text-color 30%, transparent);
			border-radius: 7px;
			cursor: pointer;

			&.selected {
				background: color-mix(in srgb, $text-color 12%, transparent);
				border-color: $text-color;
			}
		}

		img,
		.poster-placeholder {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 42px;
			height: 58px;
			object-fit: cover;
			border-radius: 4px;
			background: color-mix(in srgb, $text-color 10%, transparent);
		}
	}

	.candidate-info {
		display: flex;
		flex-flow: column;
		min-width: 0;

		strong,
		span {
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		small {
			opacity: 0.7;
		}

		.reason {
			margin-top: 4px;
			font-size: 12px;
			color: #64c894;
		}
	}

	.inline-loading,
	.empty {
		display: flex;
		align-items: center;
		justify-content: center;
		grid-column: 1 / -1;
		min-height: 180px;
		text-align: center;
	}

	.footer {
		justify-content: space-between;
		margin-top: auto;

		.pagination {
			button {
				width: max-content;
			}

			span {
				font-size: 13px;
			}
		}

		.add-selected {
			width: max-content;
		}
	}

	@media screen and (max-width: 680px) {
		.bulk-editor {
			min-height: 75dvh;
		}

		.candidate-list {
			grid-template-columns: 1fr;
		}

		.footer {
			align-items: stretch;
			flex-flow: column;

			.pagination {
				justify-content: space-between;
			}

			.add-selected {
				width: 100%;
			}
		}
	}
</style>
