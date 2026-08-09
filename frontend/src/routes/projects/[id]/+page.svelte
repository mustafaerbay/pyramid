<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ProjectFormModal from '$lib/components/ProjectFormModal.svelte';
	import ProjectMemberFormModal from '$lib/components/ProjectMemberFormModal.svelte';
	import AddTeamMembersModal from '$lib/components/AddTeamMembersModal.svelte';
	import ContributionFormModal from '$lib/components/ContributionFormModal.svelte';
	import KickoffFormModal from '$lib/components/KickoffFormModal.svelte';
	import MilestoneFormModal from '$lib/components/MilestoneFormModal.svelte';
	import MilestoneTimeline from '$lib/components/MilestoneTimeline.svelte';
	import TaskFormModal from '$lib/components/TaskFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import { badgeToneClass, healthDotClass as healthDotClassMap, type BadgeTone } from '$lib/badgeTones';
	import { daysUntil } from '$lib/dateMath';

	type Project = components['schemas']['Project'];
	type ProjectMember = components['schemas']['ProjectMember'];
	type Person = components['schemas']['Person'];
	type Contribution = components['schemas']['Contribution'];
	type Milestone = components['schemas']['Milestone'];
	type ProjectKickoff = components['schemas']['ProjectKickoff'];
	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];

	const projectId = page.params.id ?? '';

	let project = $state<Project | null>(null);
	let loadingProject = $state(true);
	let projectError = $state<string | null>(null);

	let members = $state<ProjectMember[]>([]);
	let loadingMembers = $state(true);
	let membersError = $state<string | null>(null);

	let people = $state<Record<string, Person>>({});

	let editingProject = $state(false);

	let memberModalMode = $state<'create' | 'edit' | null>(null);
	let editingMember = $state<ProjectMember | null>(null);
	let showAddMembers = $state(false);

	let removingMember = $state<ProjectMember | null>(null);
	let removing = $state(false);
	let removeError = $state<string | null>(null);

	// A project has at most one kickoff (project_id is unique at the DB
	// level), so there's no list of these to manage — just a single nullable
	// record, unlike members/milestones/tasks.
	let kickoff = $state<ProjectKickoff | null>(null);
	let loadingKickoff = $state(true);
	let kickoffError = $state<string | null>(null);

	let showKickoffModal = $state(false);

	let removingKickoff = $state(false);
	let removingKickoffBusy = $state(false);
	let removingKickoffError = $state<string | null>(null);

	// Per-action-item "Convert to task" busy indicator (index into
	// kickoff.actionItems), so only the clicked row shows a pending state.
	let convertingActionItemIndex = $state<number | null>(null);
	let convertActionItemError = $state<string | null>(null);

	let milestones = $state<Milestone[]>([]);
	let loadingMilestones = $state(true);
	let milestonesError = $state<string | null>(null);

	let milestoneModalMode = $state<'create' | 'edit' | null>(null);
	let editingMilestone = $state<Milestone | null>(null);

	let removingMilestone = $state<Milestone | null>(null);
	let removingMilestoneBusy = $state(false);
	let removingMilestoneError = $state<string | null>(null);

	let tasks = $state<Task[]>([]);
	let loadingTasks = $state(true);
	let tasksError = $state<string | null>(null);

	let taskModalMode = $state<'create' | 'edit' | null>(null);
	let editingTask = $state<Task | null>(null);
	// Prefills the milestone picker when "+ Add task" is clicked from inside
	// a specific milestone's group, rather than the section-level button.
	let taskModalMilestoneId = $state<string | null>(null);

	let removingTask = $state<Task | null>(null);
	let removingTaskBusy = $state(false);
	let removingTaskError = $state<string | null>(null);

	let taskStatusFilter = $state<TaskStatus | ''>('');
	// '' = everyone, '__unassigned__' = the unassigned sentinel, else a personId.
	let taskAssigneeFilter = $state('');

	let contributionsTotal = $state<number | null>(null);
	let recentContributions = $state<Contribution[]>([]);
	let loadingActivity = $state(true);
	let showLogContribution = $state(false);

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function loadProject() {
		loadingProject = true;
		projectError = null;
		const { data, error: err } = await api.GET('/api/v1/projects/{id}', {
			params: { path: { id: projectId } }
		});
		if (err) {
			projectError = err.message ?? 'Failed to load project';
		} else if (data) {
			project = data;
		}
		loadingProject = false;
	}

	async function loadMembers() {
		loadingMembers = true;
		membersError = null;
		const { data, error: err } = await api.GET('/api/v1/project-members', {
			params: { query: { projectId, pageSize: 100 } }
		});
		if (err) {
			membersError = err.message ?? 'Failed to load the team';
		} else if (data) {
			members = data.data;
		}
		loadingMembers = false;
	}

	async function loadPeople() {
		const { data } = await api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } });
		if (data) {
			const map: Record<string, Person> = {};
			for (const p of data.data) map[p.id] = p;
			people = map;
		}
	}

	async function loadKickoff() {
		loadingKickoff = true;
		kickoffError = null;
		const { data, error: err } = await api.GET('/api/v1/project-kickoffs', {
			params: { query: { projectId, pageSize: 1 } }
		});
		if (err) {
			kickoffError = err.message ?? 'Failed to load the kickoff.';
		} else if (data) {
			kickoff = data.data[0] ?? null;
		}
		loadingKickoff = false;
	}

	async function loadMilestones() {
		loadingMilestones = true;
		milestonesError = null;
		const { data, error: err } = await api.GET('/api/v1/milestones', {
			params: { query: { projectId, pageSize: 100 } }
		});
		if (err) {
			milestonesError = err.message ?? 'Failed to load milestones';
		} else if (data) {
			milestones = data.data;
		}
		loadingMilestones = false;
	}

	async function loadTasks() {
		loadingTasks = true;
		tasksError = null;
		const { data, error: err } = await api.GET('/api/v1/tasks', {
			params: { query: { projectId, pageSize: 100 } }
		});
		if (err) {
			tasksError = err.message ?? 'Failed to load tasks';
		} else if (data) {
			tasks = data.data;
		}
		loadingTasks = false;
	}

	async function loadActivity() {
		loadingActivity = true;
		const { data } = await api.GET('/api/v1/contributions', {
			params: { query: { projectId, page: 1, pageSize: 5 } }
		});
		if (data) {
			recentContributions = data.data;
			contributionsTotal = data.pagination.total;
		}
		loadingActivity = false;
	}

	// Active members first, then departed ones at the bottom.
	const sortedMembers = $derived(
		[...members].sort((a, b) => {
			if (!a.leftAt && b.leftAt) return -1;
			if (a.leftAt && !b.leftAt) return 1;
			return 0;
		})
	);

	const existingPersonIds = $derived(members.filter((m) => !m.leftAt).map((m) => m.personId));

	const activeMembers = $derived(members.filter((m) => !m.leftAt));

	const avgAllocation = $derived(
		activeMembers.length === 0
			? 0
			: Math.round(
					activeMembers.reduce((sum, m) => sum + m.allocationPct, 0) / activeMembers.length
				)
	);

	const milestoneOptions = $derived(milestones.map((m) => ({ id: m.id, name: m.name })));

	const assigneeOptions = $derived(
		activeMembers.map((m) => ({ id: m.personId, name: personLabel(m.personId) }))
	);

	// Tasks grouped by milestone (same order as the milestones list — due
	// date ascending, undated last), then a trailing "No milestone" group for
	// project-level backlog tasks. A milestone with zero tasks ever created
	// against it is omitted entirely rather than showing an empty group —
	// use the "+ Add task" button (section-level or per-group) to start one.
	// Progress counts (done/total) always reflect ALL of that milestone's
	// tasks, not just the ones matching the current filters, so the filter
	// can't make progress look better or worse than it really is.
	const taskGroups = $derived(
		(() => {
			const matchesFilters = (t: Task) => {
				if (taskStatusFilter && t.status !== taskStatusFilter) return false;
				if (taskAssigneeFilter === '__unassigned__' && t.assigneeIds.length > 0) return false;
				if (
					taskAssigneeFilter &&
					taskAssigneeFilter !== '__unassigned__' &&
					!t.assigneeIds.includes(taskAssigneeFilter)
				)
					return false;
				return true;
			};

			type Group = {
				key: string;
				milestoneId: string | null;
				name: string;
				totalCount: number;
				doneCount: number;
				tasks: Task[];
			};
			const groups: Group[] = [];

			for (const m of milestones) {
				const all = tasks.filter((t) => t.milestoneId === m.id);
				if (all.length === 0) continue;
				groups.push({
					key: m.id,
					milestoneId: m.id,
					name: m.name,
					totalCount: all.length,
					doneCount: all.filter((t) => t.status === 'done').length,
					tasks: all.filter(matchesFilters)
				});
			}

			const unassignedToMilestone = tasks.filter((t) => !t.milestoneId);
			if (unassignedToMilestone.length > 0) {
				groups.push({
					key: '__none__',
					milestoneId: null,
					name: 'No milestone',
					totalCount: unassignedToMilestone.length,
					doneCount: unassignedToMilestone.filter((t) => t.status === 'done').length,
					tasks: unassignedToMilestone.filter(matchesFilters)
				});
			}

			return groups;
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

	function openAddTask(milestoneId: string | null) {
		editingTask = null;
		taskModalMilestoneId = milestoneId;
		taskModalMode = 'create';
	}

	function openEditTask(t: Task) {
		editingTask = t;
		taskModalMilestoneId = null;
		taskModalMode = 'edit';
	}

	function closeTaskModal() {
		taskModalMode = null;
		editingTask = null;
		taskModalMilestoneId = null;
	}

	function handleTaskSaved(saved: Task) {
		const wasEdit = editingTask !== null;
		closeTaskModal();
		flash('success', wasEdit ? `Updated "${saved.title}".` : `Added task "${saved.title}".`);
		loadTasks();
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

	function askRemoveTask(t: Task) {
		closeTaskModal();
		removingTaskError = null;
		removingTask = t;
	}

	async function confirmRemoveTask() {
		if (!removingTask) return;
		const target = removingTask;
		removingTaskBusy = true;
		removingTaskError = null;

		const { error: err } = await api.DELETE('/api/v1/tasks/{id}', {
			params: { path: { id: target.id } }
		});

		removingTaskBusy = false;
		if (err) {
			removingTaskError = err.message ?? 'Failed to delete task.';
			return;
		}
		removingTask = null;
		flash('success', `Deleted task "${target.title}".`);
		loadTasks();
	}

	const daysToTarget = $derived(project?.targetDate ? daysUntil(project.targetDate) : null);

	function contributionTypeLabel(type: Contribution['type']): string {
		return type.replace('_', ' ');
	}

	function personLabel(id: string): string {
		const p = people[id];
		return p ? `${p.firstName} ${p.lastName}` : 'Unknown person';
	}

	function assigneesLabel(ids: string[]): string {
		return ids.length === 0 ? 'Unassigned' : ids.map((id) => personLabel(id)).join(', ');
	}

	function openAddMembers() {
		showAddMembers = true;
	}

	function handleMembersAdded(addedCount: number) {
		// Don't close here — AddTeamMembersModal closes itself once every
		// selected person succeeded, but stays open (with the failures still
		// selected) when some POSTs fail, so the PM can retry just those.
		flash('success', `Added ${addedCount} ${addedCount === 1 ? 'person' : 'people'} to the project.`);
		loadMembers();
	}

	function openEditMember(m: ProjectMember) {
		editingMember = m;
		memberModalMode = 'edit';
	}

	function closeMemberModal() {
		memberModalMode = null;
		editingMember = null;
	}

	function handleMemberSaved(saved: ProjectMember) {
		const wasEdit = editingMember !== null;
		closeMemberModal();
		flash(
			'success',
			wasEdit ? `Updated ${personLabel(saved.personId)}'s assignment.` : `Added ${personLabel(saved.personId)} to the project.`
		);
		loadMembers();
	}

	function askRemove(m: ProjectMember) {
		removeError = null;
		removingMember = m;
	}

	async function confirmRemove() {
		if (!removingMember) return;
		const target = removingMember;
		removing = true;
		removeError = null;

		const { error: err } = await api.DELETE('/api/v1/project-members/{id}', {
			params: { path: { id: target.id } }
		});

		removing = false;
		if (err) {
			removeError = err.message ?? 'Failed to remove person from project.';
			return;
		}
		removingMember = null;
		flash('success', `${personLabel(target.personId)} was removed from the project.`);
		loadMembers();
	}

	function handleProjectSaved(saved: Project) {
		editingProject = false;
		project = saved;
		flash('success', 'Project updated.');
	}

	function openKickoff() {
		showKickoffModal = true;
	}

	function closeKickoffModal() {
		showKickoffModal = false;
	}

	function handleKickoffSaved(saved: ProjectKickoff) {
		closeKickoffModal();
		flash('success', kickoff ? 'Kickoff updated.' : 'Kickoff saved.');
		kickoff = saved;
	}

	function askDeleteKickoff() {
		closeKickoffModal();
		removingKickoffError = null;
		removingKickoff = true;
	}

	async function confirmDeleteKickoff() {
		if (!kickoff) return;
		removingKickoffBusy = true;
		removingKickoffError = null;

		const { error: err } = await api.DELETE('/api/v1/project-kickoffs/{id}', {
			params: { path: { id: kickoff.id } }
		});

		removingKickoffBusy = false;
		if (err) {
			removingKickoffError = err.message ?? 'Failed to delete the kickoff.';
			return;
		}
		removingKickoff = false;
		kickoff = null;
		flash('success', 'Kickoff deleted.');
	}

	// Creates a real task from an action item (title = the item's
	// description, assignee = its owner if one was set), then PATCHes the
	// kickoff's full action-item set with that item's taskId filled in —
	// actionItems is a replace-whole-set field, so every other item's
	// current description/owner/taskId is echoed back unchanged. Refreshes
	// Tasks too, since a new task now exists on this project.
	async function convertActionItemToTask(index: number) {
		if (!kickoff) return;
		const item = kickoff.actionItems[index];
		if (!item || item.taskId) return;

		convertActionItemError = null;
		convertingActionItemIndex = index;
		try {
			const { data: task, error: taskErr } = await api.POST('/api/v1/tasks', {
				body: {
					projectId,
					title: item.description,
					assigneeIds: item.ownerId ? [item.ownerId] : undefined
				}
			});
			if (taskErr || !task) {
				convertActionItemError = taskErr?.message ?? 'Failed to create the task.';
				return;
			}

			const nextItems = kickoff.actionItems.map((a, i) =>
				i === index
					? { description: a.description, ownerId: a.ownerId ?? undefined, taskId: task.id }
					: { description: a.description, ownerId: a.ownerId ?? undefined, taskId: a.taskId ?? undefined }
			);
			const { data: updated, error: kickoffErr } = await api.PATCH('/api/v1/project-kickoffs/{id}', {
				params: { path: { id: kickoff.id } },
				body: { actionItems: nextItems }
			});
			if (kickoffErr) {
				convertActionItemError = kickoffErr.message ?? 'Task was created, but the kickoff failed to update.';
				return;
			}
			if (updated) kickoff = updated;
			flash('success', `Created task "${task.title}".`);
			loadTasks();
		} finally {
			convertingActionItemIndex = null;
		}
	}

	function openAddMilestone() {
		editingMilestone = null;
		milestoneModalMode = 'create';
	}

	function openEditMilestone(m: Milestone) {
		editingMilestone = m;
		milestoneModalMode = 'edit';
	}

	function closeMilestoneModal() {
		milestoneModalMode = null;
		editingMilestone = null;
	}

	function handleMilestoneSaved(saved: Milestone) {
		const wasEdit = editingMilestone !== null;
		closeMilestoneModal();
		flash('success', wasEdit ? `Updated "${saved.name}".` : `Added milestone "${saved.name}".`);
		loadMilestones();
	}

	function askRemoveMilestone(m: Milestone) {
		// Triggered from the "Delete milestone" link inside the edit modal —
		// close that modal first so the confirm dialog isn't stacked behind it.
		closeMilestoneModal();
		removingMilestoneError = null;
		removingMilestone = m;
	}

	async function confirmRemoveMilestone() {
		if (!removingMilestone) return;
		const target = removingMilestone;
		removingMilestoneBusy = true;
		removingMilestoneError = null;

		const { error: err } = await api.DELETE('/api/v1/milestones/{id}', {
			params: { path: { id: target.id } }
		});

		removingMilestoneBusy = false;
		if (err) {
			removingMilestoneError = err.message ?? 'Failed to delete milestone.';
			return;
		}
		removingMilestone = null;
		flash('success', `Deleted milestone "${target.name}".`);
		loadMilestones();
		// Deleting a milestone doesn't delete its tasks — they fall back to
		// "No milestone" server-side — so refresh the Tasks grouping too.
		loadTasks();
	}

	function statusTone(status: Project['status']): BadgeTone {
		switch (status) {
			case 'active':
				return 'success';
			case 'planning':
				return 'info';
			case 'on_hold':
				return 'warning';
			case 'completed':
				return 'neutralMuted';
			case 'cancelled':
				return 'danger';
		}
	}

	function roleTone(role: ProjectMember['role']): BadgeTone {
		switch (role) {
			case 'lead':
				return 'dark';
			case 'maintainer':
				return 'violet';
			case 'contributor':
				return 'neutral';
			case 'reviewer':
				return 'info';
			case 'stakeholder':
				return 'warning';
		}
	}

	onMount(() => {
		loadProject();
		loadMembers();
		loadPeople();
		loadActivity();
		loadKickoff();
		loadMilestones();
		loadTasks();
	});
</script>

<div class="space-y-6">
	<a href="/projects" class="text-sm text-slate-500 hover:text-slate-700">← All projects</a>

	{#if loadingProject}
		<p class="text-slate-500">Loading…</p>
	{:else if projectError}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
			<p class="font-medium">Couldn't load this project.</p>
			<p class="text-sm">{projectError}</p>
		</div>
	{:else if project}
		{#if banner}
			<div
				class="rounded-md border px-3 py-2 text-sm {banner.kind === 'success'
					? 'border-emerald-200 bg-emerald-50 text-emerald-700'
					: 'border-red-200 bg-red-50 text-red-700'}"
			>
				{banner.text}
			</div>
		{/if}

		<section class="rounded-lg border border-slate-200 bg-white p-6">
			<div class="flex flex-wrap items-start justify-between gap-3">
				<div>
					<div class="flex items-center gap-2">
						<span class="font-mono text-xs font-medium text-slate-500">{project.key}</span>
						<Badge tone={statusTone(project.status)}>{project.status.replace('_', ' ')}</Badge>
						<span class="inline-flex items-center gap-1.5 text-xs text-slate-500">
							<span class="h-2 w-2 rounded-full {healthDotClassMap[project.health]}"></span>
							{project.health.replace('_', ' ')}
						</span>
					</div>
					<h1 class="mt-1 text-2xl font-semibold tracking-tight">{project.name}</h1>
					{#if project.description}
						<p class="mt-2 max-w-2xl text-sm text-slate-600">{project.description}</p>
					{/if}
					<dl class="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-500">
						<div>
							<dt class="inline font-medium text-slate-600">Lead:</dt>
							<dd class="inline">{project.leadId ? personLabel(project.leadId) : 'Unassigned'}</dd>
						</div>
						<div>
							<dt class="inline font-medium text-slate-600">Priority:</dt>
							<dd class="inline capitalize">{project.priority}</dd>
						</div>
						{#if project.startDate}
							<div>
								<dt class="inline font-medium text-slate-600">Start:</dt>
								<dd class="inline">{project.startDate}</dd>
							</div>
						{/if}
						{#if project.targetDate}
							<div>
								<dt class="inline font-medium text-slate-600">Target:</dt>
								<dd class="inline">{project.targetDate}</dd>
							</div>
						{/if}
					</dl>
				</div>
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={() => (showLogContribution = true)}>
						Log contribution
					</button>
					<button type="button" class="btn-secondary" onclick={() => (editingProject = true)}>
						Edit project
					</button>
				</div>
			</div>

			<dl class="mt-6 grid grid-cols-2 gap-4 border-t border-slate-100 pt-4 sm:grid-cols-4">
				<MetricCard label="Team members" value={existingPersonIds.length} />
				<MetricCard label="Contributions" value={loadingActivity ? '—' : (contributionsTotal ?? 0)} />
				<MetricCard
					label="Avg. allocation"
					value={activeMembers.length > 0 ? `${avgAllocation}%` : '—'}
				/>
				<MetricCard
					label={daysToTarget !== null && daysToTarget < 0 ? 'Overdue by' : 'Days to target'}
					value={daysToTarget !== null ? Math.abs(daysToTarget) : '—'}
					danger={daysToTarget !== null && daysToTarget < 0}
				/>
			</dl>
		</section>

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Team</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{existingPersonIds.length}
						{existingPersonIds.length === 1 ? 'active member' : 'active members'}
					</p>
				</div>
				<button type="button" class="btn-primary" onclick={openAddMembers}>+ Add person</button>
			</div>

			{#if loadingMembers}
				<p class="text-slate-500">Loading…</p>
			{:else if membersError}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
					<p class="text-sm">{membersError}</p>
				</div>
			{:else if members.length === 0}
				<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
					No one is assigned to this project yet. Click "Add person" to build the team.
				</div>
			{:else}
				<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
					<table class="w-full text-left text-sm">
						<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
							<tr>
								<th class="px-4 py-3 font-medium">Person</th>
								<th class="px-4 py-3 font-medium">Role</th>
								<th class="px-4 py-3 font-medium">Allocation</th>
								<th class="px-4 py-3 font-medium">Joined</th>
								<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-slate-100">
							{#each sortedMembers as member (member.id)}
								<tr class="hover:bg-slate-50 {member.leftAt ? 'opacity-60' : ''}">
									<td class="px-4 py-3 font-medium">
										{personLabel(member.personId)}
										{#if member.leftAt}
											<span class="ml-1.5 rounded-full bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500">
												left {member.leftAt}
											</span>
										{/if}
									</td>
									<td class="px-4 py-3">
										<Badge tone={roleTone(member.role)}>{member.role}</Badge>
									</td>
									<td class="px-4 py-3 text-slate-600">{member.allocationPct}%</td>
									<td class="px-4 py-3 text-slate-600">{member.joinedAt}</td>
									<td class="px-4 py-3 text-right whitespace-nowrap">
										<button
											type="button"
											class="btn-ghost px-2 py-1"
											onclick={() => openEditMember(member)}
										>
											Edit
										</button>
										<button
											type="button"
											class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
											onclick={() => askRemove(member)}
										>
											Remove
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Kickoff</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{kickoff ? 'The kickoff meeting record for this project.' : 'No kickoff recorded yet.'}
					</p>
				</div>
				<button type="button" class="btn-primary" onclick={openKickoff}>
					{kickoff ? 'Edit kickoff' : 'Run kickoff'}
				</button>
			</div>

			{#if loadingKickoff}
				<p class="text-slate-500">Loading…</p>
			{:else if kickoffError}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
					<p class="text-sm">{kickoffError}</p>
				</div>
			{:else if !kickoff}
				<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
					This project hasn't been kicked off yet. Click "Run kickoff" to capture the meeting.
				</div>
			{:else}
				<div class="space-y-4 rounded-lg border border-slate-200 bg-white p-6">
					<dl class="flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-500">
						{#if kickoff.meetingDate}
							<div>
								<dt class="inline font-medium text-slate-600">Meeting date:</dt>
								<dd class="inline">{kickoff.meetingDate}</dd>
							</div>
						{/if}
						{#if kickoff.facilitatorId}
							<div>
								<dt class="inline font-medium text-slate-600">Facilitator:</dt>
								<dd class="inline">{personLabel(kickoff.facilitatorId)}</dd>
							</div>
						{/if}
						{#if kickoff.attendees}
							<div>
								<dt class="inline font-medium text-slate-600">Attendees:</dt>
								<dd class="inline">{kickoff.attendees}</dd>
							</div>
						{/if}
					</dl>

					{#if kickoff.businessProblem || kickoff.successDefinition}
						<div class="border-t border-slate-100 pt-4">
							<h3 class="text-sm font-semibold text-slate-700">Vision &amp; business goals</h3>
							{#if kickoff.businessProblem}
								<p class="mt-1 text-sm text-slate-600">{kickoff.businessProblem}</p>
							{/if}
							{#if kickoff.successDefinition}
								<p class="mt-1 text-sm text-slate-600">{kickoff.successDefinition}</p>
							{/if}
						</div>
					{/if}

					{#if kickoff.primaryUsers || kickoff.painPoints || kickoff.mvpFeature}
						<div class="border-t border-slate-100 pt-4">
							<h3 class="text-sm font-semibold text-slate-700">Target audience &amp; value proposition</h3>
							{#if kickoff.primaryUsers}
								<p class="mt-1 text-sm text-slate-600">{kickoff.primaryUsers}</p>
							{/if}
							{#if kickoff.painPoints}
								<p class="mt-1 text-sm text-slate-600">{kickoff.painPoints}</p>
							{/if}
							{#if kickoff.mvpFeature}
								<p class="mt-1 text-sm text-slate-600">
									<span class="font-medium text-slate-700">MVP:</span> {kickoff.mvpFeature}
								</p>
							{/if}
						</div>
					{/if}

					{#if kickoff.constraintsNotes || kickoff.biggestRisk}
						<div class="border-t border-slate-100 pt-4">
							<h3 class="text-sm font-semibold text-slate-700">Constraints, risks &amp; compliance</h3>
							{#if kickoff.constraintsNotes}
								<p class="mt-1 text-sm text-slate-600">{kickoff.constraintsNotes}</p>
							{/if}
							{#if kickoff.biggestRisk}
								<p class="mt-1 text-sm text-slate-600">{kickoff.biggestRisk}</p>
							{/if}
						</div>
					{/if}

					{#if kickoff.signOffOwner || kickoff.meetingCadence || kickoff.primaryTools}
						<div class="border-t border-slate-100 pt-4">
							<h3 class="text-sm font-semibold text-slate-700">Governance &amp; communication</h3>
							<dl class="mt-1 flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-600">
								{#if kickoff.signOffOwner}
									<div>
										<dt class="inline font-medium text-slate-700">Sign-off:</dt>
										<dd class="inline">{kickoff.signOffOwner}</dd>
									</div>
								{/if}
								{#if kickoff.meetingCadence}
									<div>
										<dt class="inline font-medium text-slate-700">Cadence:</dt>
										<dd class="inline">{kickoff.meetingCadence}</dd>
									</div>
								{/if}
								{#if kickoff.primaryTools}
									<div>
										<dt class="inline font-medium text-slate-700">Tools:</dt>
										<dd class="inline">{kickoff.primaryTools}</dd>
									</div>
								{/if}
							</dl>
						</div>
					{/if}

					<div class="border-t border-slate-100 pt-4">
						<h3 class="text-sm font-semibold text-slate-700">Next steps &amp; action items</h3>
						{#if kickoff.nextSessionDate}
							<p class="mt-1 text-sm text-slate-600">
								<span class="font-medium text-slate-700">Next session:</span> {kickoff.nextSessionDate}
							</p>
						{/if}

						{#if convertActionItemError}
							<div class="mt-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
								{convertActionItemError}
							</div>
						{/if}

						{#if kickoff.actionItems.length === 0}
							<p class="mt-1 text-sm text-slate-500">No action items captured.</p>
						{:else}
							<ul class="mt-2 divide-y divide-slate-100 rounded-md border border-slate-200">
								{#each kickoff.actionItems as item, i (item.id)}
									<li class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
										<span class="text-slate-700">
											{item.description}
											{#if item.ownerId}
												<span class="ml-1.5 text-xs text-slate-500">— {personLabel(item.ownerId)}</span>
											{/if}
										</span>
										{#if item.taskId}
											<Badge tone="success">Task created</Badge>
										{:else}
											<button
												type="button"
												class="btn-ghost shrink-0 px-2 py-1 text-xs"
												disabled={convertingActionItemIndex === i}
												onclick={() => convertActionItemToTask(i)}
											>
												{convertingActionItemIndex === i ? 'Creating…' : 'Convert to task'}
											</button>
										{/if}
									</li>
								{/each}
							</ul>
						{/if}
					</div>
				</div>
			{/if}
		</section>

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Milestones</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{milestones.length}
						{milestones.length === 1 ? 'milestone' : 'milestones'}
					</p>
				</div>
				<button type="button" class="btn-primary" onclick={openAddMilestone}>+ Add milestone</button>
			</div>

			{#if loadingMilestones}
				<p class="text-slate-500">Loading…</p>
			{:else if milestonesError}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
					<p class="text-sm">{milestonesError}</p>
				</div>
			{:else}
				<div class="rounded-lg border border-slate-200 bg-white p-4">
					<MilestoneTimeline
						{milestones}
						projectStartDate={project.startDate}
						projectTargetDate={project.targetDate}
						onSelect={(m) => {
							const full = milestones.find((x) => x.id === m.id);
							if (full) openEditMilestone(full);
						}}
					/>
				</div>
			{/if}
		</section>

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Tasks</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{tasks.length}
						{tasks.length === 1 ? 'task' : 'tasks'}
					</p>
				</div>
				<button type="button" class="btn-primary" onclick={() => openAddTask(null)}>+ Add task</button>
			</div>

			{#if loadingTasks}
				<p class="text-slate-500">Loading…</p>
			{:else if tasksError}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
					<p class="text-sm">{tasksError}</p>
				</div>
			{:else if tasks.length === 0}
				<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
					No tasks yet. Click "Add task" to break this project's milestones down into work.
				</div>
			{:else}
				<div class="flex flex-wrap items-center gap-3">
					<select
						class="input w-auto"
						value={taskStatusFilter}
						onchange={(e) => (taskStatusFilter = e.currentTarget.value as TaskStatus | '')}
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
					<select
						class="input w-auto"
						value={taskAssigneeFilter}
						onchange={(e) => (taskAssigneeFilter = e.currentTarget.value)}
					>
						<option value="">All assignees</option>
						<option value="__unassigned__">Unassigned</option>
						{#each assigneeOptions as person (person.id)}
							<option value={person.id}>{person.name}</option>
						{/each}
					</select>
				</div>

				<div class="space-y-4">
					{#each taskGroups as group (group.key)}
						<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
							<div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-4 py-2.5">
								<div class="flex items-center gap-2">
									<h3 class="text-sm font-semibold text-slate-800">{group.name}</h3>
									<span class="text-xs text-slate-500">
										{group.doneCount}/{group.totalCount} done
									</span>
								</div>
								<button
									type="button"
									class="btn-ghost px-2 py-1 text-xs"
									onclick={() => openAddTask(group.milestoneId)}
								>
									+ Add task
								</button>
							</div>
							{#if group.tasks.length === 0}
								<p class="px-4 py-3 text-sm text-slate-400">No tasks match the current filters.</p>
							{:else}
								<table class="w-full text-left text-sm">
									<tbody class="divide-y divide-slate-100">
										{#each group.tasks as task (task.id)}
											<tr class="hover:bg-slate-50">
												<td class="px-4 py-2.5">
													<button
														type="button"
														class="text-left font-medium text-slate-800 hover:underline"
														onclick={() => openEditTask(task)}
													>
														{task.title}
													</button>
												</td>
												<td class="px-4 py-2.5 text-slate-600" title={assigneesLabel(task.assigneeIds)}>
													<span class="line-clamp-1">{assigneesLabel(task.assigneeIds)}</span>
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
												<td class="px-4 py-2.5 text-right whitespace-nowrap">
													<button
														type="button"
														class="btn-ghost px-2 py-1"
														onclick={() => openEditTask(task)}
													>
														Edit
													</button>
													<button
														type="button"
														class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
														onclick={() => askRemoveTask(task)}
													>
														Delete
													</button>
												</td>
											</tr>
										{/each}
									</tbody>
								</table>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 class="text-lg font-semibold tracking-tight">Recent activity</h2>
				<a href="/contributions" class="text-sm text-slate-500 hover:text-slate-700">
					View all →
				</a>
			</div>

			{#if loadingActivity}
				<p class="text-slate-500">Loading…</p>
			{:else if recentContributions.length === 0}
				<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
					No contributions logged for this project yet.
				</div>
			{:else}
				<ul class="divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white">
					{#each recentContributions as contribution (contribution.id)}
						<li class="flex items-center justify-between gap-3 px-4 py-3 text-sm">
							<div class="min-w-0">
								<p class="truncate font-medium text-slate-800">
									{contribution.title ?? contributionTypeLabel(contribution.type)}
								</p>
								<p class="text-xs text-slate-500">
									{personLabel(contribution.personId)} · {contributionTypeLabel(contribution.type)}
								</p>
							</div>
							<span class="shrink-0 text-xs text-slate-400">
								{contribution.occurredAt.slice(0, 10)}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>

{#if editingProject && project}
	<ProjectFormModal {project} onClose={() => (editingProject = false)} onSaved={handleProjectSaved} />
{/if}

{#if memberModalMode}
	<ProjectMemberFormModal
		{projectId}
		member={editingMember}
		personDisplayName={editingMember ? personLabel(editingMember.personId) : ''}
		{existingPersonIds}
		onClose={closeMemberModal}
		onSaved={handleMemberSaved}
	/>
{/if}

{#if showAddMembers}
	<AddTeamMembersModal
		{projectId}
		{existingPersonIds}
		onClose={() => (showAddMembers = false)}
		onSaved={handleMembersAdded}
	/>
{/if}

{#if showLogContribution}
	<ContributionFormModal
		initialProjectId={projectId}
		onClose={() => (showLogContribution = false)}
		onSaved={() => {
			showLogContribution = false;
			flash('success', 'Contribution logged.');
			loadActivity();
		}}
	/>
{/if}

{#if showKickoffModal}
	<KickoffFormModal
		{projectId}
		{kickoff}
		onClose={closeKickoffModal}
		onSaved={handleKickoffSaved}
		onDelete={kickoff ? askDeleteKickoff : undefined}
	/>
{/if}

<ConfirmDialog
	open={removingKickoff}
	title="Delete kickoff"
	message="Delete this project's kickoff record? This can't be undone (any tasks already created from its action items are kept)."
	confirmLabel={removingKickoffBusy ? 'Deleting…' : 'Delete'}
	danger
	error={removingKickoffError}
	onConfirm={confirmDeleteKickoff}
	onCancel={() => (removingKickoff = false)}
/>

{#if milestoneModalMode}
	<MilestoneFormModal
		{projectId}
		milestone={editingMilestone}
		onClose={closeMilestoneModal}
		onSaved={handleMilestoneSaved}
		onDelete={editingMilestone ? () => askRemoveMilestone(editingMilestone!) : undefined}
	/>
{/if}

<ConfirmDialog
	open={removingMember !== null}
	title="Remove from project"
	message={removingMember
		? `Remove ${personLabel(removingMember.personId)} from this project? This can't be undone.`
		: ''}
	confirmLabel={removing ? 'Removing…' : 'Remove'}
	danger
	error={removeError}
	onConfirm={confirmRemove}
	onCancel={() => (removingMember = null)}
/>

{#if taskModalMode}
	<TaskFormModal
		{projectId}
		task={editingTask}
		{milestoneOptions}
		{assigneeOptions}
		initialMilestoneId={taskModalMilestoneId}
		onClose={closeTaskModal}
		onSaved={handleTaskSaved}
		onDelete={editingTask ? () => askRemoveTask(editingTask!) : undefined}
	/>
{/if}

<ConfirmDialog
	open={removingMilestone !== null}
	title="Delete milestone"
	message={removingMilestone
		? `Delete "${removingMilestone.name}"? This can't be undone.`
		: ''}
	confirmLabel={removingMilestoneBusy ? 'Deleting…' : 'Delete'}
	danger
	error={removingMilestoneError}
	onConfirm={confirmRemoveMilestone}
	onCancel={() => (removingMilestone = null)}
/>

<ConfirmDialog
	open={removingTask !== null}
	title="Delete task"
	message={removingTask ? `Delete "${removingTask.title}"? This can't be undone.` : ''}
	confirmLabel={removingTaskBusy ? 'Deleting…' : 'Delete'}
	danger
	error={removingTaskError}
	onConfirm={confirmRemoveTask}
	onCancel={() => (removingTask = null)}
/>
