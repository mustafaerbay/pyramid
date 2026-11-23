<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { velocity } from '$lib/sprints';
	import BurndownChart from './BurndownChart.svelte';
	import CapacityCard from './CapacityCard.svelte';
	import CfdChart from './CfdChart.svelte';
	import QueryChart from './QueryChart.svelte';
	import VelocityChart from './VelocityChart.svelte';

	type Widget = components['schemas']['DashboardWidget'];
	let {
		widget,
		queries,
		projects,
		people
	}: {
		widget: Widget;
		queries: components['schemas']['SavedQuery'][];
		projects: { id: string; key: string; name: string }[];
		people: Record<string, string>;
	} = $props();

	const names = { person: (id: string) => people[id] ?? 'Someone', project: (id: string) => projects.find((p) => p.id === id)?.name ?? 'Project' };
	const query = $derived(queries.find((q) => q.id === widget.queryId) ?? null);
	const projectName = $derived(projects.find((p) => p.id === widget.projectId)?.name ?? '');

	let rows = $state<components['schemas']['Task'][]>([]);
	let total = $state(0);
	let sprints = $state<components['schemas']['Sprint'][]>([]);
	let report = $state<components['schemas']['SprintReport'] | null>(null);
	let flow = $state<components['schemas']['ProjectFlow'] | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		const w = widget;
		error = null;
		if (w.kind === 'query') {
			if (!query) return;
			api.POST('/api/v1/queries/run', { body: query.filters }).then(({ data, error: err }) => {
				if (err || !data) error = err?.message ?? 'Could not run the query.';
				rows = data?.data ?? [];
				total = data?.total ?? 0;
			});
		} else if ((w.kind === 'burndown' || w.kind === 'velocity') && w.projectId) {
			api.GET('/api/v1/sprints', { params: { query: { projectId: w.projectId } } }).then(async ({ data }) => {
				sprints = data?.data ?? [];
				if (w.kind !== 'burndown') return;
				const s = sprints.find((x) => x.id === w.sprintId) ?? sprints.find((x) => x.status === 'active') ?? sprints.at(-1);
				if (!s) return;
				const r = await api.GET('/api/v1/sprints/{id}/report', { params: { path: { id: s.id } } });
				report = r.data ?? null;
			});
		} else if (w.kind === 'cfd' && w.projectId) {
			api.GET('/api/v1/projects/{id}/flow', { params: { path: { id: w.projectId }, query: { days: 30 } } }).then((r) => (flow = r.data ?? null));
		}
	});
	const sprint = $derived(report?.sprint ?? null);
	const heading = $derived(
		widget.title ||
			(widget.kind === 'query'
				? (query?.name ?? 'Query')
				: widget.kind === 'notes'
					? 'Notes'
					: widget.kind === 'capacity'
						? 'Capacity'
						: `${widget.kind === 'cfd' ? 'Cumulative flow' : widget.kind === 'burndown' ? 'Burndown' : 'Velocity'} · ${projectName}`)
	);
</script>

{#if widget.kind === 'capacity'}
	<CapacityCard {projects} />
{:else}
	<div class="h-full rounded-lg border border-slate-200 bg-white p-4">
		<h3 class="mb-3 truncate font-semibold text-slate-800">{heading}</h3>
		{#if error}
			<p class="text-sm text-red-700">{error}</p>
		{:else if widget.kind === 'query'}
			{#if !query}
				<p class="text-sm text-slate-500">This saved query is gone or no longer shared.</p>
			{:else}
				<QueryChart {rows} {total} chart={widget.chart ?? 'bar'} groupBy={widget.groupBy ?? 'status'} stackBy={widget.stackBy ?? 'priority'} {names} />
			{/if}
		{:else if widget.kind === 'notes'}
			<p class="whitespace-pre-line text-sm text-slate-700">{widget.text ?? ''}</p>
		{:else if widget.kind === 'burndown'}
			{#if report && sprint}
				<p class="-mt-2 mb-2 text-xs text-slate-500">{sprint.name}</p>
				<BurndownChart days={report.days} start={sprint.startDate} end={sprint.endDate} metric="manDays" mode="burndown" />
			{:else}
				<p class="text-sm text-slate-500">No sprint yet.</p>
			{/if}
		{:else if widget.kind === 'velocity'}
			<VelocityChart data={velocity(sprints)} />
		{:else if widget.kind === 'cfd'}
			{#if flow}<CfdChart days={flow.days} />{/if}
		{/if}
	</div>
{/if}
