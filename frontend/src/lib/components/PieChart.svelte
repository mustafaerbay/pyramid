<script lang="ts">
	import { OTHER, seriesColor, topWithOther } from '$lib/dashcharts';

	let { data, label }: { data: { key: string; label: string; count: number }[]; label: string } = $props();
	const slices = $derived(topWithOther(data, 7));
	const total = $derived(slices.reduce((s, d) => s + d.count, 0));
	const R = 60;
	const r = 34;
	let hover = $state<string | null>(null);
	const arcs = $derived.by(() => {
		let a = -Math.PI / 2;
		return slices.map((d, i) => {
			const sweep = total ? (d.count / total) * Math.PI * 2 : 0;
			const a0 = a;
			a += sweep;
			const p = (ang: number, rad: number) => `${70 + rad * Math.cos(ang)} ${70 + rad * Math.sin(ang)}`;
			const large = sweep > Math.PI ? 1 : 0;
			const path =
				sweep >= Math.PI * 2 - 1e-6
					? `M ${p(0, R)} A ${R} ${R} 0 1 1 ${p(Math.PI, R)} A ${R} ${R} 0 1 1 ${p(0, R)} M ${p(0, r)} A ${r} ${r} 0 1 0 ${p(Math.PI, r)} A ${r} ${r} 0 1 0 ${p(0, r)} Z`
					: `M ${p(a0, R)} A ${R} ${R} 0 ${large} 1 ${p(a, R)} L ${p(a, r)} A ${r} ${r} 0 ${large} 0 ${p(a0, r)} Z`;
			return { ...d, path, color: d.key === '__other' ? OTHER : seriesColor(i) };
		});
	});
	const hovered = $derived(arcs.find((a) => a.key === hover) ?? null);
</script>

{#if total === 0}
	<p class="text-sm text-slate-500">No results to chart.</p>
{:else}
	<div class="flex flex-wrap items-center gap-4">
		<svg viewBox="0 0 140 140" class="h-36 w-36 shrink-0" role="img" aria-label={label}>
			{#each arcs as a (a.key)}
				<path
					d={a.path}
					fill={a.color}
					stroke="white"
					stroke-width="2"
					fill-rule="evenodd"
					opacity={hover && hover !== a.key ? 0.45 : 1}
					role="presentation"
					onmouseenter={() => (hover = a.key)}
					onmouseleave={() => (hover = null)}
				>
					<title>{a.label}: {a.count} ({Math.round((a.count / total) * 100)}%)</title>
				</path>
			{/each}
			<text x="70" y="68" text-anchor="middle" class="fill-slate-900 text-[18px] font-semibold tabular-nums">{hovered ? hovered.count : total}</text>
			<text x="70" y="84" text-anchor="middle" class="fill-slate-600 text-xs">{hovered ? hovered.label.slice(0, 12) : 'items'}</text>
		</svg>
		<ul class="min-w-0 flex-1 space-y-1 text-sm">
			{#each arcs as a (a.key)}
				<li class="flex items-center gap-2" onmouseenter={() => (hover = a.key)} onmouseleave={() => (hover = null)} role="presentation">
					<span class="h-2.5 w-2.5 shrink-0 rounded-sm" style:background={a.color}></span>
					<span class="min-w-0 flex-1 truncate text-slate-700">{a.label}</span>
					<span class="tabular-nums text-slate-600">{a.count}</span>
				</li>
			{/each}
		</ul>
	</div>
{/if}
