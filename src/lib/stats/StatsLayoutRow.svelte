<script lang="ts">
	import { useSortable } from "@dnd-kit-svelte/svelte/sortable";
	import { RestrictToVerticalAxis } from "@dnd-kit-svelte/svelte/modifiers";
	import { RestrictToElement } from "@dnd-kit/dom/modifiers";
	import type { StatsSectionId } from "./sectionOrder";
	let {
		id,
		label,
		note,
		index,
		total,
		boundary,
		onMove,
	}: {
		id: StatsSectionId;
		label: string;
		note?: string;
		index: number;
		total: number;
		boundary?: HTMLElement;
		onMove: (id: StatsSectionId, index: number) => void;
	} = $props();
	const { ref, sourceRef, targetRef, handleRef } = useSortable({
		id: () => id,
		index: () => index,
		group: "stats-section-order",
		modifiers: () => [
			RestrictToVerticalAxis,
			RestrictToElement.configure({ element: () => boundary ?? null }),
		],
	});
</script>

<li
	data-stats-layout-id={id}
	{@attach ref}
	{@attach sourceRef}
	{@attach targetRef}
>
	<button
		type="button"
		class="plain handle"
		{@attach handleRef}
		aria-label={`Drag ${label}`}
		title="Drag to reorder">⠿</button
	>
	<span class="label"
		>{label}{#if note}<small>{note}</small>{/if}</span
	>
	<button
		type="button"
		class="plain"
		disabled={index === 0}
		aria-label={`Move ${label} up`}
		onclick={() => onMove(id, index - 1)}>↑</button
	>
	<button
		type="button"
		class="plain"
		disabled={index === total - 1}
		aria-label={`Move ${label} down`}
		onclick={() => onMove(id, index + 1)}>↓</button
	>
</li>

<style>
	li {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px;
		border: 1px solid var(--stats-border);
		border-radius: 7px;
		background: var(--stats-surface);
		color: var(--stats-text);
		touch-action: pan-y;
	}
	.label {
		flex: 1;
		min-width: 0;
		font-size: 14px;
	}
	small {
		display: block;
		font-size: 12px;
		color: var(--stats-muted);
	}
	button {
		min-width: 40px;
		min-height: 40px;
		color: inherit;
		font-size: 22px;
		border-radius: 4px;
	}
	.handle {
		touch-action: none;
		cursor: grab;
	}
	button:disabled {
		opacity: 0.3;
		cursor: default;
	}
	button:not(:disabled):hover {
		background: color-mix(in srgb, var(--stats-accent) 12%, transparent);
	}
	button:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 2px;
	}
</style>
