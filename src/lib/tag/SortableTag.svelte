<script lang="ts">
	import { useSortable } from "@dnd-kit-svelte/svelte/sortable";
	import type { Tag as TagT } from "@/types";
	import Tag from "./Tag.svelte";

	interface Props {
		tag: TagT;
		index: number;
	}

	let { tag, index }: Props = $props();

	const { ref, sourceRef, targetRef, handleRef } = useSortable({
		id: () => tag.id,
		index: () => index,
		group: "tag-order",
		data: () => ({ tagId: tag.id }),
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
