<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ProjectFormModal from '$lib/components/ProjectFormModal.svelte';
	import ProjectMemberFormModal from '$lib/components/ProjectMemberFormModal.svelte';
	import AddTeamMembersModal from '$lib/components/AddTeamMembersModal.svelte';
	import ContributionFormModal from '$lib/components/ContributionFormModal.svelte';
	import MilestoneFormModal from '$lib/components/MilestoneFormModal.svelte';
	import MilestoneTimeline from '$lib/components/MilestoneTimeline.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	type Project = components['schemas']['Project'];
	type ProjectMember = components['schemas']['ProjectMember'];
	type Person = components['schemas']['Person'];
	type Contribution = components['schemas']['Contribution'];
	type Milestone = components['schemas']['Milestone'];

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

	let milestones = $state<Milestone[]>([]);
	let loadingMilestones = $state(true);
	let milestonesError = $state<string | null>(null);

	let milestoneModalMode = $state<'create' | 'edit' | null>(null);
	let editingMilestone = $state<Milestone | null>(null);

	let removingMilestone = $state<Milestone | null>(null);
	let removingMilestoneBusy = $state(false);
	let removingMilestoneError = $state<string | null>(null);

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

	// UTC-safe day diff: comparing plain "YYYY-MM-DD" dates via `new Date(str)`
	// parses as UTC midnight, so as long as "today" is also built from UTC
	// components (not local wall-clock time) the diff can't be off by one
	// depending on the viewer's timezone.
	function daysUntil(dateStr: string): number {
		const target = new Date(`${dateStr}T00:00:00Z`).getTime();
		const now = new Date();
		const todayUtc = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
		return Math.round((target - todayUtc) / 86_400_000);
	}

	const daysToTarget = $derived(project?.targetDate ? daysUntil(project.targetDate) : null);

	function contributionTypeLabel(type: Contribution['type']): string {
		return type.replace('_', ' ');
	}

	function personLabel(id: string): string {
		const p = people[id];
		return p ? `${p.firstName} ${p.lastName}` : 'Unknown person';
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
	}

	function statusBadgeClass(status: Project['status']): string {
		switch (status) {
			case 'active':
				return 'bg-emerald-100 text-emerald-700';
			case 'planning':
				return 'bg-sky-100 text-sky-700';
			case 'on_hold':
				return 'bg-amber-100 text-amber-700';
			case 'completed':
				return 'bg-slate-200 text-slate-600';
			case 'cancelled':
				return 'bg-red-100 text-red-700';
		}
	}

	function healthDotClass(health: Project['health']): string {
		switch (health) {
			case 'on_track':
				return 'bg-emerald-500';
			case 'at_risk':
				return 'bg-amber-500';
			case 'off_track':
				return 'bg-red-500';
		}
	}

	function roleBadgeClass(role: ProjectMember['role']): string {
		switch (role) {
			case 'lead':
				return 'bg-slate-900 text-white';
			case 'maintainer':
				return 'bg-violet-100 text-violet-700';
			case 'contributor':
				return 'bg-slate-100 text-slate-600';
			case 'reviewer':
				return 'bg-sky-100 text-sky-700';
			case 'stakeholder':
				return 'bg-amber-100 text-amber-700';
		}
	}

	onMount(() => {
		loadProject();
		loadMembers();
		loadPeople();
		loadActivity();
		loadMilestones();
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
						<span class="rounded-full px-2 py-0.5 text-xs {statusBadgeClass(project.status)}">
							{project.status.replace('_', ' ')}
						</span>
						<span class="inline-flex items-center gap-1.5 text-xs text-slate-500">
							<span class="h-2 w-2 rounded-full {healthDotClass(project.health)}"></span>
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
				<div>
					<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">Team members</dt>
					<dd class="mt-1 text-xl font-semibold text-slate-900">{existingPersonIds.length}</dd>
				</div>
				<div>
					<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">Contributions</dt>
					<dd class="mt-1 text-xl font-semibold text-slate-900">
						{loadingActivity ? '—' : (contributionsTotal ?? 0)}
					</dd>
				</div>
				<div>
					<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">Avg. allocation</dt>
					<dd class="mt-1 text-xl font-semibold text-slate-900">
						{activeMembers.length > 0 ? `${avgAllocation}%` : '—'}
					</dd>
				</div>
				<div>
					<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">
						{daysToTarget !== null && daysToTarget < 0 ? 'Overdue by' : 'Days to target'}
					</dt>
					<dd
						class="mt-1 text-xl font-semibold {daysToTarget !== null && daysToTarget < 0
							? 'text-red-600'
							: 'text-slate-900'}"
					>
						{daysToTarget !== null ? Math.abs(daysToTarget) : '—'}
					</dd>
				</div>
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
										<span class="rounded-full px-2 py-0.5 text-xs {roleBadgeClass(member.role)}">
											{member.role}
										</span>
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
