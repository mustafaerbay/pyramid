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

const PRIORITY_RANK: Record<string, number> = { urgent: 0, high: 1, medium: 2, low: 3 };

type BoardTask = { number: number; status: string; priority: string };

/** Groups tasks into the given columns, most urgent first, then by number. */
export function boardColumns<T extends BoardTask>(
	tasks: readonly T[],
	statuses: readonly BoardStatus[]
): { status: BoardStatus; tasks: T[] }[] {
	return statuses.map((status) => ({
		status,
		tasks: tasks
			.filter((t) => t.status === status)
			.sort((a, b) => (PRIORITY_RANK[a.priority] ?? 9) - (PRIORITY_RANK[b.priority] ?? 9) || a.number - b.number)
	}));
}
