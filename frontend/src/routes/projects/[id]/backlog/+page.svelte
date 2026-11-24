<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import { afterIdToPlaceBefore } from '$lib/board';
	import { backlogLevel, forecast, rollup, velocityOf, type Level } from '$lib/backlog';
	import { formatManDays } from '$lib/capacity';
	import { teamFilter } from '$lib/teams';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type Sprint = components['schemas']['Sprint'];

	const projectId = $derived(page.params.id ?? '');
	let project = $state<components['schemas']['Project'] | null>(null);
	let tasks = $state<Task[]>([]);
	let sprints = $state<Sprint[]>([]);
	let teams = $state<components['schemas']['ProjectTeam'][]>([]);
	let areas = $state<components['schemas']['Area'][]>([]);
	let people = $state<Record<string, string>>({});
	let level = $state<Level>('stories');
	let teamId = $state('');
	let showDone = $state(false);
	let error = $state<string | null>(null);

	async function load() {
		const [p, t, s, tm, ar, pe] = await Promise.all([
			api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }),
			api.GET('/api/v1/tasks', { params: { query: { projectId, pageSize: 100 } } }),
			api.GET('/api/v1/sprints', { params: { query: { projectId } } }),
			api.GET('/api/v1/project-teams', { params: { query: { projectId } } }),
			api.GET('/api/v1/areas', { params: { query: { projectId } } }),
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } })
		]);
		project = p.data ?? null;
		tasks = t.data?.data ?? [];
		sprints = s.data?.data ?? [];
		teams = tm.data?.data ?? [];
		areas = ar.data?.data ?? [];
		people = Object.fromEntries((pe.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
	}
	$effect(() => {
		if (projectId) load();
	});

	const team = $derived(teams.find((t) => t.id === teamId) ?? null);
	const inTeam = $derived(team ? teamFilter(team, areas) : null);
	const rows = $derived(
		tasks.filter(
			(t) =>
				backlogLevel(t.type) === level &&
				t.status !== 'cancelled' &&
				(showDone || t.status !== 'done') &&
				(!inTeam || inTeam(t as never))
		)
	);
	const teamSprints = $derived(sprints.filter((s) => !teamId || s.teamId === teamId || !s.teamId));
	const velocity = $derived(velocityOf(teamSprints));
	const upcoming = $derived(teamSprints.filter((s) => s.status === 'active' || s.status === 'planned').map((s) => s.name));
	const fc = $derived(level === 'stories' ? forecast(rows, velocity, upcoming.length ? upcoming : ['Next sprint']) : new Map<string, string>());
	const stateLabel = (key: string) => project?.workflow.find((w) => w.key === key)?.label ?? key;
	const sprintName = (id?: string | null) => sprints.find((s) => s.id === id)?.name ?? '';
	const parentTitle = (id?: string | null) => tasks.find((t) => t.id === id)?.title ?? '';

	// Reorder by dragging a row onto another (it goes before that one).
	let dragging = $state<string | null>(null);
	async function placeBefore(moving: Task, target: Task) {
		if (moving.id === target.id) return;
		const order = [...tasks].sort((a, b) => a.rank - b.rank || a.number - b.number);
		const { data, error: err } = await api.POST('/api/v1/tasks/{id}/move', {
			params: { path: { id: moving.id } },
			body: { afterId: afterIdToPlaceBefore(order, moving.id, target.id) }
		});
		if (err || !data) return (error = err?.message ?? 'Could not reorder.');
		tasks = tasks.map((t) => (t.id === data.id ? { ...t, rank: data.rank } : t)).sort((a, b) => a.rank - b.rank || a.number - b.number);
	}
	const LEVELS: { key: Level; label: string }[] = [
		{ key: 'epics', label: 'Epics' },
		{ key: 'features', label: 'Features' },
		{ key: 'stories', label: 'Stories' },
		{ key: 'tasks', label: 'Tasks' }
	];
</script>

<svelte:head><title>Backlog · {project?.name ?? 'Project'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<a href="/projects/{projectId}" class="text-sm text-slate-500 hover:text-slate-700">← {project?.name ?? 'Project'}</a>
		<h1 class="mt-1 text-2xl font-semibold tracking-tight">Backlog</h1>
	</div>

	<div class="flex flex-wrap items-center gap-2">
		<div class="inline-flex rounded-md bg-slate-100 p-0.5 text-sm" role="tablist" aria-label="Backlog level">
			{#each LEVELS as l (l.key)}
				<button
					type="button"
					role="tab"
					aria-selected={level === l.key}
					class="rounded px-3 py-1.5 font-medium {level === l.key ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'}"
					onclick={() => (level = l.key)}>{l.label}</button
				>
			{/each}
		</div>
		{#if teams.length}
			<select class="input w-auto" aria-label="Team" bind:value={teamId}>
				<option value="">All teams</option>
				{#each teams as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
			</select>
		{/if}
		<label class="flex items-center gap-1.5 text-sm text-slate-600"><input type="checkbox" bind:checked={showDone} /> Show done</label>
		{#if level === 'stories'}
			<span class="ml-auto text-sm text-slate-600">
				Velocity <b class="tabular-nums">{formatManDays(velocity)}</b> pts/sprint
				<span class="text-xs text-slate-500">(last 3 finished sprints)</span>
			</span>
		{/if}
	</div>

	{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>{/if}

	<div class="overflow-x-auto rounded-lg border border-slate-200 bg-white">
		<table class="w-full min-w-[44rem] text-left text-sm">
			<thead class="border-b border-slate-200 text-xs text-slate-500">
				<tr>
					<th class="px-3 py-2 font-medium">Item</th>
					<th class="px-3 py-2 font-medium">State</th>
					<th class="px-3 py-2 text-right font-medium">Points</th>
					<th class="px-3 py-2 text-right font-medium">Man-days</th>
					<th class="px-3 py-2 font-medium">Done</th>
					<th class="px-3 py-2 font-medium">{level === 'stories' ? 'Sprint · forecast' : 'Sprint'}</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100">
				{#each rows as t, i (t.id)}
					{@const r = rollup(t, tasks)}
					{@const f = fc.get(t.id)}
					{@const newSprint = f && i > 0 && fc.get(rows[i - 1].id) !== f}
					<tr
						class="{dragging === t.id ? 'opacity-50' : ''} {newSprint ? 'border-t-2 border-t-indigo-300' : ''}"
						draggable={can('member')}
						ondragstart={() => (dragging = t.id)}
						ondragend={() => (dragging = null)}
						ondragover={(e) => dragging && dragging !== t.id && e.preventDefault()}
						ondrop={(e) => {
							e.preventDefault();
							const m = tasks.find((x) => x.id === dragging);
							dragging = null;
							if (m) placeBefore(m, t);
						}}
					>
						<td class="px-3 py-2">
							<TypeBadge type={t.type} label={t.customTypeLabel} color={t.customTypeColor} />
							<a href="/projects/{projectId}/items/{t.id}" class="font-medium text-slate-800 hover:underline">{t.title}</a>
							{#if t.parentId}<div class="text-xs text-slate-500">↳ {parentTitle(t.parentId)}</div>{/if}
							<div class="text-xs text-slate-500">{t.assigneeIds.map((p) => people[p] ?? '?').join(', ')}</div>
						</td>
						<td class="px-3 py-2 text-slate-600">{stateLabel(t.state)}</td>
						<td class="px-3 py-2 text-right tabular-nums">{r.points ? formatManDays(r.points) : '—'}</td>
						<td class="px-3 py-2 text-right tabular-nums">{r.manDays ? formatManDays(r.manDays) : '—'}</td>
						<td class="px-3 py-2">
							<div class="flex items-center gap-2">
								<div class="h-1.5 w-16 overflow-hidden rounded-full bg-slate-100"><div class="h-full bg-indigo-500" style="width:{r.pctDone}%"></div></div>
								<span class="text-xs tabular-nums text-slate-600">{r.pctDone}%</span>
							</div>
						</td>
						<td class="px-3 py-2 text-slate-600">
							{sprintName(t.sprintId) || '—'}
							{#if f && !t.sprintId}<span class="ml-1 rounded bg-indigo-50 px-1.5 text-xs text-indigo-700" title="Forecast from velocity">→ {f}</span>{/if}
						</td>
					</tr>
				{:else}
					<tr><td colspan="6" class="px-3 py-6 text-center text-slate-500">Nothing at this level.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if level === 'stories' && velocity > 0}
		<p class="text-xs text-slate-500">Forecast: open stories in backlog order fill {formatManDays(velocity)} points a sprint; a line marks where each sprint starts. Drag rows to change the order.</p>
	{/if}
</section>
