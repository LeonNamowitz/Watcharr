<script lang="ts">
	import Icon from "../Icon.svelte";

	interface Props {
		active: boolean;
		disabled?: boolean;
		onclick: () => void;
	}

	let { active, disabled = false, onclick }: Props = $props();
</script>

<button class="plain" data-active={active} {disabled} type="button" {onclick}>
	<Icon i="reel" wh={18} /> All
</button>

<style lang="scss">
	button {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 12px;
		border-radius: 8px;
		color: $text-color;
		fill: $text-color;
		font-size: 14px;
		transition:
			background-color 150ms ease,
			color 150ms ease,
			outline 150ms ease;

		&:hover,
		&[data-active="true"] {
			color: $bg-color;
			fill: $bg-color;
			background-color: $accent-color-hover;
		}

		&[data-active="true"] {
			outline: 3px solid $accent-color;
		}
	}
</style>
