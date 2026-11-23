<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import Badge from '$lib/components/Badge.svelte';
	import DonutChart from '$lib/components/DonutChart.svelte';
	import MonthlyLoadChart, { type LoadMonth } from '$lib/components/MonthlyLoadChart.svelte';
	import PersonFormModal from '$lib/components/PersonFormModal.svelte';
	import type { BadgeTone } from '$lib/badgeTones';
	import { chartColors, type ChartColor } from '$lib/chartColors';
	import { formatManDays, loadBand, monthKey, monthLabel } from '$lib/capacity';

	type Person = components['schemas']['Person'];
	type PersonWorkload = components['schemas']['PersonWorkload'];

	const id = $derived(page.params.id ?? '');

	let person = $state<Person | null>(null);
	let workload = $state<PersonWorkload | null>(null);
	let projectNames = $state<Record<string, string>>({});
	let error = $state<string | null>(null);
	let editing = $state(false);

	// Unfilled capacity in the donut: lighter than any project colour.
	const freeColor: ChartColor = {
		stroke: 'stroke-slate-200',
		fill: 'bg-slate-200',
		soft: 'bg-slate-100',
		dot: 'bg-slate-200'
	};

	const today = monthKey(new Date());
	let year = $state(Number(today.slice(0, 4)));
	let selected = $state(today);

	async function loadPerson() {
		const { data, error: err } = await api.GET('/api/v1/people/{id}', { params: { path: { id } } });
		if (err) error = err.message ?? 'Failed to load person';
		person = data ?? null;
	}

	async function loadWorkload() {
		const { data, error: err } = await api.GET('/api/v1/people/{id}/workload', {
			params: { path: { id }, query: { year } }
		});
		if (err) error = err.message ?? 'Failed to load workload';
		workload = data ?? null;
	}

	async function loadProjects() {
		const { data } = await api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } });
		projectNames = Object.fromEntries((data?.data ?? []).map((p) => [p.id, p.name]));
	}

	$effect(() => {
		if (!id) return;
		loadPerson();
		loadProjects();
	});
	$effect(() => {
		if (!id) return;
		year;
		loadWorkload();
	});

	function changeYear(delta: number) {
		year += delta;
		selected = `${year}-${selected.slice(5)}`;
	}

	// Stable colour per project across the whole year (order of first use).
	const palette: ChartColor[] = [
		chartColors.indigo,
		chartColors.emerald,
		chartColors.amber,
		chartColors.sky,
		chartColors.violet,
		chartColors.orange,
		chartColors.cyan,
		chartColors.slateDark
	];
	const colorOf = $derived(
		(() => {
			const map = new Map<string, ChartColor>();
			for (const m of workload?.months ?? [])
				for (const p of m.projects)
					if (!map.has(p.projectId)) map.set(p.projectId, palette[map.size % palette.length]);
			return map;
		})()
	);
	const projectName = (pid: string) => projectNames[pid] ?? 'Project';

	const months = $derived<LoadMonth[]>(
		(workload?.months ?? []).map((m) => ({
			month: m.month,
			planned: m.plannedManDays,
			capacity: m.capacityManDays,
			leaveDays: m.leaveDays,
			segments: m.projects.map((p) => ({
				key: p.projectId,
				label: projectName(p.projectId),
				value: p.plannedManDays,
				color: colorOf.get(p.projectId) ?? chartColors.slate
			}))
		}))
	);

	const capacity = $derived(workload?.capacityManDays ?? person?.monthlyCapacityManDays ?? 20);
	const current = $derived(workload?.months.find((m) => m.month === selected) ?? null);
	// The selected month's capacity after approved leave.
	const monthCapacity = $derived(current?.capacityManDays ?? capacity);
	const planned = $derived(current?.plannedManDays ?? 0);
	const free = $derived(Math.max(monthCapacity - planned, 0));
	const pct = $derived(monthCapacity > 0 ? (planned / monthCapacity) * 100 : planned > 0 ? Infinity : 0);
	const band = $derived(loadBand(pct));
	const donut = $derived([
		...(current?.projects ?? []).map((p) => ({
			label: projectName(p.projectId),
			value: Math.round(p.plannedManDays * 10) / 10,
			color: colorOf.get(p.projectId) ?? chartColors.slate
		})),
		...(free > 0 ? [{ label: 'Free', value: Math.round(free * 10) / 10, color: freeColor }] : [])
	]);

	const yearTotal = $derived((workload?.months ?? []).reduce((s, m) => s + m.plannedManDays, 0));

	function statusTone(status: Person['status']): BadgeTone {
		switch (status) {
			case 'active':
				return 'success';
			case 'on_leave':
				return 'warning';
			case 'offboarding':
				return 'orange';
			case 'inactive':
				return 'neutralMuted';
		}
	}

	function shortDate(d?: string | null): string {
		return d ? d.slice(5).split('-').reverse().join('.') : '—';
	}
</script>

