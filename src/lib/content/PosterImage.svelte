<script lang="ts">
	interface Props {
		src: string;
		alt?: string;
		fluid?: boolean;
		loading?: "eager" | "lazy";
	}

	let { src, alt = "", fluid = false, loading = "eager" }: Props = $props();
</script>

<img {src} {alt} {loading} class:fluid />

<style lang="scss">
	img.fluid {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: inherit;
		box-shadow: none;
	}
	img {
		width: 245px;
		height: 100%;
		box-shadow: 0px 0px 14px -4px #000;
		border-radius: 12px;
	}
</style>
