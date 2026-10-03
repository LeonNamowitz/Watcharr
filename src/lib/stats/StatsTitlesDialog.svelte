<script lang="ts">
	import StatsExpansion from "./StatsExpansion.svelte";
	import { resolve } from "$app/paths";
	import { onMount, untrack } from "svelte";
	import StatsPosters from "./StatsPosters.svelte";
	import type { StatsMediaCard } from "./types";
	import type { RatingSettings } from "@/lib/rating/helpers";
	let {
		label,
		items,
		period,
		owner,
		settings,
		personId,
		description = "watched",
		expansionRows,
		onClose,
		initialState,
	}: {
		label: string;
		items: StatsMediaCard[];
		period: string;
		owner?: { id: string; username: string };
		settings: RatingSettings;
		personId?: number;
		description?: string;
		expansionRows: number;
		onClose: () => void;
		initialState?: {
			count: number;
			scrollTop: number;
			scrollX: number;
			scrollY: number;
		};
	} = $props();
	let dialog: HTMLDialogElement;
	let count = $state(untrack(() => initialState?.count ?? 5));
	let pageScroll = { scrollX: 0, scrollY: 0 };
	export function capture() {
		return { count, scrollTop: dialog.scrollTop, ...pageScroll };
	}
	let columns = $state(5);
	onMount(() => {
		const { body, documentElement: root } = document;
		const { scrollX, scrollY } = initialState ?? window;
		pageScroll = { scrollX, scrollY };
		const previousStyles = {
			position: body.style.position,
			top: body.style.top,
			left: body.style.left,
			width: body.style.width,
			overflow: body.style.overflow,
			paddingRight: body.style.paddingRight,
		};
		const rootOverflow = root.style.overflow;
		const scrollbarWidth = window.innerWidth - root.clientWidth;
		const paddingRight = parseFloat(getComputedStyle(body).paddingRight) || 0;
		Object.assign(body.style, {
			position: "fixed",
			top: `${-scrollY}px`,
			left: `${-scrollX}px`,
			width: "100%",
			overflow: "hidden",
			paddingRight: `${paddingRight + scrollbarWidth}px`,
		});
		root.style.overflow = "hidden";
		dialog.showModal();
		dialog.scrollTop = initialState?.scrollTop ?? 0;
		const postersGrid = dialog.querySelector<HTMLElement>(".posters");
		const updateColumns = () => {
			if (postersGrid)
				columns =
					getComputedStyle(postersGrid).gridTemplateColumns.split(" ").length;
		};
		updateColumns();
		const observer = postersGrid
			? new ResizeObserver(updateColumns)
			: undefined;
		if (postersGrid) observer?.observe(postersGrid);
		return () => {
			observer?.disconnect();
			if (dialog.open) dialog.close();
			Object.assign(body.style, previousStyles);
			root.style.overflow = rootOverflow;
			window.scrollTo({ left: scrollX, top: scrollY, behavior: "instant" });
		};
	});
	function backdrop(event: MouseEvent) {
		if (event.target !== dialog) return;
		const bounds = dialog.getBoundingClientRect();
		if (
			event.clientX < bounds.left ||
			event.clientX > bounds.right ||
			event.clientY < bounds.top ||
			event.clientY > bounds.bottom
		)
			dialog.close();
	}
</script>

<dialog
	bind:this={dialog}
	aria-labelledby="stats-dialog-title"
	onclose={onClose}
	onclick={backdrop}
>
	<header>
		<div>
			<p>
				{period} · {items.length.toLocaleString()}
				{items.some((item) => item.episodeNumber !== undefined)
					? "items"
					: items.length === 1
						? "title"
						: "titles"}
				{description}
			</p>
			<h2 class="norm" id="stats-dialog-title">{label}</h2>
		</div>
		<button
			class="plain close"
			aria-label="Close title list"
			onclick={() => dialog.close()}>×</button
		>
	</header>
	<div class="body">
		<StatsPosters items={items.slice(0, count)} {owner} {settings} />
		<StatsExpansion
			{count}
			step={expansionRows * columns}
			total={items.length}
			onChange={(value) => (count = value)}
		/>
	</div>
	{#if personId}<footer>
			<a
				href={owner
					? resolve("/(public)/lists/[id]/[username]/person/[personId]", {
							...owner,
							personId: String(personId),
						})
					: resolve("/(app)/person/[id]", { id: String(personId) })}
				>View person page →</a
			>
		</footer>{/if}
</dialog>

<style>
	dialog {
		width: min(900px, calc(100vw - 32px));
		max-width: calc(100vw - 32px);
		max-height: calc(100dvh - 32px);
		padding: 0;
		margin: auto;
		box-sizing: border-box;
		border: 1px solid var(--stats-border);
		border-radius: 14px;
		color: var(--stats-text);
		background: var(--stats-surface);
		overflow: auto;
		overscroll-behavior: contain;
		box-shadow: 0 16px 70px #0005;
	}
	dialog::backdrop {
		background: #0008;
		backdrop-filter: blur(3px);
	}
	header {
		display: flex;
		align-items: start;
		justify-content: space-between;
		gap: 16px;
		padding: 22px;
		position: sticky;
		top: 0;
		z-index: 1;
		background: var(--stats-surface);
		border-bottom: 1px solid var(--stats-border);
	}
	h2 {
		font-size: 24px;
		line-height: 1.3;
		overflow-wrap: anywhere;
		margin: 0;
	}
	p {
		font-size: 13px;
		color: var(--stats-muted);
		margin: 0 0 6px;
	}
	.close {
		flex: none;
		font-size: 30px;
		line-height: 1;
		width: 36px;
		height: 36px;
		border-radius: 8px;
		color: var(--stats-muted);
	}
	.body {
		padding: 22px;
	}
	a {
		color: var(--stats-accent);
		font-size: 14px;
	}
	footer {
		padding: 16px 22px;
		border-top: 1px solid var(--stats-border);
	}
	.close:hover {
		background: var(--stats-border);
		color: var(--stats-accent);
	}
	a:hover {
		text-decoration: underline;
	}
	button:focus-visible,
	a:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	@media (max-width: 520px) {
		header,
		.body {
			padding: 16px;
		}
		h2 {
			font-size: 21px;
		}
	}
</style>
