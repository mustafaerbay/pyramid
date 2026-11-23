// Subtasks: one level deep. A task with live (non-cancelled) subtasks is a
// container — its subtasks carry the work, so its own man-days aren't
// counted (the backend's capacity does the same).

type Node = { id: string; parentId?: string | null; status: string; manDays: number };

const live = (t: Node) => t.status !== 'cancelled';

export function subtaskSummary<T extends Node>(
	id: string,
	all: readonly T[]
): { count: number; done: number; manDays: number } {
	const kids = all.filter((t) => t.parentId === id && live(t));
	return {
		count: kids.length,
		done: kids.filter((t) => t.status === 'done').length,
		manDays: kids.reduce((s, t) => s + t.manDays, 0)
	};
}

/** False for a task whose live subtasks carry its work. */
export function countsTowardTotals<T extends Node>(task: T, all: readonly T[]): boolean {
	return !all.some((t) => t.parentId === task.id && live(t));
}

/**
 * Orders `visible` so each subtask follows its parent (depth 1). A subtask
 * whose parent isn't visible stays at depth 0. `parent` is set for every
 * subtask, looked up in `all`.
 */
export function nestTasks<T extends Node>(
	visible: readonly T[],
	all: readonly T[]
): { task: T; depth: 0 | 1; parent: T | null }[] {
	const shown = new Set(visible.map((t) => t.id));
	const byId = new Map(all.map((t) => [t.id, t]));
	const rows: { task: T; depth: 0 | 1; parent: T | null }[] = [];
	for (const task of visible) {
		const nested = task.parentId != null && shown.has(task.parentId);
		if (nested) continue;
		rows.push({ task, depth: 0, parent: task.parentId ? (byId.get(task.parentId) ?? null) : null });
		for (const kid of visible) if (kid.parentId === task.id) rows.push({ task: kid, depth: 1, parent: task });
	}
	return rows;
}

/** Tasks that `taskId` (null when creating) may be put under. */
export function parentOptions<T extends Node>(all: readonly T[], taskId: string | null): T[] {
	if (taskId && all.some((t) => t.parentId === taskId)) return [];
	return all.filter((t) => !t.parentId && t.id !== taskId);
}
