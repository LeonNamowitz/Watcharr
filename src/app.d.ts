import type { HandbookEntryId } from "./lib/handbook/entries";

// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces
declare global {
	namespace App {
		interface PageState {
			publicListDepth?: number;
			handbook?: HandbookEntryId;
		}
	}
}

export {};
