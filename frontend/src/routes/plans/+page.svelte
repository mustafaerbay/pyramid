<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { planBar } from '$lib/flow';
	import { TYPE_OPTIONS, typeMeta } from '$lib/workItems';
	import ChipSelect from '$lib/components/ChipSelect.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type Sprint = components['schemas']['Sprint'];

	const DAY = 86_400_000;
	let projects = $state<{ id: string; name: string }[]>([]);
	let projectIds = $state<string[]>([]);
	let types = $state<string[]>(['epic', 'feature']);
	let months = $state(6);
	let items = $state<Task[]>([]);
	let sprints = $state<Record<string, Sprint[]>>({});

	// Range: start of last month → `months` ahead.
	const range = $derived.by(() => {
		const now = new Date();
		const from = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 1, 1);
		const to = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - 1 + months, 1);
		return { from, to };
	});
	const monthTicks = $derived.by(() => {
		const out: { label: string; left: number }[] = [];
		const d = new Date(range.from);
		while (d.getTime() < range.to) {
			out.push({
				label: d.toLocaleDateString(undefined, {
					month: 'short',
					year: out.length === 0 || d.getUTCMonth() === 0 ? 'numeric' : undefined,
					timeZone: 'UTC'
				}),
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
			if (projectIds.length === 0) projectIds = projects.slice(0, 4).map((p) => p.id);
		});
	});
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
		for (const id of ids)
			if (!sprints[id])
				api.GET('/api/v1/sprints', { params: { query: { projectId: id } } }).then((r) => (sprints = { ...sprints, [id]: r.data?.data ?? [] }));
	});

	const lanes = $derived(
		projectIds
			.map((id) => projects.find((p) => p.id === id))
			.filter((p): p is { id: string; name: string } => !!p)
			.map((p) => {
				const mine = items.filter((t) => t.projectId === p.id);
				return {
					...p,
					placed: mine.map((t) => ({ t, bar: planBar(t.startDate, t.dueDate, range) })).filter((x) => x.bar),
					undated: mine.filter((t) => !t.startDate && !t.dueDate)
				};
			})
	);
	const byId = $derived(new Map(items.map((t) => [t.id, t])));
	const BAR: Record<string, string> = {
		epic: 'bg-orange-100 border-orange-300 text-orange-900',
		feature: 'bg-violet-100 border-violet-300 text-violet-900',
		story: 'bg-sky-100 border-sky-300 text-sky-900',
		bug: 'bg-red-100 border-red-300 text-red-900',
		task: 'bg-amber-50 border-amber-300 text-amber-900'
	};
</script>

<svelte:head><title>Delivery plan · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Delivery plan</h1>
		<p class="mt-1 text-sm text-slate-500">Big pieces of work across projects on one timeline, with each project's sprints.</p>
	</div>

	<div class="grid gap-3 rounded-lg border border-slate-200 bg-white p-4 sm:grid-cols-3">
		<ChipSelect label="Projects" options={projects.map((p) => ({ value: p.id, label: p.name }))} bind:values={projectIds} />
		<ChipSelect label="Types" options={TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label }))} bind:values={types} />
		<select class="input" aria-label="Time span" bind:value={months}>
			<option value={3}>3 months</option>
			<option value={6}>6 months</option>
			<option value={12}>12 months</option>
		</select>
	</div>

	<div class="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
		<div class="min-w-[44rem] rounded-lg border border-slate-200 bg-white">
			<div class="relative ml-40 h-7 border-b border-slate-200 text-[11px] text-slate-500">
				{#each monthTicks as m (m.left)}
					<span class="absolute top-1.5 border-l border-slate-200 pl-1" style="left:{m.left}%">{m.label}</span>
				{/each}
			</div>
			{#each lanes as lane (lane.id)}
				<div class="flex border-b border-slate-100 last:border-b-0">
					<div class="w-40 shrink-0 border-r border-slate-100 p-2 text-sm">
						<a href="/projects/{lane.id}" class="font-medium text-slate-800 hover:underline">{lane.name}</a>
						{#if lane.undated.length}<div class="text-xs text-slate-500">{lane.undated.length} without dates</div>{/if}
					</div>
					<div class="relative min-h-16 flex-1 py-2" style="height:{Math.max(lane.placed.length, 1) * 30 + 28}px">
						{#each monthTicks as m (m.left)}<span class="absolute inset-y-0 border-l border-slate-100" style="left:{m.left}%"></span>{/each}
						{#each sprints[lane.id] ?? [] as s (s.id)}
							{@const b = planBar(s.startDate, s.endDate, range)}
							{#if b}
								<span class="absolute top-0 h-5 truncate rounded-b bg-slate-100 px-1 text-[10px] leading-5 text-slate-500" style="left:{b.left}%; width:{b.width}%" title="{s.name}: {s.startDate} – {s.endDate}">{s.name}</span>
							{/if}
						{/each}
						{#if todayLeft >= 0 && todayLeft <= 100}<span class="absolute inset-y-0 z-10 border-l-2 border-red-400" style="left:{todayLeft}%" title="Today"></span>{/if}
						{#each lane.placed as { t, bar }, i (t.id)}
							{@const waits = t.blockedByIds.map((id) => byId.get(id)).filter(Boolean)}
							<a
								href="/projects/{t.projectId}?task={t.id}"
								class="absolute flex h-6 items-center gap-1 overflow-hidden rounded border px-1.5 text-xs {BAR[t.type] ?? BAR.task} {t.status === 'done' ? 'opacity-60' : ''}"
								style="left:{bar!.left}%; width:{bar!.width}%; top:{26 + i * 30}px"
								title="{typeMeta(t.type).label} #{t.number} {t.title} · {t.startDate ?? '?'} → {t.dueDate ?? '?'} · {t.status.replace('_', ' ')}{waits.length ? ` · waits on ${waits.map((w) => `#${w!.number}`).join(', ')}` : ''}"
							>
								{#if waits.length}<span aria-hidden="true">⛓</span>{/if}<span class="truncate">{t.title}</span>
							</a>
						{/each}
					</div>
				</div>
			{:else}
				<p class="p-6 text-center text-sm text-slate-500">Pick one or more projects.</p>
			{/each}
		</div>
	</div>

	{#if lanes.some((l) => l.undated.length)}
		<details class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
			<summary class="cursor-pointer font-medium">Items without dates</summary>
			<ul class="mt-2 space-y-1">
				{#each lanes.flatMap((l) => l.undated.map((t) => ({ t, project: l.name }))) as { t, project } (t.id)}
					<li><TypeBadge type={t.type} /> <a class="hover:underline" href="/projects/{t.projectId}?task={t.id}">{t.title}</a> <span class="text-slate-500">· {project}</span></li>
				{/each}
			</ul>
		</details>
	{/if}
</section>
