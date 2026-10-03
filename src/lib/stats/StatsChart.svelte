<script lang="ts">
	import StatsTooltip from "./StatsTooltip.svelte";
	import { Bar, BarChart, LineChart, PieChart } from "layerchart/svg";
	import { Tooltip } from "layerchart/svg";
	import { decimal } from "./format";
	import { type ComponentProps } from "svelte";
	import type { ChartPoint } from "./types";
	let {
		title,
		points,
		kind = "bar",
		color = "#29acf4",
		height = 180,
		showData = false,
		dataLabel = "Explore chart data",
	}: {
		title: string;
		points: ChartPoint[];
		kind?: "bar" | "horizontal" | "line" | "pie";
		color?: string;
		height?: number;
		showData?: boolean;
		dataLabel?: string;
	} = $props();
	type ChartContext = NonNullable<ComponentProps<typeof BarChart>["context"]>;
	const colors = ["#29acf4", "#f5b85a", "#f47983", "#51ad79", "#b19bea"];
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
			<StatsTooltip
				label={p.tooltipLabel ?? p.label}
				value={valueLabel(p)}
				detail={p.detail}
			/>
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
			/>
		{/if}
	{:else}<p class="empty">No recorded data for this chart.</p>{/if}
	{#if showData}<details>
			<summary>{dataLabel}</summary>
			<div class="data-list">
				{#each points as p (p.label)}<div class="data-row">
						<span class="data-label"
							>{p.tooltipLabel ?? p.label}{#if p.detail}<small>{p.detail}</small
								>{/if}</span
						><strong>{valueLabel(p)}</strong>
					</div>{/each}
			</div>
		</details>{/if}
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
	.data-row {
		display: flex;
		justify-content: space-between;
		gap: 8px;
		width: 100%;
		padding: 5px 0;
		text-align: left;
		font-size: 14px;
	}
	.data-label {
		display: grid;
		min-width: 0;
	}
	.data-label small {
		margin-top: 2px;
		color: var(--stats-muted);
		font-size: 12px;
		line-height: 1.4;
	}
	.chart :global(.lc-arc-line:hover) {
		filter: brightness(1.1);
	}
	.chart :global(svg circle:hover) {
		filter: brightness(1.15);
	}
	summary:hover {
		color: var(--stats-accent);
	}
	summary:focus-visible {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.empty {
		color: var(--stats-muted);
		padding: 24px 0;
		font-size: 13px;
	}
</style>
