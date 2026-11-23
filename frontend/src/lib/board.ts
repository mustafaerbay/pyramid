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
