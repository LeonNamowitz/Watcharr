import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load = (async ({ params }) => {
	const personId = Number(params.personId);
	if (!Number.isInteger(personId) || personId <= 0)
		error(400, "Invalid person ID");

	return { personId };
}) satisfies PageLoad;