<section class="space-y-6">
	<a href="/people" class="text-sm text-slate-500 hover:text-slate-700">← People</a>

	{#if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
	{/if}

	{#if person}
		<div class="flex flex-col gap-3 rounded-lg border border-slate-200 bg-white p-4 sm:flex-row sm:items-start sm:justify-between sm:p-6">
			<div class="min-w-0">
				<div class="flex flex-wrap items-center gap-2">
					<h1 class="text-2xl font-semibold tracking-tight">{person.firstName} {person.lastName}</h1>
					<Badge tone={statusTone(person.status)}>{person.status.replace('_', ' ')}</Badge>
				</div>
				<p class="mt-1 text-sm text-slate-600">
					{person.title ?? '—'} · <span class="break-all">{person.email}</span>
				</p>
				<p class="mt-1 text-sm text-slate-600">
					Capacity <span class="font-medium text-slate-900">{formatManDays(capacity)} man-days</span> a month
				</p>
			</div>
			<button type="button" class="btn-secondary w-full sm:w-auto" onclick={() => (editing = true)}>Edit</button>
		</div>

		<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
			<div class="mb-4 flex flex-wrap items-center justify-between gap-2">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Monthly load</h2>
					<p class="text-sm text-slate-500">
						{formatManDays(yearTotal)} man-days planned in {year} · tap a month for details
					</p>
				</div>
				<div class="flex items-center gap-1" role="group" aria-label="Year">
					<button type="button" class="btn-ghost px-2" aria-label="Previous year" onclick={() => changeYear(-1)}>‹</button>
					<span class="w-12 text-center text-sm font-medium text-slate-700">{year}</span>
					<button type="button" class="btn-ghost px-2" aria-label="Next year" onclick={() => changeYear(1)}>›</button>
				</div>
			</div>

			{#if workload}
				<MonthlyLoadChart {months} {capacity} {selected} onSelect={(m) => (selected = m)} />

				{#if colorOf.size > 0 || months.some((m) => m.leaveDays > 0)}
					<div class="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-600">
						{#each [...colorOf] as [pid, color] (pid)}
							<span class="inline-flex items-center gap-1.5">
								<span class="h-2 w-2 rounded-full {color.dot}"></span>{projectName(pid)}
							</span>
						{/each}
						<span class="inline-flex items-center gap-1.5">
							<span class="w-3 border-t border-dashed border-slate-400"></span>Capacity
						</span>
						{#if months.some((m) => m.leaveDays > 0)}
							<span class="inline-flex items-center gap-1.5">
								<span class="w-3 border-t-2 border-amber-500"></span>Capacity after leave
							</span>
						{/if}
					</div>
				{/if}
			{:else}
				<p class="text-sm text-slate-500">Loading…</p>
			{/if}
		</div>

		{#if current}
			<div class="grid gap-4 md:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
				<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
					<h2 class="text-lg font-semibold tracking-tight">{monthLabel(selected)}</h2>
					<p class="mb-4 text-sm text-slate-500">
						{current.workingDays} working days
						{#if current.leaveDays > 0}
							· <span class="font-medium text-amber-700">{current.leaveDays} on leave</span>
							· capacity {formatManDays(monthCapacity)} of {formatManDays(capacity)} md
						{/if}
					</p>
					<DonutChart
						data={donut}
						size={150}
						strokeWidth={18}
						centerValue="{formatManDays(planned)} / {formatManDays(monthCapacity)}"
						centerLabel={Number.isFinite(pct) ? `${Math.round(pct)}% full` : 'on leave'}
					/>
					<p class="mt-3 text-sm font-medium {band.text}">
						{#if monthCapacity === 0 && planned === 0 && current.leaveDays > 0}
							On leave all month
						{:else if planned > monthCapacity}
							Over capacity by {formatManDays(planned - monthCapacity)} man-days
						{:else}
							{band.label} · {formatManDays(free)} man-days free
						{/if}
					</p>
				</div>

				<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
					<h2 class="mb-3 text-lg font-semibold tracking-tight">Tasks this month</h2>
					{#if current.tasks.length === 0}
						<p class="text-sm text-slate-500">
							Nothing planned. Tasks show up here once they have man-days for this person and a start or due
							date in this month.
						</p>
					{:else}
						<ul class="divide-y divide-slate-100">
							{#each current.tasks as task (task.taskId)}
								<li class="py-2.5 text-sm">
									<div class="flex items-start justify-between gap-3">
										<a href="/projects/{task.projectId}" class="min-w-0 font-medium text-slate-800 hover:underline">
											{task.title}
										</a>
										<span class="shrink-0 font-semibold tabular-nums">{formatManDays(task.monthManDays)} md</span>
									</div>
									<div class="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-slate-500">
										<span class="inline-flex items-center gap-1.5">
											<span class="h-2 w-2 rounded-full {(colorOf.get(task.projectId) ?? chartColors.slate).dot}"></span>
											{projectName(task.projectId)}
										</span>
										<span>{shortDate(task.startDate)} → {shortDate(task.dueDate)}</span>
										{#if task.monthManDays < task.manDays}
											<span>{formatManDays(task.manDays)} md in total</span>
										{/if}
										<span>{task.status.replace('_', ' ')}</span>
									</div>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</div>
		{/if}
	{:else if !error}
		<p class="text-slate-500">Loading…</p>
	{/if}
</section>

{#if editing && person}
	<PersonFormModal
		{person}
		onClose={() => (editing = false)}
		onSaved={(p: Person) => {
			person = p;
			editing = false;
			loadWorkload();
		}}
	/>
{/if}
