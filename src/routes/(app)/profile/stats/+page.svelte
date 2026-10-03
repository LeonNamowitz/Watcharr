<script lang="ts">
	import type { Snapshot } from "@sveltejs/kit";
	import StatsRoute from "@/lib/stats/StatsRoute.svelte";
	let statsRoute: StatsRoute;
	export const snapshot: Snapshot<ReturnType<StatsRoute["capture"]>> = {
		capture: () => statsRoute.capture(),
		restore: (saved) => statsRoute.restore(saved),
	};
</script>

<StatsRoute bind:this={statsRoute} />
