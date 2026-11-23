<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import { cleanFilters, groupResults, type GroupBy } from '$lib/queries';
	import { TYPE_OPTIONS } from '$lib/workItems';
	import { areaLabel } from '$lib/areas';
	import { formatManDays } from '$lib/capacity';
	import ChipSelect from '$lib/components/ChipSelect.svelte';
	import CountBars from '$lib/components/CountBars.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';
	import BulkBar from '$lib/components/BulkBar.svelte';
	import { download } from '$lib/csv';
	import { tasksToCsv } from '$lib/taskCsv';

	type Task = components['schemas']['Task'];
	type Saved = components['schemas']['SavedQuery'];
	type Filters = components['schemas']['QueryFilters'];

	const STATUSES = ['backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'];
	const PRIORITIES = ['urgent', 'high', 'medium', 'low'];

	let projects = $state<{ id: string; name: string }[]>([]);
	let people = $state<Record<string, string>>({});
	let tags = $state<string[]>([]);
	let saved = $state<Saved[]>([]);
	let sprints = $state<{ id: string; name: string }[]>([]);
	let areas = $state<{ id: string; path: string }[]>([]);

	// The filters being edited.
	let projectIds = $state<string[]>([]);
	let types = $state<string[]>([]);
	let statuses = $state<string[]>([]);
	let priorities = $state<string[]>([]);
	let assigneeIds = $state<string[]>([]);
	let unassigned = $state(false);
	let tagValues = $state<string[]>([]);
	let sprintIds = $state<string[]>([]);
	let areaIds = $state<string[]>([]);
	let text = $state('');
	let dueAfter = $state('');
	let dueBefore = $state('');
	let groupBy = $state<GroupBy>('status');

	let results = $state<Task[]>([]);
	let total = $state(0);
	let truncated = $state(false);
	let running = $state(false);
	let error = $state<string | null>(null);
	let current = $state<Saved | null>(null);
	let saveName = $state('');
	let saveShared = $state(false);
	let banner = $state<string | null>(null);
	let selected = $state<Set<string>>(new Set());
	const selectedTasks = $derived(results.filter((t) => selected.has(t.id)));
	function toggle(id: string) {
		const next = new Set(selected);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selected = next;
	}
	function exportCsv() {
		const rows = selectedTasks.length ? selectedTasks : results;
		download(
			`pyramid-${(current?.name ?? 'query').replace(/[^\w-]+/g, '-').toLowerCase()}.csv`,
			tasksToCsv(rows, {
				people: Object.entries(people).map(([id, name]) => ({ id, name, email: '' })),
				tasks: results.map((t) => ({ id: t.id, number: t.number, title: t.title })),
				milestones: [],
				sprints,
				areas,
				project: projectName
			})
		);
	}

	const filters = $derived(
		cleanFilters({
			projectIds,
			types,
			statuses,
			priorities,
			assigneeIds,
			unassigned,
			tags: tagValues,
			sprintIds,
			areaIds,
			text,
			dueAfter,
			dueBefore
		}) as Filters
	);
	const singleProject = $derived(projectIds.length === 1 ? projectIds[0] : null);
	const projectName = (id: string) => projects.find((p) => p.id === id)?.name ?? 'Project';
	const chart = $derived(groupResults(results, groupBy, { person: (id) => people[id] ?? 'Someone', project: projectName }));

	$effect(() => {
		Promise.all([
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/tags'),
			api.GET('/api/v1/queries')
		]).then(([p, pe, t, q]) => {
			projects = (p.data?.data ?? []).map((x) => ({ id: x.id, name: x.name })).sort((a, b) => a.name.localeCompare(b.name));
			people = Object.fromEntries((pe.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
			tags = t.data?.data ?? [];
			saved = q.data?.data ?? [];
		});
	});

	// Sprints and areas only make sense inside one project.
	$effect(() => {
		const pid = singleProject;
		if (!pid) {
			sprints = [];
			areas = [];
			sprintIds = [];
			areaIds = [];
			return;
		}
		api.GET('/api/v1/sprints', { params: { query: { projectId: pid } } }).then((r) => (sprints = (r.data?.data ?? []).map((s) => ({ id: s.id, name: s.name }))));
		api.GET('/api/v1/areas', { params: { query: { projectId: pid } } }).then((r) => (areas = r.data?.data ?? []));
	});

	// Run whenever the filters change (debounced).
	let timer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const f = filters;
		clearTimeout(timer);
		timer = setTimeout(() => run(f), 250);
		return () => clearTimeout(timer);
	});
	async function run(f: Filters) {
		running = true;
		const { data, error: err } = await api.POST('/api/v1/queries/run', { body: f });
		running = false;
		if (err || !data) {
			error = err?.message ?? 'Could not run the query.';
			return;
		}
		error = null;
		results = data.data;
		selected = new Set([...selected].filter((id) => data.data.some((t) => t.id === id)));
		total = data.total;
		truncated = data.truncated;
	}

	function load(q: Saved) {
		const f = q.filters;
		projectIds = [...(f.projectIds ?? [])];
		types = [...(f.types ?? [])];
		statuses = [...(f.statuses ?? [])];
		priorities = [...(f.priorities ?? [])];
		assigneeIds = [...(f.assigneeIds ?? [])];
		unassigned = !!f.unassigned;
		tagValues = [...(f.tags ?? [])];
		text = f.text ?? '';
		dueAfter = f.dueAfter ?? '';
		dueBefore = f.dueBefore ?? '';
		groupBy = q.groupBy;
		current = q;
		saveName = q.name;
		saveShared = q.shared;
		// Sprint/area lists load after the project; set them once they're there.
		setTimeout(() => {
			sprintIds = [...(f.sprintIds ?? [])];
			areaIds = [...(f.areaIds ?? [])];
		}, 300);
	}
	function reset() {
		projectIds = [];
		types = [];
		statuses = [];
		priorities = [];
		assigneeIds = [];
		tagValues = [];
		sprintIds = [];
		areaIds = [];
		unassigned = false;
		text = dueAfter = dueBefore = '';
		current = null;
		saveName = '';
		saveShared = false;
	}
	async function save(asNew: boolean) {
		const body = { name: saveName, filters, groupBy, shared: saveShared };
		const res =
			current && current.mine && !asNew
				? await api.PATCH('/api/v1/queries/{id}', { params: { path: { id: current.id } }, body })
				: await api.POST('/api/v1/queries', { body });
		if (res.error || !res.data) return (banner = res.error?.message ?? 'Could not save.');
		current = res.data;
		const q = await api.GET('/api/v1/queries');
		saved = q.data?.data ?? saved;
		banner = `Saved "${res.data.name}".`;
		setTimeout(() => (banner = null), 3000);
	}
	async function remove() {
		if (!current || !confirm(`Delete the saved query "${current.name}"?`)) return;
		const { error: err } = await api.DELETE('/api/v1/queries/{id}', { params: { path: { id: current.id } } });
		if (err) return (banner = err.message ?? 'Could not delete.');
		saved = saved.filter((q) => q.id !== current!.id);
		reset();
	}
