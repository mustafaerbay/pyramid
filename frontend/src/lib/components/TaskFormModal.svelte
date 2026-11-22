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
	// svelte-ignore state_referenced_locally
	let assigneeIds = $state<string[]>(initial?.assigneeIds ?? []);
	// Unlike milestones (always start "pending"), a task's initial status is
	// caller-settable — it's normal to log work retroactively that's already
	// in progress or done.
	let status = $state<TaskStatus>(initial?.status ?? 'todo');
	let priority = $state<TaskPriority>(initial?.priority ?? 'medium');
	let startDate = $state(initial?.startDate ?? '');
	let dueDate = $state(initial?.dueDate ?? '');
	// Each assignee's share of the task, in man-days. Kept for unchecked
	// people too, so toggling someone off and on doesn't lose their number.
	let manDays = $state<Record<string, number>>(
		// svelte-ignore state_referenced_locally
		Object.fromEntries((initial?.assignments ?? []).map((a) => [a.personId, a.manDays]))
	);
	const totalManDays = $derived(
		assigneeIds.reduce((sum, id) => sum + (Number(manDays[id]) || 0), 0)
	);

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

	function toggleAssignee(id: string) {
		assigneeIds = assigneeIds.includes(id)
			? assigneeIds.filter((x) => x !== id)
			: [...assigneeIds, id];
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!title.trim()) {
			formError = 'Title is required.';
			return;
		}
		if (startDate && dueDate && startDate > dueDate) {
			formError = 'Start date must be on or before the due date.';
			return;
		}
		const assignments = assigneeIds.map((personId) => ({
			personId,
			manDays: Number(manDays[personId]) || 0
		}));
		if (assignments.some((a) => a.manDays < 0 || a.manDays > 999)) {
			formError = 'Man-days must be between 0 and 999.';
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
						// assigneeIds always replaces the full set — sending it
						// (even as []) is how the backend knows to update it, vs.
						// omitting the field entirely to leave it untouched.
						assignments,
						status,
						priority,
						startDate: startDate || undefined,
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
						assignments,
						status,
						priority,
						startDate: startDate || undefined,
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
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 sm:p-4"
	role="presentation"
	onclick={onClose}
>
	<div
		class="h-dvh max-h-dvh w-full max-w-lg overflow-y-auto bg-white p-4 sm:p-6 shadow-xl sm:h-auto sm:max-h-[90vh] sm:rounded-lg"
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

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Milestone</span>
				<select class="input" bind:value={milestoneId}>
					<option value="">No milestone</option>
					{#each milestoneOptions as m (m.id)}
						<option value={m.id}>{m.name}</option>
					{/each}
				</select>
			</label>

			<div class="block text-sm">
				<span class="mb-1 flex items-baseline justify-between font-medium text-slate-700">
					<span>Assignees{assigneeIds.length > 0 ? ` (${assigneeIds.length})` : ''}</span>
					<span class="text-xs font-normal text-slate-500">
						Total: <span class="font-semibold text-slate-800 tabular-nums">{totalManDays}</span> man-days
					</span>
				</span>
				<div class="max-h-56 space-y-1 overflow-y-auto rounded-md border border-slate-300 px-3 py-2">
					{#if assigneeOptions.length === 0}
						<p class="text-xs text-slate-400">No active team members to assign.</p>
					{:else}
						{#each assigneeOptions as person (person.id)}
							<div class="flex min-h-10 items-center gap-2 text-sm text-slate-700">
								<label class="flex min-w-0 flex-1 items-center gap-2">
									<input
										type="checkbox"
										checked={assigneeIds.includes(person.id)}
										onchange={() => toggleAssignee(person.id)}
									/>
									<span class="truncate">{person.name}</span>
								</label>
								{#if assigneeIds.includes(person.id)}
									<input
										type="number"
										class="input w-20 text-right tabular-nums"
										min="0"
										max="999"
										step="0.5"
										inputmode="decimal"
										aria-label="Man-days for {person.name}"
										bind:value={manDays[person.id]}
										placeholder="0"
									/>
									<span class="w-6 text-xs text-slate-500">md</span>
								{/if}
							</div>
						{/each}
					{/if}
				</div>
			</div>

			<div class="grid grid-cols-2 gap-4">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Start date</span>
					<input type="date" class="input" bind:value={startDate} max={dueDate || undefined} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Due date</span>
					<input type="date" class="input" bind:value={dueDate} min={startDate || undefined} />
				</label>
			</div>
			<p class="-mt-2 text-xs text-slate-500">
				Man-days are spread over the working days between start and due date.
			</p>

			<div class="grid grid-cols-2 gap-4">
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
