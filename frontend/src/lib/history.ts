// Turns a task history change ({from, to} for one field) into a sentence.

import { formatManDays } from './capacity';

export type Names = {
	person: (id: string) => string;
	milestone: (id: string) => string;
	task: (id: string) => string;
	sprint?: (id: string) => string;
	area?: (id: string) => string;
};

type Change = { from?: unknown; to?: unknown };

const LABELS: Record<string, string> = {
	title: 'Title',
	type: 'Type',
	description: 'Description',
	status: 'Status',
	priority: 'Priority',
	milestoneId: 'Milestone',
	parentId: 'Parent task',
	startDate: 'Start date',
	dueDate: 'Due date',
	estimateManDays: 'Size',
	storyPoints: 'Story points',
	remainingManDays: 'Remaining',
	completedManDays: 'Completed',
	assignments: 'Assignees',
	tags: 'Tags',
	blockedByIds: 'Waiting on',
	relatedIds: 'Related',
	duplicateOfId: 'Duplicate of',
	sprintId: 'Sprint',
	areaId: 'Area'
};

const list = (v: unknown): string[] => (Array.isArray(v) ? v.map(String) : []);
const none = (v: unknown, show: (x: string) => string) => (v == null || v === '' ? 'none' : show(String(v)));

function addedRemoved(from: unknown, to: unknown, show: (x: string) => string): string {
	const a = list(from);
	const b = list(to);
	const lower = (xs: string[]) => xs.map((x) => x.toLowerCase());
	const added = b.filter((x) => !lower(a).includes(x.toLowerCase()));
	const removed = a.filter((x) => !lower(b).includes(x.toLowerCase()));
	const parts: string[] = [];
	if (added.length) parts.push(`added ${added.map(show).join(', ')}`);
	if (removed.length) parts.push(`removed ${removed.map(show).join(', ')}`);
	return parts.join('; ') || 'reordered';
}

function assignments(from: unknown, to: unknown, names: Names): string {
	const a = (from ?? {}) as Record<string, number>;
	const b = (to ?? {}) as Record<string, number>;
	const parts: string[] = [];
	const added = Object.keys(b).filter((id) => !(id in a));
	const removed = Object.keys(a).filter((id) => !(id in b));
	const changed = Object.keys(b).filter((id) => id in a && a[id] !== b[id]);
	if (added.length) parts.push(`added ${added.map(names.person).join(', ')}`);
	if (removed.length) parts.push(`removed ${removed.map(names.person).join(', ')}`);
	for (const id of changed) parts.push(`${names.person(id)} ${formatManDays(a[id])} → ${formatManDays(b[id])} md`);
	return parts.join('; ');
}

export function describeChange(field: string, change: Change, names: Names): string {
	const label = LABELS[field] ?? field;
	const { from, to } = change;
	switch (field) {
		case 'description':
			return 'Description edited';
		case 'title':
			return `${label}: “${from ?? ''}” → “${to ?? ''}”`;
		case 'type':
		case 'status':
		case 'priority':
			return `${label}: ${none(from, (x) => x.replace('_', ' '))} → ${none(to, (x) => x.replace('_', ' '))}`;
		case 'milestoneId':
			return `${label}: ${none(from, names.milestone)} → ${none(to, names.milestone)}`;
		case 'sprintId': {
			const n = names.sprint ?? (() => 'a sprint');
			return `${label}: ${none(from, n)} → ${none(to, n)}`;
		}
		case 'areaId': {
			const n = names.area ?? (() => 'an area');
			return `${label}: ${none(from, n)} → ${none(to, n)}`;
		}
		case 'parentId':
		case 'duplicateOfId':
			return `${label}: ${none(from, names.task)} → ${none(to, names.task)}`;
		case 'storyPoints':
			return `${label}: ${none(from, (x) => formatManDays(Number(x)))} → ${none(to, (x) => formatManDays(Number(x)))}`;
		case 'remainingManDays':
		case 'completedManDays':
		case 'estimateManDays':
			return `${label}: ${none(from, (x) => formatManDays(Number(x)))} → ${none(to, (x) => formatManDays(Number(x)))} md`;
		case 'tags':
			return `${label}: ${addedRemoved(from, to, (x) => x)}`;
		case 'blockedByIds':
		case 'relatedIds':
			return `${label}: ${addedRemoved(from, to, names.task)}`;
		case 'assignments':
			return `${label}: ${assignments(from, to, names)}`;
		default:
			return `${label}: ${none(from, (x) => x)} → ${none(to, (x) => x)}`;
	}
}
