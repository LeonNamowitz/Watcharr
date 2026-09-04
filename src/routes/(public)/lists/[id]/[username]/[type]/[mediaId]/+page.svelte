<script lang="ts">
	import { page } from "$app/state";
	import PublicMediaPage from "@/lib/public/PublicMediaPage.svelte";
</script>

<PublicMediaPage
	ownerId={page.params.id ?? ""}
	ownerName={page.params.username ?? ""}
	mediaId={page.params.mediaId ?? ""}
	mediaType={page.params.type ?? ""}
/>
