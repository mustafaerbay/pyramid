// Kanban board: one column per task status.

const ALL_STATUSES = ['backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'] as const;
export type BoardStatus = (typeof ALL_STATUSES)[number];

/** Columns shown when no status filter is set: everything but cancelled. */
export const DEFAULT_BOARD_STATUSES: BoardStatus[] = ALL_STATUSES.filter((s) => s !== 'cancelled');

/** The board's columns for a status filter (empty = default), in board order. */
export function boardStatuses(filter: readonly string[]): BoardStatus[] {
	if (filter.length === 0) return DEFAULT_BOARD_STATUSES;
	return ALL_STATUSES.filter((s) => filter.includes(s));
}

type BoardTask = { number: number; status: string; rank: number };

/** Groups tasks into the given columns, in the manual order (rank), then by number. */
export function boardColumns<T extends BoardTask>(
	tasks: readonly T[],
	statuses: readonly BoardStatus[]
): { status: BoardStatus; tasks: T[] }[] {
	return statuses.map((status) => ({
		status,
		tasks: tasks.filter((t) => t.status === status).sort((a, b) => a.rank - b.rank || a.number - b.number)
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
