<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';
	import { PROJECT_ROLE_OPTIONS, projectRoleLabel } from '$lib/projectRoles';

	type Project = components['schemas']['Project'];
	type ProjectPriority = components['schemas']['ProjectPriority'];
	type ProjectRole = components['schemas']['ProjectRole'];
	type Person = components['schemas']['Person'];

	let {
		onClose,
		onCreated
	}: {
		onClose: () => void;
		// Called once the project (and whatever staffing/milestones succeeded)
		// has actually been created — the caller navigates to it.
		onCreated: (project: Project) => void;
	} = $props();

	const STEP_LABELS = ['Basics', 'Staff the team', 'Seed milestones', 'Review & create'];
	let step = $state(1);

	// ---- Step 1: Basics (same field set ProjectFormModal uses for create) ----
	let key = $state('');
	let name = $state('');
	let description = $state('');
	let priority = $state<ProjectPriority>('medium');
	let leadId = $state('');
	let startDate = $state('');
	let targetDate = $state('');

	const priorityOptions: ProjectPriority[] = ['low', 'medium', 'high', 'critical'];

	let step1Error = $state<string | null>(null);

	function validateStep1(): boolean {
		if (!key.trim() || !name.trim()) {
			step1Error = 'Key and name are required.';
			return false;
		}
		step1Error = null;
		return true;
	}

	// ---- Step 2: Staff the team ----
	// Nothing here is persisted until the final "Create project" submit — we
	// only build up a local draft list, then batch-POST /project-members once
	// the project (and its real id) exists, mirroring AddTeamMembersModal's
	// Promise.allSettled batching pattern.
	let people = $state<Person[]>([]);
	let loadingPeople = $state(true);
	let teamSearch = $state('');
	let selected = $state<Set<string>>(new Set());
	let role = $state<ProjectRole>('contributor');
	let allocationPct = $state(100);

	type DraftMember = { personId: string; name: string; role: ProjectRole; allocationPct: number };
	let draftTeam = $state<DraftMember[]>([]);

	const stagedIds = $derived(new Set(draftTeam.map((m) => m.personId)));

	const teamCandidates = $derived(people.filter((p) => !stagedIds.has(p.id)));

	const filteredTeamCandidates = $derived(
		(() => {
			const q = teamSearch.trim().toLowerCase();
			if (!q) return teamCandidates;
			return teamCandidates.filter((p) => {
				const full = `${p.firstName} ${p.lastName}`.toLowerCase();
				return full.includes(q) || p.email.toLowerCase().includes(q);
			});
		})()
	);

	function fullName(p: Person): string {
		return `${p.firstName} ${p.lastName}`;
	}

	function leadName(id: string): string {
		const p = people.find((person) => person.id === id);
		return p ? fullName(p) : 'Selected';
	}

	function toggleCandidate(id: string) {
		const next = new Set(selected);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selected = next;
	}

	function addSelectedToTeam() {
		if (selected.size === 0) return;
		const additions: DraftMember[] = [];
		for (const id of selected) {
			const p = people.find((person) => person.id === id);
			if (!p) continue;
			additions.push({ personId: p.id, name: fullName(p), role, allocationPct });
		}
		draftTeam = [...draftTeam, ...additions];
		selected = new Set();
	}

	function removeFromTeam(personId: string) {
		draftTeam = draftTeam.filter((m) => m.personId !== personId);
	}

	// ---- Step 3: Seed milestones ----
	// Same field shape as MilestoneFormModal, minus description (kept out of
	// the compact repeatable row to keep the list scannable — a milestone
	// added this way can still get a description later via the normal edit
	// modal on the project page).
	type DraftMilestone = { name: string; startDate: string; dueDate: string };
	let draftMilestones = $state<DraftMilestone[]>([]);

	let newMilestoneName = $state('');
	let newMilestoneStart = $state('');
	let newMilestoneDue = $state('');
	let milestoneRowError = $state<string | null>(null);

	function addMilestoneRow() {
		milestoneRowError = null;
		if (!newMilestoneName.trim()) {
			milestoneRowError = 'Milestone name is required.';
			return;
		}
		if (newMilestoneStart && newMilestoneDue && newMilestoneStart > newMilestoneDue) {
			milestoneRowError = 'Start date must be on or before the due date.';
			return;
		}
		draftMilestones = [
			...draftMilestones,
			{ name: newMilestoneName.trim(), startDate: newMilestoneStart, dueDate: newMilestoneDue }
		];
		newMilestoneName = '';
		newMilestoneStart = '';
		newMilestoneDue = '';
	}

	function removeMilestone(index: number) {
		draftMilestones = draftMilestones.filter((_, i) => i !== index);
	}

	// ---- Step 4: Review & create ----
	let creating = $state(false);
	let createError = $state<string | null>(null);
	// Set once the project itself is created, even if some staffing/milestone
	// batch items failed — lets the review step switch to a "Go to project"
	// recovery state instead of re-showing the whole form.
	let createdProject = $state<Project | null>(null);

	async function handleCreate() {
		createError = null;
		creating = true;
		try {
			const { data: project, error } = await api.POST('/api/v1/projects', {
				body: {
					key,
					name,
					description: description || undefined,
					priority,
					leadId: leadId || undefined,
					startDate: startDate || undefined,
					targetDate: targetDate || undefined
				}
			});
			if (error || !project) {
				createError = error?.message ?? 'Failed to create project.';
				return;
			}
			createdProject = project;

			const [memberResults, milestoneResults] = await Promise.all([
				Promise.allSettled(
					draftTeam.map((m) =>
						api.POST('/api/v1/project-members', {
							body: {
								projectId: project.id,
								personId: m.personId,
								role: m.role,
								allocationPct: m.allocationPct
							}
						})
					)
				),
				Promise.allSettled(
					draftMilestones.map((m) =>
						api.POST('/api/v1/milestones', {
							body: {
								projectId: project.id,
								name: m.name,
								startDate: m.startDate || undefined,
								dueDate: m.dueDate || undefined
							}
						})
					)
				)
			]);

			const failedMembers = memberResults.filter(
				(r) => r.status === 'rejected' || r.value.error
			).length;
			const failedMilestones = milestoneResults.filter(
				(r) => r.status === 'rejected' || r.value.error
			).length;

			if (failedMembers > 0 || failedMilestones > 0) {
				const parts: string[] = [];
				if (failedMembers > 0) parts.push(`${failedMembers} team member${failedMembers === 1 ? '' : 's'}`);
				if (failedMilestones > 0) parts.push(`${failedMilestones} milestone${failedMilestones === 1 ? '' : 's'}`);
				createError = `${project.name} was created, but ${parts.join(' and ')} couldn't be added. You can add ${failedMembers > 0 && failedMilestones > 0 ? 'them' : parts[0]} from the project page.`;
				return;
			}

			onCreated(project);
		} finally {
			creating = false;
		}
	}

	function next() {
		if (step === 1 && !validateStep1()) return;
		step = Math.min(4, step + 1);
	}

	function back() {
		step = Math.max(1, step - 1);
	}

	onMount(async () => {
		loadingPeople = true;
		const { data } = await api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } });
		if (data) people = data.data;
		loadingPeople = false;
	});
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && !creating && onClose()} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
	role="presentation"
	onclick={() => !creating && onClose()}
