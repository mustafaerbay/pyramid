<script lang="ts">
	import { can } from '$lib/auth.svelte';
	import type { components } from '$lib/api/schema';
	import Badge from '$lib/components/Badge.svelte';
	import TagChips from '$lib/components/TagChips.svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';
	import { boardColumns, boardLanes, wipState, type BoardStatus } from '$lib/board';
	import { formatManDays } from '$lib/capacity';
	import { subtaskSummary } from '$lib/subtasks';
	import DependencyNote from '$lib/components/DependencyNote.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];
	type BoardSettings = components['schemas']['BoardSettings'];

	let {
		tasks,
		allTasks,
		statuses,
		statusOptions,
		statusTone,
		priorityTone,
		assigneesLabel,
		personName,
		fitWarnings,
		settings,
		canEditSettings = false,
		onSaveSettings,
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
		personName: (id: string) => string;
		fitWarnings: (t: Task) => string[];
		settings: BoardSettings;
		canEditSettings?: boolean;
		onSaveSettings: (s: BoardSettings) => Promise<boolean>;
		onOpen: (t: Task) => void;
		onMove: (t: Task, status: TaskStatus) => void;
		/** Dropped on a card: goes just before it in the manual order. */
		onPlaceBefore: (t: Task, targetId: string) => void;
		onTagSelect: (tag: string) => void;
	} = $props();

	const label = (s: TaskStatus) => statusOptions.find((o) => o.value === s)?.label ?? s;
	const taskTitle = (id: string) => allTasks.find((t) => t.id === id)?.title ?? 'Deleted item';
	const lanes = $derived(boardLanes(tasks, settings.swimlanes, { person: personName, task: taskTitle }));
	const totals = $derived(boardColumns(tasks, statuses));
	// Same width rules for headers and lane cells so they line up.
	const colClass = 'w-[80vw] shrink-0 snap-start sm:w-64 lg:w-auto lg:min-w-48 lg:flex-1';

	const EDGE_BY_TYPE: Record<string, string> = {
		epic: 'border-l-orange-400',
		feature: 'border-l-violet-400',
		story: 'border-l-sky-400',
		bug: 'border-l-red-500',
		task: 'border-l-amber-400'
	};
	const EDGE_BY_PRIORITY: Record<string, string> = {
		urgent: 'border-l-red-500',
		high: 'border-l-orange-400',
		medium: 'border-l-sky-400',
		low: 'border-l-slate-300'
	};
	const edge = (t: Task) =>
		settings.cardColor === 'type'
			? `border-l-4 ${EDGE_BY_TYPE[t.type] ?? ''}`
			: settings.cardColor === 'priority'
				? `border-l-4 ${EDGE_BY_PRIORITY[t.priority] ?? ''}`
				: '';

	let draggingId = $state<string | null>(null);
	let over = $state<string | null>(null); // `${lane}|${status}`

	function drop(status: TaskStatus) {
		const task = tasks.find((t) => t.id === draggingId);
		draggingId = null;
		over = null;
		if (task && task.status !== status) onMove(task, status);
	}

	// Settings panel.
	let editing = $state(false);
	let draft = $state<BoardSettings>({ wipLimits: {}, swimlanes: 'none', cardColor: 'none' });
	let saving = $state(false);
	function openSettings() {
		draft = { wipLimits: { ...settings.wipLimits }, swimlanes: settings.swimlanes, cardColor: settings.cardColor };
		editing = true;
	}
	async function save() {
		saving = true;
		const clean = Object.fromEntries(
			Object.entries(draft.wipLimits).filter(([, v]) => v != null && Number(v) >= 1).map(([k, v]) => [k, Math.round(Number(v))])
		);
		const ok = await onSaveSettings({ ...draft, wipLimits: clean });
		saving = false;
		if (ok) editing = false;
	}
</script>

