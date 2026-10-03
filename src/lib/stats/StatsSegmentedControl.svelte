<script lang="ts">
	import type { Snippet } from "svelte";
	let {
		label,
		role,
		wrap = false,
		children,
	}: {
		label?: string;
		role?: "group";
		wrap?: boolean;
		children: Snippet;
	} = $props();
</script>

<div class="segmented" class:wrap aria-label={label} {role}>
	{@render children()}
</div>

<style>
	.segmented {
		display: flex;
		background: #8881;
		border: 1px solid var(--stats-border);
		border-radius: 7px;
		padding: 3px;
		max-width: 100%;
	}
	.wrap {
		flex-wrap: wrap;
	}
	.segmented > :global(button) {
		flex: 1 0 auto;
		width: auto;
		white-space: nowrap;
		min-width: 80px;
		min-height: 40px;
		line-height: 20px;
		padding: 9px 13px;
		border: 0;
		background: transparent;
		color: inherit;
		font: inherit;
		font-size: 14px;
		border-radius: 4px;
		box-shadow: none;
	}
	.segmented > :global(button.active) {
		background: #29acf425;
		color: var(--stats-accent, #29acf4);
	}
	.segmented > :global(button:hover) {
		background: color-mix(in srgb, #29acf4 12%, transparent);
		color: var(--stats-accent, #29acf4);
	}
	.segmented > :global(button:focus-visible) {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
</style>
