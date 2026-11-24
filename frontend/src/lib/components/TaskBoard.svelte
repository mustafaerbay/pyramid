<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { can } from '$lib/auth.svelte';
	import type { components } from '$lib/api/schema';
	import Badge from '$lib/components/Badge.svelte';
	import TagChips from '$lib/components/TagChips.svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';
	import { boardColumns, boardLanes, cardRuleColor, cardShows, DEFAULT_CARD_FIELDS, splitColumn, wipState, type CardRule, type State } from '$lib/board';
	import { formatManDays } from '$lib/capacity';
	import { subtaskSummary } from '$lib/subtasks';
	import DependencyNote from '$lib/components/DependencyNote.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type TaskStatus = components['schemas']['TaskStatus'];
	type BoardSettings = components['schemas']['BoardSettings'];
	type CustomField = components['schemas']['CustomField'];
	const COLORS = ['orange', 'violet', 'sky', 'red', 'amber', 'emerald', 'pink', 'slate'] as const;
	const TINT: Record<string, string> = {
		orange: 'bg-orange-50',
		violet: 'bg-violet-50',
		sky: 'bg-sky-50',
		red: 'bg-red-50',
		amber: 'bg-amber-50',
		emerald: 'bg-emerald-50',
		pink: 'bg-pink-50',
		slate: 'bg-slate-100'
	};
	const FIELD_LABELS: Record<string, string> = {
		assignees: 'People',
		manDays: 'Man-days',
		points: 'Story points',
		due: 'Due date',
		tags: 'Tags',
		priority: 'Priority',
		parent: 'Parent',
		remaining: 'Remaining work'
	};

	let {
		tasks,
		allTasks,
		states,
		workflow,
		statusTone,
		priorityTone,
		assigneesLabel,
		personName,
		fitWarnings,
		settings,
		canEditSettings = false,
		customFields = [],
		onSaveSettings,
		onOpen,
		onMove,
		onPlaceBefore,
		onTagSelect
	}: {
		tasks: Task[];
		/** Every task in the project, for parent titles and subtask counts. */
		allTasks: Task[];
		/** Columns: workflow states shown on the board. */
		states: State[];
		/** Every state of the project's workflow (for the move picker). */
		workflow: State[];
		statusTone: (s: TaskStatus) => BadgeTone;
		priorityTone: (p: Task['priority']) => BadgeTone;
		assigneesLabel: (ids: string[]) => string;
		personName: (id: string) => string;
		fitWarnings: (t: Task) => string[];
		settings: BoardSettings;
		canEditSettings?: boolean;
		/** The project's custom fields (cards can show them). */
		customFields?: CustomField[];
		onSaveSettings: (s: BoardSettings) => Promise<boolean>;
		onOpen: (t: Task) => void;
		/** Move to a workflow state (by key); done = which half of a split column. */
		onMove: (t: Task, state: string, done?: boolean) => void;
		/** Dropped on a card: goes just before it in the manual order. */
		onPlaceBefore: (t: Task, targetId: string) => void;
		onTagSelect: (tag: string) => void;
	} = $props();

	const label = (key: string) => workflow.find((o) => o.key === key)?.label ?? key;
	const category = (key: string) => (workflow.find((o) => o.key === key)?.category ?? key) as TaskStatus;
	const stateKeys = $derived(states.map((s) => s.key));
	const taskTitle = (id: string) => allTasks.find((t) => t.id === id)?.title ?? 'Deleted item';
	const lanes = $derived(boardLanes(tasks, settings.swimlanes, { person: personName, task: taskTitle }));
	const totals = $derived(boardColumns(tasks, stateKeys));
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

	const split = $derived(new Set(settings.splitColumns ?? []));
	const tagColors = $derived(Object.fromEntries(Object.entries(settings.tagColors ?? {}).map(([k, v]) => [k.toLowerCase(), v])));
	const shows = (f: string) => cardShows(settings.cardFields, f);
	const today = new Date().toISOString().slice(0, 10);
	const tint = (t: Task) => TINT[cardRuleColor(t, settings.cardRules as CardRule[] | undefined, today) ?? ''] ?? 'bg-white';
	const customShown = $derived(customFields.filter((f) => shows(`custom:${f.key}`)));
	const customText = (t: Task, f: CustomField) => {
		const v = t.customValues?.[f.key];
		if (v == null || v === '') return '';
		if (typeof v === 'boolean') return v ? 'Yes' : 'No';
		return Array.isArray(v) ? v.join(', ') : String(v);
	};

	let draggingId = $state<string | null>(null);
	let over = $state<string | null>(null); // `${lane}|${status}|${half}`

	// done: undefined for an unsplit column, else the half it was dropped in.
	function drop(state: string, done?: boolean) {
		const task = tasks.find((t) => t.id === draggingId);
		draggingId = null;
		over = null;
		if (!task) return;
		if (task.state !== state || (done !== undefined && task.boardDone !== done)) onMove(task, state, done);
	}

	// Settings panel.
	let editing = $state(false);
	let draft = $state<BoardSettings>({ wipLimits: {}, swimlanes: 'none', cardColor: 'none' });
	let draftSplit = $state<Record<string, boolean>>({});
	let draftDod = $state<Record<string, string>>({});
	let draftFields = $state<string[]>([]);
	let draftTags = $state<{ tag: string; color: string }[]>([]);
	let draftRules = $state<CardRule[]>([]);
	let saving = $state(false);
	const allTags = $derived([...new Set(allTasks.flatMap((t) => t.tags))].sort((a, b) => a.localeCompare(b)));
	function openSettings() {
		draft = { wipLimits: { ...settings.wipLimits }, swimlanes: settings.swimlanes, cardColor: settings.cardColor };
		draftSplit = Object.fromEntries((settings.splitColumns ?? []).map((k) => [k, true]));
		draftDod = { ...(settings.definitionOfDone ?? {}) };
		draftFields = [...(settings.cardFields ?? DEFAULT_CARD_FIELDS)];
		draftTags = Object.entries(settings.tagColors ?? {}).map(([tag, color]) => ({ tag, color }));
		draftRules = ((settings.cardRules ?? []) as CardRule[]).map((r) => ({ ...r }));
		editing = true;
	}
	const toggleField = (f: string) => (draftFields = draftFields.includes(f) ? draftFields.filter((x) => x !== f) : [...draftFields, f]);
	async function save() {
		saving = true;
		const clean = Object.fromEntries(
			Object.entries(draft.wipLimits).filter(([, v]) => v != null && Number(v) >= 1).map(([k, v]) => [k, Math.round(Number(v))])
		);
		const ok = await onSaveSettings({
			...draft,
			wipLimits: clean,
			splitColumns: Object.keys(draftSplit).filter((k) => draftSplit[k] && stateKeys.includes(k)),
			definitionOfDone: Object.fromEntries(Object.entries(draftDod).filter(([, v]) => v.trim()).map(([k, v]) => [k, v.trim()])),
			cardFields: draftFields,
			tagColors: Object.fromEntries(draftTags.filter((x) => x.tag.trim()).map((x) => [x.tag.trim(), x.color])) as BoardSettings['tagColors'],
			cardRules: draftRules.filter((r) => r.value.trim()) as BoardSettings['cardRules']
		});
		saving = false;
		if (ok) editing = false;
	}
