<script lang="ts">
	import { store } from "@/store.svelte";
	import Icon from "../Icon.svelte";
	import CreateTagModal from "./CreateTagModal.svelte";
	import type { Tag as TagT } from "@/types";
	import Tag from "./Tag.svelte";
	import DeleteTagModal from "./DeleteTagModal.svelte";
	import SortableTag from "./SortableTag.svelte";
	import { reorderTags } from "./api";
	import { DragDropProvider } from "@dnd-kit-svelte/svelte";
	import type {
		DragEndEvent as DragEndHandler,
		DragOverEvent as DragOverHandler,
	} from "@dnd-kit/dom";
	import Menu, { type MenuConfig } from "../Menu.svelte";

	interface Props {
		titleText?: string | undefined;
		onTagClick?: (tag: TagT, remove: boolean) => void | undefined;
		selectedTags?: TagT[] | undefined;
		/**
		 * When `showManageBtn` is true, a manage icon will appear at top
		 * of menu for the user to click. When toggled on, clicking a tag
		 * will trigger a deletion instead of `onTagClick()`.
		 */
		showManageBtn?: boolean;
		menuConfig?: MenuConfig;
	}

	const defaultMenuConfig = {
		width: "200px",
		right: "47px",
		arrowLeft: "78px",
	};

	let {
		titleText = undefined,
		onTagClick = undefined!,
		selectedTags = undefined,
		showManageBtn = false,
		menuConfig = {},
	}: Props = $props();

	let allTags = $derived(store.tags);
	let editableTags = $state<TagT[]>([]);

	let tagModalOpen = $state(false);
	let inManageMode = $state(false);
	let inOrderEditMode = $state(false);
	let tagToDelete: TagT | undefined = $state(undefined);

	function deleteTag(t: TagT) {
		// This will show the DeleteTagModal (look below).
		tagToDelete = t;
	}

	function toggleOrderEdit() {
		inManageMode = false;
		editableTags = [...allTags];
		inOrderEditMode = true;
	}

	async function saveTagOrder() {
		if (!(await reorderTags(editableTags.map((tag) => tag.id)))) {
			return;
		}
		store.tags = editableTags;
		inOrderEditMode = false;
	}

	let lastDragTargetId: string | number | null = null;

	function moveEditableTag(
		sourceId: string | number,
		targetId: string | number,
	) {
		if (sourceId === targetId) return;
		const from = editableTags.findIndex((tag) => tag.id === sourceId);
		const to = editableTags.findIndex((tag) => tag.id === targetId);
		if (from < 0 || to < 0) return;

		const next = [...editableTags];
		const [moved] = next.splice(from, 1);
		next.splice(to, 0, moved);
		editableTags = next;
	}

	function handleDragStart() {
		lastDragTargetId = null;
	}

	function handleDragOver(event: Parameters<DragOverHandler>[0]) {
		const sourceId = event.operation.source?.id;
		const targetId = event.operation.target?.id;
		if (sourceId == null || targetId == null || targetId === lastDragTargetId)
			return;
		lastDragTargetId = targetId;
		moveEditableTag(sourceId, targetId);
	}

	function handleDragEnd(event: Parameters<DragEndHandler>[0]) {
		if (!event.canceled) {
			const sourceId = event.operation.source?.id;
			const targetId = event.operation.target?.id;
			if (
				sourceId != null &&
				targetId != null &&
				targetId !== lastDragTargetId
			) {
				moveEditableTag(sourceId, targetId);
			}
		}
		lastDragTargetId = null;
	}
</script>

<Menu conf={Object.assign(defaultMenuConfig, menuConfig)}>
	<div class="title">
		<h4 class="norm sm-caps">{titleText ? titleText : "my tags"}</h4>
		{#if showManageBtn}
			<button
				class={["plain", inManageMode ? "manage-on" : ""].join(" ")}
				disabled={inOrderEditMode}
				onclick={() => (inManageMode = !inManageMode)}
				aria-label="Delete tags"
			>
				<Icon i="trash" wh={18} />
			</button>
			<button
				class={["plain", inOrderEditMode ? "manage-on" : ""].join(" ")}
				disabled={inManageMode}
				onclick={() => (inOrderEditMode ? saveTagOrder() : toggleOrderEdit())}
				aria-label={inOrderEditMode ? "Save tag order" : "Edit tag order"}
			>
				<Icon i={inOrderEditMode ? "check" : "pencil"} wh={18} />
			</button>
		{/if}
		<button
			class="plain"
			disabled={inOrderEditMode}
			onclick={() => (tagModalOpen = !tagModalOpen)}
			aria-label="Add tag"
		>
			<Icon i="add" wh={22} />
		</button>
	</div>
	{#if allTags && allTags.length > 0}
		{#if inManageMode}
			<strong style="font-size: 12px; margin-bottom: 10px;"
				>Click a tag to delete it.</strong
			>
		{/if}
		<DragDropProvider
			onDragStart={handleDragStart}
			onDragOver={handleDragOver}
			onDragEnd={handleDragEnd}
		>
			<div class="list">
				{#each inOrderEditMode ? editableTags : allTags as t, i (t.id)}
					{#if inOrderEditMode}
						<SortableTag tag={t} index={i} />
					{:else}
						{@const isSelected = selectedTags
							? selectedTags.find((tag) => tag.id === t.id)
								? true
								: false
							: false}
						<div>
							<Tag
								tag={t}
								onClick={() =>
									inManageMode ? deleteTag(t) : onTagClick(t, isSelected)}
							/>
							{#if isSelected}
								<Icon i="check" wh={18} />
							{/if}
						</div>
					{/if}
				{/each}
			</div>
		</DragDropProvider>
	{:else}
		<span style="margin-top: 0;">You have no tags yet!</span>
	{/if}
</Menu>

{#if tagModalOpen}
	<CreateTagModal onClose={() => (tagModalOpen = false)} />
{/if}

{#if tagToDelete}
	<DeleteTagModal tag={tagToDelete} onClose={() => (tagToDelete = undefined)} />
{/if}

<style lang="scss">
	h4 {
		color: $text-color;

		&:not(:first-of-type) {
			margin-top: 8px;
		}
	}

	.title {
		display: flex;
		flex-flow: row;
		align-items: center;
		margin-bottom: 8px;
		gap: 5px;

		button.plain {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 28px;
			height: 26px;
			padding: 2px 3px;
			border-radius: 8px;

			&.manage-on {
				color: #f3555a;
				background-color: $text-color;
			}

			&:first-of-type {
				margin-left: auto;
			}
		}
	}

	.list {
		display: flex;
		flex-flow: column;
		gap: 5px;

		& > div {
			display: flex;
			align-items: center;
			gap: 5px;
			color: $text-color;

			:global(svg) {
				min-width: 18px;
			}
		}
	}
</style>
