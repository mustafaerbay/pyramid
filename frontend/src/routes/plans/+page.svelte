<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import { planBar } from '$lib/flow';
	import { dependencyLinks, laneTops } from '$lib/plans';
	import { teamFilter } from '$lib/teams';
	import { TYPE_OPTIONS, typeMeta } from '$lib/workItems';
	import ChipSelect from '$lib/components/ChipSelect.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type Sprint = components['schemas']['Sprint'];
	type Plan = components['schemas']['DeliveryPlan'];
	type Lane = components['schemas']['DeliveryPlanLane'];
	type Marker = components['schemas']['DeliveryPlanMarker'];
	type Team = components['schemas']['ProjectTeam'];
	type Area = components['schemas']['Area'];

	const SIZE = { head: 26, row: 30, pad: 6 };
	const MARK: Record<string, string> = {
		orange: '#ea580c',
		violet: '#7c3aed',
		sky: '#0284c7',
		red: '#dc2626',
		amber: '#d97706',
		emerald: '#059669',
		pink: '#db2777',
		slate: '#475569'
	};
	let projects = $state<{ id: string; name: string }[]>([]);
	let lanes = $state<Lane[]>([]);
	let types = $state<string[]>(['epic', 'feature']);
	let months = $state<3 | 6 | 12>(6);
	let markers = $state<Marker[]>([]);
	let items = $state<Task[]>([]);
	let sprints = $state<Record<string, Sprint[]>>({});
	let teams = $state<Record<string, Team[]>>({});
	let areas = $state<Record<string, Area[]>>({});
	let plans = $state<Plan[]>([]);
	let current = $state<Plan | null>(null);
	let saveName = $state('');
	let saveShared = $state(false);
	let banner = $state<string | null>(null);
	let showLines = $state(true);
	let addProject = $state('');
	let addTeam = $state('');

	const range = $derived.by(() => {
		const now = new Date();
		const from = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 1, 1);
		const to = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 1 + months, 1);
		return { from, to };
	});
	const pct = (iso: string) => ((Date.parse(iso) - range.from) / (range.to - range.from)) * 100;
	const monthTicks = $derived.by(() => {
		const out: { label: string; left: number }[] = [];
		const d = new Date(range.from);
		while (d.getTime() < range.to) {
			out.push({
				label: d.toLocaleDateString(undefined, { month: 'short', year: out.length === 0 || d.getUTCMonth() === 0 ? 'numeric' : undefined, timeZone: 'UTC' }),
				left: ((d.getTime() - range.from) / (range.to - range.from)) * 100
			});
			d.setUTCMonth(d.getUTCMonth() + 1);
		}
		return out;
	});
	const todayLeft = $derived(((Date.now() - range.from) / (range.to - range.from)) * 100);

	$effect(() => {
		api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }).then((r) => {
			projects = (r.data?.data ?? [])
				.filter((p) => p.status !== 'cancelled' && p.status !== 'completed')
				.map((p) => ({ id: p.id, name: p.name }))
				.sort((a, b) => a.name.localeCompare(b.name));
			if (lanes.length === 0 && !current) lanes = projects.slice(0, 4).map((p) => ({ projectId: p.id }));
		});
		api.GET('/api/v1/delivery-plans').then((r) => (plans = r.data?.data ?? []));
	});
	const projectIds = $derived([...new Set(lanes.map((l) => l.projectId))]);
	$effect(() => {
		const ids = projectIds;
		const t = types;
		if (ids.length === 0) {
			items = [];
			return;
		}
		api.POST('/api/v1/queries/run', { body: { projectIds: ids, types: t as Task['type'][], statuses: ['backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done'] } }).then(
			(r) => (items = r.data?.data ?? [])
		);
		for (const id of ids) loadProject(id);
	});
	function loadProject(id: string) {
		if (sprints[id]) return;
		sprints = { ...sprints, [id]: [] };
		api.GET('/api/v1/sprints', { params: { query: { projectId: id } } }).then((r) => (sprints = { ...sprints, [id]: r.data?.data ?? [] }));
		api.GET('/api/v1/project-teams', { params: { query: { projectId: id } } }).then((r) => (teams = { ...teams, [id]: r.data?.data ?? [] }));
		api.GET('/api/v1/areas', { params: { query: { projectId: id } } }).then((r) => (areas = { ...areas, [id]: r.data?.data ?? [] }));
	}
	$effect(() => {
		if (addProject) loadProject(addProject);
	});

	const rows = $derived(
		lanes.map((l) => {
			const project = projects.find((p) => p.id === l.projectId);
			const team = l.teamId ? (teams[l.projectId] ?? []).find((t) => t.id === l.teamId) : null;
			const inTeam = team ? teamFilter(team, areas[l.projectId] ?? []) : null;
			const mine = items.filter((t) => t.projectId === l.projectId && (!inTeam || inTeam(t as never)));
			return {
				key: `${l.projectId}|${l.teamId ?? ''}`,
				lane: l,
				name: project?.name ?? 'Project',
				team: team?.name ?? null,
				sprints: (sprints[l.projectId] ?? []).filter((s) => !l.teamId || !s.teamId || s.teamId === l.teamId),
				placed: mine.map((t) => ({ t, bar: planBar(t.startDate, t.dueDate, range) })).filter((x) => x.bar),
				undated: mine.filter((t) => !t.startDate && !t.dueDate)
			};
		})
	);
	const layout = $derived(laneTops(rows.map((r) => r.placed.length), SIZE));
	const placedMap = $derived(
		new Map(
			rows.flatMap((r, li) =>
				r.placed.map(({ t, bar }, i) => [
					t.id,
					{ left: bar!.left, width: bar!.width, y: layout.tops[li] + SIZE.head + i * SIZE.row + 12, start: t.startDate ?? t.dueDate ?? '', due: t.dueDate ?? t.startDate ?? '' }
				])
			)
		)
	);
	const links = $derived(showLines ? dependencyLinks(items, placedMap) : []);
	const byId = $derived(new Map(items.map((t) => [t.id, t])));
	const BAR: Record<string, string> = {
		epic: 'bg-orange-100 border-orange-300 text-orange-900',
		feature: 'bg-violet-100 border-violet-300 text-violet-900',
		story: 'bg-sky-100 border-sky-300 text-sky-900',
		bug: 'bg-red-100 border-red-300 text-red-900',
		task: 'bg-amber-50 border-amber-300 text-amber-900'
	};

	function addLane() {
		if (!addProject) return;
		const lane: Lane = { projectId: addProject, teamId: addTeam || null };
		if (!lanes.some((l) => l.projectId === lane.projectId && (l.teamId ?? null) === lane.teamId)) lanes = [...lanes, lane];
		addProject = addTeam = '';
	}
	function moveLane(i: number, d: number) {
		const j = i + d;
		if (j < 0 || j >= lanes.length) return;
		const next = [...lanes];
		[next[i], next[j]] = [next[j], next[i]];
		lanes = next;
	}
	function load(p: Plan) {
		current = p;
		lanes = p.lanes.map((l) => ({ ...l }));
		types = [...p.types];
		months = p.months as 3 | 6 | 12;
		markers = p.markers.map((m) => ({ ...m }));
		saveName = p.name;
		saveShared = p.shared;
	}
	function flash(m: string) {
		banner = m;
		setTimeout(() => (banner = null), 3000);
	}
	async function save(asNew: boolean) {
		const body = {
			name: saveName,
			shared: saveShared,
			lanes: $state.snapshot(lanes),
			types: types as Task['type'][],
			months,
			markers: markers.filter((m) => m.date && m.label.trim())
		};
		const res =
			current && current.mine && !asNew
				? await api.PATCH('/api/v1/delivery-plans/{id}', { params: { path: { id: current.id } }, body })
				: await api.POST('/api/v1/delivery-plans', { body });
		if (res.error || !res.data) return flash(res.error?.message ?? 'Could not save.');
		current = res.data;
		plans = (await api.GET('/api/v1/delivery-plans')).data?.data ?? plans;
		flash(`Saved "${res.data.name}".`);
	}
	async function remove() {
		if (!current || !confirm(`Delete the plan "${current.name}"?`)) return;
		const { error } = await api.DELETE('/api/v1/delivery-plans/{id}', { params: { path: { id: current.id } } });
		if (error) return flash(error.message ?? 'Could not delete.');
		plans = plans.filter((p) => p.id !== current!.id);
		current = null;
		saveName = '';
	}
