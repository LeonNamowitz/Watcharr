import { on } from "svelte/events";

export function holdTooltip(options: {
	target: string;
	canPin: () => boolean;
	onPin?: () => void;
	onDismiss: () => void;
}) {
	let pinned = $state(false);
	const dismiss = () => {
		pinned = false;
		options.onDismiss();
	};
	return {
		get pinned() {
			return pinned;
		},
		dismiss,
		attach(element: HTMLElement) {
			let timer: ReturnType<typeof setTimeout> | undefined;
			let pointer: number | undefined;
			let startX = 0;
			let startY = 0;
			let suppressClick = false;
			const cancel = () => {
				clearTimeout(timer);
				timer = undefined;
			};
			const pin = () => {
				cancel();
				if (!options.canPin()) return;
				pinned = true;
				suppressClick = true;
				options.onPin?.();
			};
			const stopDown = on(
				window,
				"pointerdown",
				(event) => {
					suppressClick = false;
					cancel();
					pointer = undefined;
					const target = event.target;
					const inside = target instanceof Element && element.contains(target);
					if (pinned) {
						dismiss();
						if (inside) {
							suppressClick = true;
							event.stopImmediatePropagation();
						}
						return;
					}
					if (
						event.pointerType !== "touch" ||
						!inside ||
						!target.closest(options.target)
					)
						return;
					pointer = event.pointerId;
					startX = event.clientX;
					startY = event.clientY;
					timer = setTimeout(pin, 600);
				},
				{ capture: true },
			);
			const stopMove = on(window, "pointermove", (event) => {
				if (
					event.pointerId === pointer &&
					Math.hypot(event.clientX - startX, event.clientY - startY) > 10
				)
					cancel();
			});
			const stopUp = on(window, "pointerup", cancel);
			const stopCancel = on(window, "pointercancel", cancel);
			const stopKeyboard = on(window, "keydown", (event) => {
				if (pinned && (event.key === "Escape" || event.key === "Tab"))
					dismiss();
			});
			const stopMenu = on(element, "contextmenu", (event) => {
				// Native long-press timing can precede our fallback timer.
				if (timer !== undefined) pin();
				if (pinned) event.preventDefault();
			});
			const stopClick = on(
				element,
				"click",
				(event) => {
					if (suppressClick) {
						event.preventDefault();
						event.stopImmediatePropagation();
						suppressClick = false;
					}
				},
				{ capture: true },
			);
			return () => {
				cancel();
				stopDown();
				stopMove();
				stopUp();
				stopCancel();
				stopKeyboard();
				stopMenu();
				stopClick();
				if (pinned) dismiss();
			};
		},
	};
}
