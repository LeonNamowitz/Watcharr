<script lang="ts">
	import { page } from "$app/state";
	import type { Snapshot } from "@sveltejs/kit";
	import StatsRoute from "@/lib/stats/StatsRoute.svelte";
	const publicOwner = $derived({
		id: page.params.id ?? "",
		username: page.params.username ?? "",
	});
	let statsRoute: StatsRoute;
	export const snapshot: Snapshot<ReturnType<StatsRoute["capture"]>> = {
		capture: () => statsRoute.capture(),
		restore: (saved) => statsRoute.restore(saved),
	};
</script>

<StatsRoute bind:this={statsRoute} {publicOwner} />
