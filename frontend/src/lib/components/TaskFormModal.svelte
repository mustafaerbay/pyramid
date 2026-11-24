<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import MentionTextarea from './MentionTextarea.svelte';
	import { api, baseUrl } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { formatManDays, splitEvenly } from '$lib/capacity';
	import TagInput from './TagInput.svelte';
	import TaskActivity from './TaskActivity.svelte';
	import TaskPicker from './TaskPicker.svelte';
	import { parentOptions, subtaskSummary } from '$lib/subtasks';
	import { TYPE_OPTIONS, type WorkItemType } from '$lib/workItems';
	import { areaLabel } from '$lib/areas';
	import { valuesForSave } from '$lib/customFields';
	import { blockerOptions } from '$lib/dependencies';
	import Markdown from './Markdown.svelte';
	import { fillPending } from '$lib/markdown';
	import { imageName, itemHref, uploadImage } from '$lib/itemFiles';

	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];
	type TaskPriority = components['schemas']['TaskPriority'];

	let {
		projectId,
		task = null,
		milestoneOptions = [],
		sprintOptions = [],
		areaOptions = [],
		customFields = [],
		workflow = [],
		itemTypes = [],
		assigneeOptions = [],
		initialMilestoneId = null,
		projectTasks = [],
		initialParentId = null,
		initialType = 'task',
		onAddSubtask,
		onClose,
		onSaved,
		onDelete,
		templates = [],
		transferTargets = [],
		onCopy,
		onTransfer,
		onSaveTemplate,
		onDeleteTemplate,
		variant = 'modal',
		projectKey = '',
		version = 0
	}: {
		projectId: string;
		task?: Task | null;
		// The project's milestones, for the "which milestone is this part of"
		// picker — tasks don't have to belong to one (project-level backlog).
		milestoneOptions?: { id: string; name: string }[];
		// Open sprints of the project.
		sprintOptions?: { id: string; name: string }[];
		// The project's area paths.
		areaOptions?: { id: string; path: string }[];
		// The project's custom fields.
		customFields?: components['schemas']['CustomField'][];
		// The project's workflow states; status is set through these.
		workflow?: components['schemas']['WorkflowState'][];
		// The project's own work item types.
		itemTypes?: components['schemas']['WorkItemTypeDef'][];
		// The project's active team members — a task can only be assigned to
		// someone already on the team, not any person in the org.
		assigneeOptions?: { id: string; name: string }[];
		// Prefills the milestone picker when creating a task from a specific
		// milestone group's "+ Add task" button.
		initialMilestoneId?: string | null;
		// The project's tasks: parent choices and this task's subtasks.
		projectTasks?: Task[];
		// Prefills the parent when adding a subtask.
		initialParentId?: string | null;
		// Type for a new item (a child gets one level below its parent).
		initialType?: WorkItemType;
		// Shown as "+ Add child" when editing.
		onAddSubtask?: (parent: Task) => void;
		onClose: () => void;
		onSaved: (task: Task) => void;
		// Shown as a "Delete task" link in edit mode only. The parent owns the
		// actual confirm-and-delete flow (same split as milestones/members).
		onDelete?: () => void;
		/** Item templates of this project (create mode can start from one). */
		templates?: components['schemas']['ItemTemplate'][];
		/** Projects this item can be moved to. */
		transferTargets?: { id: string; name: string }[];
		onCopy?: (withChildren: boolean) => void;
		onTransfer?: (projectId: string) => void;
		onSaveTemplate?: (name: string) => void;
		onDeleteTemplate?: (id: string) => void;
		/** "page": the item's own page, two columns, every change saved as it's made. */
		variant?: 'modal' | 'page';
		projectKey?: string;
		/** Bumped by the page after a save, so history reloads. */
		version?: number;
	} = $props();
	// svelte-ignore state_referenced_locally
	const onPage = variant === 'page';
	let templateId = $state('');
	let moveTo = $state('');
	function applyTemplate(id: string) {
		const t = templates.find((x) => x.id === id);
		if (!t) return;
		const f = t.fields;
		if (f.customType) typeChoice = `custom:${f.customType}`;
		else if (f.type) typeChoice = f.type;
		if (f.title) title = f.title;
		if (f.description) description = f.description;
		if (f.priority) priority = f.priority;
		if (f.tags) tags = [...f.tags];
		if (f.storyPoints != null) storyPoints = f.storyPoints;
		if (f.estimateManDays != null) setTotal(String(f.estimateManDays));
		if (f.customValues) custom = { ...custom, ...Object.fromEntries(Object.entries(f.customValues).map(([k, v]) => [k, v == null ? '' : String(v)])) };
	}

	// svelte-ignore state_referenced_locally
	const isEdit = task !== null;

	// svelte-ignore state_referenced_locally
	const initial = task;

	let title = $state(initial?.title ?? '');
	let description = $state(initial?.description ?? '');
	// svelte-ignore state_referenced_locally
	let tags = $state<string[]>([...(initial?.tags ?? [])]);
	// svelte-ignore state_referenced_locally
	let milestoneId = $state(initial?.milestoneId ?? initialMilestoneId ?? '');
	// svelte-ignore state_referenced_locally
	let sprintId = $state(initial?.sprintId ?? '');
	// svelte-ignore state_referenced_locally
	let areaId = $state(initial?.areaId ?? '');
	// Custom field inputs by key (strings while editing).
	let custom = $state<Record<string, string>>(
		// svelte-ignore state_referenced_locally
		Object.fromEntries(Object.entries(initial?.customValues ?? {}).map(([k, v]) => [k, v == null ? '' : String(v)]))
	);
	// svelte-ignore state_referenced_locally
	let parentId = $state(initial?.parentId ?? initialParentId ?? '');
	// svelte-ignore state_referenced_locally
	// "story", or "custom:spike" for one of the project's own types.
	// svelte-ignore state_referenced_locally
	let typeChoice = $state<string>(initial?.customType ? `custom:${initial.customType}` : ((initial?.type as WorkItemType) ?? initialType));
	const customKey = $derived(typeChoice.startsWith('custom:') ? typeChoice.slice(7) : null);
	const type = $derived<WorkItemType>(
		customKey ? ((itemTypes.find((t) => t.key === customKey)?.base as WorkItemType) ?? 'task') : (typeChoice as WorkItemType)
	);
	const parentChoices = $derived(parentOptions(projectTasks, initial?.id ?? null, type));
	const subtasks = $derived(initial ? projectTasks.filter((t) => t.parentId === initial.id) : []);
	const summary = $derived(initial ? subtaskSummary(initial.id, projectTasks) : null);
	// svelte-ignore state_referenced_locally
	let blockedByIds = $state<string[]>([...(initial?.blockedByIds ?? [])]);
	// svelte-ignore state_referenced_locally
	const blockerChoices = blockerOptions(projectTasks, initial?.id ?? null);
	// svelte-ignore state_referenced_locally
	let relatedIds = $state<string[]>([...(initial?.relatedIds ?? [])]);
	// svelte-ignore state_referenced_locally
	let duplicateOfId = $state(initial?.duplicateOfId ?? '');
	// svelte-ignore state_referenced_locally
	const linkChoices = projectTasks.filter((t) => t.id !== initial?.id);
	// A new subtask lands in its parent's milestone unless one is picked.
	function pickParent(id: string) {
		parentId = id;
		const p = projectTasks.find((t) => t.id === id);
		if (p?.milestoneId && !milestoneId) milestoneId = p.milestoneId;
	}
	// svelte-ignore state_referenced_locally
	let assigneeIds = $state<string[]>(initial?.assigneeIds ?? []);
	// Unlike milestones (always start "pending"), a task's initial status is
	// caller-settable — it's normal to log work retroactively that's already
	// in progress or done.
	let status = $state<TaskStatus>(initial?.status ?? 'todo');
	// svelte-ignore state_referenced_locally
	let wfState = $state(initial?.state ?? workflow.find((w) => w.category === 'todo')?.key ?? workflow[0]?.key ?? '');
	let priority = $state<TaskPriority>(initial?.priority ?? 'medium');
	let startDate = $state(initial?.startDate ?? '');
	let dueDate = $state(initial?.dueDate ?? '');
	// Empty input = not set (bind:value on a number input gives null).
	let storyPoints = $state<number | null>(initial?.storyPoints ?? null);
	let remaining = $state<number | null>(initial?.remainingManDays ?? null);
	let completed = $state<number | null>(initial?.completedManDays ?? null);
	const num = (v: number | null | undefined | string) => (v === null || v === undefined || v === '' ? null : Number(v));
	// Each assignee's share of the task, in man-days. Kept for unchecked
	// people too, so toggling someone off and on doesn't lose their number.
	let manDays = $state<Record<string, number>>(
		// svelte-ignore state_referenced_locally
		Object.fromEntries((initial?.assignments ?? []).map((a) => [a.personId, a.manDays]))
	);
	// The task's size in man-days, always editable. Changing it re-splits it
	// evenly across the ticked people; changing one person's share makes the
	// total the sum of the shares.
	let total = $state<number | null>(
		// svelte-ignore state_referenced_locally
		initial && initial.manDays > 0 ? initial.manDays : null
	);
	const sharesSum = $derived(assigneeIds.reduce((sum, id) => sum + (Number(manDays[id]) || 0), 0));

	function spreadTotal() {
		const parts = splitEvenly(Number(total) || 0, assigneeIds.length);
		const next = { ...manDays };
		assigneeIds.forEach((id, i) => (next[id] = parts[i]));
		manDays = next;
	}

	function setTotal(value: string) {
		total = value === '' ? null : Number(value);
		spreadTotal();
	}

	function setShare(id: string, value: string) {
		manDays = { ...manDays, [id]: value === '' ? 0 : Number(value) };
		if (!assigneeIds.includes(id) && Number(value) > 0) assigneeIds = [...assigneeIds, id];
		total = assigneeIds.reduce((sum, pid) => sum + (Number(manDays[pid]) || 0), 0);
	}

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
		if (Number(total) > 0) spreadTotal();
	}

	// Images pasted before a new item exists wait here (pending:N → file)
	// and are uploaded once it's created.
	let pending = $state<Record<string, { file: File; url: string }>>({});
	const pendingUrls = $derived(Object.fromEntries(Object.entries(pending).map(([k, v]) => [k, v.url])));
	let pendingCount = 0;
	async function addImage(file: File): Promise<string | null> {
		if (task) return uploadImage(task.id, file);
		const key = `pending:${++pendingCount}`;
		pending = { ...pending, [key]: { file, url: URL.createObjectURL(file) } };
		return `![${imageName(file).replace(/[[\]]/g, '')}](${key})`;
	}

	function check(): string | null {
		if (!title.trim()) return 'Title is required.';
		if (startDate && dueDate && startDate > dueDate) return 'Start date must be on or before the due date.';
		const estimateManDays = total == null ? undefined : Number(total) || 0;
		if (estimateManDays !== undefined && (estimateManDays < 0 || estimateManDays > 9999)) return 'Man-days must be between 0 and 9999.';
		if (assigneeIds.some((id) => (Number(manDays[id]) || 0) < 0 || (Number(manDays[id]) || 0) > 999)) return 'Man-days must be between 0 and 999.';
		return null;
	}
	function editBody() {
		return {
			title,
			description: description || undefined,
			// Explicit null (not undefined) so the backend can tell
			// "clear this" apart from "field wasn't touched".
			type,
			customType: customKey,
			milestoneId: milestoneId || null,
			sprintId: sprintId || null,
			areaId: areaId || null,
			customValues: valuesForSave(customFields, custom),
			parentId: parentId || null,
			// The full set of assignees always replaces the old one.
			assignments: assigneeIds.map((personId) => ({ personId, manDays: Number(manDays[personId]) || 0 })),
			estimateManDays: total == null ? undefined : Number(total) || 0,
			storyPoints: num(storyPoints),
			remainingManDays: num(remaining),
			completedManDays: num(completed),
			...(workflow.length ? { state: wfState } : { status }),
			priority,
			startDate: startDate || undefined,
			dueDate: dueDate || undefined,
			tags,
			blockedByIds,
			relatedIds,
			duplicateOfId: duplicateOfId || null
		};
	}

	async function saveEdit(): Promise<boolean> {
		if (!task) return false;
		const { data, error } = await api.PATCH('/api/v1/tasks/{id}', { params: { path: { id: task.id } }, body: editBody() });
		if (error) {
			formError = error.message ?? 'Failed to update task.';
			return false;
		}
		if (data) onSaved(data);
		return true;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = check();
		if (formError) return;
		saving = true;
		try {
			if (isEdit && task) {
				await saveEdit();
			} else {
				const b = editBody();
				const { data, error } = await api.POST('/api/v1/tasks', {
					body: {
						...b,
						projectId,
						customType: customKey ?? undefined,
						milestoneId: milestoneId || undefined,
						sprintId: sprintId || undefined,
						areaId: areaId || undefined,
						parentId: parentId || undefined,
						duplicateOfId: duplicateOfId || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create task.';
					return;
				}
				if (!data) return;
				let saved = data;
				const used = Object.keys(pending).filter((k) => description.includes(`(${k})`));
				if (used.length) {
					const ids: Record<string, string> = {};
					const fd = (f: File) => {
						const x = new FormData();
						x.append('file', f, imageName(f));
						return x;
					};
					for (const k of used) {
						const res = await fetch(`${baseUrl}/api/v1/tasks/${data.id}/attachments`, { method: 'POST', body: fd(pending[k].file), credentials: 'include' });
						if (res.ok) ids[k] = ((await res.json()) as { id: string }).id;
					}
					const { data: upd } = await api.PATCH('/api/v1/tasks/{id}', { params: { path: { id: data.id } }, body: { description: fillPending(description, ids) } });
					if (upd) saved = upd;
				}
				onSaved(saved);
			}
		} finally {
			saving = false;
		}
	}

	// On the page every change is saved shortly after it's made.
	let saveState = $state<'idle' | 'saving' | 'saved' | 'error'>('idle');
	let lastSent = '';
	let timer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		if (!onPage || !task) return;
		const snap = JSON.stringify(editBody());
		if (!lastSent) {
			lastSent = snap;
			return;
		}
		if (snap === lastSent) return;
		clearTimeout(timer);
		timer = setTimeout(async () => {
			const problem = check();
			if (problem) {
				formError = problem;
				saveState = 'error';
				return;
			}
			formError = null;
			lastSent = snap;
			saveState = 'saving';
			saveState = (await saveEdit()) ? 'saved' : 'error';
			if (saveState === 'error') lastSent = '';
		}, 700);
		return () => clearTimeout(timer);
	});

	// The description on the page is read until "Edit" is clicked.
	let editingDesc = $state(false);
	let descDraft = $state('');
	function startDesc() {
		descDraft = description;
		editingDesc = true;
	}
	function doneDesc() {
		description = descDraft;
		editingDesc = false;
	}
	const childLink = (t: Task) => itemHref(t.projectId, t.number);
	const itemLink = (n: number) => {
		const t = projectTasks.find((x) => x.number === n);
		return t ? childLink(t) : null;
	};
