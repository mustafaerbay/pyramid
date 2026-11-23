// Query page helpers: chart grouping and tidy filters.

export type GroupBy = 'status' | 'assignee' | 'type' | 'priority' | 'project';

type Row = { status: string; type: string; priority: string; projectId: string; assigneeIds: string[] };

/** Count results per group, biggest first (then by label). */
export function groupResults(
	rows: readonly Row[],
	by: GroupBy,
	names: { person: (id: string) => string; project: (id: string) => string }
): { key: string; label: string; count: number }[] {
	const groups = new Map<string, { key: string; label: string; count: number }>();
	const add = (key: string, label: string) => {
		const g = groups.get(key) ?? { key, label, count: 0 };
		g.count++;
		groups.set(key, g);
	};
	for (const r of rows) {
		if (by === 'assignee') {
			if (r.assigneeIds.length === 0) add('-', 'Unassigned');
			for (const id of r.assigneeIds) add(id, names.person(id));
		} else if (by === 'project') add(r.projectId, names.project(r.projectId));
		else add(r[by], r[by].replace('_', ' '));
	}
	return [...groups.values()].sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));
}

/** Leaves out empty lists, blank text and false flags. */
export function cleanFilters<T extends Record<string, unknown>>(f: T): Partial<T> {
	const out: Record<string, unknown> = {};
	for (const [k, v] of Object.entries(f)) {
		if (v == null || v === false) continue;
		if (Array.isArray(v) && v.length === 0) continue;
		if (typeof v === 'string' && v.trim() === '') continue;
		out[k] = v;
	}
	return out as Partial<T>;
}
