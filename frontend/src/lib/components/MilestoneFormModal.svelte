<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Milestone = components['schemas']['Milestone'];
	type MilestoneStatus = components['schemas']['MilestoneStatus'];

	let {
		projectId,
		milestone = null,
		onClose,
		onSaved,
		onDelete
	}: {
		projectId: string;
		milestone?: Milestone | null;
		onClose: () => void;
		onSaved: (milestone: Milestone) => void;
		// Shown as a "Delete milestone" link in edit mode only. The parent
		// owns the actual confirm-and-delete flow (same split as elsewhere in
		// the app, e.g. project member removal).
		onDelete?: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = milestone !== null;

	// svelte-ignore state_referenced_locally
	const initial = milestone;

	let name = $state(initial?.name ?? '');
	let description = $state(initial?.description ?? '');
	let startDate = $state(initial?.startDate ?? '');
	let dueDate = $state(initial?.dueDate ?? '');
	// A new milestone always starts pending server-side (matches Goal — status
	// is update-only), so this control only shows in edit mode.
	let status = $state<MilestoneStatus>(initial?.status ?? 'pending');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const statusOptions: MilestoneStatus[] = ['pending', 'in_progress', 'completed', 'missed'];

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!name.trim()) {
			formError = 'Name is required.';
			return;
		}

		if (startDate && dueDate && startDate > dueDate) {
			formError = 'Start date must be on or before the due date.';
			return;
		}

		saving = true;
		try {
			if (isEdit && milestone) {
				const { data, error } = await api.PATCH('/api/v1/milestones/{id}', {
					params: { path: { id: milestone.id } },
					body: {
						name,
						description: description || undefined,
						status,
						startDate: startDate || undefined,
						dueDate: dueDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update milestone.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/milestones', {
					body: {
						projectId,
						name,
						description: description || undefined,
						startDate: startDate || undefined,
						dueDate: dueDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create milestone.';
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
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
	role="presentation"
	onclick={onClose}
>
	<div
		class="max-h-[90vh] w-full max-w-md overflow-y-auto rounded-lg bg-white p-6 shadow-xl"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="milestone-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="milestone-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit milestone' : 'Add milestone'}
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
				<span class="mb-1 block font-medium text-slate-700">Name *</span>
				<input class="input" bind:value={name} placeholder="e.g. Beta launch" required />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Description</span>
				<textarea class="input" rows="2" bind:value={description}></textarea>
			</label>

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

			<div class="flex items-center justify-between gap-2 pt-2">
				{#if isEdit && onDelete}
					<button
						type="button"
						class="text-sm text-red-600 hover:text-red-700 hover:underline"
						onclick={onDelete}
					>
						Delete milestone
					</button>
				{:else}
					<span></span>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={saving}>
						{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add milestone'}
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
