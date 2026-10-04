<script lang="ts">
	import { tick } from "svelte";
	let {
		count,
		total,
		onChange,
		step = 5,
		separateCollapse = true,
		expandAll = false,
		collapseTo = 5,
		prominent = false,
	}: {
		count: number;
		total: number;
		step?: number;
		separateCollapse?: boolean;
		expandAll?: boolean;
		collapseTo?: number;
		prominent?: boolean;
		onChange: (count: number) => void;
	} = $props();
	let expansionTop: number | undefined;
	async function expand(event: MouseEvent) {
		const button = event.currentTarget as HTMLElement;
		const container = button.parentElement;
		const focused = document.activeElement === button;
		if (count <= collapseTo) {
			expansionTop = (
				event.currentTarget as HTMLElement
			).getBoundingClientRect().top;
		}
		onChange(expandAll ? total : Math.min(total, count + step));
		await tick();
		if (focused && !button.isConnected) {
			container
				?.querySelector<HTMLButtonElement>("button")
				?.focus({ preventScroll: true });
		}
	}
	async function collapse(event: MouseEvent) {
		const button = event.currentTarget as HTMLElement;
		const container = button.parentElement;
		const focused = document.activeElement === button;
		const top = expansionTop ?? window.innerHeight - button.offsetHeight - 24;
		onChange(collapseTo);
		await tick();
		const anchor = button.isConnected
			? button
			: container?.querySelector("button");
		if (focused && !button.isConnected) {
			anchor?.focus({ preventScroll: true });
		}
		if (anchor?.isConnected) {
			window.scrollBy({
				top: anchor.getBoundingClientRect().top - top,
				behavior: "instant",
			});
		}
		expansionTop = undefined;
	}
</script>

{#if separateCollapse}
	{#if count < total || count > collapseTo}
		<div class="expansion">
			{#if count < total}<button
					class="plain more"
					class:prominent
					onclick={expand}>Show more ({total - count} remaining)</button
				>{/if}
			{#if count > collapseTo}<button
					class="plain more less expanded"
					class:prominent
					onclick={collapse}>Show less</button
				>{/if}
		</div>
	{/if}
{:else if count < total || (total > collapseTo && count > collapseTo)}
	<button
		class="plain more"
		class:prominent
		class:expanded={count >= total}
		onclick={(event) => (count < total ? expand(event) : collapse(event))}
	>
		{#if count < total}Show more ({total - count} remaining){:else}Show less{/if}
	</button>
{/if}

<style>
	.expansion {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20px;
	}
	.less {
		margin-left: auto;
	}
	.more {
		margin-top: 18px;
		font-size: 14px;
		color: var(--stats-accent);
	}
	.more:hover {
		text-decoration: underline;
	}
	.more:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 4px;
	}
	.more.prominent {
		margin-top: 0;
		padding: 8px 12px;
		border: 1px solid
			color-mix(in srgb, var(--stats-accent) 45%, var(--stats-border));
		border-radius: 7px;
		background: color-mix(
			in srgb,
			var(--stats-accent) 9%,
			var(--stats-surface)
		);
		font-weight: 600;
	}
	.more.prominent::after {
		content: " ↓";
	}
	.more.prominent.expanded::after {
		content: " ↑";
	}
	.more.prominent:hover {
		background: color-mix(
			in srgb,
			var(--stats-accent) 15%,
			var(--stats-surface)
		);
		text-decoration: none;
	}
</style>
