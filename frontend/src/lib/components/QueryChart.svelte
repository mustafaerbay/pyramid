<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { groupResults, type GroupBy } from '$lib/queries';
	import { stackGroups, trendSeries } from '$lib/dashcharts';
	import CountBars from './CountBars.svelte';
	import PieChart from './PieChart.svelte';
	import StackedBars from './StackedBars.svelte';
	import TrendLines from './TrendLines.svelte';

	type Task = components['schemas']['Task'];
	export type ChartKind = 'bar' | 'pie' | 'stacked' | 'trend' | 'list' | 'count';
	let {
		rows,
		total = rows.length,
		chart,
		groupBy,
		stackBy = 'priority',
		names,
		days = 30
	}: {
		rows: Task[];
		total?: number;
		chart: ChartKind;
		groupBy: GroupBy;
		stackBy?: GroupBy;
		names: { person: (id: string) => string; project: (id: string) => string };
		days?: number;
	} = $props();
	const today = new Date().toISOString().slice(0, 10);
	const from = $derived(new Date(Date.now() - (days - 1) * 86400000).toISOString().slice(0, 10));
</script>

{#if chart === 'count'}
	<p class="text-4xl font-semibold tabular-nums text-slate-900">{total}</p>
	<p class="text-xs text-slate-500">{total === 1 ? 'item' : 'items'}</p>
{:else if chart === 'list'}
	<ul class="max-h-64 divide-y divide-slate-100 overflow-y-auto text-sm">
		{#each rows.slice(0, 50) as t (t.id)}
			<li class="truncate py-1.5"><a href="/projects/{t.projectId}/items/{t.id}" class="text-slate-800 hover:underline">{t.title}</a></li>
		{:else}
			<li class="py-3 text-slate-500">Nothing matches.</li>
		{/each}
	</ul>
{:else if chart === 'pie'}
	<PieChart data={groupResults(rows, groupBy, names)} label="Items by {groupBy}" />
{:else if chart === 'stacked'}
	{@const s = stackGroups(rows, groupBy, stackBy, names)}
	<StackedBars groups={s.groups} series={s.series} counts={s.counts} label="Items by {groupBy} and {stackBy}" />
{:else if chart === 'trend'}
	<TrendLines data={trendSeries(rows, from, today)} />
{:else}
	<CountBars data={groupResults(rows, groupBy, names)} label="Items by {groupBy}" />
{/if}
