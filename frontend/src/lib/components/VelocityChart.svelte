<script lang="ts">
	import { formatManDays } from '$lib/capacity';

	let { data }: { data: { name: string; committed: number; done: number }[] } = $props();

	// Validated pair on the light surface: committed = teal, done = indigo.
	const TEAL = '#0d9488';
	const INDIGO = '#6366f1';
	const H = 160;
	const max = $derived(Math.max(1, ...data.flatMap((d) => [d.committed, d.done])));
	const avg = $derived(data.length ? data.reduce((s, d) => s + d.done, 0) / data.length : 0);
</script>

{#if data.length === 0}
	<p class="text-sm text-slate-500">Velocity shows up once a sprint is marked completed.</p>
{:else}
	<div class="mb-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-600">
		<span class="inline-flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm" style="background:{TEAL}"></span>Committed (pts)</span>
		<span class="inline-flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm" style="background:{INDIGO}"></span>Done (pts)</span>
		<span class="ml-auto text-slate-700">Average done: <span class="font-semibold tabular-nums">{formatManDays(avg)}</span> pts</span>
	</div>
	<div class="flex items-end gap-4" style="height:{H}px" role="img" aria-label="Velocity by sprint">
		{#each data as d (d.name)}
			<div class="flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-1">
				<div class="flex h-full w-full items-end justify-center gap-0.5">
					<div
						class="w-1/3 max-w-6 rounded-t"
						style="height:{(d.committed / max) * 100}%; background:{TEAL}"
						title="{d.name}: {formatManDays(d.committed)} pts committed"
					></div>
					<div
						class="w-1/3 max-w-6 rounded-t"
						style="height:{(d.done / max) * 100}%; background:{INDIGO}"
						title="{d.name}: {formatManDays(d.done)} pts done"
					></div>
				</div>
				<span class="w-full truncate text-center text-[11px] text-slate-500" title={d.name}>{d.name}</span>
			</div>
		{/each}
	</div>
{/if}
