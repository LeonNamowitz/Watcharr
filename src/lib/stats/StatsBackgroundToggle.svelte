<script lang="ts">
	import { getContext } from "svelte";
	import Icon from "@/lib/Icon.svelte";
	import tooltip from "@/lib/actions/tooltip";
	import {
		STATS_BACKGROUND_CONTEXT,
		type StatsBackgroundState,
	} from "./backgroundContext";

	const background = getContext<StatsBackgroundState>(STATS_BACKGROUND_CONTEXT);
</script>

<button
	class="plain other stats-background-toggle"
	aria-label="Stats background effects"
	aria-pressed={background.enabled}
	onclick={() => (background.enabled = !background.enabled)}
	use:tooltip={{
		text: background.enabled
			? "Hide background effects"
			: "Show background effects",
		pos: "bot",
	}}
>
	<Icon i={background.enabled ? "eye" : "eye-closed"} wh={24} />
</button>

<style>
	.stats-background-toggle {
		margin-right: 8px;
	}
</style>
