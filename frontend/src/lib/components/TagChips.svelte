<script lang="ts">
	import { tagColor, TAG_COLOR_BY_NAME } from '$lib/tags';

	// colors: a board's own colour per tag (lower-case name → colour name).
	let { tags, onSelect, colors = {} }: { tags: string[]; onSelect?: (tag: string) => void; colors?: Record<string, string> } = $props();
	const cls = (tag: string) => TAG_COLOR_BY_NAME[colors[tag.toLowerCase()] ?? ''] ?? tagColor(tag);
</script>

{#if tags.length > 0}
	<span class="inline-flex flex-wrap gap-1">
		{#each tags as tag (tag)}
			{#if onSelect}
				<button
					type="button"
					class="rounded-full px-1.5 py-px text-[11px] font-medium ring-1 ring-inset hover:brightness-95 {cls(tag)}"
					title="Filter by {tag}"
					onclick={(e) => {
						e.stopPropagation();
						onSelect(tag);
					}}>{tag}</button
				>
			{:else}
				<span class="rounded-full px-1.5 py-px text-[11px] font-medium ring-1 ring-inset {cls(tag)}">{tag}</span>
			{/if}
		{/each}
	</span>
{/if}
