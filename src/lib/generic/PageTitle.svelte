<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		title: string;
		row?: boolean;
		children: Snippet;
		actions?: Snippet;
	}

	let { title, row = true, children, actions }: Props = $props();
</script>

<div class="results-filters-header" class:row>
	{#if actions}
		<div class="title-row">
			<h2>{title}</h2>
			<div class="actions">{@render actions()}</div>
		</div>
	{:else}
		<h2>{title}</h2>
	{/if}
	{@render children()}
</div>

<style lang="scss">
	.results-filters-header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 10px;
		margin: 0 15px;

		&.row {
			flex-flow: row wrap;
		}
	}

	.title-row {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
	}

	.actions {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-left: auto;
	}
</style>
