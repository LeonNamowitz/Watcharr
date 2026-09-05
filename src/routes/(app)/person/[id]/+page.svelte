<script lang="ts">
	import PersonPage from "@/lib/person/PersonPage.svelte";

	let { data } = $props();
</script>

<PersonPage personId={data.personId} />
