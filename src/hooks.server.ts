import { env } from "$env/dynamic/private";
import type { Handle } from "@sveltejs/kit";

interface PublicRoute {
	ownerId: string;
	ownerName: string;
	mediaType?: "movie" | "tv" | "game";
	mediaId?: string;
}

interface PublicUserMetadata {
	username: string;
	avatar?: { path?: string };
}

interface PublicMediaMetadata {
	media?: {
		name?: string;
		summary?: string;
		extPosterPath?: string;
		poster?: { path?: string };
	};
}

const publicRoutePattern =
	/^\/lists\/(\d+)\/([^/]+)(?:\/(movie|tv|game)\/(\d+))?\/?$/;

function parsePublicRoute(pathname: string): PublicRoute | undefined {
	const match = pathname.match(publicRoutePattern);
	if (!match) return;

	try {
		return {
			ownerId: match[1],
			ownerName: decodeURIComponent(match[2]),
			mediaType: match[3] as PublicRoute["mediaType"],
			mediaId: match[4],
		};
	} catch {
		return;
	}
}

async function getPublicJson<T>(path: string): Promise<T | undefined> {
	const apiBase = (
		env.WATCHARR_INTERNAL_API_URL ?? "http://127.0.0.1:3080/api"
	).replace(/\/$/, "");

	try {
		const response = await fetch(`${apiBase}/${path}`, {
			signal: AbortSignal.timeout(2_000),
		});
		if (!response.ok) return;
		return (await response.json()) as T;
	} catch {
		// Metadata must never prevent the public page itself from loading.
		return;
	}
}

function escapeHtml(value: string) {
	return value.replace(/[&<>"']/g, (character) => {
		const escaped: Record<string, string> = {
			"&": "&amp;",
			"<": "&lt;",
			">": "&gt;",
			'"': "&quot;",
			"'": "&#39;",
		};
		return escaped[character];
	});
}

function metadataImage(
	route: PublicRoute,
	origin: string,
	user?: PublicUserMetadata,
	details?: PublicMediaMetadata,
) {
	const media = details?.media;
	if (media?.poster?.path) {
		return new URL(`/api/${media.poster.path.replace(/^\/+/, "")}`, origin)
			.href;
	}
	if (media?.extPosterPath && route.mediaType === "game") {
		return `https://images.igdb.com/igdb/image/upload/t_cover_big/${media.extPosterPath}.jpg`;
	}
	if (media?.extPosterPath) {
		return `https://image.tmdb.org/t/p/w500${media.extPosterPath}`;
	}
	if (user?.avatar?.path) {
		return new URL(`/api/${user.avatar.path.replace(/^\/+/, "")}`, origin).href;
	}
	return new URL("/logo-sqre.png", origin).href;
}

function injectMetadata(
	html: string,
	url: URL,
	route: PublicRoute,
	user?: PublicUserMetadata,
	details?: PublicMediaMetadata,
) {
	const ownerName = user?.username ?? route.ownerName;
	const mediaName = details?.media?.name;
	const title = mediaName
		? `${mediaName} - ${ownerName}'s Watcharr`
		: `${ownerName}'s Watcharr Library`;
	const description = mediaName
		? `See ${ownerName}'s status, rating, review, and activity for ${mediaName} on Watcharr.`
		: `See ${ownerName}'s recently finished titles, complete library, ratings, and watchlist on Watcharr.`;
	const image = metadataImage(route, url.origin, user, details);
	const canonicalUrl = new URL(url.pathname, url.origin).href;
	const safeTitle = escapeHtml(title);
	const safeDescription = escapeHtml(description);
	const safeImage = escapeHtml(image);
	const safeCanonicalUrl = escapeHtml(canonicalUrl);

	const tags = `
	<meta name="description" content="${safeDescription}" />
	<link rel="canonical" href="${safeCanonicalUrl}" />
	<meta property="og:site_name" content="Watcharr" />
	<meta property="og:type" content="website" />
	<meta property="og:title" content="${safeTitle}" />
	<meta property="og:description" content="${safeDescription}" />
	<meta property="og:image" content="${safeImage}" />
	<meta property="og:url" content="${safeCanonicalUrl}" />
	<meta name="twitter:card" content="summary" />
	<meta name="twitter:title" content="${safeTitle}" />
	<meta name="twitter:description" content="${safeDescription}" />
	<meta name="twitter:image" content="${safeImage}" />`;

	return html
		.replace("<title>Watcharr</title>", `<title>${safeTitle}</title>`)
		.replace("</head>", `${tags}\n</head>`);
}

export const handle: Handle = async ({ event, resolve }) => {
	const route = parsePublicRoute(event.url.pathname);
	if (!route) return resolve(event);

	const ownerPath = `public/users/${route.ownerId}/${encodeURIComponent(route.ownerName)}`;
	const [user, details] = await Promise.all([
		getPublicJson<PublicUserMetadata>(ownerPath),
		route.mediaType && route.mediaId
			? getPublicJson<PublicMediaMetadata>(
					`${ownerPath}/content/${route.mediaType}/${route.mediaId}`,
				)
			: Promise.resolve(undefined),
	]);

	let pageHtml = "";
	return resolve(event, {
		transformPageChunk: ({ html, done }) => {
			pageHtml += html;
			if (!done) return "";
			return injectMetadata(pageHtml, event.url, route, user, details);
		},
	});
};
