// Sprint report helpers: units, ideal burndown line, velocity.

export type Metric = 'manDays' | 'points' | 'count';

type Day = {
	scopeCount: number;
	doneCount: number;
	scopePoints: number;
	donePoints: number;
	scopeManDays: number;
	doneManDays: number;
};

export function metricOf(d: Day, m: Metric): { scope: number; done: number; remaining: number } {
	const [scope, done] =
		m === 'points' ? [d.scopePoints, d.donePoints] : m === 'count' ? [d.scopeCount, d.doneCount] : [d.scopeManDays, d.doneManDays];
	return { scope, done, remaining: Math.max(scope - done, 0) };
}

const DAY = 86_400_000;

/** Days from start to end, both included (dates as YYYY-MM-DD). */
export function sprintLength(start: string, end: string): number {
	return Math.round((Date.parse(end) - Date.parse(start)) / DAY) + 1;
}

/** Ideal remaining work on day `index`: a straight line from `total` to 0 on the last day. */
export function idealRemaining(total: number, index: number, length: number): number {
	if (length <= 1) return 0;
	return Math.max(total - (total * index) / (length - 1), 0);
}

type SprintLike = { name: string; status: string; startDate: string; stats: { points: number; donePoints: number } };

/** Points committed and done in the last six completed sprints, oldest first. */
export function velocity(sprints: readonly SprintLike[]): { name: string; committed: number; done: number }[] {
	return sprints
		.filter((s) => s.status === 'completed')
		.sort((a, b) => a.startDate.localeCompare(b.startDate))
		.slice(-6)
		.map((s) => ({ name: s.name, committed: s.stats.points, done: s.stats.donePoints }));
}
