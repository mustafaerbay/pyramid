// Charts for query results on the Queries page and dashboards.

import type { GroupBy } from './queries';

/** Reference categorical palette, slots 1–8 in fixed order (validated). */
export const SERIES = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948'];
/** "Other" and anything past slot 8 (folded, never a generated hue). */
export const OTHER = '#94a3b8';
export const seriesColor = (i: number) => SERIES[i] ?? OTHER;

type Row = { status: string; type: string; priority: string; projectId: string; assigneeIds: string[] };
type Names = { person: (id: string) => string; project: (id: string) => string };

function keysOf(r: Row, by: GroupBy, names: Names): { key: string; label: string }[] {
	if (by === 'assignee') {
		return r.assigneeIds.length ? r.assigneeIds.map((id) => ({ key: id, label: names.person(id) })) : [{ key: '-', label: 'Unassigned' }];
	}
	if (by === 'project') return [{ key: r.projectId, label: names.project(r.projectId) }];
	return [{ key: r[by], label: r[by].replace('_', ' ') }];
}

/** Counts per group, split by a second field (people with several assignees count once each). */
export function stackGroups(rows: readonly Row[], by: GroupBy, stackBy: GroupBy, names: Names) {
	const groups = new Map<string, { key: string; label: string; total: number }>();
	const series = new Map<string, { key: string; label: string; total: number }>();
	const counts: Record<string, Record<string, number>> = {};
	for (const r of rows) {
		for (const g of keysOf(r, by, names)) {
			const gg = groups.get(g.key) ?? { ...g, total: 0 };
			gg.total++;
			groups.set(g.key, gg);
			for (const s of keysOf(r, stackBy, names)) {
				const ss = series.get(s.key) ?? { ...s, total: 0 };
				ss.total++;
				series.set(s.key, ss);
				counts[g.key] ??= {};
				counts[g.key][s.key] = (counts[g.key][s.key] ?? 0) + 1;
			}
		}
	}
	const order = <T extends { label: string; total: number }>(m: Map<string, T>) =>
		[...m.values()].sort((a, b) => b.total - a.total || a.label.localeCompare(b.label)).map(({ total: _t, ...x }) => x);
	const s = order(series);
	// Unassigned / none last among series.
	const last = s.findIndex((x) => x.key === '-');
	if (last >= 0) s.push(...s.splice(last, 1));
	return { groups: order(groups), series: s, counts };
}

/** Keeps the first n groups and folds the rest into "Other". */
export function topWithOther<T extends { key: string; label: string; count: number }>(data: readonly T[], n: number) {
	if (data.length <= n + 1) return [...data];
	const rest = data.slice(n).reduce((s, d) => s + d.count, 0);
	return [...data.slice(0, n), { key: '__other', label: 'Other', count: rest }];
}

const dayOf = (iso: string) => iso.slice(0, 10);
const nextDay = (d: string) => {
	const t = new Date(`${d}T00:00:00Z`);
	t.setUTCDate(t.getUTCDate() + 1);
	return t.toISOString().slice(0, 10);
};

/** Items open and finished at the end of each day in [from, to]. */
export function trendSeries(rows: readonly { createdAt: string; completedAt?: string | null }[], from: string, to: string) {
	const out: { day: string; open: number; done: number }[] = [];
	for (let d = from; d <= to; d = nextDay(d)) {
		let open = 0;
		let done = 0;
		for (const r of rows) {
			if (dayOf(r.createdAt) > d) continue;
			if (r.completedAt && dayOf(r.completedAt) <= d) done++;
			else open++;
		}
		out.push({ day: d, open, done });
	}
	return out;
}
