import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load = (async ({ params }) => {
	const { type, id } = params;
	if (type !== "movie" && type !== "tv" && type !== "game") {
		error(404, "Rating helper is only available for movies, shows, and games");
	}
	const helperType = type as "movie" | "tv" | "game";

	const contentId = Number(id);
	if (!Number.isSafeInteger(contentId) || contentId <= 0) {
		error(400, "Invalid content id");
	}

	return {
		helperType,
		contentId,
	};
}) satisfies PageLoad;
