// Copying work items and making templates from them.

type Src = {
	projectId: string;
	title: string;
	description?: string | null;
	type: string;
	customType?: string | null;
	priority: string;
	tags: string[];
	storyPoints?: number | null;
	estimateManDays?: number | null;
	assignments: { personId: string; manDays: number }[];
	sprintId?: string | null;
	areaId?: string | null;
	milestoneId?: string | null;
	startDate?: string | null;
	dueDate?: string | null;
	customValues?: Record<string, unknown>;
};

const defined = <T extends Record<string, unknown>>(o: T) =>
	Object.fromEntries(Object.entries(o).filter(([, v]) => v !== null && v !== undefined)) as Partial<T>;

/** The create body for a copy of `t` (new state, no links); children keep their titles. */
export function cloneBody(t: Src, o: { parentId?: string | null; keepTitle?: boolean }) {
	return defined({
		projectId: t.projectId,
		title: o.keepTitle ? t.title : `Copy of ${t.title}`.slice(0, 300),
		description: t.description,
		type: t.type,
		customType: t.customType,
		priority: t.priority,
		tags: [...t.tags],
		storyPoints: t.storyPoints,
		estimateManDays: t.estimateManDays,
		assignments: t.assignments.map((a) => ({ ...a })),
		sprintId: t.sprintId,
		areaId: t.areaId,
		milestoneId: t.milestoneId,
		parentId: o.parentId,
		startDate: t.startDate,
		dueDate: t.dueDate,
		customValues: t.customValues && Object.keys(t.customValues).length ? { ...t.customValues } : undefined
	});
}

/** Everything under `id`, depth first in manual order. */
export function descendants<T extends { id: string; parentId?: string | null; rank: number; number: number }>(id: string, all: readonly T[]): T[] {
	const out: T[] = [];
	const walk = (pid: string, depth: number) => {
		if (depth > 10) return;
		for (const k of all.filter((x) => x.parentId === pid).sort((a, b) => a.rank - b.rank || a.number - b.number)) {
			out.push(k);
			walk(k.id, depth + 1);
		}
	};
	walk(id, 0);
	return out;
}

/** The fields an item template keeps from an item. */
export function templateFields(t: Src) {
	return defined({
		type: t.type,
		customType: t.customType,
		title: t.title,
		description: t.description,
		priority: t.priority,
		tags: t.tags.length ? [...t.tags] : undefined,
		storyPoints: t.storyPoints,
		estimateManDays: t.estimateManDays,
		customValues: t.customValues && Object.keys(t.customValues).length ? { ...t.customValues } : undefined
	});
}
