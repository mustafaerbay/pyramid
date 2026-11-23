<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import { formatManDays } from '$lib/capacity';
	import { velocity, type Metric } from '$lib/sprints';
	import BurndownChart from '$lib/components/BurndownChart.svelte';
	import VelocityChart from '$lib/components/VelocityChart.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Sprint = components['schemas']['Sprint'];
	type SprintStatus = components['schemas']['SprintStatus'];
	type Report = components['schemas']['SprintReport'];
	type Task = components['schemas']['Task'];

	const projectId = $derived(page.params.id ?? '');
	let projectName = $state('');
	let sprints = $state<Sprint[]>([]);
	let tasks = $state<Task[]>([]);
	let people = $state<Record<string, string>>({});
	let selectedId = $state<string | null>(null);
	let report = $state<Report | null>(null);
	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let metric = $state<Metric>('manDays');
	let mode = $state<'burndown' | 'burnup'>('burndown');
	let backlogFilter = $state('');

	const selected = $derived(sprints.find((s) => s.id === selectedId) ?? null);
	const inSprint = $derived(selected ? tasks.filter((t) => t.sprintId === selected.id) : []);
	const backlog = $derived(
		tasks.filter(
			(t) =>
				!t.sprintId &&
				t.status !== 'done' &&
				t.status !== 'cancelled' &&
				(!backlogFilter || t.title.toLowerCase().includes(backlogFilter.toLowerCase()))
		)
	);

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		setTimeout(() => (banner = null), 4000);
	}

	async function loadAll() {
		const [p, s, t, pe] = await Promise.all([
			api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }),
			api.GET('/api/v1/sprints', { params: { query: { projectId } } }),
			api.GET('/api/v1/tasks', { params: { query: { projectId, pageSize: 100 } } }),
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } })
		]);
		projectName = p.data?.name ?? '';
		sprints = s.data?.data ?? [];
		tasks = t.data?.data ?? [];
		people = Object.fromEntries((pe.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
		if (!selectedId || !sprints.some((x) => x.id === selectedId)) {
			selectedId = (sprints.find((x) => x.status === 'active') ?? sprints.at(-1))?.id ?? null;
		}
	}
	async function loadReport() {
		if (!selectedId) {
			report = null;
			return;
		}
		const { data } = await api.GET('/api/v1/sprints/{id}/report', { params: { path: { id: selectedId } } });
		report = data ?? null;
	}
	$effect(() => {
		if (projectId) loadAll();
	});
	$effect(() => {
		selectedId;
		loadReport();
	});

	// New / edit sprint form.
	let formOpen = $state<'new' | 'edit' | null>(null);
	let fName = $state('');
	let fGoal = $state('');
	let fStart = $state('');
	let fEnd = $state('');
	function iso(d: Date) {
		return d.toISOString().slice(0, 10);
	}
	function openNew() {
		const last = sprints.at(-1);
		const start = last ? new Date(Date.parse(last.endDate) + 86_400_000 * 3) : new Date();
		fName = `Sprint ${sprints.length + 1}`;
		fGoal = '';
		fStart = iso(start);
		fEnd = iso(new Date(start.getTime() + 86_400_000 * 13));
		formOpen = 'new';
	}
	function openEdit() {
		if (!selected) return;
		fName = selected.name;
		fGoal = selected.goal ?? '';
		fStart = selected.startDate;
		fEnd = selected.endDate;
		formOpen = 'edit';
	}
	async function saveForm(e: SubmitEvent) {
		e.preventDefault();
		const body = { name: fName, goal: fGoal || undefined, startDate: fStart, endDate: fEnd };
		const res =
			formOpen === 'new'
				? await api.POST('/api/v1/sprints', { body: { ...body, projectId } })
				: await api.PATCH('/api/v1/sprints/{id}', { params: { path: { id: selected!.id } }, body });
		if (res.error || !res.data) return flash('error', res.error?.message ?? 'Could not save the sprint.');
		formOpen = null;
		selectedId = res.data.id;
		await loadAll();
		loadReport();
	}
	async function setStatus(status: SprintStatus) {
		if (!selected) return;
		const { error } = await api.PATCH('/api/v1/sprints/{id}', { params: { path: { id: selected.id } }, body: { status } });
		if (error) return flash('error', error.message ?? 'Could not change the status.');
		await loadAll();
	}
	async function removeSprint() {
		if (!selected || !confirm(`Delete ${selected.name}? Its items go back to the backlog.`)) return;
		const { error } = await api.DELETE('/api/v1/sprints/{id}', { params: { path: { id: selected.id } } });
		if (error) return flash('error', error.message ?? 'Could not delete the sprint.');
		selectedId = null;
		await loadAll();
	}

	async function plan(task: Task, sprintId: string | null) {
		const { data, error } = await api.PATCH('/api/v1/tasks/{id}', { params: { path: { id: task.id } }, body: { sprintId } });
		if (error || !data) return flash('error', error?.message ?? 'Could not move the item.');
		tasks = tasks.map((t) => (t.id === data.id ? data : t));
		await Promise.all([loadReport(), api.GET('/api/v1/sprints', { params: { query: { projectId } } }).then((r) => (sprints = r.data?.data ?? sprints))]);
	}

	const range = (s: Sprint) =>
		`${new Date(s.startDate).toLocaleDateString(undefined, { day: 'numeric', month: 'short', timeZone: 'UTC' })} – ${new Date(s.endDate).toLocaleDateString(undefined, { day: 'numeric', month: 'short', timeZone: 'UTC' })}`;
	const STATUS_TONE: Record<string, string> = {
		planned: 'bg-slate-100 text-slate-700',
		active: 'bg-emerald-100 text-emerald-800',
		completed: 'bg-indigo-100 text-indigo-800',
		cancelled: 'bg-slate-100 text-slate-500'
	};
	const capTotals = $derived(
		(report?.capacity ?? []).reduce((a, c) => ({ avail: a.avail + c.availableManDays, planned: a.planned + c.plannedManDays }), { avail: 0, planned: 0 })
	);
</script>

<svelte:head><title>Sprints · {projectName || 'Project'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<a href="/projects/{projectId}" class="text-sm text-slate-500 hover:text-slate-700">← {projectName || 'Project'}</a>
		<div class="mt-1 flex flex-wrap items-center justify-between gap-2">
			<h1 class="text-2xl font-semibold tracking-tight">Sprints</h1>
			{#if can('member')}<button type="button" class="btn-primary" onclick={openNew}>+ New sprint</button>{/if}
		</div>
	</div>

	{#if banner}
		<div class="rounded-md border p-3 text-sm {banner.kind === 'error' ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-800'}" role="status">
			{banner.text}
		</div>
	{/if}

	{#if formOpen}
		<form class="space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm" onsubmit={saveForm}>
			<h2 class="font-semibold">{formOpen === 'new' ? 'New sprint' : `Edit ${selected?.name}`}</h2>
			<div class="grid gap-3 sm:grid-cols-[2fr_1fr_1fr]">
				<label class="block"><span class="mb-1 block font-medium text-slate-700">Name</span><input class="input" required maxlength="100" bind:value={fName} /></label>
				<label class="block"><span class="mb-1 block font-medium text-slate-700">Start</span><input class="input" type="date" required bind:value={fStart} /></label>
				<label class="block"><span class="mb-1 block font-medium text-slate-700">End</span><input class="input" type="date" required min={fStart} bind:value={fEnd} /></label>
			</div>
			<label class="block"><span class="mb-1 block font-medium text-slate-700">Goal</span><input class="input" bind:value={fGoal} placeholder="What should this sprint achieve?" /></label>
			<div class="flex gap-2">
				<button type="submit" class="btn-primary">Save</button>
				<button type="button" class="btn-ghost" onclick={() => (formOpen = null)}>Cancel</button>
			</div>
		</form>
	{/if}

	{#if sprints.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">
			No sprints yet. A sprint is a timebox (often two weeks) you plan items into.
		</div>
	{:else}
		<div class="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1 sm:mx-0 sm:px-0" role="tablist" aria-label="Sprints">
			{#each sprints as s (s.id)}
				<button
					type="button"
					role="tab"
					aria-selected={s.id === selectedId}
					class="shrink-0 rounded-lg border px-3 py-2 text-left text-sm {s.id === selectedId ? 'border-slate-900 bg-white' : 'border-slate-200 bg-white hover:border-slate-300'}"
					onclick={() => (selectedId = s.id)}
				>
					<div class="flex items-center gap-2 font-medium">{s.name}<span class="rounded-full px-1.5 text-[10px] font-semibold {STATUS_TONE[s.status]}">{s.status}</span></div>
					<div class="text-xs text-slate-500">{range(s)} · {s.stats.doneCount}/{s.stats.taskCount} done</div>
				</button>
			{/each}
		</div>
	{/if}

	{#if selected}
		<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
			<div class="flex flex-wrap items-start justify-between gap-3">
				<div class="min-w-0">
					<h2 class="text-lg font-semibold tracking-tight">{selected.name}</h2>
					<p class="text-sm text-slate-500">{range(selected)}{selected.goal ? ` · ${selected.goal}` : ''}</p>
					<p class="mt-1 text-sm text-slate-600">
						{selected.stats.doneCount}/{selected.stats.taskCount} items · {formatManDays(selected.stats.donePoints)}/{formatManDays(selected.stats.points)} pts ·
						{formatManDays(selected.stats.doneManDays)}/{formatManDays(selected.stats.manDays)} md done
					</p>
				</div>
				{#if can('member')}
					<div class="flex flex-wrap gap-2">
						<select class="input w-auto" aria-label="Sprint status" value={selected.status} onchange={(e) => setStatus(e.currentTarget.value as SprintStatus)}>
							<option value="planned">Planned</option>
							<option value="active">Active</option>
							<option value="completed">Completed</option>
							<option value="cancelled">Cancelled</option>
						</select>
						<button type="button" class="btn-secondary" onclick={openEdit}>Edit</button>
						<button type="button" class="btn-secondary text-red-600 hover:bg-red-50" onclick={removeSprint}>Delete</button>
					</div>
				{/if}
			</div>
		</div>

		<div class="grid gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
			<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
				<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
					<h3 class="font-semibold">{mode === 'burndown' ? 'Burndown' : 'Burnup'}</h3>
					<div class="flex gap-2">
						<select class="input w-auto" aria-label="Chart" bind:value={mode}>
							<option value="burndown">Burndown</option>
							<option value="burnup">Burnup</option>
						</select>
						<select class="input w-auto" aria-label="Unit" bind:value={metric}>
							<option value="manDays">Man-days</option>
							<option value="points">Story points</option>
							<option value="count">Item count</option>
						</select>
					</div>
				</div>
				{#if report}
					<BurndownChart days={report.days} start={selected.startDate} end={selected.endDate} {metric} {mode} />
				{/if}
			</div>

			<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
				<h3 class="font-semibold">Capacity</h3>
				<p class="mb-3 text-xs text-slate-500">Working days in the sprint, minus approved leave, against each person's planned man-days.</p>
				{#if report && report.capacity.length > 0}
					<ul class="space-y-2 text-sm">
						{#each report.capacity as c (c.personId)}
							{@const over = c.plannedManDays > c.availableManDays + 1e-9}
							<li>
								<div class="flex justify-between gap-2">
									<span class="truncate">{people[c.personId] ?? 'Someone'}</span>
									<span class="shrink-0 tabular-nums {over ? 'font-semibold text-red-700' : 'text-slate-600'}">
										{formatManDays(c.plannedManDays)} / {formatManDays(c.availableManDays)} md{c.leaveDays ? ` · ${c.leaveDays}d leave` : ''}
									</span>
								</div>
								<div class="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100">
									<div class="h-full rounded-full {over ? 'bg-red-500' : 'bg-indigo-500'}" style="width:{Math.min((c.plannedManDays / Math.max(c.availableManDays, 0.01)) * 100, 100)}%"></div>
								</div>
							</li>
						{/each}
					</ul>
					<p class="mt-3 border-t border-slate-100 pt-2 text-sm font-medium tabular-nums">
						Team: {formatManDays(capTotals.planned)} / {formatManDays(capTotals.avail)} md
					</p>
				{:else}
					<p class="text-sm text-slate-500">Add team members to the project or assign people to sprint items.</p>
				{/if}
			</div>
		</div>

		<div class="grid gap-4 lg:grid-cols-2">
			<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
				<h3 class="mb-2 font-semibold">In this sprint ({inSprint.length})</h3>
				{#if inSprint.length === 0}
					<p class="text-sm text-slate-500">Nothing planned yet. Add items from the backlog.</p>
				{/if}
				<ul class="divide-y divide-slate-100 text-sm">
					{#each inSprint as t (t.id)}
						<li class="flex items-center gap-2 py-2">
							<TypeBadge type={t.type} />
							<a href="/projects/{projectId}?task={t.id}" class="min-w-0 flex-1 truncate hover:underline {t.status === 'done' ? 'text-slate-400 line-through' : ''}">{t.title}</a>
							<span class="shrink-0 text-xs text-slate-500">{t.status.replace('_', ' ')}{t.storyPoints != null ? ` · ${formatManDays(t.storyPoints)} pts` : ''}</span>
							{#if can('member')}
								<button type="button" class="shrink-0 px-1 text-slate-400 hover:text-red-600" aria-label="Take {t.title} out of the sprint" onclick={() => plan(t, null)}>×</button>
							{/if}
						</li>
					{/each}
				</ul>
			</div>
			<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
				<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
					<h3 class="font-semibold">Backlog ({backlog.length})</h3>
					<input class="input w-full sm:w-48" placeholder="Search…" aria-label="Search the backlog" bind:value={backlogFilter} />
				</div>
				<ul class="max-h-96 divide-y divide-slate-100 overflow-y-auto text-sm">
					{#each backlog as t (t.id)}
						<li class="flex items-center gap-2 py-2">
							<TypeBadge type={t.type} />
							<span class="min-w-0 flex-1 truncate">{t.title}</span>
							<span class="shrink-0 text-xs text-slate-500">{t.storyPoints != null ? `${formatManDays(t.storyPoints)} pts` : ''}</span>
							{#if can('member')}
								<button type="button" class="btn-secondary shrink-0 px-2 py-1 text-xs" onclick={() => plan(t, selected.id)}>Add</button>
							{/if}
						</li>
					{:else}
						<li class="py-2 text-slate-500">No open items outside a sprint.</li>
					{/each}
				</ul>
			</div>
		</div>
	{/if}

	{#if sprints.length > 0}
		<div class="rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
			<h3 class="mb-3 font-semibold">Velocity</h3>
			<VelocityChart data={velocity(sprints)} />
		</div>
	{/if}
</section>
