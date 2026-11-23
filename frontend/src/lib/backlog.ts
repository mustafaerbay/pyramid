// Backlog levels, roll-ups and a velocity forecast.

export type Level = 'epics' | 'features' | 'stories' | 'tasks';

/** Which backlog a (base) type belongs to; bugs sit with stories. */
export function backlogLevel(type: string): Level {
	return type === 'epic' ? 'epics' : type === 'feature' ? 'features' : type === 'story' || type === 'bug' ? 'stories' : 'tasks';
}

type Item = { id: string; parentId?: string | null; status: string; storyPoints?: number | null; manDays: number };
const live = (t: Item) => t.status !== 'cancelled';

/**
 * Points and man-days of an item and everything below it (cancelled left
 * out). Points count where they're set; man-days count on items with no
 * live children. % done is by points when there are any, else by man-days.
 */
export function rollup(item: Item, all: readonly Item[]) {
	let points = 0,
		donePoints = 0,
		manDays = 0,
		doneManDays = 0;
	const visit = (t: Item, depth: number) => {
		if (!live(t) || depth > 6) return;
		const done = t.status === 'done';
		if (t.storyPoints != null) {
			points += t.storyPoints;
			if (done) donePoints += t.storyPoints;
		}
		const kids = all.filter((x) => x.parentId === t.id && live(x));
		if (kids.length === 0) {
			manDays += t.manDays;
			if (done) doneManDays += t.manDays;
		}
		kids.forEach((k) => visit(k, depth + 1));
	};
	visit(item, 0);
	const pctDone = points > 0 ? Math.round((donePoints / points) * 100) : manDays > 0 ? Math.round((doneManDays / manDays) * 100) : 0;
	return { points, donePoints, manDays, doneManDays, pctDone };
}

/** Average done points of the last three finished sprints. */
export function velocityOf(sprints: readonly { status: string; startDate: string; stats: { donePoints: number } }[]): number {
	const done = sprints
		.filter((s) => s.status === 'completed')
		.sort((a, b) => a.startDate.localeCompare(b.startDate))
		.slice(-3);
	return done.length ? done.reduce((s, x) => s + x.stats.donePoints, 0) / done.length : 0;
}

/**
 * In backlog order, puts open items into upcoming sprints, `velocity`
 * points each, and names the sprint each one finishes in (an item starts a
 * new sprint when it doesn't fit, unless the sprint is empty; a bigger one
 * spills over). Unestimated items ride along. Past the named sprints
 * they're "Last + n".
 */
export function forecast(
	items: readonly { id: string; status: string; storyPoints?: number | null }[],
	velocity: number,
	sprintNames: readonly string[]
): Map<string, string> {
	const out = new Map<string, string>();
	if (velocity <= 0) return out;
	let sprint = 0,
		used = 0;
	const name = (i: number) =>
		i < sprintNames.length ? sprintNames[i] : `${sprintNames.at(-1) ?? 'Sprint'} + ${i - sprintNames.length + 1}`;
	for (const it of items) {
		if (it.status === 'done' || it.status === 'cancelled') continue;
		const p = it.storyPoints ?? 0;
		while (used > 0 && used + p > velocity) {
			sprint++;
			used = Math.max(used - velocity, 0);
		}
		used += p;
		// A big item spills over: it lands in the sprint where it finishes.
		while (used > velocity) {
			sprint++;
			used -= velocity;
		}
		out.set(it.id, name(sprint));
	}
	return out;
}
