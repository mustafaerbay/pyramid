// Task dependencies: blockedByIds lists the tasks that must finish first.

type Dep = { id: string; status: string; blockedByIds: string[] };

const finished = (t: Dep) => t.status === 'done' || t.status === 'cancelled';

/** Blockers of `task` that aren't finished yet. */
export function openBlockers<T extends Dep>(task: T, all: readonly T[]): T[] {
	return task.blockedByIds.map((id) => all.find((t) => t.id === id)).filter((b): b is T => !!b && !finished(b));
}

/** Tasks that wait on `id`. */
export function blocksOf<T extends Dep>(id: string, all: readonly T[]): T[] {
	return all.filter((t) => t.blockedByIds.includes(id));
}

/**
 * Tasks `taskId` may wait on: not itself, and not any task that already
 * waits on it (directly or further down), which would make a loop.
 */
export function blockerOptions<T extends Dep>(all: readonly T[], taskId: string | null): T[] {
	if (!taskId) return [...all];
	const waiting = new Set<string>([taskId]);
	let grew = true;
	while (grew) {
		grew = false;
		for (const t of all)
			if (!waiting.has(t.id) && t.blockedByIds.some((b) => waiting.has(b))) {
				waiting.add(t.id);
				grew = true;
			}
	}
	return all.filter((t) => !waiting.has(t.id));
}
