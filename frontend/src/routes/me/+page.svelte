<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { session } from '$lib/auth.svelte';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';

	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];
	type Project = components['schemas']['Project'];
	type Milestone = components['schemas']['Milestone'];

	// Starts on the signed-in person; the picker still lets you look at
	// someone else's tasks.
	let viewingAsId = $state(session.me?.person.id ?? '');

	let tasks = $state<Task[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let hasLoadedOnce = $state(false);

	let projects = $state<Record<string, Project>>({});
	let milestones = $state<Record<string, Milestone>>({});

	let statusFilter = $state<TaskStatus | ''>('');

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function loadLookups() {
		const [projectsRes, milestonesRes] = await Promise.all([
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/milestones', { params: { query: { page: 1, pageSize: 100 } } })
		]);
		if (projectsRes.data) {
			const map: Record<string, Project> = {};
			for (const p of projectsRes.data.data) map[p.id] = p;
			projects = map;
		}
		if (milestonesRes.data) {
			const map: Record<string, Milestone> = {};
			for (const m of milestonesRes.data.data) map[m.id] = m;
			milestones = map;
		}
	}

	async function loadTasks() {
		if (!viewingAsId) {
			tasks = [];
			return;
		}
		loading = true;
		error = null;
		const { data, error: err } = await api.GET('/api/v1/tasks', {
			params: { query: { assigneeId: viewingAsId, pageSize: 100 } }
		});
		if (err) {
			error = err.message ?? 'Failed to load tasks';
		} else if (data) {
			tasks = data.data;
		}
		loading = false;
		hasLoadedOnce = true;
	}

	function handleViewingAsChange() {
		loadTasks();
	}

	const filtered = $derived(
		statusFilter ? tasks.filter((t) => t.status === statusFilter) : tasks
	);

	// Grouped by project — key sort so the groups line up with the Projects
	// list ordering elsewhere in the app.
	type Group = { projectId: string; key: string; name: string; tasks: Task[] };
	const groups = $derived(
		(() => {
			const byProject = new Map<string, Task[]>();
			for (const t of filtered) {
				const list = byProject.get(t.projectId) ?? [];
				list.push(t);
				byProject.set(t.projectId, list);
			}
			const out: Group[] = [];
			for (const [projectId, list] of byProject) {
				const p = projects[projectId];
				out.push({ projectId, key: p?.key ?? '—', name: p?.name ?? 'Unknown project', tasks: list });
			}
			out.sort((a, b) => a.key.localeCompare(b.key));
			return out;
		})()
	);

	function taskStatusTone(status: TaskStatus): BadgeTone {
		switch (status) {
			case 'backlog':
				return 'neutral';
			case 'todo':
				return 'neutralMuted';
			case 'in_progress':
				return 'warning';
			case 'in_review':
				return 'info';
			case 'blocked':
				return 'danger';
			case 'done':
				return 'success';
			case 'cancelled':
				return 'neutralFaint';
		}
	}

	function taskPriorityTone(priority: Task['priority']): BadgeTone {
		switch (priority) {
			case 'low':
				return 'neutral';
			case 'medium':
				return 'info';
			case 'high':
				return 'warning';
			case 'urgent':
				return 'danger';
		}
	}

	async function quickUpdateStatus(task: Task, status: TaskStatus) {
		const { data, error: err } = await api.PATCH('/api/v1/tasks/{id}', {
			params: { path: { id: task.id } },
			body: { status }
		});
		if (err) {
			flash('error', err.message ?? 'Failed to update task status.');
			return;
		}
		if (data) {
			tasks = tasks.map((t) => (t.id === data.id ? data : t));
		}
	}

	onMount(() => {
		loadLookups();
		if (viewingAsId) loadTasks();
	});
</script>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">My Tasks</h1>
		<p class="mt-0.5 text-sm text-slate-500">Every task assigned to you, across all projects.</p>
	</div>

	{#if banner}
		<div
			class="rounded-md border px-3 py-2 text-sm {banner.kind === 'success'
				? 'border-emerald-200 bg-emerald-50 text-emerald-700'
				: 'border-red-200 bg-red-50 text-red-700'}"
		>
			{banner.text}
		</div>
	{/if}

	<div class="max-w-xs">
		<PersonSelect
			bind:value={viewingAsId}
			label="Viewing as"
			emptyLabel="Select a person…"
			onchange={handleViewingAsChange}
		/>
	</div>

	{#if !viewingAsId}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			Pyramid doesn't have logins yet — pick your name above to see your tasks.
		</div>
	{:else if loading}
		<p class="text-slate-500">Loading…</p>
	{:else if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
			<p class="font-medium">Couldn't reach the API.</p>
			<p class="text-sm">{error}</p>
		</div>
	{:else if hasLoadedOnce && tasks.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			No tasks assigned to you right now.
		</div>
	{:else if tasks.length > 0}
		<div class="flex flex-wrap items-center gap-3">
			<select
				class="input w-auto"
				value={statusFilter}
				onchange={(e) => (statusFilter = e.currentTarget.value as TaskStatus | '')}
			>
				<option value="">All statuses</option>
				<option value="backlog">Backlog</option>
				<option value="todo">To do</option>
				<option value="in_progress">In progress</option>
				<option value="in_review">In review</option>
				<option value="blocked">Blocked</option>
				<option value="done">Done</option>
				<option value="cancelled">Cancelled</option>
			</select>
		</div>

		{#if groups.length === 0}
			<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
				No tasks match the current filter.
			</div>
		{:else}
			<div class="space-y-4">
				{#each groups as group (group.projectId)}
					<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
						<div class="flex flex-wrap items-center gap-x-2 gap-y-0.5 border-b border-slate-200 bg-slate-50 px-4 py-2.5">
							<a
								href="/projects/{group.projectId}"
								class="text-sm font-semibold text-slate-800 hover:underline"
							>
								{group.name}
							</a>
							<span class="font-mono text-xs text-slate-400">{group.key}</span>
							<span class="text-xs text-slate-500">
								{group.tasks.length}
								{group.tasks.length === 1 ? 'task' : 'tasks'}
							</span>
						</div>
						<ul class="divide-y divide-slate-100 md:hidden">
							{#each group.tasks as task (task.id)}
								<li class="space-y-2 px-4 py-3 text-sm">
									<div>
										<a href="/projects/{group.projectId}" class="font-medium text-slate-800 hover:underline">
											{task.title}
										</a>
										{#if task.milestoneId && milestones[task.milestoneId]}
											<p class="text-xs text-slate-500">{milestones[task.milestoneId].name}</p>
										{/if}
									</div>
									<div class="flex items-center gap-2">
										<Badge tone={taskPriorityTone(task.priority)}>{task.priority}</Badge>
										{#if task.dueDate}<span class="text-xs text-slate-500">Due {task.dueDate}</span>{/if}
										<select
											class="ml-auto min-h-10 rounded-md border-0 px-3 text-sm font-medium {badgeToneClass[
												taskStatusTone(task.status)
											]}"
											aria-label="Status of {task.title}"
											value={task.status}
											onchange={(e) => quickUpdateStatus(task, e.currentTarget.value as TaskStatus)}
										>
											<option value="backlog">backlog</option>
											<option value="todo">todo</option>
											<option value="in_progress">in progress</option>
											<option value="in_review">in review</option>
											<option value="blocked">blocked</option>
											<option value="done">done</option>
											<option value="cancelled">cancelled</option>
										</select>
									</div>
								</li>
							{/each}
						</ul>
						<table class="hidden w-full text-left text-sm md:table">
							<thead class="border-b border-slate-200 text-xs text-slate-500">
								<tr>
									<th class="px-4 py-2 font-medium">Task</th>
									<th class="px-4 py-2 font-medium">Priority</th>
									<th class="px-4 py-2 font-medium">Status</th>
									<th class="px-4 py-2 font-medium">Due</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-slate-100">
								{#each group.tasks as task (task.id)}
									<tr class="hover:bg-slate-50">
										<td class="px-4 py-2.5">
											<a
												href="/projects/{group.projectId}"
												class="font-medium text-slate-800 hover:underline"
											>
												{task.title}
											</a>
											{#if task.milestoneId && milestones[task.milestoneId]}
												<span class="ml-1.5 text-xs text-slate-400">
													· {milestones[task.milestoneId].name}
												</span>
											{/if}
										</td>
										<td class="px-4 py-2.5">
											<Badge tone={taskPriorityTone(task.priority)}>{task.priority}</Badge>
										</td>
										<td class="px-4 py-2.5">
											<select
												class="rounded-full border-0 px-2 py-0.5 text-xs font-medium {badgeToneClass[
													taskStatusTone(task.status)
												]}"
												value={task.status}
												onchange={(e) =>
													quickUpdateStatus(task, e.currentTarget.value as TaskStatus)}
											>
												<option value="backlog">backlog</option>
												<option value="todo">todo</option>
												<option value="in_progress">in progress</option>
												<option value="in_review">in review</option>
												<option value="blocked">blocked</option>
												<option value="done">done</option>
												<option value="cancelled">cancelled</option>
											</select>
										</td>
										<td class="px-4 py-2.5 text-xs text-slate-400">
											{task.dueDate ?? ''}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</section>
