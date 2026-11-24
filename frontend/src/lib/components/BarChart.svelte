<script lang="ts">
	import type { ChartColor } from '$lib/chartColors';

	type Item = { label: string; value: number; color: ChartColor };

	let { data }: { data: Item[] } = $props();

	const max = $derived(Math.max(1, ...data.map((d) => d.value)));
</script>

<div class="space-y-2.5">
	{#if data.length === 0}
		<p class="text-sm text-slate-500">No data yet.</p>
	{:else}
		{#each data as item (item.label)}
			<div class="flex items-center gap-3 text-sm">
				<span class="w-32 shrink-0 truncate text-slate-600" title={item.label}>{item.label}</span>
				<div class="h-2 flex-1 overflow-hidden rounded-full bg-slate-100">
					<div
						class="h-full rounded-full {item.color.fill}"
						style="width: {(item.value / max) * 100}%"
					></div>
				</div>
				<span class="w-8 shrink-0 text-right font-medium text-slate-700">{item.value}</span>
			</div>
		{/each}
	{/if}
</div>
