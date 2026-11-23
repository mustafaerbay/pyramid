<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ProjectFormModal from '$lib/components/ProjectFormModal.svelte';
	import ProjectInfoFormModal from '$lib/components/ProjectInfoFormModal.svelte';
	import ProjectMemberFormModal from '$lib/components/ProjectMemberFormModal.svelte';
	import AddTeamMembersModal from '$lib/components/AddTeamMembersModal.svelte';
	import ContributionFormModal from '$lib/components/ContributionFormModal.svelte';
	import KickoffFormModal from '$lib/components/KickoffFormModal.svelte';
	import MilestoneFormModal from '$lib/components/MilestoneFormModal.svelte';
	import MilestoneTimeline from '$lib/components/MilestoneTimeline.svelte';
	import ReleasePlan from '$lib/components/ReleasePlan.svelte';
	import VersionFormModal from '$lib/components/VersionFormModal.svelte';
	import WeeklySummaryFormModal from '$lib/components/WeeklySummaryFormModal.svelte';
	import TaskFormModal from '$lib/components/TaskFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import TagChips from '$lib/components/TagChips.svelte';
	import TaskBoard from '$lib/components/TaskBoard.svelte';
	import Markdown from '$lib/components/Markdown.svelte';
	import { can } from '$lib/auth.svelte';
	import DependencyNote from '$lib/components/DependencyNote.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';
	import AreaManager from '$lib/components/AreaManager.svelte';
	import ImportCsvModal from '$lib/components/ImportCsvModal.svelte';
	import CustomFieldsManager from '$lib/components/CustomFieldsManager.svelte';
	import WorkflowManager from '$lib/components/WorkflowManager.svelte';
	import TypesManager from '$lib/components/TypesManager.svelte';
	import TeamsManager from '$lib/components/TeamsManager.svelte';
	import RulesManager from '$lib/components/RulesManager.svelte';
	import ProcessManager from '$lib/components/ProcessManager.svelte';
	import HooksManager from '$lib/components/HooksManager.svelte';
	import { cloneBody, descendants, templateFields } from '$lib/clone';
	import { teamFilter } from '$lib/teams';
	import { download, downloadBytes, parseCsv, XLSX_TYPE } from '$lib/csv';
	import { writeXlsx } from '$lib/xlsx';
	import { tasksToCsv, type Lookups } from '$lib/taskCsv';
	import { areaIdsUnder, areaLabel } from '$lib/areas';
	import { TYPE_OPTIONS, childType, type WorkItemType } from '$lib/workItems';
	import { afterIdToPlaceBefore, boardStates } from '$lib/board';
	import { countsTowardTotals, nestTasks, subtaskSummary } from '$lib/subtasks';
	import { hasAllTags, tagColor } from '$lib/tags';
	import { fitText, formatManDays, shortfalls } from '$lib/capacity';
	import { badgeToneClass, healthDotClass as healthDotClassMap, type BadgeTone } from '$lib/badgeTones';
	import { projectRoleLabel, projectRoleTone } from '$lib/projectRoles';
	import { daysUntil } from '$lib/dateMath';

	type Project = components['schemas']['Project'];
	type ProjectMember = components['schemas']['ProjectMember'];
	type Person = components['schemas']['Person'];
	type Contribution = components['schemas']['Contribution'];
	type Milestone = components['schemas']['Milestone'];
	type Version = components['schemas']['Version'];
	type WeeklySummary = components['schemas']['WeeklySummary'];
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
	// The stakeholder/reporting metadata block (Sponsor, PM contacts, team
	// size, …) shown in the header — edited via its own modal, separate from
	// the core Edit Project modal above, since it's a large optional block.
	let editingProjectInfo = $state(false);

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

	let versions = $state<Version[]>([]);
	let loadingVersions = $state(true);
	let versionsError = $state<string | null>(null);

	let versionModalMode = $state<'create' | 'edit' | null>(null);
	let editingVersion = $state<Version | null>(null);

	let removingVersion = $state<Version | null>(null);
	let removingVersionBusy = $state(false);
	let removingVersionError = $state<string | null>(null);

	let weeklySummaries = $state<WeeklySummary[]>([]);
	let loadingWeeklySummaries = $state(true);
	let weeklySummariesError = $state<string | null>(null);

	let weeklySummaryModalMode = $state<'create' | 'edit' | null>(null);
	let editingWeeklySummary = $state<WeeklySummary | null>(null);

	let removingWeeklySummary = $state<WeeklySummary | null>(null);
	let removingWeeklySummaryBusy = $state(false);
	let removingWeeklySummaryError = $state<string | null>(null);

	let tasks = $state<Task[]>([]);
	let loadingTasks = $state(true);
	let tasksError = $state<string | null>(null);

	let taskModalMode = $state<'create' | 'edit' | null>(null);
	let editingTask = $state<Task | null>(null);
	// Prefills the milestone picker when "+ Add task" is clicked from inside
	// a specific milestone's group, rather than the section-level button.
	let taskModalMilestoneId = $state<string | null>(null);
	let taskModalParentId = $state<string | null>(null);

	let removingTask = $state<Task | null>(null);
	let removingTaskBusy = $state(false);
	let removingTaskError = $state<string | null>(null);

	// Toggleable chips, not a dropdown — several statuses can be shown at
	// once. Empty array = no filter (show every status), same default
	// meaning the old single-select's '' value had. A plain array that gets
	// reassigned on every toggle, not a Set — $state(new Set()) only makes
	// the Set *reference* reactive (swapping it triggers updates), not its
	// .add()/.delete() mutations; those need the SvelteSet class from
	// svelte/reactivity, or (simpler, and consistent with the
	// reassign-a-new-array pattern already used elsewhere on this page,
	// e.g. draftMilestones) just reassign a new array each time.
	let taskStatusFilters = $state<TaskStatus[]>([]);
	const statusFilterOptions: { value: TaskStatus; label: string }[] = [
		{ value: 'backlog', label: 'Backlog' },
		{ value: 'todo', label: 'To do' },
		{ value: 'in_progress', label: 'In progress' },
		{ value: 'in_review', label: 'In review' },
		{ value: 'blocked', label: 'Blocked' },
		{ value: 'done', label: 'Done' },
		{ value: 'cancelled', label: 'Cancelled' }
	];

	function toggleStatusFilter(status: TaskStatus) {
		taskStatusFilters = taskStatusFilters.includes(status)
			? taskStatusFilters.filter((s) => s !== status)
			: [...taskStatusFilters, status];
	}

	// '' = everyone, '__unassigned__' = the unassigned sentinel, else a personId.
	let taskAssigneeFilter = $state('');
	// Tasks must carry every selected tag.
	let taskTagFilters = $state<string[]>([]);
	let taskTypeFilter = $state<string>('');
	let taskModalType = $state<WorkItemType>('task');
	// List (grouped by milestone) or Kanban board; remembered per browser.
	let taskView = $state<'list' | 'board'>('list');
	onMount(() => {
		try {
			if (localStorage.getItem('pyramid.taskView') === 'board') taskView = 'board';
		} catch {
			// storage blocked: stay on the list
		}
	});
	function setTaskView(v: 'list' | 'board') {
		taskView = v;
		try {
			localStorage.setItem('pyramid.taskView', v);
		} catch {
			// not remembered, still switches
		}
	}
	function addTagFilter(tag: string) {
		if (!taskTagFilters.some((t) => t.toLowerCase() === tag.toLowerCase())) taskTagFilters = [...taskTagFilters, tag];
	}

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

	async function loadVersions() {
		loadingVersions = true;
		versionsError = null;
		const { data, error: err } = await api.GET('/api/v1/versions', {
			params: { query: { projectId, pageSize: 100 } }
		});
		if (err) {
			versionsError = err.message ?? 'Failed to load versions';
		} else if (data) {
			versions = data.data;
		}
		loadingVersions = false;
	}

	async function loadWeeklySummaries() {
		loadingWeeklySummaries = true;
		weeklySummariesError = null;
		const { data, error: err } = await api.GET('/api/v1/weekly-summaries', {
			params: { query: { projectId, pageSize: 100 } }
		});
		if (err) {
			weeklySummariesError = err.message ?? 'Failed to load weekly summaries';
		} else if (data) {
			weeklySummaries = data.data;
		}
		loadingWeeklySummaries = false;
	}

	let sprints = $state<components['schemas']['Sprint'][]>([]);
	async function loadSprints() {
		const { data } = await api.GET('/api/v1/sprints', { params: { query: { projectId } } });
		sprints = data?.data ?? [];
	}
	let taskSprintFilter = $state('');
	let areas = $state<components['schemas']['Area'][]>([]);
	async function loadAreas() {
		const { data } = await api.GET('/api/v1/areas', { params: { query: { projectId } } });
		areas = data?.data ?? [];
	}
	let taskAreaFilter = $state('');
	let managingAreas = $state(false);
	let importing = $state(false);
	let managingFields = $state(false);
	let managingWorkflow = $state(false);
	let managingTypes = $state(false);
	let managingTeams = $state(false);
	let managingRules = $state(false);
	let managingProcess = $state(false);
	let managingHooks = $state(false);
	async function saveRules(rules: components['schemas']['WorkItemRules']): Promise<string | null> {
		if (!project) return 'No project';
		const { data, error: err } = await api.PATCH('/api/v1/projects/{id}', { params: { path: { id: project.id } }, body: { rules } });
		if (err || !data) return err?.message ?? 'Could not save the rules.';
		project = data;
		flash('success', 'Rules saved.');
		return null;
	}
	let teams = $state<components['schemas']['ProjectTeam'][]>([]);
	async function loadTeams() {
		const { data } = await api.GET('/api/v1/project-teams', { params: { query: { projectId } } });
		teams = data?.data ?? [];
	}
	let taskTeamFilter = $state('');
	const teamMatch = $derived.by(() => {
		const t = teams.find((x) => x.id === taskTeamFilter);
		return t ? teamFilter(t, areas) : null;
	});
	async function setRestricted(restricted: boolean) {
		if (!project) return;
		const { data, error: err } = await api.PATCH('/api/v1/projects/{id}', { params: { path: { id: project.id } }, body: { restricted } });
		if (err || !data) return flash('error', err?.message ?? 'Could not change access.');
		project = data;
		flash('success', restricted ? 'Only members with write access can change this project now.' : 'Everyone who can edit may change this project again.');
	}
	async function saveTypes(workItemTypes: components['schemas']['WorkItemTypeDef'][]): Promise<string | null> {
		if (!project) return 'No project';
		const { data, error: err } = await api.PATCH('/api/v1/projects/{id}', { params: { path: { id: project.id } }, body: { workItemTypes } });
		if (err || !data) return err?.message ?? 'Could not save the types.';
		project = data;
		flash('success', 'Work item types saved.');
		loadTasks();
		return null;
	}
	async function saveWorkflow(states: components['schemas']['WorkflowState'][]): Promise<string | null> {
		if (!project) return 'No project';
		const { data, error: err } = await api.PATCH('/api/v1/projects/{id}', { params: { path: { id: project.id } }, body: { workflow: states } });
		if (err || !data) return err?.message ?? 'Could not save the workflow.';
		project = data;
		flash('success', 'Workflow saved.');
		return null;
	}
	async function saveFields(customFields: components['schemas']['CustomField'][]): Promise<string | null> {
		if (!project) return 'No project';
		const { data, error: err } = await api.PATCH('/api/v1/projects/{id}', { params: { path: { id: project.id } }, body: { customFields } });
		if (err || !data) return err?.message ?? 'Could not save the fields.';
		project = data;
		flash('success', 'Custom fields saved.');
		return null;
	}
	const csvLookups = $derived<Lookups>({
		people: Object.values(people).map((p) => ({ id: p.id, name: `${p.firstName} ${p.lastName}`, email: p.email })),
		tasks: tasks.map((t) => ({ id: t.id, number: t.number, title: t.title })),
		milestones: milestones.map((m) => ({ id: m.id, name: m.name })),
		sprints: sprints.map((s) => ({ id: s.id, name: s.name })),
		areas,
		customFields: project?.customFields ?? []
	});
	function exportXlsx() {
		const csv = tasksToCsv(tasks.filter(matchesTaskFilters), { ...csvLookups, project: () => project?.name ?? '' });
		downloadBytes(`${(project?.key ?? 'project').toLowerCase()}-items.xlsx`, writeXlsx(parseCsv(csv), project?.key ?? 'Items'), XLSX_TYPE);
	}
	function exportTasks() {
		download(
			`${(project?.key ?? 'project').toLowerCase()}-items.csv`,
			tasksToCsv(tasks.filter(matchesTaskFilters), { ...csvLookups, project: () => project?.name ?? '' })
		);
	}
	const areaFilterIds = $derived(taskAreaFilter ? areaIdsUnder(areas, taskAreaFilter) : null);

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
	// Man-days: cancelled tasks don't count as work; done ones count as done.
	function manDaySums(list: Task[]): { manDays: number; doneManDays: number } {
		let manDays = 0;
		let doneManDays = 0;
		for (const t of list) {
			if (t.status === 'cancelled' || !countsTowardTotals(t, tasks)) continue;
			manDays += t.manDays;
			if (t.status === 'done') doneManDays += t.manDays;
		}
		return { manDays, doneManDays };
	}
	const projectManDays = $derived(manDaySums(tasks));

	// Every tag used in this project, for the tag filter.
	const projectTags = $derived(
		[...new Map(tasks.flatMap((t) => t.tags).map((tag) => [tag.toLowerCase(), tag])).values()].sort((a, b) =>
			a.localeCompare(b, undefined, { sensitivity: 'base' })
		)
	);

	function matchesTaskFilters(t: Task): boolean {
		if (taskStatusFilters.length > 0 && !taskStatusFilters.includes(t.status)) return false;
		if (taskAssigneeFilter === '__unassigned__' && t.assigneeIds.length > 0) return false;
		if (taskAssigneeFilter && taskAssigneeFilter !== '__unassigned__' && !t.assigneeIds.includes(taskAssigneeFilter))
			return false;
		if (taskTypeFilter && t.type !== taskTypeFilter) return false;
		if (areaFilterIds && !(t.areaId && areaFilterIds.has(t.areaId))) return false;
		if (teamMatch && !teamMatch(t)) return false;
		if (taskSprintFilter === '__none__' ? t.sprintId : taskSprintFilter && t.sprintId !== taskSprintFilter) return false;
		return hasAllTags(t.tags, taskTagFilters);
	}

	const taskGroups = $derived(
		(() => {
			const matchesFilters = matchesTaskFilters;

			type Group = {
				key: string;
				milestoneId: string | null;
				name: string;
				totalCount: number;
				doneCount: number;
				manDays: number;
				doneManDays: number;
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
					...manDaySums(all),
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
					...manDaySums(unassignedToMilestone),
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
		taskModalParentId = null;
		taskModalMode = 'edit';
	}

	// A link with ?task=<id> (from a notification) opens that task once.
	let openedFromLink = $state<string | null>(null);
	$effect(() => {
		const wanted = page.url.searchParams.get('task');
		if (!wanted || wanted === openedFromLink || tasks.length === 0) return;
		const t = tasks.find((x) => x.id === wanted);
		if (t) {
			openedFromLink = wanted;
			openEditTask(t);
		}
	});

	function openAddSubtask(parent: Task) {
		editingTask = null;
		taskModalMilestoneId = parent.milestoneId ?? null;
		taskModalParentId = parent.id;
		taskModalType = childType(parent.type);
		taskModalMode = 'create';
	}

	// Copy, move and item templates.
	let itemTemplates = $state<components['schemas']['ItemTemplate'][]>([]);
	let transferTargets = $state<{ id: string; name: string }[]>([]);
	$effect(() => {
		if (!projectId) return;
		api.GET('/api/v1/item-templates', { params: { query: { projectId } } }).then((r) => (itemTemplates = r.data?.data ?? []));
		api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }).then(
			(r) => (transferTargets = (r.data?.data ?? []).filter((p) => p.id !== projectId).map((p) => ({ id: p.id, name: p.name })).sort((a, b) => a.name.localeCompare(b.name)))
		);
	});
	async function copyTask(t: Task, withChildren: boolean) {
		const post = (body: ReturnType<typeof cloneBody>) => api.POST('/api/v1/tasks', { body: body as components['schemas']['TaskCreate'] });
		const { data: copy, error: err } = await post(cloneBody(t, { parentId: t.parentId }));
		if (err || !copy) return flash('error', err?.message ?? 'Could not copy the item.');
		if (withChildren) {
			const ids = new Map([[t.id, copy.id]]);
			for (const k of descendants(t.id, tasks)) {
				const { data } = await post(cloneBody(k, { parentId: ids.get(k.parentId ?? '') ?? copy.id, keepTitle: true }));
				if (data) ids.set(k.id, data.id);
			}
		}
		await loadTasks();
		flash('success', `Copied as #${copy.number}.`);
		openEditTask(tasks.find((x) => x.id === copy.id) ?? copy);
	}
	async function transferTask(t: Task, pid: string) {
		const { data, error: err } = await api.POST('/api/v1/tasks/{id}/transfer', { params: { path: { id: t.id } }, body: { projectId: pid } });
		if (err || !data) return flash('error', err?.message ?? 'Could not move the item.');
		closeTaskModal();
		await loadTasks();
		flash('success', `Moved to ${transferTargets.find((p) => p.id === pid)?.name ?? 'the project'} as #${data.number}.`);
	}
	async function saveTemplate(t: Task, name: string) {
		const { data, error: err } = await api.POST('/api/v1/item-templates', {
			body: { projectId, name, fields: templateFields(t) as components['schemas']['ItemTemplateFields'] }
		});
		if (err || !data) return flash('error', err?.message ?? 'Could not save the template.');
		itemTemplates = [...itemTemplates, data].sort((a, b) => a.name.localeCompare(b.name));
		flash('success', `Template "${data.name}" saved — pick it when adding an item.`);
	}
	async function deleteTemplate(id: string) {
		const t = itemTemplates.find((x) => x.id === id);
		if (!t || !confirm(`Delete the template "${t.name}"?`)) return;
		await api.DELETE('/api/v1/item-templates/{id}', { params: { path: { id } } });
		itemTemplates = itemTemplates.filter((x) => x.id !== id);
	}

	function closeTaskModal() {
		taskModalMode = null;
		editingTask = null;
		taskModalMilestoneId = null;
		taskModalParentId = null;
	}

	// A task with subtasks shows their total; its own man-days aren't counted.
	function shownManDays(task: Task): number {
		const sub = subtaskSummary(task.id, tasks);
		return sub.count > 0 ? sub.manDays : task.manDays;
	}
	function subtaskLabel(task: Task): string {
		const sub = subtaskSummary(task.id, tasks);
		return sub.count > 0 ? `${sub.done}/${sub.count} children done` : '';
	}

	function handleTaskSaved(saved: Task) {
		const wasEdit = editingTask !== null;
		closeTaskModal();
		flash('success', wasEdit ? `Updated "${saved.title}".` : `Added task "${saved.title}".`);
		loadTasks();
	}

	// Manual order: put `task` after `afterId` (null = first).
	async function placeTask(task: Task, afterId: string | null) {
		const { data, error: err } = await api.POST('/api/v1/tasks/{id}/move', {
			params: { path: { id: task.id } },
			body: { afterId }
		});
		if (err || !data) {
			flash('error', err?.message ?? 'Failed to reorder task.');
			return;
		}
		// Only the rank: a status change sent at the same time (board drop)
		// must not be overwritten by this response.
		tasks = tasks
			.map((t) => (t.id === data.id ? { ...t, rank: data.rank } : t))
			.sort((a, b) => a.rank - b.rank || a.number - b.number);
	}
	function placeBefore(task: Task, targetId: string) {
		if (task.id !== targetId) placeTask(task, afterIdToPlaceBefore(tasks, task.id, targetId));
	}
	// ↑/↓ on phones: swap with the neighbour in the visible list.
	function shift(task: Task, siblings: Task[], dir: -1 | 1) {
		const j = siblings.findIndex((x) => x.id === task.id) + dir;
		if (j < 0 || j >= siblings.length) return;
		if (dir < 0) placeBefore(task, siblings[j].id);
		else placeTask(task, siblings[j].id);
	}
	let draggingRow = $state<string | null>(null);

	async function saveBoardSettings(settings: components['schemas']['BoardSettings']): Promise<boolean> {
		if (!project) return false;
		const { data, error: err } = await api.PATCH('/api/v1/projects/{id}', {
			params: { path: { id: project.id } },
			body: { boardSettings: settings }
		});
		if (err || !data) {
			flash('error', err?.message ?? 'Failed to save board settings.');
			return false;
		}
		project = data;
		flash('success', 'Board settings saved.');
		return true;
	}

	// Board moves update the card at once and put it back if saving fails.
	// The project's workflow states (default: one per status).
	const workflow = $derived(project?.workflow ?? []);
	const categoryOf = (key: string) => (workflow.find((w) => w.key === key)?.category ?? key) as TaskStatus;

	// Board and row moves: the item changes state at once and goes back if
	// saving fails.
	// done: which half of a split board column it lands in.
	async function moveTask(task: Task, state: string, done?: boolean) {
		const before = { state: task.state, status: task.status, boardDone: task.boardDone };
		const boardDone = done ?? (before.state === state ? task.boardDone : false);
		if (before.state === state && before.boardDone === boardDone) return;
		tasks = tasks.map((t) => (t.id === task.id ? { ...t, state, status: categoryOf(state), boardDone } : t));
		const { data, error: err } = await api.PATCH('/api/v1/tasks/{id}', {
			params: { path: { id: task.id } },
			body: done === undefined ? { state } : { state, boardDone: done }
		});
		if (err || !data) {
			tasks = tasks.map((t) => (t.id === task.id ? { ...t, ...before } : t));
			flash('error', err?.message ?? 'Failed to move task.');
			return;
		}
		tasks = tasks.map((t) => (t.id === data.id ? data : t));
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

	// "Name: 12 md in 5 available days" for each assignee whose share doesn't
	// fit before the due date (approved leave taken out).
	function fitWarnings(task: Task): string[] {
		if (!countsTowardTotals(task, tasks)) return [];
		return shortfalls(task.assignments).map(
			(a) => `${personLabel(a.personId)}: ${fitText(a.manDays, a.availableDays ?? 0)}`
		);
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

	function handleProjectInfoSaved(saved: Project) {
		editingProjectInfo = false;
		project = saved;
		flash('success', 'Project info updated.');
	}

	// Rows for the read-only "project info" block in the header, in the same
	// order as the edit modal. Filtered down to only the fields that have a
	// value when rendered — undefined/'' entries are skipped.
	const projectInfoRows = $derived(
		project
			? [
					{ label: 'Sponsor', value: project.sponsor },
					{ label: 'Project Director', value: project.projectDirector },
					{ label: 'Report Period', value: project.reportPeriod },
					{ label: 'Account Name', value: project.accountName },
					{ label: 'Project Code', value: project.projectCode },
					{ label: 'HQ PM', value: project.hqPm },
					{ label: 'Turkey PM', value: project.turkeyPm },
					{ label: 'HQ SE', value: project.hqSe },
					{ label: 'Turkey SE', value: project.turkeySe },
					{ label: 'HQ Interface', value: project.hqInterface },
					{ label: 'Chinese Interface', value: project.chineseInterface },
					{ label: 'Turkey QA', value: project.turkeyQa },
					{
						label: 'Team size (Actual/Planned)',
						value:
							project.teamSizeActual != null || project.teamSizePlanned != null
								? `${project.teamSizeActual ?? '—'} / ${project.teamSizePlanned ?? '—'}`
								: null
					}
				].filter((row): row is { label: string; value: string } => !!row.value)
			: []
	);

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

	function openAddVersion() {
		editingVersion = null;
		versionModalMode = 'create';
	}

	function openEditVersion(v: Version) {
		editingVersion = v;
		versionModalMode = 'edit';
	}

	function closeVersionModal() {
		versionModalMode = null;
		editingVersion = null;
	}

	function handleVersionSaved(saved: Version) {
		const wasEdit = editingVersion !== null;
		closeVersionModal();
		flash('success', wasEdit ? `Updated "${saved.name}".` : `Added version "${saved.name}".`);
		loadVersions();
	}

	function askRemoveVersion(v: Version) {
		closeVersionModal();
		removingVersionError = null;
		removingVersion = v;
	}

	async function confirmRemoveVersion() {
		if (!removingVersion) return;
		const target = removingVersion;
		removingVersionBusy = true;
		removingVersionError = null;

		const { error: err } = await api.DELETE('/api/v1/versions/{id}', {
			params: { path: { id: target.id } }
		});

		removingVersionBusy = false;
		if (err) {
			removingVersionError = err.message ?? 'Failed to delete version.';
			return;
		}
		removingVersion = null;
		flash('success', `Deleted version "${target.name}".`);
		loadVersions();
		// Its milestones were unassigned server-side; reload so none still
		// points at the deleted version.
		loadMilestones();
	}

	function openAddWeeklySummary() {
		editingWeeklySummary = null;
		weeklySummaryModalMode = 'create';
	}

	function openEditWeeklySummary(w: WeeklySummary) {
		editingWeeklySummary = w;
		weeklySummaryModalMode = 'edit';
	}

	function closeWeeklySummaryModal() {
		weeklySummaryModalMode = null;
		editingWeeklySummary = null;
	}

	function handleWeeklySummarySaved(saved: WeeklySummary) {
		const wasEdit = editingWeeklySummary !== null;
		closeWeeklySummaryModal();
		flash('success', wasEdit ? 'Updated weekly summary.' : 'Added weekly summary.');
		loadWeeklySummaries();
	}

	function askRemoveWeeklySummary(w: WeeklySummary) {
		// Same split as askRemoveMilestone — close the edit modal first so the
		// confirm dialog isn't stacked behind it.
		closeWeeklySummaryModal();
		removingWeeklySummaryError = null;
		removingWeeklySummary = w;
	}

	async function confirmRemoveWeeklySummary() {
		if (!removingWeeklySummary) return;
		const target = removingWeeklySummary;
		removingWeeklySummaryBusy = true;
		removingWeeklySummaryError = null;

		const { error: err } = await api.DELETE('/api/v1/weekly-summaries/{id}', {
			params: { path: { id: target.id } }
		});

		removingWeeklySummaryBusy = false;
		if (err) {
			removingWeeklySummaryError = err.message ?? 'Failed to delete weekly summary.';
			return;
		}
		removingWeeklySummary = null;
		flash('success', 'Deleted weekly summary.');
		loadWeeklySummaries();
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

	onMount(() => {
		loadProject();
		loadMembers();
		loadPeople();
		loadActivity();
		loadKickoff();
		loadMilestones();
		loadVersions();
		loadWeeklySummaries();
		loadTasks();
		loadSprints();
		loadAreas();
		loadTeams();
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
						<Markdown text={project.description} class="mt-2 max-w-2xl text-sm text-slate-600" />
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
				<div class="grid w-full grid-cols-1 gap-2 sm:flex sm:w-auto">
					<button data-write type="button" class="btn-secondary" onclick={() => (showLogContribution = true)}>
						Log contribution
					</button>
					<button data-write type="button" class="btn-secondary" onclick={() => (editingProject = true)}>
						Edit project
					</button>
					<button data-write
						type="button"
						class="btn-secondary"
						onclick={() => (editingProjectInfo = true)}
					>
						{projectInfoRows.length > 0 ? 'Edit project info' : '+ Add project info'}
					</button>
				</div>
			</div>

			<dl class="mt-6 grid grid-cols-2 gap-4 border-t border-slate-100 pt-4 sm:grid-cols-3 lg:grid-cols-6">
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
				<MetricCard label="Total work" value="{formatManDays(projectManDays.manDays)} md" />
				<MetricCard label="Done work" value="{formatManDays(projectManDays.doneManDays)} md" />
			</dl>

			{#if projectInfoRows.length > 0}
				<dl class="mt-6 grid grid-cols-1 gap-x-6 gap-y-2 border-t border-slate-100 pt-4 text-sm sm:grid-cols-3">
					{#each projectInfoRows as row (row.label)}
						<div>
							<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">{row.label}</dt>
							<dd class="mt-0.5 text-slate-800">{row.value}</dd>
						</div>
					{/each}
				</dl>
			{/if}
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
				<button data-write type="button" class="btn-primary" onclick={openAddMembers}>+ Add person</button>
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
				<ul class="m-list">
					{#each sortedMembers as member (member.id)}
						<li class="m-card {member.leftAt ? 'opacity-60' : ''}">
							<div class="flex items-start justify-between gap-2">
								<p class="font-medium text-slate-900">{personLabel(member.personId)}</p>
								<Badge tone={projectRoleTone[member.role]}>{projectRoleLabel[member.role]}</Badge>{#if member.access === 'read'} <Badge tone="neutralMuted">read-only</Badge>{/if}
							</div>
							<div class="m-meta">
								<span>{member.allocationPct}% allocation</span>
								<span>Joined {member.joinedAt}</span>
								{#if member.leftAt}<span>Left {member.leftAt}</span>{/if}
							</div>
							<div class="m-actions">
								<button data-write type="button" class="btn-secondary" onclick={() => openEditMember(member)}>Edit</button>
								<button data-write
									type="button"
									class="btn-secondary text-red-600 hover:bg-red-50"
									onclick={() => askRemove(member)}
								>
									Remove
								</button>
							</div>
						</li>
					{/each}
				</ul>

				<div class="table-wrap">
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
								<!-- svelte-ignore a11y_click_events_have_key_events -->
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<tr
									class="cursor-pointer hover:bg-slate-50 {member.leftAt ? 'opacity-60' : ''}"
									onclick={() => openEditMember(member)}
								>
									<td class="px-4 py-3 font-medium">
										{personLabel(member.personId)}
										{#if member.leftAt}
											<span class="ml-1.5 rounded-full bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500">
												left {member.leftAt}
											</span>
										{/if}
									</td>
									<td class="px-4 py-3">
										<Badge tone={projectRoleTone[member.role]}>{projectRoleLabel[member.role]}</Badge>{#if member.access === 'read'} <Badge tone="neutralMuted">read-only</Badge>{/if}{#if member.access === 'read'} <Badge tone="neutralMuted">read-only</Badge>{/if}
									</td>
									<td class="px-4 py-3 text-slate-600">{member.allocationPct}%</td>
									<td class="px-4 py-3 text-slate-600">{member.joinedAt}</td>
									<td
										class="px-4 py-3 text-right whitespace-nowrap"
										onclick={(e) => e.stopPropagation()}
									>
										<button data-write
											type="button"
											class="btn-ghost px-2 py-1"
											onclick={() => openEditMember(member)}
										>
											Edit
										</button>
										<button data-write
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
				<button data-write type="button" class="btn-primary" onclick={openKickoff}>
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
											<button data-write
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
					<h2 class="text-lg font-semibold tracking-tight">Releases</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{versions.length}
						{versions.length === 1 ? 'version' : 'versions'}
					</p>
				</div>
				<button data-write type="button" class="btn-primary" onclick={openAddVersion}>+ Add version</button>
			</div>

			{#if loadingVersions || loadingMilestones}
				<p class="text-slate-500">Loading…</p>
			{:else if versionsError}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
					<p class="text-sm">{versionsError}</p>
				</div>
			{:else}
				<ReleasePlan
					{versions}
					{milestones}
					{tasks}
					onSelectVersion={openEditVersion}
					onSelectMilestone={openEditMilestone}
				/>
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
				<button data-write type="button" class="btn-primary" onclick={openAddMilestone}>+ Add milestone</button>
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
						{versions}
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
					<h2 class="text-lg font-semibold tracking-tight">
						Tasks <a href="/projects/{projectId}/sprints" class="ml-2 text-sm font-normal text-slate-500 hover:text-slate-800 hover:underline">Sprints →</a>
						<a href="/projects/{projectId}/backlog" class="ml-2 text-sm font-normal text-slate-500 hover:text-slate-800 hover:underline">Backlog →</a>
						<a href="/projects/{projectId}/analytics" class="ml-2 text-sm font-normal text-slate-500 hover:text-slate-800 hover:underline">Analytics →</a>
						<button type="button" class="ml-2 text-sm font-normal text-slate-500 hover:text-slate-800 hover:underline" onclick={exportTasks}>Export CSV</button>
						<button type="button" class="ml-2 text-sm font-normal text-slate-500 hover:text-slate-800 hover:underline" onclick={exportXlsx}>Excel</button>
						{#if can('member')}
							<button type="button" class="ml-2 text-sm font-normal text-slate-500 hover:text-slate-800 hover:underline" onclick={() => (importing = true)}>Import CSV / Excel</button>
						{/if}
					</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{tasks.length}
						{tasks.length === 1 ? 'task' : 'tasks'}
					</p>
				</div>
				<div class="flex items-center gap-2">
					<div class="inline-flex rounded-md bg-slate-100 p-0.5 text-sm" role="group" aria-label="Task view">
						{#each [['list', 'List'], ['board', 'Board']] as [v, l] (v)}
							<button
								type="button"
								class="rounded px-3 py-1.5 font-medium {taskView === v
									? 'bg-white text-slate-900 shadow-sm'
									: 'text-slate-500 hover:text-slate-700'}"
								aria-pressed={taskView === v}
								onclick={() => setTaskView(v as 'list' | 'board')}>{l}</button
							>
						{/each}
					</div>
					<button data-write type="button" class="btn-primary" onclick={() => openAddTask(null)}>+ Add task</button>
				</div>
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
					<div class="flex flex-wrap items-center gap-1.5" role="group" aria-label="Filter by status">
						{#each statusFilterOptions as opt (opt.value)}
							{@const selected = taskStatusFilters.includes(opt.value)}
							<button
								type="button"
								class="rounded-full px-2.5 py-1 text-xs font-medium transition-colors {selected
									? badgeToneClass[taskStatusTone(opt.value)] + ' ring-1 ring-inset ring-transparent'
									: 'bg-white text-slate-500 ring-1 ring-inset ring-slate-200 hover:bg-slate-50'}"
								aria-pressed={selected}
								onclick={() => toggleStatusFilter(opt.value)}
							>
								{opt.label}
							</button>
						{/each}
						{#if taskStatusFilters.length > 0}
							<button
								type="button"
								class="px-1 text-xs text-slate-500 hover:text-slate-700 hover:underline"
								onclick={() => (taskStatusFilters = [])}
							>
								Clear
							</button>
						{/if}
					</div>
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
					{#if sprints.length > 0}
						<select class="input w-auto" aria-label="Filter by sprint" bind:value={taskSprintFilter}>
							<option value="">All sprints</option>
							<option value="__none__">No sprint</option>
							{#each sprints as sp (sp.id)}
								<option value={sp.id}>{sp.name}{sp.status === 'active' ? ' (active)' : ''}</option>
							{/each}
						</select>
					{/if}
					{#if teams.length}
						<select class="input w-auto" aria-label="Filter by team" bind:value={taskTeamFilter}>
							<option value="">All teams</option>
							{#each teams as tm (tm.id)}<option value={tm.id}>{tm.name}</option>{/each}
						</select>
					{/if}
					<select class="input w-auto" aria-label="Filter by area" bind:value={taskAreaFilter}>
						<option value="">All areas</option>
						{#each areas as a (a.id)}
							<option value={a.id}>{areaLabel(a.path)}</option>
						{/each}
					</select>
					{#if can('member')}
						<details class="relative">
							<summary class="btn-ghost cursor-pointer list-none px-2 py-1 text-xs">Settings ▾</summary>
							<div class="absolute right-0 z-30 mt-1 w-44 rounded-md border border-slate-200 bg-white py-1 text-sm shadow-lg">
								{#each [['Workflow', () => (managingWorkflow = true)], ['Work item types', () => (managingTypes = true)], ['Custom fields', () => (managingFields = true)], ['Rules', () => (managingRules = true)], ['Areas', () => (managingAreas = true)], ['Teams', () => (managingTeams = true)], ['Process templates', () => (managingProcess = true)], ...(can('manager') ? [['Integrations', () => (managingHooks = true)]] : [])] as [label, open] (label)}
									<button
										type="button"
										class="block w-full px-3 py-1.5 text-left hover:bg-slate-50"
										onclick={(e) => {
											(e.currentTarget.closest('details') as HTMLDetailsElement).open = false;
											(open as () => void)();
										}}>{label}</button
									>
								{/each}
							</div>
						</details>
					{/if}
					{#if can('manager') && project}
						<label class="flex items-center gap-1 text-xs text-slate-600" title="Only project members with write access (and managers) can change a restricted project">
							<input type="checkbox" checked={project.restricted} onchange={(e) => setRestricted(e.currentTarget.checked)} /> Restricted
						</label>
					{/if}
					<select class="input w-auto" aria-label="Filter by type" bind:value={taskTypeFilter}>
						<option value="">All types</option>
						{#each TYPE_OPTIONS as o (o.value)}
							<option value={o.value}>{o.label}</option>
						{/each}
					</select>
					{#if projectTags.length > 0}
						<select
							class="input w-auto"
							aria-label="Filter by tag"
							value=""
							onchange={(e) => {
								if (e.currentTarget.value) addTagFilter(e.currentTarget.value);
								e.currentTarget.value = '';
							}}
						>
							<option value="">Filter by tag…</option>
							{#each projectTags.filter((t) => !hasAllTags(taskTagFilters, [t])) as tag (tag)}
								<option value={tag}>{tag}</option>
							{/each}
						</select>
					{/if}
					{#if taskTagFilters.length > 0}
						<div class="flex flex-wrap items-center gap-1.5" aria-label="Selected tags">
							{#each taskTagFilters as tag (tag)}
								<button
									type="button"
									class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium ring-1 ring-inset {tagColor(tag)}"
									aria-label="Remove tag filter {tag}"
									onclick={() => (taskTagFilters = taskTagFilters.filter((t) => t !== tag))}
								>
									{tag} <span aria-hidden="true">×</span>
								</button>
							{/each}
							<button
								type="button"
								class="px-1 text-xs text-slate-500 hover:text-slate-700 hover:underline"
								onclick={() => (taskTagFilters = [])}
							>
								Clear tags
							</button>
						</div>
					{/if}
				</div>

				{#if taskView === 'board'}
					<TaskBoard
						tasks={tasks.filter(matchesTaskFilters)}
						allTasks={tasks}
						states={boardStates(workflow, taskStatusFilters)}
						{workflow}
						statusTone={taskStatusTone}
						priorityTone={taskPriorityTone}
						{assigneesLabel}
						{fitWarnings}
						onOpen={openEditTask}
						onMove={moveTask}
						customFields={project.customFields}
						onPlaceBefore={placeBefore}
						personName={personLabel}
						settings={project.boardSettings}
						canEditSettings={can('member')}
						onSaveSettings={saveBoardSettings}
						onTagSelect={addTagFilter}
					/>
				{:else}
				<div class="space-y-4">
					{#each taskGroups as group (group.key)}
						<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
							<div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-4 py-2.5">
								<div class="flex items-center gap-2">
									<h3 class="text-sm font-semibold text-slate-800">{group.name}</h3>
									<span class="text-xs text-slate-500">
										{group.doneCount}/{group.totalCount} done
										{#if group.manDays > 0}
											· {formatManDays(group.doneManDays)}/{formatManDays(group.manDays)} md
										{/if}
									</span>
								</div>
								<button data-write
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
								<ul class="divide-y divide-slate-100 md:hidden">
									{#each nestTasks(group.tasks, tasks) as row, i (row.task.id)}
										{@const task = row.task}
										{@const siblings = nestTasks(group.tasks, tasks).map((r) => r.task)}
										<li class="space-y-2 py-3 pr-4 text-sm" style="padding-left: {1 + row.depth * 1.25}rem">
											<button
												type="button"
												class="block w-full text-left"
												onclick={() => openEditTask(task)}
											>
												<span class="font-medium text-slate-800"
													>{#if row.depth}<span class="text-slate-400" aria-hidden="true">↳ </span>{/if}<TypeBadge type={task.type} label={task.customTypeLabel} color={task.customTypeColor} /> {task.title}</span
												>
												{#if row.depth === 0 && row.parent}
													<span class="block text-xs text-slate-500">↳ subtask of {row.parent.title}</span>
												{/if}
												{#if subtaskLabel(task)}
													<span class="block text-xs text-slate-500">{subtaskLabel(task)}</span>
												{/if}
												<DependencyNote {task} all={tasks} />
												{#if task.tags.length}
													<span class="mt-1 block"><TagChips tags={task.tags} /></span>
												{/if}
												<span class="mt-0.5 block text-xs text-slate-500">
													{assigneesLabel(task.assigneeIds)}{shownManDays(task) > 0 ? ` · ${formatManDays(shownManDays(task))} md` : ''}{task.storyPoints != null ? ` · ${formatManDays(task.storyPoints)} pts` : ''}{task.dueDate ? ` · Due ${task.dueDate}` : ''}
												</span>
												{#each fitWarnings(task) as warning (warning)}
													<span class="mt-0.5 block text-xs font-medium text-red-700">⚠ {warning}</span>
												{/each}
											</button>
											<div class="flex items-center gap-2">
												<Badge tone={taskPriorityTone(task.priority)}>{task.priority}</Badge>
												<select data-write-lock tabindex={can('member') ? 0 : -1}
													class="ml-auto min-h-10 rounded-md border-0 px-3 text-sm font-medium {badgeToneClass[
														taskStatusTone(task.status)
													]}"
													aria-label="Status of {task.title}"
													value={task.state}
													onchange={(e) => moveTask(task, e.currentTarget.value)}
												>
													{#each workflow as w (w.key)}<option value={w.key}>{w.label}</option>{/each}
</select>
											</div>
											<div class="flex gap-2">
												<button data-write
													type="button"
													class="btn-secondary px-3"
													aria-label="Move {task.title} up"
													disabled={i === 0}
													onclick={() => shift(task, siblings, -1)}>↑</button
												>
												<button data-write
													type="button"
													class="btn-secondary px-3"
													aria-label="Move {task.title} down"
													disabled={i === siblings.length - 1}
													onclick={() => shift(task, siblings, 1)}>↓</button
												>
												<button data-write type="button" class="btn-secondary flex-1" onclick={() => openEditTask(task)}>Edit</button>
												<button data-write
													type="button"
													class="btn-secondary flex-1 text-red-600 hover:bg-red-50"
													onclick={() => askRemoveTask(task)}
												>
													Delete
												</button>
											</div>
										</li>
									{/each}
								</ul>
								<table class="hidden w-full text-left text-sm md:table">
									<thead class="border-b border-slate-200 text-xs text-slate-500">
										<tr>
											<th class="px-4 py-2 font-medium">Task</th>
											<th class="px-4 py-2 font-medium">Assignees</th>
											<th class="px-4 py-2 text-right font-medium">Man-days</th>
											<th class="px-4 py-2 font-medium">Priority</th>
											<th class="px-4 py-2 font-medium">Status</th>
											<th class="px-4 py-2 font-medium">Due</th>
											<th class="px-4 py-2 font-medium"><span class="sr-only">Actions</span></th>
										</tr>
									</thead>
									<tbody class="divide-y divide-slate-100">
										{#each nestTasks(group.tasks, tasks) as row (row.task.id)}
											{@const task = row.task}
											{@const warnings = fitWarnings(task)}
											<tr
												class="hover:bg-slate-50 {draggingRow === task.id ? 'opacity-50' : ''}"
												draggable={can('member')}
												title="Drag to reorder"
												ondragstart={(e) => {
													draggingRow = task.id;
													if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
												}}
												ondragend={() => (draggingRow = null)}
												ondragover={(e) => {
													if (draggingRow && draggingRow !== task.id) e.preventDefault();
												}}
												ondrop={(e) => {
													e.preventDefault();
													const moving = tasks.find((x) => x.id === draggingRow);
													draggingRow = null;
													if (moving) placeBefore(moving, task.id);
												}}
											>
												<td class="py-2.5 pr-4" style="padding-left: {1 + row.depth * 1.5}rem">
													{#if row.depth}<span class="text-slate-400" aria-hidden="true">↳ </span>{/if}<TypeBadge type={task.type} label={task.customTypeLabel} color={task.customTypeColor} /> <button
														type="button"
														class="text-left font-medium text-slate-800 hover:underline"
														onclick={() => openEditTask(task)}
													>
														{task.title}
													</button>
													{#if row.depth === 0 && row.parent}
														<div class="text-xs text-slate-500">↳ subtask of {row.parent.title}</div>
													{/if}
													{#if subtaskLabel(task) || task.storyPoints != null || task.remainingManDays != null}
														<div class="text-xs text-slate-500">
															{[
																subtaskLabel(task),
																task.storyPoints != null ? `${formatManDays(task.storyPoints)} pts` : '',
																task.remainingManDays != null ? `${formatManDays(task.remainingManDays)} md left` : ''
															]
																.filter(Boolean)
																.join(' · ')}
														</div>
													{/if}
													<DependencyNote {task} all={tasks} />
													{#if task.tags.length}
														<div class="mt-1"><TagChips tags={task.tags} onSelect={addTagFilter} /></div>
													{/if}
												</td>
												<td class="px-4 py-2.5 text-slate-600" title={assigneesLabel(task.assigneeIds)}>
													<span class="line-clamp-1">{assigneesLabel(task.assigneeIds)}</span>
												</td>
												<td
													class="px-4 py-2.5 text-right tabular-nums {warnings.length ? 'font-medium text-red-700' : 'text-slate-600'}"
													title={warnings.length
														? `Doesn't fit before the due date — ${warnings.join('; ')}`
														: task.assignments.map((a) => `${personLabel(a.personId)}: ${formatManDays(a.manDays)}`).join(', ')}
												>
													{#if warnings.length}<span aria-hidden="true">⚠ </span>{/if}{shownManDays(task) > 0 ? formatManDays(shownManDays(task)) : '—'}
													{#if warnings.length}<span class="sr-only">Doesn't fit before the due date: {warnings.join('; ')}</span>{/if}
												</td>
												<td class="px-4 py-2.5">
													<Badge tone={taskPriorityTone(task.priority)}>{task.priority}</Badge>
												</td>
												<td class="px-4 py-2.5">
													<select data-write-lock tabindex={can('member') ? 0 : -1}
														class="rounded-full border-0 px-2 py-0.5 text-xs font-medium {badgeToneClass[
															taskStatusTone(task.status)
														]}"
														value={task.state}
													onchange={(e) => moveTask(task, e.currentTarget.value)}
												>
													{#each workflow as w (w.key)}<option value={w.key}>{w.label}</option>{/each}
</select>
												</td>
												<td class="px-4 py-2.5 text-xs text-slate-400">
													{task.dueDate ?? ''}
												</td>
												<td class="px-4 py-2.5 text-right whitespace-nowrap">
													<button data-write
														type="button"
														class="btn-ghost px-2 py-1"
														onclick={() => openEditTask(task)}
													>
														Edit
													</button>
													<button data-write
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
			{/if}
		</section>

		<section class="space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold tracking-tight">Weekly summary</h2>
					<p class="mt-0.5 text-sm text-slate-500">
						{weeklySummaries.length}
						{weeklySummaries.length === 1 ? 'entry' : 'entries'}
					</p>
				</div>
				<button data-write type="button" class="btn-primary" onclick={openAddWeeklySummary}>
					+ Add weekly summary
				</button>
			</div>

			{#if loadingWeeklySummaries}
				<p class="text-slate-500">Loading…</p>
			{:else if weeklySummariesError}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
					<p class="text-sm">{weeklySummariesError}</p>
				</div>
			{:else if weeklySummaries.length === 0}
				<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
					No weekly summaries yet. Click "Add weekly summary" to log this week's task progress.
				</div>
			{:else}
				<ul class="divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200 bg-white">
					{#each weeklySummaries as entry (entry.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
						<li
							class="cursor-pointer px-4 py-3 hover:bg-slate-50"
							onclick={() => openEditWeeklySummary(entry)}
						>
							<p class="text-sm font-medium text-slate-800">Week of {entry.weekOf}</p>
							<p class="mt-1 whitespace-pre-wrap text-sm text-slate-600">{entry.summary}</p>
						</li>
					{/each}
				</ul>
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

{#if editingProjectInfo && project}
	<ProjectInfoFormModal
		{project}
		onClose={() => (editingProjectInfo = false)}
		onSaved={handleProjectInfoSaved}
	/>
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
		{versions}
		onClose={closeMilestoneModal}
		onSaved={handleMilestoneSaved}
		onDelete={editingMilestone ? () => askRemoveMilestone(editingMilestone!) : undefined}
	/>
{/if}

{#if weeklySummaryModalMode}
	<WeeklySummaryFormModal
		{projectId}
		entry={editingWeeklySummary}
		onClose={closeWeeklySummaryModal}
		onSaved={handleWeeklySummarySaved}
		onDelete={editingWeeklySummary ? () => askRemoveWeeklySummary(editingWeeklySummary!) : undefined}
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
	{#key `${taskModalMode}:${editingTask?.id ?? taskModalParentId ?? ''}`}
	<TaskFormModal
		{projectId}
		task={editingTask}
		{milestoneOptions}
		areaOptions={areas}
		customFields={project?.customFields ?? []}
		workflow={project?.workflow ?? []}
		itemTypes={project?.workItemTypes ?? []}
		sprintOptions={sprints.filter((sp) => sp.status !== 'completed' && sp.status !== 'cancelled').map((sp) => ({ id: sp.id, name: sp.name }))}
		{assigneeOptions}
		initialMilestoneId={taskModalMilestoneId}
		projectTasks={tasks}
		initialParentId={taskModalParentId}
		initialType={taskModalParentId ? taskModalType : 'task'}
		onAddSubtask={openAddSubtask}
		onClose={closeTaskModal}
		onSaved={handleTaskSaved}
		onDelete={editingTask ? () => askRemoveTask(editingTask!) : undefined}
		templates={itemTemplates}
		{transferTargets}
		onCopy={editingTask ? (kids) => copyTask(editingTask!, kids) : undefined}
		onTransfer={editingTask ? (pid) => transferTask(editingTask!, pid) : undefined}
		onSaveTemplate={editingTask ? (name) => saveTemplate(editingTask!, name) : undefined}
		onDeleteTemplate={deleteTemplate}
	/>
	{/key}
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

{#if versionModalMode}
	<VersionFormModal
		{projectId}
		version={editingVersion}
		onClose={closeVersionModal}
		onSaved={handleVersionSaved}
		onDelete={editingVersion ? () => askRemoveVersion(editingVersion!) : undefined}
	/>
{/if}

<ConfirmDialog
	open={removingVersion !== null}
	title="Delete version"
	message={removingVersion
		? `Delete "${removingVersion.name}"? Milestones in "${removingVersion.name}" will move to "No version".`
		: ''}
	confirmLabel={removingVersionBusy ? 'Deleting…' : 'Delete'}
	danger
	error={removingVersionError}
	onConfirm={confirmRemoveVersion}
	onCancel={() => (removingVersion = null)}
/>

<ConfirmDialog
	open={removingWeeklySummary !== null}
	title="Delete weekly summary"
	message={removingWeeklySummary ? `Delete the entry for week of ${removingWeeklySummary.weekOf}? This can't be undone.` : ''}
	confirmLabel={removingWeeklySummaryBusy ? 'Deleting…' : 'Delete'}
	danger
	error={removingWeeklySummaryError}
	onConfirm={confirmRemoveWeeklySummary}
	onCancel={() => (removingWeeklySummary = null)}
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

{#if managingAreas}
	<AreaManager {projectId} {areas} onChange={loadAreas} onClose={() => (managingAreas = false)} />
{/if}

{#if importing}
	<ImportCsvModal
		{projectId}
		lookups={csvLookups}
		onClose={() => (importing = false)}
		onDone={(m) => {
			importing = false;
			flash(m.includes('failed') ? 'error' : 'success', m);
			loadTasks();
		}}
	/>
{/if}

{#if managingFields && project}
	<CustomFieldsManager fields={project.customFields} onSave={saveFields} onClose={() => (managingFields = false)} />
{/if}

{#if managingWorkflow && project}
	<WorkflowManager states={project.workflow} onSave={saveWorkflow} onClose={() => (managingWorkflow = false)} />
{/if}

{#if managingTypes && project}
	<TypesManager types={project.workItemTypes} onSave={saveTypes} onClose={() => (managingTypes = false)} />
{/if}

{#if managingTeams}
	<TeamsManager
		{projectId}
		{teams}
		people={Object.values(people).map((x) => ({ id: x.id, name: `${x.firstName} ${x.lastName}` })).sort((a, b) => a.name.localeCompare(b.name))}
		{areas}
		onChange={loadTeams}
		onClose={() => (managingTeams = false)}
	/>
{/if}

{#if managingRules && project}
	<RulesManager rules={project.rules} states={project.workflow} fields={project.customFields} onSave={saveRules} onClose={() => (managingRules = false)} />
{/if}

{#if managingHooks && project}
	<HooksManager projectId={project.id} onClose={() => (managingHooks = false)} />
{/if}

{#if managingProcess && project}
	<ProcessManager
		{project}
		onApplied={(p) => {
			project = p;
			flash('success', 'Process applied.');
			loadTasks();
		}}
		onClose={() => (managingProcess = false)}
	/>
{/if}
