<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];
	type TaskPriority = components['schemas']['TaskPriority'];

	let {
		projectId,
		task = null,
		milestoneOptions = [],
		assigneeOptions = [],
		initialMilestoneId = null,
		onClose,
		onSaved,
		onDelete
	}: {
		projectId: string;
		task?: Task | null;
		// The project's milestones, for the "which milestone is this part of"
		// picker — tasks don't have to belong to one (project-level backlog).
		milestoneOptions?: { id: string; name: string }[];
		// The project's active team members — a task can only be assigned to
		// someone already on the team, not any person in the org.
		assigneeOptions?: { id: string; name: string }[];
		// Prefills the milestone picker when creating a task from a specific
		// milestone group's "+ Add task" button.
		initialMilestoneId?: string | null;
		onClose: () => void;
		onSaved: (task: Task) => void;
		// Shown as a "Delete task" link in edit mode only. The parent owns the
		// actual confirm-and-delete flow (same split as milestones/members).
		onDelete?: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = task !== null;

	// svelte-ignore state_referenced_locally
	const initial = task;

	let title = $state(initial?.title ?? '');
	let description = $state(initial?.description ?? '');
	// svelte-ignore state_referenced_locally
	let milestoneId = $state(initial?.milestoneId ?? initialMilestoneId ?? '');
	let assigneeId = $state(initial?.assigneeId ?? '');
	// Unlike milestones (always start "pending"), a task's initial status is
	// caller-settable — it's normal to log work retroactively that's already
	// in progress or done.
	let status = $state<TaskStatus>(initial?.status ?? 'todo');
	let priority = $state<TaskPriority>(initial?.priority ?? 'medium');
	let dueDate = $state(initial?.dueDate ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const statusOptions: TaskStatus[] = [
		'backlog',
		'todo',
		'in_progress',
		'in_review',
		'blocked',
		'done',
		'cancelled'
	];
	const priorityOptions: TaskPriority[] = ['low', 'medium', 'high', 'urgent'];

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!title.trim()) {
			formError = 'Title is required.';
			return;
		}

		saving = true;
		try {
			if (isEdit && task) {
				const { data, error } = await api.PATCH('/api/v1/tasks/{id}', {
					params: { path: { id: task.id } },
					body: {
						title,
						description: description || undefined,
						// Explicit null (not undefined) so the backend can tell
						// "clear this" apart from "field wasn't touched".
						milestoneId: milestoneId || null,
						assigneeId: assigneeId || null,
						status,
						priority,
						dueDate: dueDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update task.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/tasks', {
					body: {
						projectId,
						milestoneId: milestoneId || undefined,
						title,
						description: description || undefined,
						assigneeId: assigneeId || undefined,
						status,
						priority,
						dueDate: dueDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create task.';
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
		aria-labelledby="task-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="task-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit task' : 'Add task'}
			</h3>
			<button type="button" class="text-slate-400 hover:text-slate-600" onclick={onClose} aria-label="Close">
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
				<input class="input" bind:value={title} placeholder="e.g. Wire up auth middleware" required />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Description</span>
				<textarea class="input" rows="2" bind:value={description}></textarea>
			</label>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Milestone</span>
					<select class="input" bind:value={milestoneId}>
						<option value="">No milestone</option>
						{#each milestoneOptions as m (m.id)}
							<option value={m.id}>{m.name}</option>
						{/each}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Assignee</span>
					<select class="input" bind:value={assigneeId}>
						<option value="">Unassigned</option>
						{#each assigneeOptions as person (person.id)}
							<option value={person.id}>{person.name}</option>
						{/each}
					</select>
				</label>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Status</span>
					<select class="input" bind:value={status}>
						{#each statusOptions as opt (opt)}
							<option value={opt}>{opt.replace('_', ' ')}</option>
						{/each}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Priority</span>
					<select class="input" bind:value={priority}>
						{#each priorityOptions as opt (opt)}
							<option value={opt}>{opt}</option>
						{/each}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Due date</span>
					<input type="date" class="input" bind:value={dueDate} />
				</label>
			</div>

			<div class="flex items-center justify-between gap-2 pt-2">
				{#if isEdit && onDelete}
					<button type="button" class="text-sm text-red-600 hover:text-red-700 hover:underline" onclick={onDelete}>
						Delete task
					</button>
				{:else}
					<span></span>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={saving}>
						{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add task'}
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
