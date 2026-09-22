<script lang="ts">
	import { useSortable } from "@dnd-kit-svelte/svelte/sortable";
	import { RestrictToVerticalAxis } from "@dnd-kit-svelte/svelte/modifiers";
	import { RestrictToElement } from "@dnd-kit/dom/modifiers";
	import type { Tag as TagT } from "@/types";
	import Tag from "./Tag.svelte";

	interface Props {
		tag: TagT;
		index: number;
		boundary?: HTMLElement;
	}

	let { tag, index, boundary }: Props = $props();
	const restrictToTagMenu = RestrictToElement.configure({
		element: () => boundary ?? null,
	});

	const { ref, sourceRef, targetRef, handleRef } = useSortable({
		id: () => tag.id,
		index: () => index,
		group: "tag-order",
		data: () => ({ tagId: tag.id }),
		modifiers: () => [RestrictToVerticalAxis, restrictToTagMenu],
	});
</script>

<div class="tag-row" {@attach ref} {@attach sourceRef} {@attach targetRef}>
	<Tag {tag} draggable dragHandleRef={handleRef} />
</div>

<style lang="scss">
	.tag-row {
		display: flex;
		align-items: center;
		gap: 5px;
		color: $text-color;
		touch-action: pan-y;
	}
</style>