>
	<div
		class="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-lg bg-white shadow-xl"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="wizard-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between border-b border-slate-200 px-6 py-4">
			<div>
				<h3 id="wizard-modal-title" class="text-lg font-semibold text-slate-900">New project</h3>
				<p class="mt-0.5 text-xs text-slate-500">Step {step} of 4 — {STEP_LABELS[step - 1]}</p>
			</div>
			<button
				type="button"
				class="text-slate-400 hover:text-slate-600"
				onclick={onClose}
				aria-label="Close"
				disabled={creating}
			>
				✕
			</button>
		</div>

		<div class="flex items-center gap-2 px-6 pt-4">
			{#each STEP_LABELS as label, i (label)}
				{@const n = i + 1}
				<div class="flex flex-1 items-center gap-2">
					<span
						class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-medium {n <= step
							? 'bg-slate-900 text-white'
							: 'bg-slate-200 text-slate-500'}"
					>
						{n}
					</span>
					{#if n < 4}
						<span class="h-px flex-1 {n < step ? 'bg-slate-900' : 'bg-slate-200'}"></span>
					{/if}
				</div>
			{/each}
		</div>

		<div class="min-h-0 flex-1 overflow-y-auto px-6 py-4">
			{#if step === 1}
				<div class="space-y-4">
					{#if step1Error}
						<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
							{step1Error}
						</div>
					{/if}

					<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
						<label class="block text-sm sm:col-span-1">
							<span class="mb-1 block font-medium text-slate-700">Key *</span>
							<input class="input uppercase" bind:value={key} placeholder="PYR" required />
						</label>
						<label class="block text-sm sm:col-span-2">
							<span class="mb-1 block font-medium text-slate-700">Name *</span>
							<input class="input" bind:value={name} required />
						</label>
					</div>

					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Description</span>
						<textarea class="input" rows="2" bind:value={description}></textarea>
					</label>

					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Priority</span>
						<select class="input" bind:value={priority}>
							{#each priorityOptions as opt (opt)}
								<option value={opt}>{opt}</option>
							{/each}
						</select>
					</label>

					<PersonSelect bind:value={leadId} label="Lead" emptyLabel="No lead assigned" />

					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">Start date</span>
							<input type="date" class="input" bind:value={startDate} />
						</label>
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">Target date</span>
							<input type="date" class="input" bind:value={targetDate} />
						</label>
					</div>
				</div>
			{:else if step === 2}
				<div class="space-y-4">
					<p class="text-sm text-slate-500">
						Nothing is saved yet — staffing is applied once you create the project.
					</p>

					{#if draftTeam.length > 0}
						<ul class="divide-y divide-slate-100 rounded-md border border-slate-200">
							{#each draftTeam as m (m.personId)}
								<li class="flex items-center justify-between px-3 py-2 text-sm">
									<span class="text-slate-800">
										{m.name}
										<span class="ml-1.5 text-xs text-slate-500"
											>{projectRoleLabel[m.role]} · {m.allocationPct}%</span
										>
									</span>
									<button
										type="button"
										class="text-xs text-red-600 hover:underline"
										onclick={() => removeFromTeam(m.personId)}
									>
										Remove
									</button>
								</li>
							{/each}
						</ul>
					{/if}

					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">Role for selected</span>
							<select class="input" bind:value={role}>
								{#each PROJECT_ROLE_OPTIONS as opt (opt)}
									<option value={opt}>{projectRoleLabel[opt]}</option>
								{/each}
							</select>
						</label>
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">Allocation for selected</span>
							<div class="flex items-center gap-2">
								<input
									type="number"
									min="0"
									max="100"
									step="5"
									class="input"
									bind:value={allocationPct}
								/>
								<span class="text-sm text-slate-500">%</span>
							</div>
						</label>
					</div>

					<input
						class="input"
						type="search"
						placeholder="Search by name or email…"
						bind:value={teamSearch}
					/>

					<div class="max-h-48 overflow-y-auto rounded-md border border-slate-200">
						{#if loadingPeople}
							<p class="p-4 text-sm text-slate-500">Loading people…</p>
						{:else if filteredTeamCandidates.length === 0}
							<p class="p-4 text-sm text-slate-500">
								{teamCandidates.length === 0 ? 'Everyone has been staged.' : 'No one matches your search.'}
							</p>
						{:else}
							<ul class="divide-y divide-slate-100">
								{#each filteredTeamCandidates as p (p.id)}
									<li>
										<label class="flex cursor-pointer items-center gap-2 px-3 py-2 text-sm hover:bg-slate-50">
											<input
												type="checkbox"
												checked={selected.has(p.id)}
												onchange={() => toggleCandidate(p.id)}
											/>
											<span class="text-slate-800">{fullName(p)}</span>
											<span class="text-xs text-slate-400">{p.email}</span>
										</label>
									</li>
								{/each}
							</ul>
						{/if}
					</div>

					<div class="flex justify-end">
						<button
							type="button"
							class="btn-secondary"
							disabled={selected.size === 0}
							onclick={addSelectedToTeam}
						>
							Add {selected.size || ''} to team
						</button>
					</div>
				</div>
			{:else if step === 3}
				<div class="space-y-4">
					<p class="text-sm text-slate-500">
						Nothing is saved yet — milestones are created once you create the project.
					</p>

					{#if draftMilestones.length > 0}
						<ul class="divide-y divide-slate-100 rounded-md border border-slate-200">
							{#each draftMilestones as m, i (i)}
								<li class="flex items-center justify-between px-3 py-2 text-sm">
									<span class="text-slate-800">
										{m.name}
										{#if m.startDate || m.dueDate}
											<span class="ml-1.5 text-xs text-slate-500">
												{m.startDate || '—'} → {m.dueDate || '—'}
											</span>
										{/if}
									</span>
									<button
										type="button"
										class="text-xs text-red-600 hover:underline"
										onclick={() => removeMilestone(i)}
									>
										Remove
									</button>
								</li>
							{/each}
						</ul>
					{/if}

					{#if milestoneRowError}
						<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
							{milestoneRowError}
						</div>
					{/if}

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
						<label class="block text-sm sm:col-span-1">
							<span class="mb-1 block font-medium text-slate-700">Name</span>
							<input class="input" bind:value={newMilestoneName} placeholder="e.g. Beta launch" />
						</label>
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">Start date</span>
							<input type="date" class="input" bind:value={newMilestoneStart} />
						</label>
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">Due date</span>
							<input type="date" class="input" bind:value={newMilestoneDue} />
						</label>
					</div>

					<div class="flex justify-end">
						<button type="button" class="btn-secondary" onclick={addMilestoneRow}>
							+ Add milestone
						</button>
					</div>
				</div>
			{:else if step === 4}
				<div class="space-y-4">
					{#if createError}
						<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
							{createError}
							{#if createdProject}
								<div class="mt-2">
									<button
										type="button"
										class="btn-secondary"
										onclick={() => createdProject && onCreated(createdProject)}
									>
										Go to project
									</button>
								</div>
							{/if}
						</div>
					{/if}

					<div class="rounded-md border border-slate-200 p-4">
						<h4 class="text-sm font-semibold text-slate-900">{name || '(untitled project)'}</h4>
						<p class="mt-0.5 font-mono text-xs text-slate-500">{key || '—'}</p>
						{#if description}
							<p class="mt-2 text-sm text-slate-600">{description}</p>
						{/if}
						<dl class="mt-3 grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
							<div>
								<dt class="text-xs text-slate-400">Priority</dt>
								<dd class="text-slate-700 capitalize">{priority}</dd>
							</div>
							<div>
								<dt class="text-xs text-slate-400">Start</dt>
								<dd class="text-slate-700">{startDate || '—'}</dd>
							</div>
							<div>
								<dt class="text-xs text-slate-400">Target</dt>
								<dd class="text-slate-700">{targetDate || '—'}</dd>
							</div>
							<div>
								<dt class="text-xs text-slate-400">Lead</dt>
								<dd class="text-slate-700">
									{leadId ? leadName(leadId) : '—'}
								</dd>
							</div>
						</dl>
					</div>

					<div>
						<h5 class="text-xs font-medium tracking-wide text-slate-500 uppercase">
							Team ({draftTeam.length})
						</h5>
						{#if draftTeam.length === 0}
							<p class="mt-1 text-sm text-slate-400">No team members staged yet.</p>
						{:else}
							<ul class="mt-1 space-y-1 text-sm text-slate-700">
								{#each draftTeam as m (m.personId)}
									<li>{m.name} — {projectRoleLabel[m.role]}, {m.allocationPct}%</li>
								{/each}
							</ul>
						{/if}
						<button type="button" class="mt-1 text-xs text-slate-500 hover:underline" onclick={() => (step = 2)}>
							Edit team
						</button>
					</div>

					<div>
						<h5 class="text-xs font-medium tracking-wide text-slate-500 uppercase">
							Milestones ({draftMilestones.length})
						</h5>
						{#if draftMilestones.length === 0}
							<p class="mt-1 text-sm text-slate-400">No milestones staged yet.</p>
						{:else}
							<ul class="mt-1 space-y-1 text-sm text-slate-700">
								{#each draftMilestones as m, i (i)}
									<li>{m.name}</li>
								{/each}
							</ul>
						{/if}
						<button type="button" class="mt-1 text-xs text-slate-500 hover:underline" onclick={() => (step = 3)}>
							Edit milestones
						</button>
					</div>
				</div>
			{/if}
		</div>

		<div class="flex items-center justify-between border-t border-slate-200 px-6 py-4">
			<button type="button" class="btn-secondary" onclick={onClose} disabled={creating}>Cancel</button>
			<div class="flex gap-2">
				{#if step > 1}
					<button type="button" class="btn-secondary" onclick={back} disabled={creating}>Back</button>
				{/if}
				{#if step < 4}
					<button type="button" class="btn-primary" onclick={next}>
						{step === 1 ? 'Next' : 'Next'}
					</button>
				{:else}
					<button type="button" class="btn-primary" onclick={handleCreate} disabled={creating || createdProject !== null}>
						{creating ? 'Creating…' : 'Create project'}
					</button>
				{/if}
			</div>
		</div>
	</div>
</div>
