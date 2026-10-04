<script module lang="ts">
	import { SvelteMap } from "svelte/reactivity";
	// Keep large responses out of SvelteKit's sessionStorage snapshots.
	const savedResponses = new SvelteMap<string, StatsResponse>();
	let nextToken = 0;
</script>

<script lang="ts">
	import { resolve } from "$app/paths";
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import { tick } from "svelte";
	import { req, noAuthReq } from "@/lib/util/api";
	import Spinner from "@/lib/Spinner.svelte";
	import Error from "@/lib/Error.svelte";
	import StatsPage from "./StatsPage.svelte";
	import type { StatsResponse, StatsMedia } from "./types";
	import type { StatsSectionId } from "./sectionOrder";
	let { publicOwner }: { publicOwner?: { id: string; username: string } } =
		$props();
	let data = $state<StatsResponse>();
	let loadedOwnerKey = $state<string>();
	const ownerKey = $derived(
		`${publicOwner?.id ?? "private"}:${publicOwner?.username ?? ""}`,
	);
	let error = $state<unknown>();
	let loading = $state(true);
	let focusAfterLoad: string | undefined;
	let statsPage = $state<StatsPage>();
	let requestVersion = 0;
	let restoredKey: string | undefined;
	let pendingContext: ReturnType<StatsPage["capture"]> | undefined;
	const year = $derived(
		page.url.searchParams.get("year") ?? String(new Date().getUTCFullYear()),
	);
	const media = $derived(
		page.url.searchParams.get("media") === "game"
			? "game"
			: page.url.searchParams.get("media") === "tv"
				? "tv"
				: "movie",
	);
	const requestKey = $derived(
		`${publicOwner?.id ?? "private"}:${publicOwner?.username ?? ""}:${year}:${media}`,
	);

	export function capture() {
		if (loading || error || !data || !statsPage) return null;
		const token = String(nextToken++);
		savedResponses.set(token, $state.snapshot(data));
		while (savedResponses.size > 10) {
			const oldest = savedResponses.keys().next().value;
			if (oldest === undefined) break;
			savedResponses.delete(oldest);
		}
		return { key: requestKey, token, context: statsPage.capture() };
	}

	async function restoreContext(version: number) {
		await tick();
		if (version !== requestVersion || !pendingContext || !statsPage) return;
		const context = pendingContext;
		pendingContext = undefined;
		await statsPage.restore(context);
	}

	export function restore(saved: ReturnType<typeof capture>) {
		if (!saved || saved.key !== requestKey) return;
		pendingContext = saved.context;
		const response = savedResponses.get(saved.token);
		if (!response) {
			// After a reload or cache eviction, apply context once data arrives.
			if (!loading && data) void restoreContext(requestVersion);
			return;
		}
		restoredKey = saved.key;
		const version = ++requestVersion;
		data = response;
		loadedOwnerKey = ownerKey;
		error = undefined;
		loading = false;
		void restoreContext(version);
	}
	$effect(() => {
		const owner = publicOwner;
		const requestedOwnerKey = ownerKey;
		const key = requestKey;
		if (restoredKey === key) {
			restoredKey = undefined;
			return;
		}
		restoredKey = undefined;
		const version = ++requestVersion;
		const isActive = () => active && version === requestVersion;
		const params = new URLSearchParams({ year, media });
		const path = owner
			? `/public/users/${encodeURIComponent(owner.id)}/${encodeURIComponent(owner.username)}/stats`
			: "/stats";
		let active = true;
		loading = true;
		error = undefined;
		(owner ? noAuthReq : req)
			.get<StatsResponse>(`${path}?${params}`)
			.then((result) => {
				if (isActive()) {
					data = result;
					loadedOwnerKey = requestedOwnerKey;
				}
			})
			.catch((err) => {
				if (isActive()) error = err;
			})
			.finally(async () => {
				if (!isActive()) return;
				loading = false;
				await restoreContext(version);
				const control = focusAfterLoad;
				focusAfterLoad = undefined;
				await tick();
				if (isActive() && control && document.activeElement === document.body) {
					document
						.querySelector<HTMLElement>(
							`.stats-page [data-stats-control="${control}"]`,
						)
						?.focus({ preventScroll: true });
				}
			});
		return () => {
			active = false;
		};
	});
	async function saveLayout(sectionOrder: StatsSectionId[]) {
		if (publicOwner || loading || !data)
			throw new globalThis.Error("Stats layout is unavailable.");
		const current = data;
		const result = await req.put<{ sectionOrder: string[] }>("/stats/layout", {
			media: current.media,
			sectionOrder,
		});
		if (data === current) data.sectionOrder = result.sectionOrder;
		// Back navigation must not restore an order from before the save.
		for (const [token, response] of savedResponses) {
			if (
				response.owner.id === current.owner.id &&
				response.media === current.media
			)
				savedResponses.set(token, {
					...response,
					sectionOrder: result.sectionOrder,
				});
		}
	}
	function changeSelection(nextYear: string, nextMedia: StatsMedia) {
		focusAfterLoad =
			document.activeElement instanceof HTMLElement
				? document.activeElement.dataset.statsControl
				: undefined;
		const url = new URL(page.url);
		url.searchParams.set("year", nextYear);
		url.searchParams.set("media", nextMedia);
		const query = url.searchParams.toString();
		goto(
			publicOwner
				? resolve(`/(public)/lists/[id]/[username]/stats?${query}`, publicOwner)
				: resolve(`/profile/stats?${query}`),
			{ noScroll: true, keepFocus: true },
		);
	}
</script>

{#if data && loadedOwnerKey === ownerKey}
	<StatsPage
		bind:this={statsPage}
		{data}
		{publicOwner}
		{loading}
		{error}
		requestedYear={year}
		requestedMedia={media}
		onSelectionChange={changeSelection}
		onSaveLayout={saveLayout}
	/>
{:else if loading}<div class="loading" role="status">
		<Spinner />
		<p>Gathering your stats…</p>
	</div>{:else if error}<div class="error">
		<Error {error} pretty="Unable to load these stats." />
	</div>{/if}

<style>
	.loading,
	.error {
		max-width: 1000px;
		margin: 40px auto;
		padding: 24px;
		text-align: center;
	}
	.loading p {
		font-size: 13px;
		opacity: 0.6;
		margin-top: 15px;
	}
</style>
