<script lang="ts">
	import { page } from '$app/state';
	import { afterNavigate, goto, replaceState } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import TaskFormModal from '$lib/components/TaskFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';
	import { cloneBody, descendants, templateFields } from '$lib/clone';
	import { childType, type WorkItemType } from '$lib/workItems';
	import { itemHref } from '$lib/itemFiles';

	type Task = components['schemas']['Task'];
	type Project = components['schemas']['Project'];

	const projectId = $derived(page.params.id ?? '');
	// Came from the board? Then "← Board" goes back in history, which keeps
	// the board's filters and scroll position.
	let cameFromBoard = false;
	afterNavigate((nav) => {
		cameFromBoard = nav.from?.url.pathname === `/projects/${projectId}/board`;
	});
	function backToBoard(e: MouseEvent) {
		if (!cameFromBoard || e.metaKey || e.ctrlKey || e.shiftKey) return;
		e.preventDefault();
		history.back();
	}
	const ref = $derived(page.params.ref ?? '');

	let project = $state<Project | null>(null);
	let tasks = $state<Task[]>([]);
	let members = $state<components['schemas']['ProjectMember'][]>([]);
	let people = $state<Record<string, string>>({});
	let milestones = $state<{ id: string; name: string }[]>([]);
	let sprints = $state<components['schemas']['Sprint'][]>([]);
	let areas = $state<{ id: string; path: string }[]>([]);
	let templates = $state<components['schemas']['ItemTemplate'][]>([]);
	let transferTargets = $state<{ id: string; name: string }[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let notice = $state<string | null>(null);
	let version = $state(0);

	async function loadTasks() {
		const all: Task[] = [];
		for (let p = 1; p <= 20; p++) {
			const { data } = await api.GET('/api/v1/tasks', { params: { query: { projectId, page: p, pageSize: 100 } } });
			all.push(...(data?.data ?? []));
			if (!data || data.data.length < 100) break;
		}
		tasks = all;
	}

	async function load() {
		loading = true;
		error = null;
		const [pr, mem, ppl, ms, sp, ar, tpl, prs] = await Promise.all([
			api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }),
			api.GET('/api/v1/project-members', { params: { query: { projectId, pageSize: 100 } } }),
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/milestones', { params: { query: { projectId, pageSize: 100 } } }),
			api.GET('/api/v1/sprints', { params: { query: { projectId } } }),
			api.GET('/api/v1/areas', { params: { query: { projectId } } }),
			api.GET('/api/v1/item-templates', { params: { query: { projectId } } }),
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }),
			loadTasks()
		]);
		if (pr.error || !pr.data) {
			error = pr.error?.message ?? 'Could not load the project.';
			loading = false;
			return;
		}
		project = pr.data;
		members = mem.data?.data ?? [];
		people = Object.fromEntries((ppl.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
		milestones = (ms.data?.data ?? []).map((m) => ({ id: m.id, name: m.name }));
		sprints = sp.data?.data ?? [];
		areas = (ar.data?.data ?? []).map((a) => ({ id: a.id, path: a.path }));
		templates = tpl.data?.data ?? [];
		transferTargets = (prs.data?.data ?? []).filter((p) => p.id !== projectId).map((p) => ({ id: p.id, name: p.name })).sort((a, b) => a.name.localeCompare(b.name));
		loading = false;
	}
	$effect(() => {
		if (projectId) load();
	});

	// The page is addressed by number; older links (?task=<id>) carry the id.
	const task = $derived(/^\d+$/.test(ref) ? tasks.find((t) => t.number === Number(ref)) : tasks.find((t) => t.id === ref));
	$effect(() => {
		if (task && !/^\d+$/.test(ref)) replaceState(itemHref(projectId, task.number), {});
	});
	const parent = $derived(task?.parentId ? tasks.find((t) => t.id === task!.parentId) : undefined);

	const assigneeOptions = $derived(members.filter((m) => !m.leftAt).map((m) => ({ id: m.personId, name: people[m.personId] ?? 'Unknown person' })));
	const sprintOptions = $derived(sprints.filter((s) => s.status !== 'completed' && s.status !== 'cancelled').map((s) => ({ id: s.id, name: s.name })));

	function saved(t: Task) {
		tasks = tasks.map((x) => (x.id === t.id ? t : x));
		version++;
	}

	// Adding a child opens the quick form on top of the page.
	let childOf = $state<Task | null>(null);
	let childKind = $state<WorkItemType>('task');

	let removing = $state(false);
	let removeBusy = $state(false);
	let removeError = $state<string | null>(null);
	async function remove() {
		if (!task) return;
		removeBusy = true;
		const { error: err } = await api.DELETE('/api/v1/tasks/{id}', { params: { path: { id: task.id } } });
		removeBusy = false;
		if (err) {
			removeError = err.message ?? 'Could not delete the item.';
			return;
		}
		goto(`/projects/${projectId}/board`);
	}
	async function copy(withChildren: boolean) {
		if (!task) return;
		const t = task;
		const post = (body: ReturnType<typeof cloneBody>) => api.POST('/api/v1/tasks', { body: body as components['schemas']['TaskCreate'] });
		const { data: c, error: err } = await post(cloneBody(t, { parentId: t.parentId }));
		if (err || !c) return (notice = err?.message ?? 'Could not copy the item.');
		if (withChildren) {
			const ids = new Map([[t.id, c.id]]);
			for (const k of descendants(t.id, tasks)) {
				const { data } = await post(cloneBody(k, { parentId: ids.get(k.parentId ?? '') ?? c.id, keepTitle: true }));
				if (data) ids.set(k.id, data.id);
			}
		}
		await loadTasks();
		goto(itemHref(projectId, c.number));
	}
	async function transfer(pid: string) {
		if (!task) return;
		const { data, error: err } = await api.POST('/api/v1/tasks/{id}/transfer', { params: { path: { id: task.id } }, body: { projectId: pid } });
		if (err || !data) return (notice = err?.message ?? 'Could not move the item.');
		goto(itemHref(pid, data.number));
	}
	async function saveTemplate(name: string) {
		if (!task) return;
		const { data, error: err } = await api.POST('/api/v1/item-templates', {
			body: { projectId, name, fields: templateFields(task) as components['schemas']['ItemTemplateFields'] }
		});
		if (err || !data) return (notice = err?.message ?? 'Could not save the template.');
		templates = [...templates, data].sort((a, b) => a.name.localeCompare(b.name));
		notice = `Template "${data.name}" saved. Pick it when adding an item.`;
	}
	async function deleteTemplate(id: string) {
		const t = templates.find((x) => x.id === id);
		if (!t || !confirm(`Delete the template "${t.name}"?`)) return;
		await api.DELETE('/api/v1/item-templates/{id}', { params: { path: { id } } });
		templates = templates.filter((x) => x.id !== id);
	}
</script>

<svelte:head><title>{task ? `${project?.key ?? ''}-${task.number} ${task.title}` : 'Item'} · Pyramid</title></svelte:head>

<div>
	<nav class="mb-4 flex flex-wrap items-center gap-2 text-sm text-slate-500" aria-label="Item">
		<a class="inline-flex min-h-6 items-center hover:text-slate-800 hover:underline" href="/projects/{projectId}/board" onclick={backToBoard}>← Board</a>
		{#if parent}
			<span aria-hidden="true">·</span>
			<span>Parent:</span>
			<a class="inline-flex min-h-6 items-center gap-1 hover:text-slate-800 hover:underline" href={itemHref(projectId, parent.number)}
				><TypeBadge type={parent.type} /> #{parent.number} {parent.title}</a
			>
		{/if}
	</nav>

	{#if notice}
		<div class="mb-4 flex items-center justify-between rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
			<span>{notice}</span><button type="button" class="text-slate-400 hover:text-slate-600" aria-label="Dismiss" onclick={() => (notice = null)}>✕</button>
		</div>
	{/if}

	{#if loading}
		<p class="text-sm text-slate-500">Loading…</p>
	{:else if error}
		<p class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
	{:else if !task}
		<p class="text-sm text-slate-600">This item doesn't exist in this project (it may have been deleted or moved). <a class="text-indigo-700 hover:underline" href="/projects/{projectId}/board">Back to the board</a></p>
	{:else}
		{#key task.id}
			<TaskFormModal
				variant="page"
				{projectId}
				projectKey={project?.key ?? ''}
				{task}
				{version}
				milestoneOptions={milestones}
				areaOptions={areas}
				customFields={project?.customFields ?? []}
				workflow={project?.workflow ?? []}
				itemTypes={project?.workItemTypes ?? []}
				{sprintOptions}
				{assigneeOptions}
				projectTasks={tasks}
				onAddSubtask={(p) => {
					childKind = childType(p.type);
					childOf = p;
				}}
				onClose={() => goto(`/projects/${projectId}/board`)}
				onSaved={saved}
				onDelete={() => {
					removeError = null;
					removing = true;
				}}
				{templates}
				{transferTargets}
				onCopy={copy}
				onTransfer={transfer}
				onSaveTemplate={saveTemplate}
				onDeleteTemplate={deleteTemplate}
			/>
		{/key}
	{/if}
</div>

{#if childOf}
	<TaskFormModal
		{projectId}
		milestoneOptions={milestones}
		areaOptions={areas}
		customFields={project?.customFields ?? []}
		workflow={project?.workflow ?? []}
		itemTypes={project?.workItemTypes ?? []}
		{sprintOptions}
		{assigneeOptions}
		projectTasks={tasks}
		initialParentId={childOf.id}
		initialMilestoneId={childOf.milestoneId ?? null}
		initialType={childKind}
		{templates}
		onClose={() => (childOf = null)}
		onSaved={async (t) => {
			childOf = null;
			await loadTasks();
			notice = `Added #${t.number} ${t.title}.`;
		}}
	/>
{/if}

<ConfirmDialog
	open={removing}
	title="Delete item"
	message={task ? `Delete "${task.title}"? This can't be undone.` : ''}
	confirmLabel={removeBusy ? 'Deleting…' : 'Delete'}
	danger
	error={removeError}
	onConfirm={remove}
	onCancel={() => (removing = false)}
/>
