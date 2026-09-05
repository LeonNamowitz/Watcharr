<script lang="ts">
	import { page } from "$app/state";
	import PersonPage from "@/lib/person/PersonPage.svelte";

	let { data } = $props();
</script>

<PersonPage
	personId={data.personId}
	publicOwner={{
		id: page.params.id ?? "",
		username: page.params.username ?? "",
	}}
/>
