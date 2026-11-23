<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import DashWidget from '$lib/components/DashWidget.svelte';

	type Dashboard = components['schemas']['Dashboard'];
	type Widget = components['schemas']['DashboardWidget'];

	const id = $derived(page.params.id ?? '');
	let dash = $state<Dashboard | null>(null);
	let queries = $state<components['schemas']['SavedQuery'][]>([]);
	let projects = $state<{ id: string; key: string; name: string }[]>([]);
	let people = $state<Record<string, string>>({});
	let editing = $state(false);
	let draft = $state<Widget[]>([]);
	let name = $state('');
	let shared = $state(false);
	let error = $state<string | null>(null);
	let saving = $state(false);

	$effect(() => {
		if (!id) return;
		Promise.all([
			api.GET('/api/v1/dashboards/{id}', { params: { path: { id } } }),
			api.GET('/api/v1/queries'),
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } })
		]).then(([d, q, p, pe]) => {
			if (d.error) return (error = d.error.message ?? 'Dashboard not found.');
			dash = d.data ?? null;
			queries = q.data?.data ?? [];
			projects = (p.data?.data ?? []).map((x) => ({ id: x.id, key: x.key, name: x.name })).sort((a, b) => a.name.localeCompare(b.name));
			people = Object.fromEntries((pe.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
			if (dash?.mine && page.url.searchParams.get('edit')) startEdit();
		});
	});

	function startEdit() {
		if (!dash) return;
		draft = structuredClone($state.snapshot(dash.widgets)) as Widget[];
		name = dash.name;
		shared = dash.shared;
		editing = true;
	}
	const newId = () => Math.random().toString(36).slice(2, 10);
	function add(kind: Widget['kind']) {
		const w: Widget = { id: newId(), kind, size: kind === 'notes' || kind === 'cfd' ? 'wide' : 'small' };
		if (kind === 'query') Object.assign(w, { queryId: queries[0]?.id, chart: 'bar', groupBy: 'status' });
		if (kind === 'burndown' || kind === 'velocity' || kind === 'cfd') w.projectId = projects[0]?.id;
		if (kind === 'notes') w.text = '';
		draft = [...draft, w];
	}
	function moveW(i: number, d: number) {
		const j = i + d;
		if (j < 0 || j >= draft.length) return;
		const next = [...draft];
		[next[i], next[j]] = [next[j], next[i]];
		draft = next;
	}
	async function save() {
		saving = true;
		const { data, error: err } = await api.PATCH('/api/v1/dashboards/{id}', {
			params: { path: { id } },
			body: { name, shared, widgets: draft.map((w) => ({ ...w, title: w.title?.trim() || undefined })) }
		});
		saving = false;
		if (err || !data) return (error = err?.message ?? 'Could not save.');
		dash = data;
		editing = false;
		error = null;
	}
	async function remove() {
		if (!dash || !confirm(`Delete the dashboard "${dash.name}"?`)) return;
		const { error: err } = await api.DELETE('/api/v1/dashboards/{id}', { params: { path: { id } } });
		if (err) return (error = err.message ?? 'Could not delete.');
		goto('/dashboards');
	}
	const KINDS: { kind: Widget['kind']; label: string }[] = [
		{ kind: 'query', label: 'Query chart' },
		{ kind: 'burndown', label: 'Burndown' },
		{ kind: 'velocity', label: 'Velocity' },
		{ kind: 'cfd', label: 'Cumulative flow' },
		{ kind: 'capacity', label: 'Capacity' },
		{ kind: 'notes', label: 'Notes' }
	];
	const GROUPS = [
		['status', 'status'],
		['assignee', 'person'],
		['type', 'type'],
		['priority', 'priority'],
		['project', 'project']
	] as const;
</script>

