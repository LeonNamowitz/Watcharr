export interface StayInViewOptions {
	/**
	 * If the `node` contains an element (like an arrow for menus),
	 * that should be shifted over to account for any shifting
	 * of the `node` itself, then pass a selector for it here.
	 */
	elToShiftSelector?: string;
	anchor?: HTMLElement;
	viewportPadding?: number;
}

export default function stayInView(node: HTMLElement, opts: StayInViewOptions) {
	console.debug("stayInView: Initial opts:", opts);
	let { elToShiftSelector, anchor, viewportPadding = anchor ? 40 : 10 } = opts;
	let viewDeb: ReturnType<typeof setTimeout>;

	/**
	 * Move element to in view, if it isn't.
	 *
	 * Keep the configured right edge stable for anchored menus, shifting them
	 * back into the viewport only when necessary. Keep the optional arrow
	 * attached to the anchor through any adjustment.
	 */
	let currentShift = 0;

	const getInView = () => {
		const nrect = node.getBoundingClientRect();
		const viewportWidth = document.documentElement.clientWidth;
		console.debug("stayInView->getInView: Called.", nrect, viewportWidth);
		const baseLeft = nrect.left - currentShift;
		const baseRight = nrect.right - currentShift;
		const inset = Math.max(0, viewportPadding);
		const minLeft = inset;
		const maxRight = viewportWidth - inset;
		let nextShift = anchor ? maxRight - baseRight : 0;
		const anchorCenter = anchor
			? anchor.getBoundingClientRect().left +
				anchor.getBoundingClientRect().width / 2
			: undefined;
		const arrowInset = 20 + node.clientLeft;

		if (anchorCenter !== undefined) {
			const anchoredLeft = baseLeft + nextShift;
			const anchoredRight = baseRight + nextShift;
			if (anchorCenter < anchoredLeft + arrowInset) {
				nextShift += anchorCenter - (anchoredLeft + arrowInset);
			} else if (anchorCenter > anchoredRight - arrowInset) {
				nextShift += anchorCenter - (anchoredRight - arrowInset);
			}
		}

		const adjustedLeft = baseLeft + nextShift;
		const adjustedRight = baseRight + nextShift;

		if (adjustedLeft < minLeft) {
			nextShift += minLeft - adjustedLeft;
		} else if (adjustedRight > maxRight) {
			nextShift += maxRight - adjustedRight;
		}

		if (nextShift !== currentShift) {
			console.debug("stayInView->getInView: Shifting node by:", nextShift);
			node.style.setProperty("translate", `${nextShift}px 0`);
			currentShift = nextShift;
		}

		if (elToShiftSelector) {
			const elToShift = node.querySelector(elToShiftSelector) as HTMLElement;
			if (elToShift) {
				if (anchor) {
					const anchorRect = anchor.getBoundingClientRect();
					const menuRect = node.getBoundingClientRect();
					const arrowLeft =
						anchorRect.left + anchorRect.width / 2 - menuRect.left - 10;
					elToShift.style.setProperty(
						"left",
						`${arrowLeft - node.clientLeft}px`,
					);
					elToShift.style.setProperty("right", "unset");
					elToShift.style.removeProperty("translate");
				} else {
					elToShift.style.setProperty("translate", `${-nextShift}px 0`);
				}
			} else {
				console.warn("elToShift not found.", elToShiftSelector);
			}
		}
	};

	const getInViewDeb = () => {
		clearTimeout(viewDeb);
		viewDeb = setTimeout(getInView, 200);
	};

	window.addEventListener("resize", getInViewDeb);
	getInView();

	return {
		update(opts: StayInViewOptions) {
			console.debug("stayInView: Opts updated", opts);
			elToShiftSelector = opts.elToShiftSelector;
			anchor = opts.anchor;
			viewportPadding = opts.viewportPadding ?? (anchor ? 40 : 10);
			getInView();
		},
		destroy() {
			window.removeEventListener("resize", getInViewDeb);
		},
	};
}
