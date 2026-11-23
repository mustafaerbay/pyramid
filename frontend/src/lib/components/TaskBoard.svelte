<script lang="ts">
	import type { components } from '$lib/api/schema';
	import Badge from '$lib/components/Badge.svelte';
	import TagChips from '$lib/components/TagChips.svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';
	import { boardColumns, type BoardStatus } from '$lib/board';
	import { formatManDays } from '$lib/capacity';
	import { subtaskSummary } from '$lib/subtasks';
	import DependencyNote from '$lib/components/DependencyNote.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];

	let {
		tasks,
		allTasks,
		statuses,
		statusOptions,
		statusTone,
		priorityTone,
		assigneesLabel,
		fitWarnings,
		onOpen,
		onMove,
		onPlaceBefore,
		onTagSelect
	}: {
		tasks: Task[];
		/** Every task in the project, for parent titles and subtask counts. */
		allTasks: Task[];
		statuses: BoardStatus[];
		statusOptions: { value: TaskStatus; label: string }[];
		statusTone: (s: TaskStatus) => BadgeTone;
		priorityTone: (p: Task['priority']) => BadgeTone;
		assigneesLabel: (ids: string[]) => string;
		fitWarnings: (t: Task) => string[];
		onOpen: (t: Task) => void;
		onMove: (t: Task, status: TaskStatus) => void;
		/** Dropped on a card: goes just before it in the manual order. */
		onPlaceBefore: (t: Task, targetId: string) => void;
		onTagSelect: (tag: string) => void;
	} = $props();

	const columns = $derived(boardColumns(tasks, statuses));
	const label = (s: TaskStatus) => statusOptions.find((o) => o.value === s)?.label ?? s;

	let draggingId = $state<string | null>(null);
	let overStatus = $state<TaskStatus | null>(null);

	function drop(status: TaskStatus) {
		const task = tasks.find((t) => t.id === draggingId);
		draggingId = null;
		overStatus = null;
		if (task && task.status !== status) onMove(task, status);
	}
</script>

<div class="-mx-4 flex snap-x snap-mandatory scroll-px-4 gap-3 overflow-x-auto px-4 pb-2 sm:mx-0 sm:scroll-px-0 sm:px-0 lg:snap-none">
	{#each columns as col (col.status)}
		{@const pts = col.tasks.reduce((s, t) => s + (t.storyPoints ?? 0), 0)}
		{@const md = col.tasks.reduce((s, t) => s + (allTasks.some((c) => c.parentId === t.id && c.status !== 'cancelled') ? 0 : t.manDays), 0)}
		<section
			class="flex w-[80vw] shrink-0 snap-start flex-col rounded-lg border bg-slate-50 sm:w-64 lg:w-auto lg:min-w-48 lg:flex-1 {overStatus ===
			col.status
				? 'border-slate-400 ring-2 ring-slate-300'
				: 'border-slate-200'}"
			aria-label="{label(col.status)} column"
			ondragover={(e) => {
				if (!draggingId) return;
				e.preventDefault();
				overStatus = col.status;
			}}
			ondragleave={(e) => {
				if (!e.currentTarget.contains(e.relatedTarget as Node | null)) overStatus = null;
			}}
			ondrop={(e) => {
				e.preventDefault();
				drop(col.status);
			}}
		>
			<header class="flex items-center justify-between gap-2 border-b border-slate-200 px-3 py-2">
				<span class="rounded-full px-2 py-0.5 text-xs font-semibold {badgeToneClass[statusTone(col.status)]}">
					{label(col.status)}
				</span>
				<span class="text-xs text-slate-500 tabular-nums">
					{col.tasks.length}{md > 0 ? ` · ${formatManDays(md)} md` : ''}{pts > 0 ? ` · ${formatManDays(pts)} pts` : ''}
				</span>
			</header>
			<ul class="flex min-h-24 flex-1 flex-col gap-2 p-2">
				{#each col.tasks as task (task.id)}
					{@const warnings = fitWarnings(task)}
					{@const parent = task.parentId ? allTasks.find((t) => t.id === task.parentId) : null}
					{@const sub = subtaskSummary(task.id, allTasks)}
					<li
						class="cursor-grab rounded-md border border-slate-200 bg-white p-2.5 text-sm shadow-sm active:cursor-grabbing {draggingId ===
						task.id
							? 'opacity-50'
							: ''}"
						draggable="true"
						ondragstart={(e) => {
							draggingId = task.id;
							e.dataTransfer?.setData('text/plain', task.id);
							if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
						}}
						ondragend={() => {
							draggingId = null;
							overStatus = null;
						}}
						ondragover={(e) => {
							if (draggingId && draggingId !== task.id) e.preventDefault();
						}}
						ondrop={(e) => {
							e.preventDefault();
							e.stopPropagation();
							const moving = tasks.find((t) => t.id === draggingId);
							draggingId = null;
							overStatus = null;
							if (!moving || moving.id === task.id) return;
							if (moving.status !== col.status) onMove(moving, col.status);
							onPlaceBefore(moving, task.id);
						}}
					>
						{#if parent}
							<p class="mb-0.5 truncate text-xs text-slate-500">↳ {parent.title}</p>
						{/if}
						<button type="button" class="block w-full text-left font-medium text-slate-800 hover:underline" onclick={() => onOpen(task)}>
							<TypeBadge type={task.type} /> {task.title}
						</button>
						{#if sub.count > 0}
							<p class="mt-1 text-xs text-slate-500">{sub.done}/{sub.count} children done</p>
						{/if}
						<DependencyNote {task} all={allTasks} />
						{#if task.tags.length}
							<div class="mt-1.5"><TagChips tags={task.tags} onSelect={onTagSelect} /></div>
						{/if}
						<p class="mt-1.5 line-clamp-2 text-xs text-slate-500">
							{assigneesLabel(task.assigneeIds)}{(sub.count > 0 ? sub.manDays : task.manDays) > 0
								? ` · ${formatManDays(sub.count > 0 ? sub.manDays : task.manDays)} md`
								: ''}{task.dueDate
								? ` · Due ${task.dueDate}`
								: ''}
						</p>
						{#each warnings as w (w)}
							<p class="mt-1 text-xs font-medium text-red-700">⚠ {w}</p>
						{/each}
						<div class="mt-2 flex flex-wrap items-center justify-between gap-1.5">
							<span class="inline-flex items-center gap-1.5">
								<Badge tone={priorityTone(task.priority)}>{task.priority}</Badge>
								{#if task.storyPoints != null}<span class="rounded bg-slate-100 px-1.5 text-[11px] font-medium text-slate-600" title="Story points">{formatManDays(task.storyPoints)}</span>{/if}
							</span>
							<select
								class="min-h-8 min-w-0 max-w-full rounded-md border-0 bg-slate-100 py-0 pl-2 pr-6 text-xs text-slate-600"
								aria-label="Move {task.title} to"
								value={task.status}
								onchange={(e) => onMove(task, e.currentTarget.value as TaskStatus)}
							>
								{#each statusOptions as opt (opt.value)}
									<option value={opt.value}>{opt.label}</option>
								{/each}
							</select>
						</div>
					</li>
				{:else}
					<li class="rounded-md border border-dashed border-slate-200 p-3 text-center text-xs text-slate-400">
						Drop tasks here
					</li>
				{/each}
			</ul>
		</section>
	{/each}
</div>
