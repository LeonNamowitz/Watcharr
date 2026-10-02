<script lang="ts">
	import { BarChart, LineChart, PieChart } from "layerchart/svg";
	import { Tooltip } from "layerchart/svg";
	import type { ChartPoint } from "./types";
	let {
		title,
		points,
		kind = "bar",
		color = "#39cfa2",
		height = 180,
	}: {
		title: string;
		points: ChartPoint[];
		kind?: "bar" | "horizontal" | "line" | "pie";
		color?: string;
		height?: number;
	} = $props();
	let selected = $state<ChartPoint>();
	const colors = ["#39cfa2", "#6495ed", "#f5b85a", "#f47983", "#b19bea"];
	const maximum = $derived(Math.max(1, ...points.map((p) => p.value ?? 0)));
	const minimum = $derived(
		Math.max(
			0,
			Math.floor(
				Math.min(
					...points
						.filter((p) => p.value !== null)
						.map((p) => p.value as number),
				) - 0.5,
			),
		),
	);
	const lineMaximum = $derived(Math.min(10, Math.ceil(maximum + 0.5)));
	function valueLabel(p: ChartPoint) {
		return p.value === null
			? "Unrated"
			: p.value.toLocaleString(undefined, { maximumFractionDigits: 2 });
	}
	const series = $derived([
		{ key: "value", label: title, value: "value", color },
	]);
	const common = $derived({
		height,
		motion: "none" as const,
		rule: false,
		legend: false,
		grid: false,
		highlight: false,
		onTooltipClick: (_e: MouseEvent, { data }: { data: ChartPoint }) =>
			(selected = data),
		padding: { left: 32, right: 10, top: 10, bottom: 26 },
		props: {
			xAxis: { tickSpacing: 70, tickOcclusion: true, tickMarks: false },
			yAxis: { ticks: 3, tickMarks: false },
			bars: { radius: 2 },
		},
	});
</script>

{#snippet tip({ context }: { context: { tooltip: { data?: ChartPoint } } })}
	<Tooltip.Root
		contained="window"
		motion="none"
		fadeDuration={0}
		variant="none"
	>
		{#if context.tooltip.data}
			{@const p = context.tooltip.data}
			<div class="tooltip">
				<strong>{p.label}</strong><span>{valueLabel(p)}</span>{#if p.detail}<p>
						{p.detail}
					</p>{/if}
			</div>
		{/if}
	</Tooltip.Root>
{/snippet}

<div class="chart" role="group" aria-label={title}>
	{#if points.length && points.some((p) => (p.value ?? 0) > 0)}
		{#if kind === "pie"}
			<PieChart
				data={points.map((p, i) => ({
					...p,
					color: colors[i % colors.length],
				}))}
				key="label"
				label="label"
				value="value"
				c="color"
				{height}
				motion="none"
				legend={false}
				tooltip={tip}
				props={{ arc: { stroke: "transparent" } }}
				onTooltipClick={(_e, { data }) => (selected = data)}
			/>
		{:else if kind === "horizontal"}
			<BarChart
				{...common}
				data={points}
				orientation="horizontal"
				x="value"
				y="label"
				xDomain={[0, maximum]}
				yReverse={true}
				{series}
				axis="x"
				padding={{ left: 0, right: 10, top: 4, bottom: 26 }}
				tooltip={tip}
				onBarClick={(_e, { data }) => (selected = data)}
			/>
		{:else if kind === "line"}
			<LineChart
				{...common}
				data={points}
				x="label"
				y="value"
				yDomain={[minimum, lineMaximum]}
				{series}
				points={true}
				tooltip={tip}
				onTooltipClick={(_e, { data }) => (selected = data)}
			/>
		{:else}
			<BarChart
				{...common}
				data={points}
				x="label"
				y="value"
				yDomain={[0, maximum]}
				bandPadding={0.22}
				{series}
				tooltip={tip}
				onBarClick={(_e, { data }) => (selected = data)}
			/>
		{/if}
	{:else}<p class="empty">No recorded data for this chart.</p>{/if}
	{#if selected}<div class="selection" aria-live="polite">
			<strong>{selected.label}: {valueLabel(selected)}</strong
			>{#if selected.detail}<span>{selected.detail}</span>{/if}<button
				class="plain"
				onclick={() => (selected = undefined)}
				aria-label="Dismiss chart selection">×</button
			>
		</div>{/if}
	<details>
		<summary>Explore chart data</summary>
		<div class="data-list">
			{#each points as p (p.label)}<button
					class="plain"
					onclick={() => (selected = p)}
					><span>{p.label}</span><strong>{valueLabel(p)}</strong></button
				>{/each}
		</div>
	</details>
</div>

<style>
	.chart {
		min-width: 0;
		width: 100%;
		--color-primary: #39cfa2;
	}
	.chart :global(svg) {
		font-family: inherit;
	}
	.chart :global(svg.lc-layout-svg) {
		overflow: hidden;
	}
	.chart :global(svg text) {
		fill: currentColor;
		font-size: 10px;
		opacity: 0.7;
	}
	.tooltip {
		background: var(--bg-color, #20242b);
		color: var(--text-color, #f4f5f7);
		border: 1px solid #7775;
		border-radius: 8px;
		padding: 10px 12px;
		width: max-content;
		max-width: min(280px, calc(100vw - 32px));
		box-shadow: 0 6px 24px #0005;
		overflow-wrap: anywhere;
	}
	.tooltip span {
		margin-left: 12px;
		color: #39cfa2;
	}
	.tooltip p {
		margin: 6px 0 0;
		font-size: 12px;
		max-height: 180px;
		overflow: auto;
	}
	summary {
		cursor: pointer;
		font-size: 11px;
		opacity: 0.6;
		margin-top: 8px;
	}
	.data-list {
		max-height: 200px;
		overflow: auto;
		margin-top: 10px;
	}
	.data-list button {
		display: flex;
		justify-content: space-between;
		gap: 8px;
		width: 100%;
		padding: 5px 0;
		text-align: left;
		font-size: 12px;
	}
	.selection {
		position: relative;
		padding: 10px 32px 10px 10px;
		background: #6495ed15;
		border-radius: 6px;
		display: grid;
		gap: 4px;
		font-size: 12px;
		overflow-wrap: anywhere;
	}
	.selection button {
		position: absolute;
		right: 8px;
		top: 8px;
	}
	.empty {
		opacity: 0.6;
		padding: 24px 0;
		font-size: 13px;
	}
</style>