</script>

<div class="mb-2 flex justify-end">
	{#if canEditSettings}
		<button type="button" class="btn-ghost px-2 py-1 text-xs" aria-expanded={editing} onclick={() => (editing ? (editing = false) : openSettings())}>
			<Icon name="settings" size={14} class="mr-1 align-[-2px]" />Board settings
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
				{#each stateKeys as s (s)}
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
		<div>
			<div class="mb-1 font-medium text-slate-700">Columns</div>
			<p class="mb-2 text-xs text-slate-500">Split a column into Doing and Done; a definition of done shows on its header.</p>
			<div class="space-y-2">
				{#each states.filter((st) => st.category !== 'done' && st.category !== 'cancelled') as st (st.key)}
					<div class="flex flex-wrap items-start gap-3">
						<label class="flex w-36 items-center gap-1.5 pt-2 text-xs"><input type="checkbox" bind:checked={draftSplit[st.key]} /> Split {st.label}</label>
						<textarea
							class="input min-h-9 flex-1 text-xs"
							rows="1"
							maxlength="500"
							placeholder="Definition of done for {st.label}"
							aria-label="Definition of done for {st.label}"
							bind:value={draftDod[st.key]}
						></textarea>
					</div>
				{/each}
			</div>
		</div>
		<div>
			<div class="mb-1 font-medium text-slate-700">Card fields</div>
			<div class="flex flex-wrap gap-x-4 gap-y-1 text-xs">
				{#each Object.keys(FIELD_LABELS) as f (f)}
					<label class="flex items-center gap-1.5"><input type="checkbox" checked={draftFields.includes(f)} onchange={() => toggleField(f)} /> {FIELD_LABELS[f]}</label>
				{/each}
				{#each customFields as f (f.key)}
					<label class="flex items-center gap-1.5"
						><input type="checkbox" checked={draftFields.includes(`custom:${f.key}`)} onchange={() => toggleField(`custom:${f.key}`)} /> {f.label}</label
					>
				{/each}
			</div>
		</div>
		<div>
			<div class="mb-1 font-medium text-slate-700">Tag colours</div>
			<div class="space-y-1.5">
				{#each draftTags as row, i (i)}
					<div class="flex items-center gap-2">
						<select class="input w-auto text-xs" aria-label="Tag" bind:value={row.tag}>
							<option value="">Tag…</option>
							{#each allTags as tg (tg)}<option value={tg}>{tg}</option>{/each}
						</select>
						<select class="input w-auto text-xs" aria-label="Colour" bind:value={row.color}>
							{#each COLORS as c (c)}<option value={c}>{c}</option>{/each}
						</select>
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (draftTags = draftTags.filter((_, j) => j !== i))}>Remove</button>
					</div>
				{/each}
				<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (draftTags = [...draftTags, { tag: '', color: 'sky' }])}>+ Tag colour</button>
			</div>
		</div>
		<div>
			<div class="mb-1 font-medium text-slate-700">Card styling rules</div>
			<p class="mb-2 text-xs text-slate-500">The first rule a card matches tints it.</p>
			<div class="space-y-1.5">
				{#each draftRules as rule, i (i)}
					<div class="flex flex-wrap items-center gap-2">
						<select class="input w-auto text-xs" aria-label="Rule field" bind:value={rule.field} onchange={() => (rule.value = rule.field === 'overdue' || rule.field === 'blocked' ? 'true' : '')}>
							<option value="priority">Priority is</option>
							<option value="type">Type is</option>
							<option value="tag">Has tag</option>
							<option value="assignee">Assigned to</option>
							<option value="overdue">Is overdue</option>
							<option value="blocked">Is blocked</option>
						</select>
						{#if rule.field === 'priority'}
							<select class="input w-auto text-xs" aria-label="Priority" bind:value={rule.value}>
								<option value="">…</option>
								{#each ['urgent', 'high', 'medium', 'low'] as pr (pr)}<option value={pr}>{pr}</option>{/each}
							</select>
						{:else if rule.field === 'tag'}
							<select class="input w-auto text-xs" aria-label="Tag" bind:value={rule.value}>
								<option value="">…</option>
								{#each allTags as tg (tg)}<option value={tg}>{tg}</option>{/each}
							</select>
						{:else if rule.field === 'assignee'}
							<select class="input w-auto text-xs" aria-label="Person" bind:value={rule.value}>
								<option value="">…</option>
								{#each [...new Set(allTasks.flatMap((t) => t.assigneeIds))] as pid (pid)}<option value={pid}>{personName(pid)}</option>{/each}
							</select>
						{:else if rule.field === 'type'}
							<input class="input w-32 text-xs" aria-label="Type" placeholder="bug, story…" bind:value={rule.value} />
						{/if}
						<span class="text-xs text-slate-500">→</span>
						<select class="input w-auto text-xs" aria-label="Colour" bind:value={rule.color}>
							{#each COLORS as c (c)}<option value={c}>{c}</option>{/each}
						</select>
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (draftRules = draftRules.filter((_, j) => j !== i))}>Remove</button>
					</div>
				{/each}
				{#if draftRules.length < 20}
					<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (draftRules = [...draftRules, { field: 'priority', value: 'urgent', color: 'red' }])}>+ Rule</button>
				{/if}
			</div>
		</div>
		<div class="flex gap-2">
			<button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Saving…' : 'Save for everyone'}</button>
			<button type="button" class="btn-ghost" onclick={() => (editing = false)}>Cancel</button>
		</div>
	</form>
{/if}

{#snippet card(task: Task, colState: string, half: boolean | undefined)}
	{@const warnings = fitWarnings(task)}
	{@const parent = task.parentId ? allTasks.find((t) => t.id === task.parentId) : null}
	{@const sub = subtaskSummary(task.id, allTasks)}
	{@const md = sub.count > 0 ? sub.manDays : task.manDays}
	{@const meta = [
		shows('assignees') ? assigneesLabel(task.assigneeIds) : '',
		shows('manDays') && md > 0 ? `${formatManDays(md)} md` : '',
		shows('remaining') && task.remainingManDays != null ? `${formatManDays(task.remainingManDays)} md left` : '',
		shows('due') && task.dueDate ? `Due ${task.dueDate}` : '',
		...customShown.map((f) => (customText(task, f) ? `${f.label}: ${customText(task, f)}` : ''))
	].filter(Boolean)}
	<li
		class="cursor-grab rounded-md border border-slate-200 p-2.5 text-sm shadow-sm active:cursor-grabbing {tint(task)} {edge(task)} {draggingId ===
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
			if (moving.state !== colState || (half !== undefined && moving.boardDone !== half)) onMove(moving, colState, half);
			onPlaceBefore(moving, task.id);
		}}
	>
		{#if parent && settings.swimlanes !== 'parent' && shows('parent')}
			<p class="mb-0.5 truncate text-xs text-slate-500">↳ {parent.title}</p>
		{/if}
		<button type="button" class="block min-h-6 w-full text-left font-medium text-slate-800 hover:underline" onclick={() => onOpen(task)}>
			<TypeBadge type={task.type} label={task.customTypeLabel} color={task.customTypeColor} /> {task.title}
		</button>
		{#if sub.count > 0}
			<p class="mt-1 text-xs text-slate-500">{sub.done}/{sub.count} children done</p>
		{/if}
		<DependencyNote {task} all={allTasks} />
		{#if task.tags.length && shows('tags')}
			<div class="mt-1.5"><TagChips tags={task.tags} onSelect={onTagSelect} colors={tagColors} /></div>
		{/if}
		{#if meta.length}<p class="mt-1.5 line-clamp-2 text-xs text-slate-500">{meta.join(' · ')}</p>{/if}
		{#each warnings as w (w)}
			<p class="mt-1 text-xs font-medium text-red-700">⚠ {w}</p>
		{/each}
		<div class="mt-2 flex flex-wrap items-center justify-between gap-1.5">
			<span class="inline-flex items-center gap-1.5">
				{#if shows('priority')}<Badge tone={priorityTone(task.priority)}>{task.priority}</Badge>{/if}
				{#if task.storyPoints != null && shows('points')}<span
						class="rounded bg-slate-100 px-1.5 text-xs font-medium text-slate-600"
						title="Story points">{formatManDays(task.storyPoints)}</span
					>{/if}
			</span>
			<select data-write-lock tabindex={can('member') ? 0 : -1}
				class="min-h-8 min-w-0 max-w-full rounded-md border-0 bg-slate-100 py-0 pl-2 pr-6 text-xs text-slate-600"
				aria-label="Move {task.title} to"
				value={split.has(task.state) ? `${task.state}|${task.boardDone ? 1 : 0}` : task.state}
				onchange={(e) => {
					const [key, h] = e.currentTarget.value.split('|');
					onMove(task, key, h === undefined ? (key === task.state ? task.boardDone : undefined) : h === '1');
				}}
			>
				{#each workflow as opt (opt.key)}
					{#if split.has(opt.key)}
						<option value="{opt.key}|0">{opt.label} · Doing</option>
						<option value="{opt.key}|1">{opt.label} · Done</option>
					{:else}
						<option value={opt.key}>{opt.label}</option>
					{/if}
				{/each}
			</select>
		</div>
	</li>
{/snippet}

<div class="-mx-4 snap-x snap-mandatory scroll-px-4 overflow-x-auto px-4 pb-2 sm:mx-0 sm:scroll-px-0 sm:px-0 lg:snap-none">
	<div class="space-y-3">
		<!-- column headers, once -->
		<div class="flex gap-3">
			{#each totals as col (col.state)}
				{@const pts = col.tasks.reduce((s, t) => s + (t.storyPoints ?? 0), 0)}
				{@const md = col.tasks.reduce((s, t) => s + (allTasks.some((c) => c.parentId === t.id && c.status !== 'cancelled') ? 0 : t.manDays), 0)}
				{@const limit = settings.wipLimits[col.state]}
				{@const wip = wipState(col.tasks.length, limit)}
				<header
					class="{colClass} flex items-center justify-between gap-2 rounded-lg border px-3 py-2 {wip === 'over'
						? 'border-red-300 bg-red-50'
						: 'border-slate-200 bg-slate-50'}"
				>
					<span class="inline-flex items-center gap-1">
						<span class="rounded-full px-2 py-0.5 text-xs font-semibold {badgeToneClass[statusTone(category(col.state))]}">
							{label(col.state)}
						</span>
						{#if settings.definitionOfDone?.[col.state]}
							<details class="relative">
								<summary class="cursor-pointer list-none text-xs text-slate-500 hover:text-slate-700" title="Definition of done" aria-label="Definition of done">ⓘ</summary>
								<div class="absolute left-0 z-20 mt-1 w-64 whitespace-pre-line rounded-md border border-slate-200 bg-white p-2 text-xs text-slate-700 shadow-lg">
									<b class="block">Definition of done</b>{settings.definitionOfDone[col.state]}
								</div>
							</details>
						{/if}
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
				{#each boardColumns(lane.tasks, stateKeys) as col (col.state)}
					{@const key = `${lane.key}|${col.state}`}
					<section
						class="{colClass} flex flex-col rounded-lg border bg-slate-50 {over === key
							? 'border-slate-400 ring-2 ring-slate-300'
							: 'border-slate-200'}"
						aria-label="{label(col.state)} column{lane.label ? `, ${lane.label}` : ''}"
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
							drop(col.state);
						}}
					>
						{#snippet list(items: Task[], half: boolean | undefined)}
							<ul class="flex min-h-20 flex-1 flex-col gap-2 p-2">
								{#each items as task (task.id)}
									{@render card(task, col.state, half)}
								{:else}
									<li class="rounded-md border border-dashed border-slate-200 p-3 text-center text-xs text-slate-500">
										Drop tasks here
									</li>
								{/each}
							</ul>
						{/snippet}
						{#if split.has(col.state)}
							{@const halves = splitColumn(col.tasks)}
							{#each [false, true] as half (half)}
								<div
									class="flex flex-1 flex-col {half ? 'border-t border-dashed border-slate-300' : ''} {over === `${key}|${half}` ? 'bg-slate-100' : ''}"
									role="group"
									aria-label="{label(col.state)} {half ? 'Done' : 'Doing'}"
									ondragover={(e) => {
										if (!draggingId) return;
										e.preventDefault();
										e.stopPropagation();
										over = `${key}|${half}`;
									}}
									ondrop={(e) => {
										e.preventDefault();
										e.stopPropagation();
										drop(col.state, half);
									}}
								>
									<div class="px-2 pt-1.5 text-xs font-semibold uppercase tracking-wide text-slate-500">
										{half ? 'Done' : 'Doing'} · {(half ? halves.done : halves.doing).length}
									</div>
									{@render list(half ? halves.done : halves.doing, half)}
								</div>
							{/each}
						{:else}
							{@render list(col.tasks, undefined)}
						{/if}
					</section>
				{/each}
			</div>
		{/each}
	</div>
</div>
