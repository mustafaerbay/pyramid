<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { FLOW_COLORS, FLOW_ORDER, stackDay } from '$lib/flow';

	type Day = components['schemas']['FlowDay'];
	let { days }: { days: Day[] } = $props();

	const W = 640;
	const H = 220;
	const pad = { l: 32, r: 8, t: 8, b: 22 };
	const stacks = $derived(days.map((d) => stackDay(d.counts)));
	const yMax = $derived(Math.max(1, ...stacks.map((s) => s.at(-1)?.y1 ?? 0)));
	const x = (i: number) => pad.l + (days.length <= 1 ? 0 : (i / (days.length - 1)) * (W - pad.l - pad.r));
	const y = (v: number) => pad.t + (1 - v / yMax) * (H - pad.t - pad.b);
	const label = (s: string) => s.replace('_', ' ');
	const fmt = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short', timeZone: 'UTC' });

	// One closed area per status: its top edge left→right, its bottom edge back.
	function area(k: number): string {
		const top = stacks.map((s, i) => `${x(i)},${y(s[k].y1)}`);
		const bottom = stacks.map((s, i) => `${x(i)},${y(s[k].y0)}`).reverse();
		return `M${top.join(' L')} L${bottom.join(' L')} Z`;
	}

	let hover = $state<number | null>(null);
	const at = $derived(hover ?? days.length - 1);
</script>

{#if days.length === 0}
	<p class="text-sm text-slate-500">No data yet.</p>
{:else}
	<div class="mb-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-600">
		{#each [...FLOW_ORDER].reverse() as s (s)}
			<span class="inline-flex items-center gap-1.5">
				<span class="h-2.5 w-2.5 rounded-sm" style="background:{FLOW_COLORS[s]}"></span>{label(s)}
				<span class="tabular-nums text-slate-800">{days[at].counts[s] ?? 0}</span>
			</span>
		{/each}
		<span class="ml-auto text-slate-500">{fmt(days[at].date)}</span>
	</div>
	<svg viewBox="0 0 {W} {H}" class="h-auto w-full" role="img" aria-label="Cumulative flow diagram">
		{#each [0, yMax / 2, yMax] as t (t)}
			<line x1={pad.l} x2={W - pad.r} y1={y(t)} y2={y(t)} stroke="#e2e8f0" />
			<text x={pad.l - 5} y={y(t) + 3} text-anchor="end" class="fill-slate-500 text-[10px]">{Math.round(t)}</text>
		{/each}
		{#each FLOW_ORDER as s, k (s)}
			<!-- 1px surface line between bands keeps neighbours apart -->
			<path d={area(k)} fill={FLOW_COLORS[s]} stroke="white" stroke-width="1" />
		{/each}
		<text x={pad.l} y={H - 6} class="fill-slate-500 text-[10px]">{fmt(days[0].date)}</text>
		<text x={W - pad.r} y={H - 6} text-anchor="end" class="fill-slate-500 text-[10px]">{fmt(days[days.length - 1].date)}</text>
		{#if hover != null}
			<line x1={x(hover)} x2={x(hover)} y1={pad.t} y2={H - pad.b} stroke="#0f172a" stroke-width="1" stroke-dasharray="3 3" />
		{/if}
		{#each days as _, i (i)}
			<rect
				x={x(i) - (W - pad.l - pad.r) / Math.max(days.length - 1, 1) / 2}
				y={pad.t}
				width={(W - pad.l - pad.r) / Math.max(days.length - 1, 1)}
				height={H - pad.t - pad.b}
				fill="transparent"
				role="presentation"
				onmouseenter={() => (hover = i)}
				onmouseleave={() => (hover = null)}
			/>
		{/each}
	</svg>
	<details class="mt-2 text-xs text-slate-600">
		<summary class="cursor-pointer text-slate-500">Show as table</summary>
		<div class="overflow-x-auto">
			<table class="mt-2 w-full text-left tabular-nums">
				<thead class="text-slate-500"><tr><th class="py-1 pr-2 font-medium">Day</th>{#each [...FLOW_ORDER].reverse() as s (s)}<th class="pr-2 font-medium">{label(s)}</th>{/each}</tr></thead>
				<tbody>
					{#each days as d (d.date)}
						<tr class="border-t border-slate-100"><td class="py-1 pr-2">{fmt(d.date)}</td>{#each [...FLOW_ORDER].reverse() as s (s)}<td class="pr-2">{d.counts[s] ?? 0}</td>{/each}</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</details>
{/if}
