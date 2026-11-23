<script lang="ts">
	import { OTHER, seriesColor } from '$lib/dashcharts';

	let {
		groups,
		series,
		counts,
		label
	}: {
		groups: { key: string; label: string }[];
		series: { key: string; label: string }[];
		counts: Record<string, Record<string, number>>;
		label: string;
	} = $props();
	// Up to 8 series get their own hue; the rest fold into Other.
	const shown = $derived(series.slice(0, series.length > 8 ? 7 : 8));
	const folded = $derived(series.slice(shown.length).map((s) => s.key));
	const legend = $derived([...shown.map((s, i) => ({ ...s, color: seriesColor(i) })), ...(folded.length ? [{ key: '__other', label: 'Other', color: OTHER }] : [])]);
	const segs = (g: string) =>
		legend
			.map((s) => ({
				...s,
				n: s.key === '__other' ? folded.reduce((t, k) => t + (counts[g]?.[k] ?? 0), 0) : (counts[g]?.[s.key] ?? 0)
			}))
			.filter((s) => s.n > 0);
	const totalOf = (g: string) => Object.values(counts[g] ?? {}).reduce((a, b) => a + b, 0);
	const max = $derived(Math.max(1, ...groups.map((g) => totalOf(g.key))));
</script>

{#if groups.length === 0}
	<p class="text-sm text-slate-500">No results to chart.</p>
{:else}
	<div class="space-y-3" role="img" aria-label={label}>
		<ul class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-600">
			{#each legend as s (s.key)}
				<li class="flex items-center gap-1.5"><span class="h-2.5 w-2.5 rounded-sm" style:background={s.color}></span>{s.label}</li>
			{/each}
		</ul>
		<ul class="space-y-1.5">
			{#each groups as g (g.key)}
				{@const total = totalOf(g.key)}
				<li class="grid grid-cols-[minmax(0,7rem)_minmax(0,1fr)_2.5rem] items-center gap-2 text-sm">
					<span class="truncate text-slate-700" title={g.label}>{g.label}</span>
					<span class="flex h-4 gap-[2px]" style:width="{(total / max) * 100}%">
						{#each segs(g.key) as s, i (s.key)}
							<span
								class="h-full {i === segs(g.key).length - 1 ? 'rounded-r' : ''}"
								style:background={s.color}
								style:flex-grow={s.n}
								title="{g.label} · {s.label}: {s.n}"
							></span>
						{/each}
					</span>
					<span class="text-right tabular-nums text-slate-600">{total}</span>
				</li>
			{/each}
		</ul>
	</div>
{/if}
