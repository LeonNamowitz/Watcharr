<script lang="ts">
	import type { Tag } from "@/types";
	import type { Attachment } from "svelte/attachments";

	interface Props {
		tag: Tag;
		onClick?: () => void | undefined;
		dragHandleRef?: Attachment<Element>;
		draggable?: boolean;
	}

	let {
		tag,
		onClick = undefined!,
		dragHandleRef = () => {},
		draggable = false,
	}: Props = $props();
</script>

<button
	class="plain"
	class:drag-handle={draggable}
	style:color={tag.color}
	style:background={tag.bgColor}
	{@attach dragHandleRef}
	onclick={() => {
		if (typeof onClick === "function") {
			onClick();
		}
	}}
>
	{tag.name}
</button>

<style lang="scss">
	button {
		text-transform: capitalize;
		position: relative;
		width: max-content;
		border-radius: 8px;
		padding: 5px 8px;
		text-wrap: wrap;
		word-break: break-word;
		transition: opacity 150ms ease-in-out;

		&:hover {
			opacity: 0.8;
		}

		&.drag-handle {
			cursor: grab;
			touch-action: none;

			&:active {
				cursor: grabbing;
			}
		}
	}
</style>
