// Kanban board: one column per task status.

export type State = { key: string; label: string; category: string };

/** Board columns: the workflow's states, cancelled ones only when the status filter asks. */
export function boardStates<S extends State>(workflow: readonly S[], statusFilter: readonly string[]): S[] {
	if (statusFilter.length === 0) return workflow.filter((s) => s.category !== 'cancelled');
	return workflow.filter((s) => statusFilter.includes(s.category));
}

type BoardTask = { number: number; state: string; rank: number };

/** Groups tasks into columns by state, in the manual order (rank), then by number. */
export function boardColumns<T extends BoardTask>(tasks: readonly T[], states: readonly string[]): { state: string; tasks: T[] }[] {
	return states.map((state) => ({
		state,
		tasks: tasks.filter((t) => t.state === state).sort((a, b) => a.rank - b.rank || a.number - b.number)
	}));
}

/**
 * To put `movingId` just before `targetId` in `order` (the project's tasks
 * in rank order), it goes after this task; null means first.
 */
export function afterIdToPlaceBefore(order: readonly { id: string }[], movingId: string, targetId: string): string | null {
	const rest = order.filter((t) => t.id !== movingId);
	const i = rest.findIndex((t) => t.id === targetId);
	return i > 0 ? rest[i - 1].id : null;
}

/** How full a column is against its WIP limit. */
export function wipState(count: number, limit: number | undefined): 'none' | 'ok' | 'full' | 'over' {
	if (!limit) return 'none';
	if (count > limit) return 'over';
	return count === limit ? 'full' : 'ok';
}

export type LaneMode = 'none' | 'assignee' | 'priority' | 'type' | 'parent';
type LaneTask = { id: string; assigneeIds: string[]; priority: string; type: string; parentId?: string | null };

const PRIORITY_ORDER = ['urgent', 'high', 'medium', 'low'];
const TYPE_ORDER = ['epic', 'feature', 'story', 'bug', 'task'];
const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

/**
 * Splits tasks into swimlanes. People lanes use each task's first assignee
 * (by name) with Unassigned last; priority and type lanes go top-down;
 * parent lanes are by parent title with No parent last. Empty lanes are left out.
 */
export function boardLanes<T extends LaneTask>(
	tasks: readonly T[],
	mode: LaneMode,
	names: { person: (id: string) => string; task: (id: string) => string }
): { key: string; label: string; tasks: T[] }[] {
	if (mode === 'none') return [{ key: 'all', label: '', tasks: [...tasks] }];
	const lanes = new Map<string, { key: string; label: string; order: string; tasks: T[] }>();
	for (const t of tasks) {
		let key: string, label: string, order: string;
		if (mode === 'assignee') {
			const first = [...t.assigneeIds].sort((a, b) => names.person(a).localeCompare(names.person(b)))[0];
			key = first ?? '-';
			label = first ? names.person(first) : 'Unassigned';
			order = first ? `0${label}` : '1';
		} else if (mode === 'priority') {
			key = t.priority;
			label = cap(t.priority);
			order = String(PRIORITY_ORDER.indexOf(t.priority));
		} else if (mode === 'type') {
			key = t.type;
			label = cap(t.type);
			order = String(TYPE_ORDER.indexOf(t.type));
		} else {
			key = t.parentId ?? '-';
			label = t.parentId ? names.task(t.parentId) : 'No parent';
			order = t.parentId ? `0${label}` : '1';
		}
		if (!lanes.has(key)) lanes.set(key, { key, label, order, tasks: [] });
		lanes.get(key)!.tasks.push(t);
	}
	return [...lanes.values()].sort((a, b) => a.order.localeCompare(b.order)).map(({ key, label, tasks }) => ({ key, label, tasks }));
}

export type CardRule = { field: 'priority' | 'type' | 'tag' | 'overdue' | 'blocked' | 'assignee'; value: string; color: string };
type RuleTask = {
	priority: string;
	type: string;
	customType?: string | null;
	tags: string[];
	dueDate?: string | null;
	status: string;
	blockedByIds: string[];
	assigneeIds: string[];
};

/** The colour of the first card rule the task matches (today is YYYY-MM-DD), or null. */
export function cardRuleColor(t: RuleTask, rules: readonly CardRule[] | undefined, today: string): string | null {
	const open = t.status !== 'done' && t.status !== 'cancelled';
	const v = (s: string) => s.trim().toLowerCase();
	for (const r of rules ?? []) {
		const want = v(r.value);
		const hit =
			r.field === 'priority'
				? t.priority === want
				: r.field === 'type'
					? (t.customType ?? t.type) === r.value || t.type === want
					: r.field === 'tag'
						? t.tags.some((x) => v(x) === want)
						: r.field === 'overdue'
							? want === 'true' && open && !!t.dueDate && t.dueDate < today
							: r.field === 'blocked'
								? want === 'true' && open && t.blockedByIds.length > 0
								: t.assigneeIds.includes(r.value);
		if (hit) return r.color;
	}
	return null;
}

export const DEFAULT_CARD_FIELDS = ['assignees', 'manDays', 'points', 'due', 'tags', 'priority', 'parent'];

/** Whether cards show this field, by the board's card field list (missing = the defaults). */
export function cardShows(fields: readonly string[] | undefined | null, f: string): boolean {
	return (fields ?? DEFAULT_CARD_FIELDS).includes(f);
}

/** The Doing and Done halves of a split column. */
export function splitColumn<T extends { boardDone: boolean }>(tasks: readonly T[]): { doing: T[]; done: T[] } {
	return { doing: tasks.filter((t) => !t.boardDone), done: tasks.filter((t) => t.boardDone) };
}
