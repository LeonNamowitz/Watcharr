<script lang="ts">
	import { page } from "$app/state";
	import StatsRoute from "@/lib/stats/StatsRoute.svelte";
	const publicOwner = $derived({
		id: page.params.id ?? "",
		username: page.params.username ?? "",
	});
</script>

<StatsRoute {publicOwner} />
