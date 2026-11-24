<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { formatManDays } from '$lib/capacity';
	import { percentile } from '$lib/flow';
	import CfdChart from '$lib/components/CfdChart.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	const projectId = $derived(page.params.id ?? '');
	let projectName = $state('');
	let days = $state(30);
	let flow = $state<components['schemas']['ProjectFlow'] | null>(null);

	$effect(() => {
		if (!projectId) return;
		api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }).then((r) => (projectName = r.data?.name ?? ''));
	});
	$effect(() => {
		if (!projectId) return;
		const n = days;
		api.GET('/api/v1/projects/{id}/flow', { params: { path: { id: projectId }, query: { days: n } } }).then((r) => (flow = r.data ?? null));
	});

	const lead = $derived((flow?.completed ?? []).map((c) => c.leadDays));
	const cycle = $derived((flow?.completed ?? []).flatMap((c) => (c.cycleDays == null ? [] : [c.cycleDays])));
	const avg = (v: number[]) => (v.length ? v.reduce((a, b) => a + b, 0) / v.length : null);
	const show = (v: number | null) => (v == null ? '—' : `${formatManDays(v)} d`);
	const tiles = $derived([
		{ label: 'Finished', value: String(flow?.completed.length ?? 0), hint: `in the last ${days} days` },
		{ label: 'Lead time', value: show(avg(lead)), hint: `average · 85% within ${show(percentile(lead, 85))}` },
		{ label: 'Cycle time', value: show(avg(cycle)), hint: `average · 85% within ${show(percentile(cycle, 85))}` }
	]);
</script>

<svelte:head><title>Analytics · {projectName || 'Project'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<a href="/projects/{projectId}" class="text-sm text-slate-500 hover:text-slate-700">← {projectName || 'Project'}</a>
		<div class="mt-1 flex flex-wrap items-center justify-between gap-2">
			<h1 class="text-2xl font-semibold tracking-tight">Analytics</h1>
			<select class="input w-auto" aria-label="Period" bind:value={days}>
				<option value={30}>Last 30 days</option>
				<option value={60}>Last 60 days</option>
				<option value={90}>Last 90 days</option>
			</select>
		</div>
	</div>

	<div class="grid gap-3 sm:grid-cols-3">
		{#each tiles as t (t.label)}
			<div class="rounded-lg border border-slate-200 bg-white p-4">
				<div class="text-sm text-slate-500">{t.label}</div>
				<div class="mt-1 text-2xl font-semibold tabular-nums">{t.value}</div>
				<div class="mt-0.5 text-xs text-slate-500">{t.hint}</div>
			</div>
		{/each}
	</div>

	<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
		<h2 class="font-semibold">Cumulative flow</h2>
		<p class="mb-3 text-xs text-slate-500">Items in each status at the end of every day. Widening bands show where work piles up.</p>
		{#if flow}<CfdChart days={flow.days} />{/if}
	</div>

	<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
		<h2 class="mb-1 font-semibold">Finished items</h2>
		<p class="mb-3 text-xs text-slate-500">Lead time: created → done. Cycle time: first in progress → done. Both come from the change history, which started on 23 Nov 2026.</p>
		{#if flow && flow.completed.length}
			<div class="overflow-x-auto">
				<table class="w-full text-left text-sm">
					<thead class="text-xs text-slate-500"><tr><th class="py-1 font-medium">Item</th><th class="font-medium">Finished</th><th class="text-right font-medium">Lead</th><th class="text-right font-medium">Cycle</th></tr></thead>
					<tbody class="divide-y divide-slate-100">
						{#each flow.completed as c (c.taskId)}
							<tr>
								<td class="py-1.5 pr-2"><TypeBadge type={c.type} /> <a class="hover:underline" href="/projects/{projectId}/items/{c.taskId}">#{c.number} {c.title}</a></td>
								<td class="whitespace-nowrap pr-2 text-slate-600">{new Date(c.completedAt).toLocaleDateString()}</td>
								<td class="text-right tabular-nums">{show(c.leadDays)}</td>
								<td class="text-right tabular-nums">{show(c.cycleDays ?? null)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<p class="text-sm text-slate-500">Nothing finished in this period.</p>
		{/if}
	</div>
</section>
