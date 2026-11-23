<script lang="ts">
	// One series of counts as horizontal bars, biggest first, value at the end.
	let { data, label }: { data: { key: string; label: string; count: number }[]; label: string } = $props();
	const max = $derived(Math.max(1, ...data.map((d) => d.count)));
	const total = $derived(data.reduce((s, d) => s + d.count, 0));
</script>

{#if data.length === 0}
	<p class="text-sm text-slate-500">No results to chart.</p>
{:else}
	<ul class="space-y-1.5" role="list" aria-label={label}>
		{#each data as d (d.key)}
			<li class="grid grid-cols-[minmax(0,8rem)_minmax(0,1fr)_2.5rem] items-center gap-2 text-sm" title="{d.label}: {d.count} of {total}">
				<span class="truncate text-slate-700">{d.label}</span>
				<span class="h-4 rounded-r" style="width:{Math.max((d.count / max) * 100, 1)}%; background:#6366f1"></span>
				<span class="text-right tabular-nums text-slate-600">{d.count}</span>
			</li>
		{/each}
	</ul>
{/if}
