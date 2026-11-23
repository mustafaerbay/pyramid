<script lang="ts">
	let { data }: { data: { day: string; open: number; done: number }[] } = $props();

	// Validated pair on the light surface: open = indigo, done = teal.
	const INDIGO = '#6366f1';
	const TEAL = '#0d9488';
	const W = 480;
	const H = 160;
	const pad = { l: 28, r: 8, t: 8, b: 20 };
	const max = $derived(Math.max(1, ...data.flatMap((d) => [d.open, d.done])));
	const x = (i: number) => pad.l + (data.length <= 1 ? 0 : (i / (data.length - 1)) * (W - pad.l - pad.r));
	const y = (v: number) => pad.t + (1 - v / max) * (H - pad.t - pad.b);
	const line = (k: 'open' | 'done') => data.map((d, i) => `${i ? 'L' : 'M'}${x(i)} ${y(d[k])}`).join(' ');
	const fmt = (d: string) => new Date(`${d}T00:00:00Z`).toLocaleDateString(undefined, { day: 'numeric', month: 'short', timeZone: 'UTC' });
	let hover = $state<number | null>(null);
	function move(e: MouseEvent) {
		const svg = e.currentTarget as SVGSVGElement;
		const px = ((e.clientX - svg.getBoundingClientRect().left) / svg.clientWidth) * W;
		hover = Math.max(0, Math.min(data.length - 1, Math.round(((px - pad.l) / (W - pad.l - pad.r)) * (data.length - 1))));
	}
</script>

<div class="space-y-1">
	<ul class="flex gap-3 text-xs text-slate-600">
		<li class="flex items-center gap-1.5"><span class="h-0.5 w-3" style:background={INDIGO}></span>Open</li>
		<li class="flex items-center gap-1.5"><span class="h-0.5 w-3" style:background={TEAL}></span>Done</li>
		{#if hover != null}<li class="ml-auto tabular-nums">{fmt(data[hover].day)}: {data[hover].open} open · {data[hover].done} done</li>{/if}
	</ul>
	<svg viewBox="0 0 {W} {H}" class="w-full" role="img" aria-label="Open and done items over time" onmousemove={move} onmouseleave={() => (hover = null)}>
		{#each [0, 0.5, 1] as f (f)}
			<line x1={pad.l} x2={W - pad.r} y1={y(max * f)} y2={y(max * f)} stroke="#e2e8f0" />
			<text x={pad.l - 4} y={y(max * f) + 3} text-anchor="end" class="fill-slate-500 text-[9px] tabular-nums">{Math.round(max * f)}</text>
		{/each}
		{#if data.length}
			<text x={pad.l} y={H - 4} class="fill-slate-500 text-[9px]">{fmt(data[0].day)}</text>
			<text x={W - pad.r} y={H - 4} text-anchor="end" class="fill-slate-500 text-[9px]">{fmt(data[data.length - 1].day)}</text>
		{/if}
		<path d={line('open')} fill="none" stroke={INDIGO} stroke-width="2" />
		<path d={line('done')} fill="none" stroke={TEAL} stroke-width="2" />
		{#if hover != null}
			<line x1={x(hover)} x2={x(hover)} y1={pad.t} y2={H - pad.b} stroke="#0f172a" stroke-dasharray="3 3" />
			<circle cx={x(hover)} cy={y(data[hover].open)} r="4" fill={INDIGO} stroke="white" stroke-width="2" />
			<circle cx={x(hover)} cy={y(data[hover].done)} r="4" fill={TEAL} stroke="white" stroke-width="2" />
		{/if}
	</svg>
</div>
