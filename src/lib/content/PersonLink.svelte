<script lang="ts">
	import { goto } from "$app/navigation";

	export let id: number | undefined;
	export let name: string | undefined;
	export let role: string | undefined;
	export let className: string | undefined = "";

	const link = id ? `/person/${id}` : undefined;

	function handleClick(e: MouseEvent) {
		if (!link) return;
		e.preventDefault();
		goto(link);
	}
</script>

<div class={className}>
	<span>
		{#if link}
			<a
				data-sveltekit-preload-data="tap"
				href={link}
				on:click={handleClick}
				aria-label={`View ${name} profile`}
			>
				{name}
			</a>
		{:else}
			{name}
		{/if}
	</span>
	{#if role}
		<span class="role">{role}</span>
	{/if}
</div>

<style lang="scss">
	:global(.person-link) {
		display: flex;
		flex-flow: column;

		span:first-child {
			font-weight: bold;
		}

		.role {
			font-size: 0.9em;
			color: rgba(255, 255, 255, 0.85);
		}
	}
</style>
