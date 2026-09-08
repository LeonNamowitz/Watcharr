<script lang="ts">
	import Icon from "../Icon.svelte";
	import Modal from "../Modal.svelte";
	import { notify } from "../util/notify";

	interface Props {
		hours?: number;
		onChange: (newHours?: number) => Promise<boolean>;
	}

	let { hours, onChange }: Props = $props();

	let modalOpen = $state(false);
	let saving = $state(false);
	let input: HTMLInputElement | undefined = $state();
	let playtimeToDisplay = $derived(
		typeof hours === "number"
			? `${hours} ${hours === 1 ? "hour" : "hours"} played`
			: "Set hours played",
	);

	async function closeEditor() {
		if (saving) {
			return;
		}
		if (!input) {
			notify({
				type: "error",
				text: "Failed to find the playtime input. Please try again.",
			});
			return;
		}

		const value = input.value.trim();
		let newHours: number | undefined;
		if (value !== "") {
			newHours = Number(value);
			if (
				!input.reportValidity() ||
				!Number.isSafeInteger(newHours) ||
				newHours < 0
			) {
				notify({
					type: "error",
					text: "Playtime must be a non-negative whole number of hours.",
				});
				return;
			}
		}

		if (hours === newHours) {
			modalOpen = false;
			return;
		}

		saving = true;
		try {
			if (await onChange(newHours)) {
				modalOpen = false;
			}
		} finally {
			saving = false;
		}
	}

	async function clearPlaytime() {
		if (saving) {
			return;
		}
		if (typeof hours !== "number") {
			modalOpen = false;
			return;
		}

		saving = true;
		try {
			if (await onChange(undefined)) {
				modalOpen = false;
			}
		} finally {
			saving = false;
		}
	}
</script>

<button
	class:placeholdered={typeof hours !== "number"}
	class="plain playtime"
	onclick={() => (modalOpen = true)}
>
	<i><Icon i="pencil" wh={24} /></i>
	<p>{playtimeToDisplay}</p>
</button>

{#if modalOpen}
	<Modal
		title="Your Playtime"
		desc="View or modify your total time played"
		maxWidth="400px"
		onClose={closeEditor}
	>
		<label for="playtime-hours">Hours played</label>
		<input
			id="playtime-hours"
			name="playtime-hours"
			type="number"
			min="0"
			step="1"
			inputmode="numeric"
			value={hours ?? ""}
			placeholder="0"
			bind:this={input}
		/>
		<button
			class="clear"
			type="button"
			onclick={clearPlaytime}
			disabled={saving || typeof hours !== "number"}
		>
			Clear hours
		</button>
	</Modal>
{/if}

<style lang="scss">
	button.playtime {
		position: relative;
		width: 100%;
		text-align: start;
		padding: 7px 10px;
		border: 2px solid $text-color;
		border-radius: 5px;
		opacity: 0.5;
		transition: opacity 150ms ease-in-out;

		&:hover {
			opacity: 1;
		}

		i {
			display: flex;
			position: absolute;
			bottom: 4px;
			right: 5px;
			opacity: 0;
			transform: scale(0.5);
			transition:
				opacity 150ms ease-in-out,
				transform 150ms ease-in-out;
		}

		&:hover i {
			transform: scale(1);
			opacity: 1;
		}

		&.placeholdered {
			padding: 12px;
		}
	}

	label {
		display: block;
		margin: 10px 0 5px;
	}

	input {
		width: 100%;
	}

	button.clear {
		align-self: flex-start;
		margin-top: 15px;
	}
</style>
