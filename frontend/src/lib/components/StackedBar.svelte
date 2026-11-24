<script lang="ts">
	import type { ChartColor } from '$lib/chartColors';

	type Segment = { label: string; value: number; color: ChartColor };

	let { data, height = 10 }: { data: Segment[]; height?: number } = $props();

	const total = $derived(data.reduce((sum, d) => sum + d.value, 0));
</script>

<div class="space-y-2.5">
	<div class="flex w-full overflow-hidden rounded-full bg-slate-100" style="height: {height}px">
		{#each data as seg (seg.label)}
			{#if seg.value > 0}
				<div
					class="{seg.color.fill} border-r-2 border-white last:border-r-0"
					style="width: {(seg.value / Math.max(total, 1)) * 100}%"
					title="{seg.label}: {seg.value}"
				></div>
			{/if}
		{/each}
	</div>
	<div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
		{#if total === 0}
			<span class="text-slate-500">No data yet.</span>
		{:else}
			{#each data as seg (seg.label)}
				<span class="inline-flex items-center gap-1.5">
					<span class="h-2 w-2 rounded-full {seg.color.dot}"></span>
					{seg.label} ({seg.value})
				</span>
			{/each}
		{/if}
	</div>
</div>