</script>

<svelte:window onkeydown={(e) => !onPage && e.key === 'Escape' && onClose()} />

{#snippet templatePicker()}
			{#if !isEdit && templates.length}
				<div class="flex items-center gap-2 text-sm">
					<span class="text-slate-600">Start from</span>
					<select
						class="input w-auto py-1"
						aria-label="Template"
						bind:value={templateId}
						onchange={() => applyTemplate(templateId)}
					>
						<option value="">No template</option>
						{#each templates as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
					</select>
					{#if templateId && onDeleteTemplate}
						<button
							type="button"
							class="text-xs text-slate-500 hover:text-red-600"
							onclick={() => {
								onDeleteTemplate(templateId);
								templateId = '';
							}}>Delete template</button
						>
					{/if}
				</div>
			{/if}
{/snippet}

{#snippet headF()}
			<div class="flex flex-wrap gap-2">
				<label class="block w-32 shrink-0 text-sm">
					<span class="mb-1 block font-medium text-slate-700">Type</span>
					<select class="input" bind:value={typeChoice}>
						{#each TYPE_OPTIONS as o (o.value)}
							<option value={o.value}>{o.label}</option>
						{/each}
						{#if itemTypes.length}
							<optgroup label="This project">
								{#each itemTypes as t (t.key)}
									<option value="custom:{t.key}">{t.label}</option>
								{/each}
							</optgroup>
						{/if}
					</select>
				</label>
				<label class="block min-w-48 flex-1 text-sm">
					<span class="mb-1 block font-medium text-slate-700">Title *</span>
					<input class="input {onPage ? 'text-base font-semibold' : ''}" bind:value={title} placeholder="e.g. Wire up auth middleware" required />
				</label>
			</div>
{/snippet}

{#snippet tagsF()}
			<div class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Tags</span>
				<TagInput bind:tags />
			</div>
{/snippet}

{#snippet milestoneF()}
			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Milestone</span>
				<select class="input" bind:value={milestoneId}>
					<option value="">No milestone</option>
					{#each milestoneOptions as m (m.id)}
						<option value={m.id}>{m.name}</option>
					{/each}
				</select>
			</label>
{/snippet}

{#snippet customF()}
			{#if customFields.length > 0}
				<div class="grid gap-3 sm:grid-cols-2">
					{#each customFields as f (f.key)}
						<label class="block text-sm">
							<span class="mb-1 block font-medium text-slate-700">{f.label}{f.required ? ' *' : ''}</span>
							{#if f.type === 'select'}
								<select class="input" required={f.required} bind:value={custom[f.key]}>
									<option value="">—</option>
									{#each f.options ?? [] as o (o)}<option value={o}>{o}</option>{/each}
								</select>
							{:else if f.type === 'number'}
								<input class="input" type="number" step="any" required={f.required} bind:value={custom[f.key]} />
							{:else if f.type === 'date'}
								<input class="input" type="date" required={f.required} bind:value={custom[f.key]} />
							{:else}
								<input class="input" maxlength="500" required={f.required} bind:value={custom[f.key]} />
							{/if}
						</label>
					{/each}
				</div>
			{/if}
{/snippet}

{#snippet areaF()}
			{#if areaOptions.length > 0}
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Area</span>
					<select class="input" bind:value={areaId}>
						<option value="">No area</option>
						{#each areaOptions as a (a.id)}
							<option value={a.id}>{areaLabel(a.path)}</option>
						{/each}
					</select>
				</label>
			{/if}
{/snippet}

{#snippet sprintF()}
			{#if sprintOptions.length > 0 || sprintId}
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Sprint</span>
					<select class="input" bind:value={sprintId}>
						<option value="">Not in a sprint</option>
						{#each sprintOptions as sp (sp.id)}
							<option value={sp.id}>{sp.name}</option>
						{/each}
						{#if sprintId && !sprintOptions.some((sp) => sp.id === sprintId)}
							<option value={sprintId}>(closed sprint)</option>
						{/if}
					</select>
				</label>
			{/if}
{/snippet}

{#snippet parentF()}
			{#if parentChoices.length > 0 || parentId}
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Parent task</span>
					<select class="input" value={parentId} onchange={(e) => pickParent(e.currentTarget.value)}>
						<option value="">None (top-level task)</option>
						{#each parentChoices as p (p.id)}
							<option value={p.id}>#{p.number} {p.title}</option>
						{/each}
					</select>
				</label>
			{/if}
{/snippet}

{#snippet linksF()}
			{#if blockerChoices.length > 0 || blockedByIds.length > 0}
				<TaskPicker
					label="Waiting on"
					placeholder="Add a task that must finish first…"
					bind:ids={blockedByIds}
					options={blockerChoices}
					all={projectTasks}
					tone="amber"
				/>
			{/if}

			{#if linkChoices.length > 0}
				<TaskPicker label="Related" placeholder="Link a related task…" bind:ids={relatedIds} options={linkChoices} all={projectTasks} />
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Duplicate of</span>
					<select class="input" bind:value={duplicateOfId}>
						<option value="">Not a duplicate</option>
						{#each linkChoices as t (t.id)}
							<option value={t.id}>#{t.number} {t.title}</option>
						{/each}
					</select>
				</label>
			{/if}
{/snippet}

{#snippet childrenF()}
			{#if isEdit && initial}
				<div class="rounded-md border border-slate-200 p-3 text-sm">
					<div class="flex items-center justify-between gap-2">
						<span class="font-medium text-slate-700">
							Children{#if summary && summary.count > 0}<span class="ml-1 font-normal text-slate-500"
									>{summary.done}/{summary.count} done · {formatManDays(summary.manDays)} md</span
								>{/if}
						</span>
						{#if onAddSubtask}
							<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => onAddSubtask(initial)}>
								+ Add child
							</button>
						{/if}
					</div>
					{#if subtasks.length > 0}
						<ul class="mt-2 space-y-1">
							{#each subtasks as sub (sub.id)}
								<li class="flex items-center justify-between gap-2 text-xs">
									<span class="min-w-0 truncate {sub.status === 'done' || sub.status === 'cancelled' ? 'text-slate-400 line-through' : 'text-slate-700'}">
										{#if onPage}<a class="hover:underline" href={childLink(sub)}>#{sub.number} {sub.title}</a>{:else}#{sub.number} {sub.title}{/if}
									</span>
									<span class="shrink-0 text-slate-500">{sub.status.replace('_', ' ')}</span>
								</li>
							{/each}
						</ul>
						{#if summary && summary.count > 0}
							<p class="mt-2 text-xs text-slate-500">
								This task's own man-days aren't counted in capacity; its subtasks carry the work.
							</p>
						{/if}
					{/if}
				</div>
			{/if}
{/snippet}

{#snippet effortF()}
			<div class="block text-sm">
				<label class="mb-3 block">
					<span class="mb-1 block font-medium text-slate-700">Man-days (task size)</span>
					<div class="flex items-center gap-2">
						<input
							type="number"
							class="input w-32 tabular-nums"
							min="0"
							max="9999"
							step="0.5"
							inputmode="decimal"
							placeholder="e.g. 40"
							value={total ?? ''}
							oninput={(e) => setTotal(e.currentTarget.value)}
						/>
						<span class="text-xs text-slate-500">
							{#if assigneeIds.length > 1}
								split across {assigneeIds.length} people below
							{:else if assigneeIds.length === 1}
								all for the person below
							{:else}
								tick people below to split it
							{/if}
						</span>
					</div>
				</label>
				<span class="mb-1 flex items-baseline justify-between font-medium text-slate-700">
					<span>Assignees{assigneeIds.length > 0 ? ` (${assigneeIds.length})` : ''}</span>
					{#if assigneeIds.length > 0 && total != null && Math.abs(sharesSum - Number(total)) > 0.001}
						<span class="text-xs font-normal text-amber-700">
							Shares add up to {formatManDays(sharesSum)} md
						</span>
					{/if}
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
								<input
									type="number"
									class="input w-20 text-right tabular-nums {assigneeIds.includes(person.id)
										? ''
										: 'bg-slate-50 text-slate-400'}"
									min="0"
									max="999"
									step="0.5"
									inputmode="decimal"
									aria-label="Man-days for {person.name}"
									value={assigneeIds.includes(person.id) ? (manDays[person.id] ?? 0) : ''}
									oninput={(e) => setShare(person.id, e.currentTarget.value)}
									placeholder="0"
								/>
								<span class="w-6 text-xs text-slate-500">md</span>
							</div>
						{/each}
					{/if}
				</div>
			</div>
{/snippet}

{#snippet datesF()}
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
{/snippet}

{#snippet pointsF()}
			<div class="grid grid-cols-3 gap-3">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Story points</span>
					<input type="number" class="input" min="0" max="999" step="0.5" bind:value={storyPoints} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Remaining (md)</span>
					<input type="number" class="input" min="0" max="9999" step="0.5" bind:value={remaining} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Completed (md)</span>
					<input type="number" class="input" min="0" max="9999" step="0.5" bind:value={completed} />
				</label>
			</div>
{/snippet}

{#snippet statePriF()}
			<div class="grid grid-cols-2 gap-4">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">State</span>
					{#if workflow.length}
						<select class="input" bind:value={wfState}>
							{#each workflow as w (w.key)}
								<option value={w.key}>{w.label}</option>
							{/each}
						</select>
					{:else}
						<select class="input" bind:value={status}>
							{#each statusOptions as opt (opt)}
								<option value={opt}>{opt.replace('_', ' ')}</option>
							{/each}
						</select>
					{/if}
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
{/snippet}

{#snippet actionsF()}
	{#if isEdit && onDelete}
		<span class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
						<button type="button" class="text-red-600 hover:text-red-700 hover:underline" onclick={onDelete}>Delete task</button>
						{#if onCopy}
							<button type="button" class="text-slate-600 hover:underline" onclick={() => onCopy(false)}>Copy</button>
							{#if projectTasks.some((x) => x.parentId === task?.id)}
								<button type="button" class="text-slate-600 hover:underline" onclick={() => onCopy(true)}>Copy with children</button>
							{/if}
						{/if}
						{#if onSaveTemplate}
							<button
								type="button"
								class="text-slate-600 hover:underline"
								onclick={() => {
									const n = prompt('Template name', task?.title ?? '');
									if (n?.trim()) onSaveTemplate(n.trim());
								}}>Save as template</button
							>
						{/if}
						{#if onTransfer && transferTargets.length}
							<select
								class="rounded border-slate-200 py-0.5 text-xs"
								aria-label="Move to project"
								bind:value={moveTo}
								onchange={() => {
									const target = transferTargets.find((p) => p.id === moveTo);
									if (target && confirm(`Move this item (and its children) to ${target.name}? Sprint, area and milestone are cleared.`)) onTransfer(moveTo);
									moveTo = '';
								}}
							>
								<option value="">Move to project…</option>
								{#each transferTargets as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
							</select>
						{/if}
					</span>
	{/if}
{/snippet}

{#snippet descModal()}
			<div class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Description</span>
				<MentionTextarea bind:value={description} people={assigneeOptions.map((p) => p.name)} rows={4} label="Description" onImage={addImage} images={pendingUrls} />
			</div>
{/snippet}

{#snippet descPage()}
	<section class="text-sm">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="font-semibold text-slate-800">Description</h2>
			{#if !editingDesc}<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={startDesc}><Icon name="pencil" size={14} class="mr-1 align-[-2px]" />Edit</button>{/if}
		</div>
		{#if editingDesc}
			<MentionTextarea bind:value={descDraft} people={assigneeOptions.map((p) => p.name)} rows={12} label="Description" onImage={addImage} autofocus />
			<div class="mt-2 flex gap-2">
				<button type="button" class="btn-primary px-3 py-1 text-xs" onclick={doneDesc}>Save description</button>
				<button type="button" class="btn-ghost px-3 py-1 text-xs" onclick={() => (editingDesc = false)}>Cancel</button>
			</div>
		{:else if description.trim()}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="rounded-md border border-transparent p-1 hover:border-slate-200" ondblclick={startDesc}>
				<Markdown text={description} people={assigneeOptions.map((p) => p.name)} class="text-slate-700" item={itemLink} />
			</div>
		{:else}
			<button
				type="button"
				class="w-full rounded-md border border-dashed border-slate-300 px-3 py-6 text-slate-500 hover:border-slate-400 hover:text-slate-700"
				onclick={startDesc}>Add a description: steps, expected and actual result, screenshots (paste or drop them in)…</button
			>
		{/if}
	</section>
{/snippet}

{#if onPage}
	<form class="space-y-4" onsubmit={(e) => e.preventDefault()}>
		<div class="flex flex-wrap items-center gap-2 text-sm">
			<span class="font-mono text-slate-500">{projectKey ? `${projectKey}-` : '#'}{task?.number}</span>
			<span class="ml-auto text-xs {saveState === 'error' ? 'text-red-600' : 'text-slate-500'}" aria-live="polite">
				{saveState === 'saving' ? 'Saving…' : saveState === 'saved' ? '✓ Saved' : saveState === 'error' ? 'Not saved' : 'Changes save automatically'}
			</span>
		</div>
		{#if formError}
			<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{formError}</div>
		{/if}
		<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
			<div class="min-w-0 space-y-5">
				{@render headF()}
				{@render descPage()}
				{@render childrenF()}
				{#if initial}<TaskActivity task={initial} {projectTasks} {milestoneOptions} {version} />{/if}
			</div>
			<aside class="space-y-4 rounded-lg border border-slate-200 bg-slate-50/60 p-4 lg:self-start">
				{@render statePriF()}
				{@render effortF()}
				{@render sprintF()}
				{@render milestoneF()}
				{@render areaF()}
				{@render datesF()}
				{@render pointsF()}
				{@render tagsF()}
				{@render parentF()}
				{@render linksF()}
				{@render customF()}
				<div class="border-t border-slate-200 pt-3">{@render actionsF()}</div>
			</aside>
		</div>
	</form>
{:else}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 sm:p-4" role="presentation" onclick={onClose}>
		<div
			class="h-dvh max-h-dvh w-full max-w-lg overflow-y-auto bg-white p-4 sm:p-6 shadow-xl sm:h-auto sm:max-h-[90vh] sm:rounded-lg"
			role="dialog"
			tabindex="-1"
			aria-modal="true"
			aria-labelledby="task-modal-title"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex items-center justify-between">
				<h3 id="task-modal-title" class="text-lg font-semibold text-slate-900">{isEdit ? 'Edit task' : 'Add task'}</h3>
				<button type="button" class="text-slate-400 hover:text-slate-600" onclick={onClose} aria-label="Close">✕</button>
			</div>
			<form class="mt-4 space-y-4" onsubmit={handleSubmit}>
				{#if formError}
					<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{formError}</div>
				{/if}
				{@render templatePicker()}
				{@render headF()}
				{@render descModal()}
				{@render tagsF()}
				{@render milestoneF()}
				{@render customF()}
				{@render areaF()}
				{@render sprintF()}
				{@render parentF()}
				{@render linksF()}
				{@render childrenF()}
				{@render effortF()}
				{@render datesF()}
				{@render pointsF()}
				{@render statePriF()}
				<div class="flex items-center justify-between gap-2 pt-2">
					{#if isEdit && onDelete}{@render actionsF()}{:else}<span></span>{/if}
					<div class="flex gap-2">
						<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
						<button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add task'}</button>
					</div>
				</div>
			</form>
			{#if isEdit && initial}
				<TaskActivity task={initial} {projectTasks} {milestoneOptions} />
			{/if}
		</div>
	</div>
{/if}
