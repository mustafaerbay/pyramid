<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';

	type Goal = components['schemas']['Goal'];
	type GoalScope = components['schemas']['GoalScope'];
	type GoalStatus = components['schemas']['GoalStatus'];

	let {
		goal = null,
		onClose,
		onSaved
	}: {
		goal?: Goal | null;
		onClose: () => void;
		onSaved: (goal: Goal) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = goal !== null;

	// svelte-ignore state_referenced_locally
	const initial = goal;

	let title = $state(initial?.title ?? '');
	let description = $state(initial?.description ?? '');
	let scope = $state<GoalScope>(initial?.scope ?? 'individual');
	let status = $state<GoalStatus>(initial?.status ?? 'draft');
	let ownerPersonId = $state(initial?.ownerPersonId ?? '');
	let progressPct = $state(initial?.progressPct ?? 0);
	let startDate = $state(initial?.startDate ?? '');
	let dueDate = $state(initial?.dueDate ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const scopeOptions: GoalScope[] = ['individual', 'team', 'department', 'company'];
	const statusOptions: GoalStatus[] = [
		'draft',
		'active',
		'achieved',
		'partially_achieved',
		'missed',
		'cancelled'
	];

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!title.trim()) {
			formError = 'Title is required.';
			return;
		}

		saving = true;
		try {
			if (isEdit && goal) {
				const { data, error } = await api.PATCH('/api/v1/goals/{id}', {
					params: { path: { id: goal.id } },
					body: {
						title,
						description: description || undefined,
						scope,
						status,
						ownerPersonId: ownerPersonId || undefined,
						progressPct,
						startDate: startDate || undefined,
						dueDate: dueDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update goal.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/goals', {
					body: {
						title,
						description: description || undefined,
						scope,
						ownerPersonId: ownerPersonId || undefined,
						startDate: startDate || undefined,
						dueDate: dueDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create goal.';
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
		class="h-dvh max-h-dvh w-full max-w-lg overflow-y-auto bg-white p-4 sm:p-6 shadow-xl sm:h-auto sm:max-h-[90vh] sm:rounded-lg"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="goal-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="goal-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit goal' : 'Add goal'}
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

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Title *</span>
				<input class="input" bind:value={title} required />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Description</span>
				<textarea class="input" rows="2" bind:value={description}></textarea>
			</label>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Scope</span>
					<select class="input" bind:value={scope}>
						{#each scopeOptions as opt (opt)}
							<option value={opt}>{opt}</option>
						{/each}
					</select>
				</label>
				{#if isEdit}
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Status</span>
						<select class="input" bind:value={status}>
							{#each statusOptions as opt (opt)}
								<option value={opt}>{opt.replace('_', ' ')}</option>
							{/each}
						</select>
					</label>
				{/if}
			</div>

			<PersonSelect bind:value={ownerPersonId} label="Owner" emptyLabel="No individual owner" />

			{#if isEdit}
				<label class="block text-sm">
					<span class="mb-1 flex items-center justify-between font-medium text-slate-700">
						<span>Progress</span>
						<span class="text-slate-500">{progressPct}%</span>
					</span>
					<input type="range" min="0" max="100" step="5" class="w-full" bind:value={progressPct} />
				</label>
			{/if}

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Start date</span>
					<input type="date" class="input" bind:value={startDate} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Due date</span>
					<input type="date" class="input" bind:value={dueDate} />
				</label>
			</div>

			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add goal'}
				</button>
			</div>
		</form>
	</div>
</div>
