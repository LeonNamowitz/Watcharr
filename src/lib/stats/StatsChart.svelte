<script lang="ts">
	import StatsTooltip from "./StatsTooltip.svelte";
	import StatsBrowseList from "./StatsBrowseList.svelte";
	import { Bar, BarChart, LineChart, PieChart } from "layerchart/svg";
	import { Tooltip } from "layerchart/svg";
	import { averageRating, type StatsUnit } from "./format";
	import { type ComponentProps } from "svelte";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import type { ChartPoint } from "./types";
	let {
		title,
		points,
		kind = "bar",
		color = "#29acf4",
		height = 180,
		showData,
		valueUnit = "titles",
		titleUnit = "titles",
		settings,
		onSelect,
	}: {
		title: string;
		points: ChartPoint[];
		kind?: "bar" | "horizontal" | "line" | "pie";
		color?: string;
		height?: number;
		showData?: boolean;
		titleUnit?: "titles" | "games";
		valueUnit?: StatsUnit;
		settings?: RatingSettings;
		onSelect?: (point: ChartPoint) => void;
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
	const browseData = $derived(
		showData ?? (kind === "bar" || kind === "horizontal"),
	);
	const browsePoints = $derived(points.filter((p) => (p.value ?? 0) > 0));
	const series = $derived([
		{ key: "value", label: title, value: "value", color },
	]);
	const common = $derived({
		height,
		onTooltipClick: (_event: MouseEvent, detail: { data: ChartPoint }) =>
			onSelect?.(detail.data),
		// LayerChart's domain-motion state requires at least two categories.
		motion: points.length > 1 ? ("none" as const) : undefined,
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
			onclick={() => onSelect?.(p)}
			role={onSelect ? "button" : undefined}
			tabindex={onSelect ? 0 : undefined}
			aria-label={onSelect ? `Explore ${p.tooltipLabel ?? p.label}` : undefined}
			onkeydown={(e) => {
				if (e.key === "Enter" || e.key === " ") {
					e.preventDefault();
					onSelect?.(p);
				}
			}}
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
				titleCount={p.titleCount}
				{titleUnit}
				averageRating={averageRating(p.averageRating, settings)}
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
					props: onSelect
						? {
								role: "button",
								tabindex: p.value ? 0 : -1,
								"aria-label": `Explore ${p.label}`,
								onkeydown: (event: KeyboardEvent) => {
									if (event.key === "Enter" || event.key === " ") {
										event.preventDefault();
										onSelect(p);
									}
								},
							}
						: {},
				}))}
				onArcClick={(_event, detail) => onSelect?.(detail.data)}
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
				onPointClick={(_event, detail) => {
					const label =
						"label" in detail.data ? detail.data.label : detail.data.x;
					const point = points.find((point) => point.label === String(label));
					if (point) onSelect?.(point);
				}}
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
	{#if browseData}
		<StatsBrowseList points={browsePoints} {settings} {valueUnit} {onSelect} />
	{/if}
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
	.chart :global(.lc-arc-line:focus) {
		outline: none;
	}
	.chart :global(.lc-arc-line:focus-visible) {
		stroke: var(--stats-accent);
		stroke-width: 2px;
	}
	.chart :global(.lc-arc-line:hover) {
		filter: brightness(1.1);
	}
	.chart :global(svg circle:hover) {
		filter: brightness(1.15);
	}
	.empty {
		color: var(--stats-muted);
		padding: 24px 0;
		font-size: 13px;
	}
</style>
