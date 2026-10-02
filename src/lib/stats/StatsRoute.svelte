<script lang="ts">
	import { resolve } from "$app/paths";
	import { goto } from "$app/navigation";
	import { page } from "$app/state";
	import { req, noAuthReq } from "@/lib/util/api";
	import Spinner from "@/lib/Spinner.svelte";
	import Error from "@/lib/Error.svelte";
	import StatsPage from "./StatsPage.svelte";
	import type { StatsResponse } from "./types";
	let { publicOwner }: { publicOwner?: { id: string; username: string } } =
		$props();
	let data = $state<StatsResponse>();
	let error = $state<unknown>();
	let loading = $state(true);
	const year = $derived(
		page.url.searchParams.get("year") ?? String(new Date().getUTCFullYear()),
	);
	const media = $derived(
		page.url.searchParams.get("media") === "tv" ? "tv" : "movie",
	);
	$effect(() => {
		const owner = publicOwner;
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
				if (active) data = result;
			})
			.catch((err) => {
				if (active) error = err;
			})
			.finally(() => {
				if (active) loading = false;
			});
		return () => {
			active = false;
		};
	});
	function changeSelection(nextYear: string, nextMedia: "movie" | "tv") {
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

{#if loading}<div class="loading">
		<Spinner />
		<p>Gathering your viewing journal…</p>
	</div>{:else if error}<div class="error">
		<Error {error} pretty="Unable to load these stats." />
	</div>{:else if data}{#key `${publicOwner?.id ?? "private"}:${data.scope}:${data.year}:${data.media}`}<StatsPage
			{data}
			{publicOwner}
			onSelectionChange={changeSelection}
		/>{/key}{/if}

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