<svelte:head><title>{dash?.name ?? 'Dashboard'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div class="flex flex-wrap items-end justify-between gap-2">
		<div>
			<a href="/dashboards" class="text-sm text-slate-500 hover:text-slate-700">← Dashboards</a>
			<h1 class="mt-1 text-2xl font-semibold tracking-tight">{dash?.name ?? 'Dashboard'}</h1>
		</div>
		{#if dash?.mine && can('member') && !editing}
			<div class="flex gap-2">
				<button type="button" class="btn-secondary" onclick={startEdit}>Edit</button>
				<button type="button" class="btn-ghost text-red-600" onclick={remove}>Delete</button>
			</div>
		{/if}
	</div>
	{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>{/if}

	{#if editing}
		<div class="space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm">
			<div class="flex flex-wrap items-center gap-3">
				<input class="input w-full sm:w-64" aria-label="Dashboard name" required maxlength="100" bind:value={name} />
				<label class="flex items-center gap-1.5 text-slate-600"><input type="checkbox" bind:checked={shared} /> Share with everyone</label>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<span class="text-slate-600">Add:</span>
				{#each KINDS as k (k.kind)}
					<button type="button" class="btn-ghost px-2 py-1 text-xs ring-1 ring-inset ring-slate-200" disabled={k.kind === 'query' && !queries.length} onclick={() => add(k.kind)}
						>+ {k.label}</button
					>
				{/each}
				{#if !queries.length}<span class="text-xs text-slate-500">(save a query on the Queries page to chart it)</span>{/if}
			</div>
			<ul class="space-y-2">
				{#each draft as w, i (w.id)}
					<li class="flex flex-wrap items-center gap-2 rounded-md border border-slate-200 p-2">
						<span class="w-28 font-medium text-slate-700">{KINDS.find((k) => k.kind === w.kind)?.label}</span>
						<input class="input w-40 py-1 text-xs" placeholder="Title (optional)" aria-label="Title" maxlength="100" bind:value={w.title} />
						{#if w.kind === 'query'}
							<select class="input w-auto py-1 text-xs" aria-label="Saved query" bind:value={w.queryId}>
								{#each queries as q (q.id)}<option value={q.id}>{q.name}</option>{/each}
							</select>
							<select class="input w-auto py-1 text-xs" aria-label="Chart" bind:value={w.chart}>
								<option value="bar">Bars</option>
								<option value="pie">Pie</option>
								<option value="stacked">Stacked bars</option>
								<option value="trend">Trend</option>
								<option value="list">List</option>
								<option value="count">Count</option>
							</select>
							{#if w.chart !== 'trend' && w.chart !== 'list' && w.chart !== 'count'}
								<select class="input w-auto py-1 text-xs" aria-label="Group by" bind:value={w.groupBy}>
									{#each GROUPS as [v, l] (v)}<option value={v}>by {l}</option>{/each}
								</select>
							{/if}
							{#if w.chart === 'stacked'}
								<select class="input w-auto py-1 text-xs" aria-label="Stack by" bind:value={w.stackBy}>
									{#each GROUPS as [v, l] (v)}<option value={v}>split by {l}</option>{/each}
								</select>
							{/if}
						{:else if w.kind === 'burndown' || w.kind === 'velocity' || w.kind === 'cfd'}
							<select class="input w-auto py-1 text-xs" aria-label="Project" bind:value={w.projectId}>
								{#each projects as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
							</select>
						{:else if w.kind === 'notes'}
							<textarea class="input min-w-60 flex-1 py-1 text-xs" rows="2" maxlength="4000" aria-label="Notes" bind:value={w.text}></textarea>
						{/if}
						<select class="input w-auto py-1 text-xs" aria-label="Size" bind:value={w.size}>
							<option value="small">Small</option>
							<option value="wide">Wide</option>
						</select>
						<span class="ml-auto flex gap-1">
							<button type="button" class="btn-ghost px-2 py-1 text-xs" aria-label="Move up" onclick={() => moveW(i, -1)}>↑</button>
							<button type="button" class="btn-ghost px-2 py-1 text-xs" aria-label="Move down" onclick={() => moveW(i, 1)}>↓</button>
							<button type="button" class="btn-ghost px-2 py-1 text-xs text-red-600" onclick={() => (draft = draft.filter((x) => x.id !== w.id))}>Remove</button>
						</span>
					</li>
				{:else}
					<li class="text-slate-500">No widgets yet — add some above.</li>
				{/each}
			</ul>
			<div class="flex gap-2">
				<button type="button" class="btn-primary" disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save dashboard'}</button>
				<button type="button" class="btn-ghost" onclick={() => (editing = false)}>Cancel</button>
			</div>
		</div>
	{/if}

	{#if dash}
		<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
			{#each editing ? draft : dash.widgets as w (w.id)}
				<div class={w.size === 'wide' ? 'md:col-span-2' : ''}>
					<DashWidget widget={w} {queries} {projects} {people} />
				</div>
			{:else}
				{#if !editing}<p class="text-sm text-slate-500">This dashboard is empty.</p>{/if}
			{/each}
		</div>
	{/if}
</section>
