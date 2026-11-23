<script lang="ts">
	import { typeMeta } from '$lib/workItems';

	let { type }: { type?: string } = $props();
	const meta = $derived(typeMeta(type));
</script>

<span class="inline-flex shrink-0 rounded px-1.5 py-px text-[10px] font-semibold uppercase tracking-wide ring-1 ring-inset {meta.badge}"
	>{meta.label}</span
>