<div class="mb-2 flex justify-end">
	{#if canEditSettings}
		<button type="button" class="btn-ghost px-2 py-1 text-xs" aria-expanded={editing} onclick={() => (editing ? (editing = false) : openSettings())}>
			⚙ Board settings
		</button>
	{/if}
</div>

{#if editing}
	<form
		class="mb-3 space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm"
		onsubmit={(e) => {
			e.preventDefault();
			save();
		}}
	>
		<div>
			<div class="mb-1 font-medium text-slate-700">WIP limits</div>
			<p class="mb-2 text-xs text-slate-500">Most items a column should hold at once. Leave empty for no limit.</p>
			<div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
				{#each statuses as s (s)}
					<label class="block text-xs">
						<span class="mb-0.5 block text-slate-600">{label(s)}</span>
						<input type="number" min="1" max="999" class="input" bind:value={draft.wipLimits[s]} />
					</label>
				{/each}
			</div>
		</div>
		<div class="flex flex-wrap gap-3">
			<label class="block">
				<span class="mb-1 block font-medium text-slate-700">Swimlanes</span>
				<select class="input w-auto" bind:value={draft.swimlanes}>
					<option value="none">None</option>
					<option value="assignee">By person</option>
					<option value="priority">By priority</option>
					<option value="type">By type</option>
					<option value="parent">By parent</option>
				</select>
			</label>
			<label class="block">
				<span class="mb-1 block font-medium text-slate-700">Card colour</span>
				<select class="input w-auto" bind:value={draft.cardColor}>
					<option value="none">None</option>
					<option value="type">By type</option>
					<option value="priority">By priority</option>
				</select>
			</label>
		</div>
		<div class="flex gap-2">
			<button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Saving…' : 'Save for everyone'}</button>
			<button type="button" class="btn-ghost" onclick={() => (editing = false)}>Cancel</button>
		</div>
	</form>
{/if}

<div class="-mx-4 snap-x snap-mandatory scroll-px-4 overflow-x-auto px-4 pb-2 sm:mx-0 sm:scroll-px-0 sm:px-0 lg:snap-none">
	<div class="space-y-3">
		<!-- column headers, once -->
		<div class="flex gap-3">
			{#each totals as col (col.status)}
				{@const pts = col.tasks.reduce((s, t) => s + (t.storyPoints ?? 0), 0)}
				{@const md = col.tasks.reduce((s, t) => s + (allTasks.some((c) => c.parentId === t.id && c.status !== 'cancelled') ? 0 : t.manDays), 0)}
				{@const limit = settings.wipLimits[col.status]}
				{@const wip = wipState(col.tasks.length, limit)}
				<header
					class="{colClass} flex items-center justify-between gap-2 rounded-lg border px-3 py-2 {wip === 'over'
						? 'border-red-300 bg-red-50'
						: 'border-slate-200 bg-slate-50'}"
				>
					<span class="rounded-full px-2 py-0.5 text-xs font-semibold {badgeToneClass[statusTone(col.status)]}">
						{label(col.status)}
					</span>
					<span
						class="text-xs tabular-nums {wip === 'over' ? 'font-semibold text-red-700' : 'text-slate-500'}"
						title={limit ? `WIP limit ${limit}` : undefined}
					>
						{col.tasks.length}{limit ? `/${limit}` : ''}{md > 0 ? ` · ${formatManDays(md)} md` : ''}{pts > 0
							? ` · ${formatManDays(pts)} pts`
							: ''}
					</span>
				</header>
			{/each}
		</div>

		{#each lanes as lane (lane.key)}
			{#if settings.swimlanes !== 'none'}
				<div class="sticky left-0 text-xs font-semibold text-slate-600">{lane.label} · {lane.tasks.length}</div>
			{/if}
			<div class="flex gap-3">
				{#each boardColumns(lane.tasks, statuses) as col (col.status)}
					{@const key = `${lane.key}|${col.status}`}
					<section
						class="{colClass} flex flex-col rounded-lg border bg-slate-50 {over === key
							? 'border-slate-400 ring-2 ring-slate-300'
							: 'border-slate-200'}"
						aria-label="{label(col.status)} column{lane.label ? `, ${lane.label}` : ''}"
						ondragover={(e) => {
							if (!draggingId) return;
							e.preventDefault();
							over = key;
						}}
						ondragleave={(e) => {
							if (!e.currentTarget.contains(e.relatedTarget as Node | null)) over = null;
						}}
						ondrop={(e) => {
							e.preventDefault();
							drop(col.status);
						}}
					>
						<ul class="flex min-h-20 flex-1 flex-col gap-2 p-2">
							{#each col.tasks as task (task.id)}
								{@const warnings = fitWarnings(task)}
								{@const parent = task.parentId ? allTasks.find((t) => t.id === task.parentId) : null}
								{@const sub = subtaskSummary(task.id, allTasks)}
								<li
									class="cursor-grab rounded-md border border-slate-200 bg-white p-2.5 text-sm shadow-sm active:cursor-grabbing {edge(task)} {draggingId ===
									task.id
										? 'opacity-50'
										: ''}"
									draggable={can('member')}
									ondragstart={(e) => {
										draggingId = task.id;
										e.dataTransfer?.setData('text/plain', task.id);
										if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
									}}
									ondragend={() => {
										draggingId = null;
										over = null;
									}}
									ondragover={(e) => {
										if (draggingId && draggingId !== task.id) e.preventDefault();
									}}
									ondrop={(e) => {
										e.preventDefault();
										e.stopPropagation();
										const moving = tasks.find((t) => t.id === draggingId);
										draggingId = null;
										over = null;
										if (!moving || moving.id === task.id) return;
										if (moving.status !== col.status) onMove(moving, col.status);
										onPlaceBefore(moving, task.id);
									}}
								>
									{#if parent && settings.swimlanes !== 'parent'}
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
											: ''}{task.dueDate ? ` · Due ${task.dueDate}` : ''}
									</p>
									{#each warnings as w (w)}
										<p class="mt-1 text-xs font-medium text-red-700">⚠ {w}</p>
									{/each}
									<div class="mt-2 flex flex-wrap items-center justify-between gap-1.5">
										<span class="inline-flex items-center gap-1.5">
											<Badge tone={priorityTone(task.priority)}>{task.priority}</Badge>
											{#if task.storyPoints != null}<span
													class="rounded bg-slate-100 px-1.5 text-[11px] font-medium text-slate-600"
													title="Story points">{formatManDays(task.storyPoints)}</span
												>{/if}
										</span>
										<select data-write-lock tabindex={can('member') ? 0 : -1}
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
		{/each}
	</div>
</div>
