<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';
	import { PROJECT_ROLE_OPTIONS, projectRoleLabel } from '$lib/projectRoles';

	type ProjectMember = components['schemas']['ProjectMember'];
	type ProjectRole = components['schemas']['ProjectRole'];

	let {
		projectId,
		member = null,
		personDisplayName = '',
		existingPersonIds = [],
		onClose,
		onSaved
	}: {
		projectId: string;
		member?: ProjectMember | null;
		// Read-only name shown in edit mode, since the person can't be changed
		// after assignment — reassigning means removing and re-adding.
		personDisplayName?: string;
		// People already on this project, hidden from the picker when adding.
		existingPersonIds?: string[];
		onClose: () => void;
		onSaved: (member: ProjectMember) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = member !== null;

	// svelte-ignore state_referenced_locally
	const initial = member;

	let personId = $state(initial?.personId ?? '');
	let role = $state<ProjectRole>(initial?.role ?? 'contributor');
	let allocationPct = $state(initial?.allocationPct ?? 100);
	let leftAt = $state(initial?.leftAt ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	// Over-allocation hint: once a person is picked, show how much of their
	// time is already committed elsewhere so a PM can catch a conflict before
	// it happens rather than after.
	let otherAllocationPct = $state<number | null>(null);
	let otherProjectCount = $state(0);

	async function checkOtherAllocations(id: string) {
		otherAllocationPct = null;
		otherProjectCount = 0;
		if (!id) return;

		const { data } = await api.GET('/api/v1/project-members', {
			params: { query: { personId: id, pageSize: 100 } }
		});
		if (!data) return;

		const active = data.data.filter((m) => !m.leftAt && m.projectId !== projectId);
		otherAllocationPct = active.reduce((sum, m) => sum + m.allocationPct, 0);
		otherProjectCount = active.length;
	}

	if (initial?.personId) {
		checkOtherAllocations(initial.personId);
	}

	const wouldExceedCapacity = $derived(
		otherAllocationPct !== null && otherAllocationPct + allocationPct > 100
	);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!isEdit && !personId) {
			formError = 'Select a person to add.';
			return;
		}

		saving = true;
		try {
			if (isEdit && member) {
				const { data, error } = await api.PATCH('/api/v1/project-members/{id}', {
					params: { path: { id: member.id } },
					body: {
						role,
						allocationPct,
						leftAt: leftAt || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update assignment.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/project-members', {
					body: {
						projectId,
						personId,
						role,
						allocationPct
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to add person to project.';
					return;
				}
				if (data) onSaved(data);
			}
		} finally {
			saving = false;
		}
	}
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onClose()} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 sm:p-4"
	role="presentation"
	onclick={onClose}
>
	<div
		class="h-dvh max-h-dvh w-full max-w-md overflow-y-auto bg-white p-4 sm:p-6 shadow-xl sm:h-auto sm:max-h-[90vh] sm:rounded-lg"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="member-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="member-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit assignment' : 'Add person to project'}
			</h3>
			<button
				type="button"
				class="text-slate-400 hover:text-slate-600"
				onclick={onClose}
				aria-label="Close"
			>
				✕
			</button>
		</div>

		<form class="mt-4 space-y-4" onsubmit={handleSubmit}>
			{#if formError}
				<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
					{formError}
				</div>
			{/if}

			{#if isEdit}
				<div class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700">
					{personDisplayName}
				</div>
			{:else}
				<PersonSelect
					bind:value={personId}
					label="Person"
					required
					emptyLabel="Select a person"
					excludeIds={existingPersonIds}
					onchange={checkOtherAllocations}
				/>
			{/if}

			{#if otherAllocationPct !== null && otherProjectCount > 0}
				<p class="text-xs {wouldExceedCapacity ? 'text-amber-600' : 'text-slate-500'}">
					Already allocated {otherAllocationPct}% across {otherProjectCount}
					{otherProjectCount === 1 ? 'other project' : 'other projects'}.
					{#if wouldExceedCapacity}
						This assignment would put them over 100%.
					{/if}
				</p>
			{/if}

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Role</span>
					<select class="input" bind:value={role}>
						{#each PROJECT_ROLE_OPTIONS as opt (opt)}
							<option value={opt}>{projectRoleLabel[opt]}</option>
						{/each}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Allocation</span>
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

			{#if isEdit}
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Left project on</span>
					<input type="date" class="input" bind:value={leftAt} />
					<span class="mt-1 block text-xs text-slate-400">
						Leave blank while they're still active on this project.
					</span>
				</label>
			{/if}

			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add to project'}
				</button>
			</div>
		</form>
	</div>
</div>
