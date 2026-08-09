<script lang="ts">
	type Point = { label: string; value: number };

	let {
		data,
		colorClass = 'bg-slate-700',
		height = 96
	}: {
		data: Point[];
		colorClass?: string;
		height?: number;
	} = $props();

	const max = $derived(Math.max(1, ...data.map((d) => d.value)));
</script>

<div class="flex items-end gap-2" style="height: {height}px">
	{#if data.length === 0}
		<p class="self-center text-sm text-slate-400">No data yet.</p>
	{:else}
		{#each data as point (point.label)}
			<div class="flex h-full flex-1 flex-col items-center justify-end gap-1">
				<span class="text-[10px] text-slate-400">{point.value || ''}</span>
				<div class="flex w-full flex-1 items-end">
					<div
						class="w-full rounded-t {colorClass}"
						style="height: {(point.value / max) * 100}%"
						title="{point.label}: {point.value}"
					></div>
				</div>
				<span class="text-[10px] text-slate-400">{point.label}</span>
			</div>
		{/each}
	{/if}
</div>
