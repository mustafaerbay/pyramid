<script lang="ts">
	import { tagColor } from '$lib/tags';

	let { tags, onSelect }: { tags: string[]; onSelect?: (tag: string) => void } = $props();
</script>

{#if tags.length > 0}
	<span class="inline-flex flex-wrap gap-1">
		{#each tags as tag (tag)}
			{#if onSelect}
				<button
					type="button"
					class="rounded-full px-1.5 py-px text-[11px] font-medium ring-1 ring-inset hover:brightness-95 {tagColor(tag)}"
					title="Filter by {tag}"
					onclick={(e) => {
						e.stopPropagation();
						onSelect(tag);
					}}>{tag}</button
				>
			{:else}
				<span class="rounded-full px-1.5 py-px text-[11px] font-medium ring-1 ring-inset {tagColor(tag)}">{tag}</span>
			{/if}
		{/each}
	</span>
{/if}
