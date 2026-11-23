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
	// Work sits with the items that have no live children of their own.
	const rollup = (t: T, depth: number): number => {
		const sub = all.filter((x) => x.parentId === t.id && live(x));
		return sub.length === 0 || depth > 6 ? t.manDays : sub.reduce((s, x) => s + rollup(x, depth + 1), 0);
	};
	return {
		count: kids.length,
		done: kids.filter((t) => t.status === 'done').length,
		manDays: kids.reduce((s, t) => s + rollup(t, 0), 0)
	};
}

/** False for a task whose live subtasks carry its work. */
export function countsTowardTotals<T extends Node>(task: T, all: readonly T[]): boolean {
	return !all.some((t) => t.parentId === task.id && live(t));
}

/** Work item levels, top first; a child can't outrank its parent. */
export const TYPE_LEVEL: Record<string, number> = { epic: 4, feature: 3, story: 2, bug: 2, task: 1 };
const levelOf = (type?: string) => TYPE_LEVEL[type ?? 'task'] ?? 1;

/**
 * Orders `visible` so every item follows its parent, indented one level
 * deeper. An item whose parent isn't visible starts at depth 0. `parent`
 * is set for every child, looked up in `all`.
 */
export function nestTasks<T extends Node>(
	visible: readonly T[],
	all: readonly T[]
): { task: T; depth: number; parent: T | null }[] {
	const shown = new Set(visible.map((t) => t.id));
	const byId = new Map(all.map((t) => [t.id, t]));
	const rows: { task: T; depth: number; parent: T | null }[] = [];
	const placed = new Set<string>();
	const add = (task: T, depth: number) => {
		if (placed.has(task.id)) return;
		placed.add(task.id);
		rows.push({ task, depth, parent: task.parentId ? (byId.get(task.parentId) ?? null) : null });
		for (const kid of visible) if (kid.parentId === task.id) add(kid, depth + 1);
	};
	for (const task of visible) if (!(task.parentId != null && shown.has(task.parentId))) add(task, 0);
	return rows;
}

/**
 * Items `taskId` (null when creating) of `type` may be put under: not
 * itself or anything below it, and not a lower-level type.
 */
export function parentOptions<T extends Node & { type?: string }>(
	all: readonly T[],
	taskId: string | null,
	type: string = 'task'
): T[] {
	const below = new Set<string>(taskId ? [taskId] : []);
	let grew = true;
	while (grew) {
		grew = false;
		for (const t of all)
			if (t.parentId && below.has(t.parentId) && !below.has(t.id)) {
				below.add(t.id);
				grew = true;
			}
	}
	return all.filter((t) => !below.has(t.id) && levelOf(t.type) >= levelOf(type));
}