</script>

<svelte:head><title>Delivery plans · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Delivery plans</h1>
		<p class="mt-1 text-sm text-slate-500">Big pieces of work across projects and teams on one timeline, with their sprints, dependencies and key dates.</p>
	</div>

	{#if plans.length}
		<div class="flex flex-wrap items-center gap-2 text-sm">
			<span class="text-slate-500">Saved:</span>
			{#each plans as p (p.id)}
				<button
					type="button"
					class="rounded-full px-3 py-1 ring-1 ring-inset {current?.id === p.id ? 'bg-slate-900 text-white ring-slate-900' : 'bg-white text-slate-700 ring-slate-200 hover:bg-slate-50'}"
					onclick={() => load(p)}>{p.name}{p.shared ? ' 👥' : ''}</button
				>
			{/each}
		</div>
	{/if}

	<div class="grid gap-4 rounded-lg border border-slate-200 bg-white p-4 lg:grid-cols-3">
		<div class="space-y-2 text-sm">
			<div class="font-medium text-slate-700">Lanes</div>
			<ul class="space-y-1">
				{#each rows as r, i (r.key)}
					<li class="flex items-center gap-1">
						<span class="min-w-0 flex-1 truncate">{r.name}{r.team ? ` · ${r.team}` : ''}</span>
						<button type="button" class="px-1 text-slate-400 hover:text-slate-700" aria-label="Move up" onclick={() => moveLane(i, -1)}>↑</button>
						<button type="button" class="px-1 text-slate-400 hover:text-slate-700" aria-label="Move down" onclick={() => moveLane(i, 1)}>↓</button>
						<button type="button" class="px-1 text-slate-400 hover:text-red-600" aria-label="Remove lane" onclick={() => (lanes = lanes.filter((_, j) => j !== i))}>✕</button>
					</li>
				{/each}
			</ul>
			<div class="flex flex-wrap gap-1.5">
				<select class="input w-auto py-1 text-xs" aria-label="Project for a new lane" bind:value={addProject}>
					<option value="">Project…</option>
					{#each projects as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
				</select>
				{#if addProject && (teams[addProject] ?? []).length}
					<select class="input w-auto py-1 text-xs" aria-label="Team" bind:value={addTeam}>
						<option value="">Whole project</option>
						{#each teams[addProject] as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
					</select>
				{/if}
				<button type="button" class="btn-ghost px-2 py-1 text-xs" disabled={!addProject} onclick={addLane}>+ Lane</button>
			</div>
		</div>
		<div class="space-y-3">
			<ChipSelect label="Types" options={TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label }))} bind:values={types} />
			<div class="flex flex-wrap items-center gap-3 text-sm">
				<select class="input w-auto" aria-label="Time span" bind:value={months}>
					<option value={3}>3 months</option>
					<option value={6}>6 months</option>
					<option value={12}>12 months</option>
				</select>
				<label class="flex items-center gap-1.5 text-slate-600"><input type="checkbox" bind:checked={showLines} /> Dependency lines</label>
			</div>
		</div>
		<div class="space-y-2 text-sm">
			<div class="font-medium text-slate-700">Markers</div>
			{#each markers as m, i (i)}
				<div class="flex flex-wrap items-center gap-1.5">
					<input type="date" class="input w-auto py-1 text-xs" aria-label="Marker date" bind:value={m.date} />
					<input class="input w-32 py-1 text-xs" aria-label="Marker label" maxlength="60" placeholder="Code freeze" bind:value={m.label} />
					<select class="input w-auto py-1 text-xs" aria-label="Marker colour" bind:value={m.color}>
						{#each Object.keys(MARK) as c (c)}<option value={c}>{c}</option>{/each}
					</select>
					<button type="button" class="px-1 text-slate-400 hover:text-red-600" aria-label="Remove marker" onclick={() => (markers = markers.filter((_, j) => j !== i))}>✕</button>
				</div>
			{/each}
			<button
				type="button"
				class="btn-ghost px-2 py-1 text-xs"
				onclick={() => (markers = [...markers, { date: new Date().toISOString().slice(0, 10), label: '', color: 'red' }])}>+ Marker</button
			>
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
			<input class="input w-full sm:w-56" placeholder="Name this plan" aria-label="Plan name" required maxlength="100" bind:value={saveName} />
			<label class="flex items-center gap-1.5 text-slate-600"><input type="checkbox" bind:checked={saveShared} /> Share with everyone</label>
			<button type="submit" class="btn-primary">{current?.mine ? 'Save' : 'Save plan'}</button>
			{#if current?.mine}
				<button type="button" class="btn-secondary" onclick={() => save(true)}>Save as new</button>
				<button type="button" class="btn-ghost text-red-600" onclick={remove}>Delete</button>
			{/if}
			{#if banner}<span class="text-slate-600" role="status">{banner}</span>{/if}
		</form>
	{/if}

	<div class="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
		<div class="min-w-[44rem] rounded-lg border border-slate-200 bg-white">
			<div class="relative ml-40 h-7 border-b border-slate-200 text-[11px] text-slate-500">
				{#each monthTicks as m (m.left)}
					<span class="absolute top-1.5 border-l border-slate-200 pl-1" style="left:{m.left}%">{m.label}</span>
				{/each}
				{#each markers.filter((m) => m.date && m.label.trim()) as m, i (i)}
					{@const left = pct(m.date)}
					{#if left >= 0 && left <= 100}
						<span class="absolute top-0 z-20 -translate-x-1/2 whitespace-nowrap rounded px-1 text-[10px] font-medium text-white" style="left:{left}%; background:{MARK[m.color]}" title="{m.label} · {m.date}">{m.label}</span>
					{/if}
				{/each}
			</div>
			{#if rows.length === 0}
				<p class="p-6 text-center text-sm text-slate-500">Add one or more lanes.</p>
			{:else}
				<div class="flex">
					<div class="w-40 shrink-0">
						{#each rows as r, li (r.key)}
							<div class="border-b border-r border-slate-100 p-2 text-sm" style="height:{(layout.tops[li + 1] ?? layout.height) - layout.tops[li]}px">
								<a href="/projects/{r.lane.projectId}" class="font-medium text-slate-800 hover:underline">{r.name}</a>
								{#if r.team}<div class="text-xs text-slate-600">{r.team}</div>{/if}
								{#if r.undated.length}<div class="text-xs text-slate-500">{r.undated.length} without dates</div>{/if}
							</div>
						{/each}
					</div>
					<div class="relative flex-1" style="height:{layout.height}px">
						{#each monthTicks as m (m.left)}<span class="absolute inset-y-0 border-l border-slate-100" style="left:{m.left}%"></span>{/each}
						{#each rows as r, li (r.key)}
							<span class="absolute inset-x-0 border-b border-slate-100" style="top:{(layout.tops[li + 1] ?? layout.height) - 1}px"></span>
							{#each r.sprints as s (s.id)}
								{@const b = planBar(s.startDate, s.endDate, range)}
								{#if b}
									<span
										class="absolute h-5 truncate rounded-b bg-slate-100 px-1 text-[10px] leading-5 text-slate-500"
										style="left:{b.left}%; width:{b.width}%; top:{layout.tops[li]}px"
										title="{s.name}: {s.startDate} – {s.endDate}">{s.name}</span
									>
								{/if}
							{/each}
							{#each r.placed as { t, bar }, i (t.id)}
								{@const waits = t.blockedByIds.map((id) => byId.get(id)).filter(Boolean)}
								<a
									href="/projects/{t.projectId}/items/{t.id}"
									class="absolute z-10 flex h-6 items-center gap-1 overflow-hidden rounded border px-1.5 text-xs {BAR[t.type] ?? BAR.task} {t.status === 'done' ? 'opacity-60' : ''}"
									style="left:{bar!.left}%; width:{bar!.width}%; top:{layout.tops[li] + SIZE.head + i * SIZE.row}px"
									title="{typeMeta(t.type).label} #{t.number} {t.title} · {t.startDate ?? '?'} → {t.dueDate ?? '?'} · {t.status.replace('_', ' ')}{waits.length
										? ` · waits on ${waits.map((w) => `#${w!.number}`).join(', ')}`
										: ''}"
								>
									<span class="truncate">{t.title}</span>
								</a>
							{/each}
						{/each}
						{#if todayLeft >= 0 && todayLeft <= 100}<span class="absolute inset-y-0 z-10 border-l-2 border-red-400" style="left:{todayLeft}%" title="Today"></span>{/if}
						{#each markers.filter((m) => m.date && m.label.trim()) as m, i (i)}
							{@const left = pct(m.date)}
							{#if left >= 0 && left <= 100}
								<span class="absolute inset-y-0 z-10 border-l-2 border-dashed" style="left:{left}%; border-color:{MARK[m.color]}" title="{m.label} · {m.date}"></span>
							{/if}
						{/each}
						{#if links.length}
							<svg class="pointer-events-none absolute inset-0 z-20 h-full w-full" viewBox="0 0 100 {layout.height}" preserveAspectRatio="none" aria-hidden="true">
								{#each links as l (l.from + l.to)}
									{@const mx = (l.x1 + l.x2) / 2}
									<path
										d="M {l.x1} {l.y1} C {Math.max(mx, l.x1 + 1.5)} {l.y1}, {Math.min(mx, l.x2 - 1.5)} {l.y2}, {l.x2} {l.y2}"
										fill="none"
										stroke={l.conflict ? '#dc2626' : '#475569'}
										stroke-width="1.5"
										stroke-dasharray={l.conflict ? '4 3' : undefined}
										vector-effect="non-scaling-stroke"
									/>
								{/each}
							</svg>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>
	{#if links.some((l) => l.conflict)}
		<p class="text-xs text-red-700">Dashed red lines: the item starts before the work it waits on is due.</p>
	{/if}

	{#if rows.some((r) => r.undated.length)}
		<details class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
			<summary class="cursor-pointer font-medium">Items without dates</summary>
			<ul class="mt-2 space-y-1">
				{#each rows.flatMap((r) => r.undated.map((t) => ({ t, lane: r.name + (r.team ? ` · ${r.team}` : ''), key: r.key }))) as { t, lane, key } (key + t.id)}
					<li><TypeBadge type={t.type} label={t.customTypeLabel} color={t.customTypeColor} /> <a class="hover:underline" href="/projects/{t.projectId}/items/{t.id}">{t.title}</a> <span class="text-slate-500">· {lane}</span></li>
				{/each}
			</ul>
		</details>
	{/if}
</section>
