<script lang="ts">
	import { tick } from "svelte";
	import {
		DragDropProvider,
		KeyboardSensor,
		PointerSensor,
	} from "@dnd-kit-svelte/svelte";
	import {
		PointerActivationConstraints,
		type DragEndEvent,
	} from "@dnd-kit/dom";
	import { isSortable } from "@dnd-kit/dom/sortable";
	import Error from "@/lib/Error.svelte";
	import StatsLayoutRow from "./StatsLayoutRow.svelte";
	import {
		normalizeSectionOrder,
		sectionLabel,
		sectionAvailability,
		type StatsSectionId,
	} from "./sectionOrder";
	import type { StatsResponse } from "./types";
	let {
		data,
		disabled = false,
		editing = $bindable(false),
		onSave,
	}: {
		data: StatsResponse;
		disabled?: boolean;
		editing?: boolean;
		onSave: (order: StatsSectionId[]) => Promise<void>;
	} = $props();
	let draft = $state<StatsSectionId[]>([]);
	let saving = $state(false);
	let error = $state<unknown>();
	let announcement = $state("");
	let list = $state<HTMLOListElement>();
	let editButton: HTMLButtonElement;
	let heading = $state<HTMLHeadingElement>();
	const sensors = [
		PointerSensor.configure({
			activationConstraints: [
				new PointerActivationConstraints.Delay({ value: 150, tolerance: 8 }),
			],
		}),
		KeyboardSensor,
	];
	async function start() {
		draft = normalizeSectionOrder(data.media, data.sectionOrder);
		error = undefined;
		announcement = "";
		editing = true;
		await tick();
		heading?.focus({ preventScroll: true });
	}
	async function close() {
		editing = false;
		error = undefined;
		await tick();
		editButton.focus({ preventScroll: true });
	}
	async function move(id: StatsSectionId, to: number) {
		const from = draft.indexOf(id);
		if (from < 0 || to < 0 || to >= draft.length || from === to) return;
		const focused = document.activeElement;
		const next = [...draft];
		next.splice(from, 1);
		next.splice(to, 0, id);
		draft = next;
		announcement = `${sectionLabel(id, data)} moved to position ${to + 1} of ${draft.length}.`;
		await tick();
		if (focused instanceof HTMLButtonElement && focused.disabled) {
			list
				?.querySelector<HTMLButtonElement>(
					`[data-stats-layout-id="${id}"] .handle`,
				)
				?.focus();
		}
	}
	function dragEnd(event: Parameters<DragEndEvent>[0]) {
		if (event.canceled) return;
		const source = event.operation.source;
		if (source && isSortable(source))
			move(source.id as StatsSectionId, source.sortable.index);
	}
	async function save() {
		saving = true;
		error = undefined;
		try {
			await onSave([...draft]);
			await close();
		} catch (err) {
			error = err;
		} finally {
			saving = false;
		}
	}
</script>

<div class="layout-editor">
	<button
		bind:this={editButton}
		type="button"
		class="layout-button"
		disabled={disabled || editing}
		onclick={start}>Edit layout</button
	>
	{#if editing}
		<div class="editor-panel" aria-labelledby="stats-layout-heading">
			<h2
				bind:this={heading}
				id="stats-layout-heading"
				tabindex="-1"
				class="norm"
			>
				Edit {data.media === "game"
					? "Games"
					: data.media === "tv"
						? "TV"
						: "Movies"} layout
			</h2>
			<p>
				Drag sections or use the arrows. Saved order applies to all periods and
				your public stats page.
			</p>
			<p class="instructions">
				For keyboard dragging, focus a handle, press Space, use the arrow keys,
				then press Space to drop or Escape to cancel.
			</p>
			<div inert={saving} aria-busy={saving}>
				<DragDropProvider {sensors} onDragEnd={dragEnd}>
					<ol bind:this={list}>
						{#each draft as id, index (id)}
							<StatsLayoutRow
								{id}
								{index}
								total={draft.length}
								boundary={list}
								label={sectionLabel(id, data)}
								note={sectionAvailability(id, data)}
								onMove={move}
							/>
						{/each}
					</ol>
				</DragDropProvider>
			</div>
			<span class="sr-only" aria-live="polite">{announcement}</span>
			{#if error}<div role="alert">
					<Error
						{error}
						pretty="Unable to save your layout. Try saving again."
					/>
				</div>{/if}
			<div class="actions">
				<button
					type="button"
					class="layout-button primary"
					disabled={saving}
					onclick={save}>{saving ? "Saving…" : "Save"}</button
				>
				<button
					type="button"
					class="layout-button"
					disabled={saving}
					onclick={close}>Cancel</button
				>
				<button
					type="button"
					class="layout-button"
					disabled={saving}
					onclick={() => {
						draft = normalizeSectionOrder(data.media);
						announcement = "Default order restored. Save to apply it.";
					}}>Reset to default</button
				>
			</div>
		</div>
	{/if}
</div>

<style>
	.layout-editor {
		margin-bottom: 20px;
	}
	.layout-editor > button {
		width: 100%;
	}
	.layout-button {
		width: auto;
		padding: 9px 14px;
		min-height: 40px;
		border: 1px solid var(--stats-border);
		border-radius: 7px;
		background: var(--stats-surface);
		color: var(--stats-text);
		box-shadow: none;
		font: inherit;
		font-size: 14px;
	}
	.layout-button:not(:disabled):hover {
		border-color: var(--stats-accent);
	}
	.layout-button:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.layout-button:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.primary {
		color: var(--stats-accent);
		font-weight: 600;
	}
	.editor-panel {
		max-width: 640px;
		margin-top: 12px;
		padding: 18px;
		background: var(--stats-surface);
		border: 1px solid var(--stats-border);
		border-radius: 10px;
	}
	h2 {
		font-size: 20px;
	}
	p {
		font-size: 14px;
		color: var(--stats-muted);
		margin: 10px 0;
	}
	.instructions {
		font-size: 12px;
	}
	ol {
		display: grid;
		gap: 6px;
		padding: 0;
		margin: 16px 0;
		list-style: none;
		max-height: 420px;
		overflow-y: auto;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-top: 14px;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
</style>
