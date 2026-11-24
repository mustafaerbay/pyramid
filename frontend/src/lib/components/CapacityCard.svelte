<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { formatManDays, loadBand, monthKey, monthLabel, shiftMonth } from '$lib/capacity';

	type CapacityReport = components['schemas']['CapacityReport'];

	let { projects = [] }: { projects?: { id: string; key: string; name: string }[] } = $props();

	let month = $state(monthKey(new Date()));
	let report = $state<CapacityReport | null>(null);
	let error = $state<string | null>(null);

	async function load() {
		error = null;
		const { data, error: err } = await api.GET('/api/v1/capacity', { params: { query: { month } } });
		if (err) error = err.message ?? 'Failed to load capacity';
		report = data ?? null;
	}

	function changeMonth(delta: number) {
		month = shiftMonth(month, delta);
		load();
	}

	onMount(load);

	const projectById = $derived(new Map(projects.map((p) => [p.id, p])));
	const pct = $derived(
		report && report.capacityManDays > 0 ? (report.plannedManDays / report.capacityManDays) * 100 : 0
	);
	const band = $derived(loadBand(pct));
	const free = $derived(report ? report.capacityManDays - report.plannedManDays : 0);
	const overPlanned = $derived(
		(report?.people ?? []).filter((p) => p.plannedManDays > p.capacityManDays).length
	);
</script>

<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-5">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h2 class="text-lg font-semibold tracking-tight">Man-day capacity</h2>
		<div class="flex items-center gap-1" role="group" aria-label="Capacity month">
			<button type="button" class="btn-ghost h-9 w-9 px-0" aria-label="Previous month" onclick={() => changeMonth(-1)}><Icon name="chevron-left" /></button>
			<span class="min-w-32 text-center text-sm font-medium text-slate-700">{monthLabel(month)}</span>
			<button type="button" class="btn-ghost h-9 w-9 px-0" aria-label="Next month" onclick={() => changeMonth(1)}><Icon name="chevron-right" /></button>
		</div>
	</div>

	{#if error}
		<p class="mt-3 text-sm text-red-600">{error}</p>
	{:else if report}
		<dl class="mt-3 grid grid-cols-3 gap-3">
			<div>
				<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">Capacity</dt>
				<dd class="text-xl font-semibold tabular-nums">{formatManDays(report.capacityManDays)}</dd>
			</div>
			<div>
				<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">Planned</dt>
				<dd class="text-xl font-semibold tabular-nums {band.text}">{formatManDays(report.plannedManDays)}</dd>
			</div>
			<div>
				<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">{free < 0 ? 'Over' : 'Free'}</dt>
				<dd class="text-xl font-semibold tabular-nums {free < 0 ? 'text-red-700' : ''}">
					{formatManDays(Math.abs(free))}
				</dd>
			</div>
		</dl>
		<div
			class="mt-3 h-2.5 overflow-hidden rounded-full bg-slate-100"
			role="meter"
			aria-label="Planned share of capacity"
			aria-valuemin={0}
			aria-valuemax={100}
			aria-valuenow={Math.round(pct)}
		>
			<div class="h-full rounded-full {band.bar}" style="width: {Math.min(pct, 100)}%"></div>
		</div>
		<p class="mt-1.5 text-xs text-slate-500">
			{Math.round(pct)}% planned · {report.people.length} active people · {report.workingDays} working days
			{#if overPlanned > 0}
				· <a href="/people" class="font-medium text-red-700 hover:underline">{overPlanned} over capacity</a>
			{/if}
		</p>

		{#if report.projects.length > 0}
			<ul class="mt-4 space-y-2">
				{#each report.projects as row (row.projectId)}
					{@const project = projectById.get(row.projectId)}
					{@const share = report.plannedManDays > 0 ? (row.plannedManDays / report.plannedManDays) * 100 : 0}
					<li class="text-sm">
						<div class="flex items-baseline justify-between gap-2">
							<a href="/projects/{row.projectId}" class="truncate text-slate-700 hover:underline">
								{project ? project.name : 'Project'}
							</a>
							<span class="shrink-0 font-medium tabular-nums">{formatManDays(row.plannedManDays)} md</span>
						</div>
						<div class="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100">
							<div class="h-full rounded-full bg-indigo-400" style="width: {share}%"></div>
						</div>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="mt-4 text-sm text-slate-500">
				No man-days planned for this month yet. Add start/due dates and man-days to tasks to see the load here.
			</p>
		{/if}
	{:else}
		<p class="mt-3 text-sm text-slate-500">Loading…</p>
	{/if}
</div>
