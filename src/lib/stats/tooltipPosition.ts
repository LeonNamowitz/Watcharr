export interface StatsTooltipPosition {
	left: number;
	top: number;
}

export function statsTooltipPosition(
	event: PointerEvent | FocusEvent,
	anchor: HTMLElement,
): StatsTooltipPosition {
	if ("clientX" in event && typeof event.clientX === "number") {
		const left = Math.max(
			16,
			Math.min(event.clientX + 14, window.innerWidth - 296),
		);
		const top =
			event.clientY + 14 + 116 < window.innerHeight
				? event.clientY + 14
				: Math.max(16, event.clientY - 130);
		return { left, top };
	}

	const rect = anchor.getBoundingClientRect();
	return {
		left: Math.max(16, Math.min(rect.left, window.innerWidth - 296)),
		top:
			rect.bottom + 8 + 116 < window.innerHeight
				? rect.bottom + 8
				: Math.max(16, rect.top - 116),
	};
}
