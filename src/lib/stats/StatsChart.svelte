<script lang="ts">
	import { Bar, BarChart, LineChart, PieChart } from "layerchart/svg";
	import { Tooltip } from "layerchart/svg";
	import { decimal } from "./format";
	import { getContext, type ComponentProps } from "svelte";
	import type { ChartPoint, StatsChartSelection } from "./types";
	let {
		title,
		points,
		kind = "bar",
		color = "#29acf4",
		height = 180,
	}: {
		title: string;
		points: ChartPoint[];
		kind?: "bar" | "horizontal" | "line" | "pie";
		color?: string;
		height?: number;
	} = $props();
	let selected = $state<ChartPoint>();
	const chartId = Symbol();
	const chartSelection = getContext<StatsChartSelection>(
		"watcharr:stats:selection",
	);
	function selectPoint(point: ChartPoint) {
		selected = point;
		chartSelection?.select(chartId);
	}
	type ChartContext = NonNullable<ComponentProps<typeof BarChart>["context"]>;
	const colors = ["#29acf4", "#6495ed", "#f5b85a", "#f47983", "#b19bea"];
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
			: kind === "line"
				? decimal(p.value)
				: p.value.toLocaleString(undefined, { maximumFractionDigits: 0 });
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
			selectPoint(data),
		padding: { left: 32, right: 10, top: 10, bottom: 26 },
		props: {
			xAxis: { tickSpacing: 70, tickOcclusion: true, tickMarks: false },
			yAxis: { ticks: 3, tickMarks: false },
			bars: {
				radius: 3,
				stroke: "transparent",
				strokeWidth: 0,
			},
		},
	});
</script>

{#snippet barMarks({ context }: { context: ChartContext })}
	{#each points as p (p.label)}<Bar
			data={p}
			seriesKey="value"
			radius={3}
			stroke="transparent"
			strokeWidth={0}
			fill={context.tooltip.data?.label === p.label
				? `color-mix(in srgb,${color} 78%,white)`
				: color}
			onclick={() => selectPoint(p)}
		/>{/each}
{/snippet}

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
				<strong>{p.tooltipLabel ?? p.label}</strong><span>{valueLabel(p)}</span
				>{#if p.detail}<p>
						{p.detail}
					</p>{/if}
			</div>
		{/if}
	</Tooltip.Root>
{/snippet}

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") selected = undefined;
	}}
/>

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
				onTooltipClick={(_e, { data }) => selectPoint(data)}
			/>
		{:else if kind === "horizontal"}
			<BarChart
				marks={barMarks}
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
				onBarClick={(_e, { data }) => selectPoint(data)}
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
				highlight={{ points: { r: 5, fill: color, stroke: "transparent" } }}
				tooltip={tip}
				onTooltipClick={(_e, { data }) => selectPoint(data)}
			/>
		{:else}
			<BarChart
				marks={barMarks}
				{...common}
				data={points}
				x="label"
				y="value"
				yDomain={[0, maximum]}
				bandPadding={0.22}
				{series}
				tooltip={tip}
				onBarClick={(_e, { data }) => selectPoint(data)}
			/>
		{/if}
	{:else}<p class="empty">No recorded data for this chart.</p>{/if}
	{#if selected && (!chartSelection || chartSelection.current() === chartId)}<div
			class="selection"
			role="region"
			aria-label="Selected chart data"
			aria-live="polite"
		>
			<strong
				>{selected.tooltipLabel ?? selected.label}: {valueLabel(
					selected,
				)}</strong
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
					onclick={() => selectPoint(p)}
					><span>{p.label}</span><strong>{valueLabel(p)}</strong></button
				>{/each}
		</div>
	</details>
</div>

<style>
	.chart {
		min-width: 0;
		width: 100%;
		--color-primary: #29acf4;
	}
	.chart :global(svg) {
		font-family: inherit;
	}
	.chart :global(svg.lc-layout-svg) {
		overflow: hidden;
	}
	.chart :global(svg text) {
		fill: var(--stats-muted);
		font-size: 12px;
	}
	.tooltip {
		background: var(--stats-surface, var(--bg-color));
		color: var(--stats-text, var(--text-color));
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
		color: var(--stats-accent, #29acf4);
	}
	.tooltip p {
		margin: 6px 0 0;
		font-size: 14px;
		max-height: 180px;
		overflow: auto;
	}
	summary {
		cursor: pointer;
		font-size: 13px;
		color: var(--stats-muted);
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
		font-size: 14px;
	}
	.selection {
		position: fixed;
		right: 16px;
		bottom: 16px;
		z-index: 90000;
		width: min(360px, calc(100vw - 32px));
		box-sizing: border-box;
		max-height: min(280px, calc(100dvh - 32px));
		overflow: auto;
		border: 1px solid var(--stats-border);
		box-shadow: 0 8px 32px #0004;
		padding: 10px 32px 10px 10px;
		background: var(--stats-surface);
		border-radius: 6px;
		display: grid;
		gap: 4px;
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.chart :global(.lc-arc-line:hover) {
		filter: brightness(1.1);
	}
	.chart :global(svg circle:hover) {
		filter: brightness(1.15);
	}
	.data-list button:hover,
	summary:hover {
		color: var(--stats-accent);
	}
	button:focus-visible,
	summary:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.selection button {
		position: absolute;
		right: 8px;
		top: 8px;
	}
	.empty {
		color: var(--stats-muted);
		padding: 24px 0;
		font-size: 13px;
	}
</style>
