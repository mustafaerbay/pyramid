// Query language: groups of conditions (AND / OR) edited on the Queries page.

export type Op = 'eq' | 'ne' | 'contains' | 'not_contains' | 'gt' | 'gte' | 'lt' | 'lte' | 'empty' | 'not_empty';
export type Condition = { field: string; op: Op; value?: string };
export type Group = { match: 'all' | 'any'; conditions: Condition[]; groups?: Group[] };

export const needsValue = (op: Op) => op !== 'empty' && op !== 'not_empty';

/** Leaves out conditions without a value and groups with nothing in them; null when nothing is left. */
export function cleanGroup(g: Group): Group | null {
	const conditions = g.conditions
		.map((c) => (needsValue(c.op) ? { field: c.field, op: c.op, value: (c.value ?? '').trim() } : { field: c.field, op: c.op }))
		.filter((c) => !needsValue(c.op) || c.value);
	const groups = (g.groups ?? []).map(cleanGroup).filter((x): x is Group => x !== null);
	if (!conditions.length && !groups.length) return null;
	return groups.length ? { match: g.match, conditions, groups } : { match: g.match, conditions };
}

/** How far to indent each row of a tree query: children sit under the row their parent is on. */
export function treeDepth(rows: readonly { id: string; parentId?: string | null }[], childIds: Set<string>): Record<string, number> {
	const out: Record<string, number> = {};
	for (const r of rows) out[r.id] = childIds.has(r.id) && r.parentId && r.parentId in out ? out[r.parentId] + 1 : 0;
	return out;
}

export type FieldKind = 'text' | 'choice' | 'person' | 'tag' | 'number' | 'date' | 'id';
export const FIELDS: { key: string; label: string; kind: FieldKind }[] = [
	{ key: 'title', label: 'Title', kind: 'text' },
	{ key: 'description', label: 'Description', kind: 'text' },
	{ key: 'type', label: 'Type', kind: 'choice' },
	{ key: 'status', label: 'Status', kind: 'choice' },
	{ key: 'state', label: 'State', kind: 'text' },
	{ key: 'priority', label: 'Priority', kind: 'choice' },
	{ key: 'assignee', label: 'Assigned to', kind: 'person' },
	{ key: 'tag', label: 'Tag', kind: 'tag' },
	{ key: 'storyPoints', label: 'Story points', kind: 'number' },
	{ key: 'manDays', label: 'Man-days', kind: 'number' },
	{ key: 'remaining', label: 'Remaining work', kind: 'number' },
	{ key: 'dueDate', label: 'Due date', kind: 'date' },
	{ key: 'startDate', label: 'Start date', kind: 'date' },
	{ key: 'createdAt', label: 'Created', kind: 'date' }
];

export const OPS: Record<FieldKind, { op: Op; label: string }[]> = {
	text: [
		{ op: 'contains', label: 'contains' },
		{ op: 'not_contains', label: "doesn't contain" },
		{ op: 'eq', label: '=' },
		{ op: 'ne', label: '≠' },
		{ op: 'empty', label: 'is empty' },
		{ op: 'not_empty', label: 'is not empty' }
	],
	choice: [
		{ op: 'eq', label: '=' },
		{ op: 'ne', label: '≠' }
	],
	person: [
		{ op: 'eq', label: 'is' },
		{ op: 'ne', label: 'is not' },
		{ op: 'empty', label: 'nobody' },
		{ op: 'not_empty', label: 'somebody' }
	],
	tag: [
		{ op: 'eq', label: 'has' },
		{ op: 'ne', label: "doesn't have" },
		{ op: 'empty', label: 'no tags' }
	],
	id: [
		{ op: 'eq', label: '=' },
		{ op: 'ne', label: '≠' },
		{ op: 'empty', label: 'is empty' }
	],
	number: [
		{ op: 'eq', label: '=' },
		{ op: 'ne', label: '≠' },
		{ op: 'gt', label: '>' },
		{ op: 'gte', label: '≥' },
		{ op: 'lt', label: '<' },
		{ op: 'lte', label: '≤' },
		{ op: 'empty', label: 'is empty' },
		{ op: 'not_empty', label: 'is not empty' }
	],
	date: [
		{ op: 'eq', label: 'on' },
		{ op: 'lt', label: 'before' },
		{ op: 'lte', label: 'on or before' },
		{ op: 'gt', label: 'after' },
		{ op: 'gte', label: 'on or after' },
		{ op: 'empty', label: 'is empty' },
		{ op: 'not_empty', label: 'is set' }
	]
};
