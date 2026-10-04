<script lang="ts">
	import StatsTooltip from "./StatsTooltip.svelte";
	import StatsBrowseList from "./StatsBrowseList.svelte";
	import { Bar, BarChart, LineChart, PieChart } from "layerchart/svg";
	import { Tooltip } from "layerchart/svg";
	import { averageRating, type StatsUnit } from "./format";
	import { type ComponentProps } from "svelte";
	import { on } from "svelte/events";
	import type { RatingSettings } from "@/lib/rating/helpers";
	import type { ChartPoint } from "./types";
	import { holdTooltip } from "./holdTooltip.svelte";
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
		xAxisTicks,
		barSeries,
		alignFirstXAxisTick = false,
		onSelect,
	}: {
		title: string;
		points: ChartPoint[];
		kind?: "bar" | "grouped" | "horizontal" | "line" | "pie";
		color?: string;
		height?: number;
		showData?: boolean;
		titleUnit?: "titles" | "games";
		valueUnit?: StatsUnit;
		settings?: RatingSettings;
		xAxisTicks?: string[];
		barSeries?: { key: string; label: string; color: string }[];
		alignFirstXAxisTick?: boolean;
		onSelect?: (point: ChartPoint) => void;
	} = $props();
	type ChartContext = NonNullable<ComponentProps<typeof BarChart>["context"]>;
	type TooltipContext = {
		tooltip: { data?: ChartPoint; hide: () => void };
	};
	let activeTooltip: TooltipContext | undefined;
	const touchTooltip = holdTooltip({
		target: "svg",
		canPin: () => !!activeTooltip?.tooltip.data,
		onDismiss: () => {
			activeTooltip?.tooltip.hide();
		},
	});
	let keyboardNavigation = $state(false);
	function dismissGroupedTooltip(context: ChartContext, element: Element) {
		if (kind !== "grouped") return;
		let touchPointer: number | undefined;
		const hide = () => context.tooltip.hide();
		const stopScroll = on(window, "scroll", hide, { capture: true });
		const stopResize = on(window, "resize", hide);
		const stopKeyboard = on(window, "keydown", (event) => {
			if (event.key === "Tab") keyboardNavigation = true;
		});
		const stopPointer = on(window, "pointerdown", (event) => {
			keyboardNavigation = false;
			touchPointer =
				event.pointerType === "touch" &&
				event.target instanceof Element &&
				element.contains(event.target)
					? event.pointerId
					: undefined;
		});
		const stopMove = on(window, "pointermove", (event) => {
			if (event.pointerId !== touchPointer || !event.buttons) return;
			// Touch events stay captured by the starting bar. Hit-test the finger
			// position instead so dragging can browse the other bars.
			const bar = document
				.elementFromPoint(event.clientX, event.clientY)
				?.closest("[data-stats-point]");
			if (!bar || !element.contains(bar)) return;
			const point = points.find(
				(p) => p.label === bar.getAttribute("data-stats-point"),
			);
			if (!point?.value) return;
			if (touchTooltip.pinned) {
				// LayerChart locks both showing and hiding while pinned. Keep the
				// pin, but update its data and position for this active gesture.
				const bounds = element
					.closest(".lc-root-container")
					?.getBoundingClientRect();
				if (!bounds) return;
				context.tooltip.data = point;
				context.tooltip.x = event.clientX - bounds.left;
				context.tooltip.y = event.clientY - bounds.top;
			} else {
				context.tooltip.show(event, point);
			}
		});
		const endTouch = (event: PointerEvent) => {
			if (event.pointerId === touchPointer) touchPointer = undefined;
		};
		const stopUp = on(window, "pointerup", endTouch);
		const stopCancel = on(window, "pointercancel", endTouch);
		return () => {
			stopScroll();
			stopResize();
			stopKeyboard();
			stopPointer();
			stopMove();
			stopUp();
			stopCancel();
			hide();
		};
	}
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
		showData ?? (kind === "bar" || kind === "grouped" || kind === "horizontal"),
	);
	const browsePoints = $derived(points.filter((p) => (p.value ?? 0) > 0));
	const series = $derived(
		kind === "grouped"
			? (barSeries ?? []).map((s) => ({ ...s, value: "value" }))
			: [{ key: "value", label: title, value: "value", color }],
	);
	const common = $derived({
		height,
		tooltipContext: { locked: touchTooltip.pinned },
		onTooltipClick: (_event: MouseEvent, detail: { data: ChartPoint }) =>
			onSelect?.(detail.data),
		// LayerChart's domain-motion state requires at least two categories.
		motion:
			new Set(points.map((p) => p.group ?? p.label)).size > 1
				? ("none" as const)
				: undefined,
		rule: false,
		legend: false,
		grid: false,
		highlight: false,
		padding: { left: 32, right: 10, top: 10, bottom: 26 },
		props: {
			xAxis: {
				tickSpacing: 70,
				tickOcclusion: true,
				tickMarks: false,
				...(xAxisTicks ? { ticks: xAxisTicks } : {}),
			},
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
	<g {@attach (element: Element) => dismissGroupedTooltip(context, element)}>
		{#each points as p (p.label)}
			{@const grouped = kind === "grouped"}
			{@const barColor = grouped
				? (barSeries?.find((s) => s.key === p.seriesKey)?.color ?? color)
				: color}
			<Bar
				data={p}
				data-stats-point={grouped ? p.label : undefined}
				x1={grouped ? (point: ChartPoint) => point.seriesKey : undefined}
				tooltip={grouped && !!p.value}
				onclick={() => {
					if (grouped) context.tooltip.hide();
					if (!grouped || p.value) onSelect?.(p);
				}}
				role={onSelect ? "button" : undefined}
				aria-disabled={grouped && !p.value ? true : undefined}
				tabindex={onSelect ? (grouped && !p.value ? -1 : 0) : undefined}
				aria-label={onSelect
					? `Explore ${p.tooltipLabel ?? p.label}`
					: undefined}
				onfocus={(event) => {
					if (grouped && event.currentTarget.matches(":focus-visible"))
						context.tooltip.show({ data: p });
				}}
				onblur={() => {
					if (grouped) context.tooltip.hide();
				}}
				onpointerup={(event) => {
					if (grouped && event.pointerType !== "mouse") context.tooltip.hide();
				}}
				onpointercancel={() => {
					if (grouped) context.tooltip.hide();
				}}
				onkeydown={(e) => {
					if (grouped && e.key === "Escape") context.tooltip.hide();
					if (e.key === "Enter" || e.key === " ") {
						e.preventDefault();
						if (grouped) context.tooltip.hide();
						if (!grouped || p.value) onSelect?.(p);
					}
				}}
				seriesKey={grouped ? p.seriesKey : "value"}
				radius={3}
				stroke="transparent"
				strokeWidth={0}
				fill={context.tooltip.data?.label === p.label
					? `color-mix(in srgb,${barColor} 78%,white)`
					: barColor}
			/>{/each}
	</g>
{/snippet}

{#snippet tip({ context }: { context: TooltipContext })}
	<span
		hidden
		{@attach () => {
			activeTooltip = context;
			return () => {
				activeTooltip = undefined;
			};
		}}
	></span>
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

<div
	{@attach touchTooltip.attach}
	class="chart"
	class:grouped={kind === "grouped"}
	class:keyboard-navigation={keyboardNavigation}
	class:align-first-x-tick={alignFirstXAxisTick || xAxisTicks?.length}
	role="group"
	aria-label={title}
>
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
				tooltipContext={{ locked: touchTooltip.pinned }}
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
		{:else if kind === "grouped"}
			<BarChart
				marks={barMarks}
				{...common}
				data={points}
				x={(p: ChartPoint) => p.group ?? p.label}
				x1="seriesKey"
				x1Domain={barSeries?.map((s) => s.key)}
				y="value"
				yDomain={[0, maximum]}
				seriesLayout="group"
				groupPadding={0.15}
				bandPadding={0.22}
				{series}
				tooltipContext={{ mode: "manual", locked: touchTooltip.pinned }}
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
	.chart.align-first-x-tick
		:global(
			.lc-axis.placement-bottom
				.lc-axis-tick-group:first-child
				.lc-axis-tick-label
		) {
		text-anchor: start;
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
	.chart.grouped :global(.lc-bar[tabindex="0"]) {
		cursor: pointer;
	}
	.chart.grouped :global(svg.lc-layout-svg) {
		touch-action: pan-y pinch-zoom;
	}
	.chart.grouped :global(.lc-bar:focus) {
		outline: none;
	}
	.chart.grouped.keyboard-navigation :global(.lc-bar:focus-visible) {
		outline: 2px solid var(--stats-accent);
		outline-offset: 3px;
	}
	.empty {
		color: var(--stats-muted);
		padding: 24px 0;
		font-size: 13px;
	}
</style>
