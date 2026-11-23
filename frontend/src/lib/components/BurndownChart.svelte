<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { formatManDays } from '$lib/capacity';
	import { idealRemaining, metricOf, sprintLength, type Metric } from '$lib/sprints';

	type Day = components['schemas']['SprintDay'];

	let {
		days,
		start,
		end,
		metric,
		mode
	}: { days: Day[]; start: string; end: string; metric: Metric; mode: 'burndown' | 'burnup' } = $props();

	// Colours (validated pair, light surface): done/remaining = indigo, scope = teal.
	const INDIGO = '#6366f1';
	const TEAL = '#0d9488';

	const W = 640;
	const H = 220;
	const pad = { l: 36, r: 12, t: 12, b: 26 };
	const unit = $derived(metric === 'points' ? 'pts' : metric === 'count' ? 'items' : 'md');

	const length = $derived(Math.max(sprintLength(start, end), 1));
	const values = $derived(days.map((d) => metricOf(d, metric)));
	const startTotal = $derived(values.find((v) => v.scope > 0)?.scope ?? values.at(-1)?.scope ?? 0);
	const yMax = $derived(Math.max(1, ...values.map((v) => v.scope), startTotal));

	const x = (i: number) => pad.l + (length <= 1 ? 0 : (i / (length - 1)) * (W - pad.l - pad.r));
	const y = (v: number) => pad.t + (1 - v / yMax) * (H - pad.t - pad.b);
	const line = (pts: number[]) => pts.map((v, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(' ');

	const series = $derived(
		mode === 'burndown'
			? [{ key: 'remaining', label: `Remaining (${unit})`, color: INDIGO, values: values.map((v) => v.remaining) }]
			: [
					{ key: 'scope', label: `Scope (${unit})`, color: TEAL, values: values.map((v) => v.scope) },
					{ key: 'done', label: `Done (${unit})`, color: INDIGO, values: values.map((v) => v.done) }
				]
	);
	const ticks = $derived([0, yMax / 2, yMax]);
	const label = (i: number) =>
		new Date(Date.parse(start) + i * 86_400_000).toLocaleDateString(undefined, { day: 'numeric', month: 'short', timeZone: 'UTC' });

	let hover = $state<number | null>(null);
	const hovered = $derived(hover != null && hover < values.length ? hover : values.length - 1);
</script>

<div>
	<div class="mb-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-600">
		{#each series as s (s.key)}
			<span class="inline-flex items-center gap-1.5"><span class="h-0.5 w-4 rounded" style="background:{s.color}"></span>{s.label}</span>
		{/each}
		{#if mode === 'burndown'}
			<span class="inline-flex items-center gap-1.5"><span class="w-4 border-t-2 border-dashed border-slate-400"></span>Ideal</span>
		{/if}
		{#if values.length > 0}
			<span class="ml-auto tabular-nums text-slate-700">
				{label(hovered)}:
				{#if mode === 'burndown'}
					{formatManDays(values[hovered].remaining)} {unit} left (ideal {formatManDays(idealRemaining(startTotal, hovered, length))})
				{:else}
					{formatManDays(values[hovered].done)} of {formatManDays(values[hovered].scope)} {unit} done
				{/if}
			</span>
		{/if}
	</div>

	{#if values.length === 0}
		<p class="py-8 text-center text-sm text-slate-500">The sprint hasn't started yet.</p>
	{:else}
		<svg viewBox="0 0 {W} {H}" class="h-auto w-full" role="img" aria-label="{mode === 'burndown' ? 'Burndown' : 'Burnup'} chart">
			{#each ticks as t (t)}
				<line x1={pad.l} x2={W - pad.r} y1={y(t)} y2={y(t)} stroke="#e2e8f0" stroke-width="1" />
				<text x={pad.l - 6} y={y(t) + 3} text-anchor="end" class="fill-slate-500 text-[10px]">{formatManDays(t)}</text>
			{/each}
			<text x={pad.l} y={H - 8} class="fill-slate-500 text-[10px]">{label(0)}</text>
			<text x={W - pad.r} y={H - 8} text-anchor="end" class="fill-slate-500 text-[10px]">{label(length - 1)}</text>

			{#if mode === 'burndown'}
				<line x1={x(0)} y1={y(startTotal)} x2={x(length - 1)} y2={y(0)} stroke="#94a3b8" stroke-width="1.5" stroke-dasharray="5 4" />
			{/if}
			{#each series as s (s.key)}
				<path d={line(s.values)} fill="none" stroke={s.color} stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
				<circle cx={x(s.values.length - 1)} cy={y(s.values.at(-1) ?? 0)} r="4" fill={s.color} stroke="white" stroke-width="2" />
			{/each}

			{#if hover != null && hover < values.length}
				<line x1={x(hover)} x2={x(hover)} y1={pad.t} y2={H - pad.b} stroke="#94a3b8" stroke-width="1" />
			{/if}
			<!-- hit areas wider than the marks -->
			{#each values as _, i (i)}
				<rect
					x={x(i) - (W - pad.l - pad.r) / Math.max(length - 1, 1) / 2}
					y={pad.t}
					width={(W - pad.l - pad.r) / Math.max(length - 1, 1)}
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
			<table class="mt-2 w-full text-left tabular-nums">
				<thead class="text-slate-500"><tr><th class="py-1 font-medium">Day</th><th class="font-medium">Scope</th><th class="font-medium">Done</th><th class="font-medium">Left</th></tr></thead>
				<tbody>
					{#each values as v, i (i)}
						<tr class="border-t border-slate-100">
							<td class="py-1">{label(i)}</td><td>{formatManDays(v.scope)}</td><td>{formatManDays(v.done)}</td><td>{formatManDays(v.remaining)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</details>
	{/if}
</div>