</script>

<svelte:head><title>Queries · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Queries</h1>
		<p class="mt-1 text-sm text-slate-500">Find work items across every project, chart them, and save the query for later.</p>
	</div>

	{#if saved.length > 0}
		<div class="flex flex-wrap items-center gap-2 text-sm">
			<span class="text-slate-500">Saved:</span>
			{#each saved as q (q.id)}
				<button
					type="button"
					class="rounded-full px-3 py-1 ring-1 ring-inset {current?.id === q.id ? 'bg-slate-900 text-white ring-slate-900' : 'bg-white text-slate-700 ring-slate-200 hover:bg-slate-50'}"
					onclick={() => load(q)}
					title={q.shared ? (q.mine ? 'Shared by you' : `Shared by ${people[q.ownerPersonId ?? ''] ?? 'someone'}`) : 'Only you'}
					>{q.name}{q.shared ? ' 👥' : ''}</button
				>
			{/each}
			<button type="button" class="text-xs text-slate-500 hover:underline" onclick={reset}>New query</button>
		</div>
	{/if}

	<div class="grid gap-3 rounded-lg border border-slate-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-4">
		<ChipSelect label="Projects" options={projects.map((p) => ({ value: p.id, label: p.name }))} bind:values={projectIds} />
		<ChipSelect label="Types" options={TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label }))} bind:values={types} />
		<ChipSelect label="Statuses" options={STATUSES.map((s) => ({ value: s, label: s.replace('_', ' ') }))} bind:values={statuses} />
		<ChipSelect label="Priorities" options={PRIORITIES.map((p) => ({ value: p, label: p }))} bind:values={priorities} />
		<div>
			<ChipSelect label="People" options={Object.entries(people).map(([value, label]) => ({ value, label })).sort((a, b) => a.label.localeCompare(b.label))} bind:values={assigneeIds} />
			<label class="mt-1 flex items-center gap-1.5 text-xs text-slate-600"><input type="checkbox" bind:checked={unassigned} /> Only unassigned</label>
		</div>
		<ChipSelect label="Tags" options={tags.map((t) => ({ value: t, label: t }))} bind:values={tagValues} />
		{#if singleProject}
			<ChipSelect label="Sprints" options={sprints.map((s) => ({ value: s.id, label: s.name }))} bind:values={sprintIds} />
			<ChipSelect label="Areas" options={areas.map((a) => ({ value: a.id, label: a.path }))} bind:values={areaIds} />
		{/if}
		<input class="input" placeholder="Title contains…" aria-label="Title contains" bind:value={text} />
		<div class="flex gap-2">
			<label class="block min-w-0 flex-1 text-xs text-slate-600">Due from<input class="input" type="date" bind:value={dueAfter} /></label>
			<label class="block min-w-0 flex-1 text-xs text-slate-600">to<input class="input" type="date" bind:value={dueBefore} /></label>
		</div>
	</div>

	{#if can('member')}
		<form
			class="flex flex-wrap items-center gap-2 text-sm"
			onsubmit={(e) => {
				e.preventDefault();
				save(false);
			}}
		>
			<input class="input w-full sm:w-56" placeholder="Name this query" aria-label="Query name" required maxlength="100" bind:value={saveName} />
			<label class="flex items-center gap-1.5 text-slate-600"><input type="checkbox" bind:checked={saveShared} /> Share with everyone</label>
			<button type="submit" class="btn-primary">{current?.mine ? 'Save' : 'Save query'}</button>
			{#if current?.mine}
				<button type="button" class="btn-secondary" onclick={() => save(true)}>Save as new</button>
				<button type="button" class="btn-ghost text-red-600" onclick={remove}>Delete</button>
			{/if}
			{#if banner}<span class="text-slate-600" role="status">{banner}</span>{/if}
		</form>
	{/if}

	{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>{/if}

	<div class="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
		<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
			<div class="flex items-center justify-between border-b border-slate-100 px-4 py-2 text-sm">
				<span class="flex items-center gap-2">
					{#if can('member') && results.length}
						<input
							type="checkbox"
							aria-label="Select all"
							checked={selected.size > 0 && selected.size === results.length}
							onchange={(e) => (selected = e.currentTarget.checked ? new Set(results.map((t) => t.id)) : new Set())}
						/>
					{/if}
					<span class="font-semibold">{total} {total === 1 ? 'item' : 'items'}{running ? ' · …' : ''}</span>
				</span>
				<span class="flex items-center gap-2">
					{#if truncated}<span class="text-xs text-amber-700">Showing the first 500</span>{/if}
					{#if results.length}
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={exportCsv}>Export CSV{selected.size ? ` (${selected.size})` : ''}</button>
					{/if}
				</span>
			</div>
			<ul class="max-h-[32rem] divide-y divide-slate-100 overflow-y-auto text-sm">
				{#each results as t (t.id)}
					<li class="flex flex-wrap items-center gap-x-2 gap-y-0.5 px-4 py-2 {selected.has(t.id) ? 'bg-indigo-50' : ''}">
						{#if can('member')}
							<input type="checkbox" aria-label="Select {t.title}" checked={selected.has(t.id)} onchange={() => toggle(t.id)} />
						{/if}
						<TypeBadge type={t.type} />
						<a href="/projects/{t.projectId}?task={t.id}" class="min-w-0 flex-1 truncate font-medium text-slate-800 hover:underline">{t.title}</a>
						<span class="w-full pl-12 text-xs text-slate-500">
							{projectName(t.projectId)} · {t.status.replace('_', ' ')} · {t.assigneeIds.map((id) => people[id] ?? '?').join(', ') || 'Unassigned'}{t.dueDate
								? ` · due ${t.dueDate}`
								: ''}{t.storyPoints != null ? ` · ${formatManDays(t.storyPoints)} pts` : ''}
						</span>
					</li>
				{:else}
					<li class="px-4 py-6 text-center text-slate-500">Nothing matches.</li>
				{/each}
			</ul>
		</div>
		<div class="rounded-lg border border-slate-200 bg-white p-4 lg:self-start">
			<div class="mb-3 flex items-center justify-between gap-2">
				<h2 class="font-semibold">Chart</h2>
				<select class="input w-auto" aria-label="Group by" bind:value={groupBy}>
					<option value="status">By status</option>
					<option value="assignee">By person</option>
					<option value="type">By type</option>
					<option value="priority">By priority</option>
					<option value="project">By project</option>
				</select>
			</div>
			<CountBars data={chart} label="Items by {groupBy}" />
		</div>
	</div>
	{#if selectedTasks.length > 0}
		<BulkBar
			tasks={selectedTasks}
			people={Object.entries(people).map(([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name))}
			onClear={() => (selected = new Set())}
			onDone={(m) => {
				banner = m;
				setTimeout(() => (banner = null), 5000);
				run(filters);
			}}
		/>
	{/if}
</section>
