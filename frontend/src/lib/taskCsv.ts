// Work items to and from CSV (export from lists, import into a project).

import { toCsv } from './csv';

export type Lookups = {
	people: { id: string; name: string; email: string }[];
	tasks: { id: string; number: number; title: string }[];
	milestones: { id: string; name: string }[];
	sprints: { id: string; name: string }[];
	areas: { id: string; path: string }[];
};

const TYPES = ['epic', 'feature', 'story', 'task', 'bug'];
const STATUSES = ['backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'];
const PRIORITIES = ['low', 'medium', 'high', 'urgent'];

const COLUMNS: Record<string, string[]> = {
	title: ['title', 'name of item'],
	type: ['type', 'work item type'],
	status: ['status', 'state'],
	priority: ['priority'],
	assignees: ['assignees', 'assignee', 'assigned to'],
	manDays: ['man-days', 'man days', 'mandays', 'size'],
	storyPoints: ['story points', 'points'],
	start: ['start', 'start date'],
	due: ['due', 'due date'],
	tags: ['tags', 'tag'],
	parent: ['parent'],
	milestone: ['milestone'],
	sprint: ['sprint', 'iteration'],
	area: ['area', 'area path'],
	description: ['description']
};

const same = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase();
const list = (v: string) =>
	v
		.split(/[;,]/)
		.map((x) => x.trim())
		.filter(Boolean);

export type ImportRow = { line: number; body: Record<string, unknown> & { title: string; projectId: string } };

/**
 * Turns CSV rows (first row = headers) into task bodies for `projectId`.
 * Each line with a problem is reported (first problem only) and left out.
 */
export function rowsToTasks(
	rows: string[][],
	projectId: string,
	lk: Lookups
): { ok: ImportRow[]; errors: { line: number; message: string }[] } {
	const ok: ImportRow[] = [];
	const errors: { line: number; message: string }[] = [];
	const header = (rows[0] ?? []).map((h) => h.trim().toLowerCase());
	const col: Record<string, number> = {};
	for (const [key, names] of Object.entries(COLUMNS)) {
		const i = header.findIndex((h) => names.includes(h));
		if (i >= 0) col[key] = i;
	}
	if (col.title === undefined) return { ok, errors: [{ line: 1, message: 'No "Title" column' }] };

	rows.slice(1).forEach((r, idx) => {
		const line = idx + 2;
		const get = (k: string) => (col[k] === undefined ? '' : (r[col[k]] ?? '').trim());
		try {
			const body: ImportRow['body'] = { projectId, title: get('title') };
			if (!body.title) throw new Error('Title is empty');
			const enumOf = (k: string, allowed: string[], what: string) => {
				const v = get(k).toLowerCase().replace(/[\s-]+/g, '_');
				if (!v) return;
				if (!allowed.includes(v)) throw new Error(`Unknown ${what} "${get(k)}"`);
				body[k] = v;
			};
			enumOf('type', TYPES, 'type');
			enumOf('status', STATUSES, 'status');
			enumOf('priority', PRIORITIES, 'priority');
			const people = list(get('assignees'));
			if (people.length) {
				body.assigneeIds = people.map((p) => {
					const m = lk.people.find((x) => same(x.email, p) || same(x.name, p));
					if (!m) throw new Error(`Unknown person "${p}"`);
					return m.id;
				});
			}
			const num = (k: string, field: string) => {
				const v = get(k);
				if (!v) return;
				const n = Number(v.replace(',', '.'));
				if (!Number.isFinite(n) || n < 0) throw new Error(`"${v}" is not a number`);
				body[field] = n;
			};
			num('manDays', 'estimateManDays');
			num('storyPoints', 'storyPoints');
			const date = (k: string, field: string) => {
				const v = get(k);
				if (!v) return;
				if (!/^\d{4}-\d{2}-\d{2}$/.test(v) || Number.isNaN(Date.parse(v))) throw new Error(`Dates must look like 2026-12-31, not "${v}"`);
				body[field] = v;
			};
			date('start', 'startDate');
			date('due', 'dueDate');
			const tags = list(get('tags'));
			if (tags.length) body.tags = tags;
			const parent = get('parent');
			if (parent) {
				const m = /^#\d+$/.test(parent)
					? lk.tasks.find((t) => t.number === Number(parent.slice(1)))
					: lk.tasks.find((t) => same(t.title, parent));
				if (!m) throw new Error(`No existing item "${parent}" to put this under`);
				body.parentId = m.id;
			}
			const named = (k: string, items: { id: string; name: string }[], field: string, what: string) => {
				const v = get(k);
				if (!v) return;
				const m = items.find((x) => same(x.name, v));
				if (!m) throw new Error(`Unknown ${what} "${v}"`);
				body[field] = m.id;
			};
			named('milestone', lk.milestones, 'milestoneId', 'milestone');
			named('sprint', lk.sprints, 'sprintId', 'sprint');
			named('area', lk.areas.map((a) => ({ id: a.id, name: a.path })), 'areaId', 'area');
			const description = get('description');
			if (description) body.description = description;
			ok.push({ line, body });
		} catch (e) {
			errors.push({ line, message: (e as Error).message });
		}
	});
	return { ok, errors };
}

type ExportTask = {
	number: number;
	title: string;
	type: string;
	status: string;
	priority: string;
	assigneeIds: string[];
	manDays: number;
	storyPoints?: number | null;
	startDate?: string | null;
	dueDate?: string | null;
	tags: string[];
	parentId?: string | null;
	milestoneId?: string | null;
	sprintId?: string | null;
	areaId?: string | null;
	description?: string | null;
	projectId: string;
};

export const EXPORT_HEADER = ['Project', '#', 'Title', 'Type', 'Status', 'Priority', 'Assignees', 'Man-days', 'Story points', 'Start', 'Due', 'Tags', 'Parent', 'Milestone', 'Sprint', 'Area', 'Description'];

export function tasksToCsv(tasks: readonly ExportTask[], lk: Lookups & { project: (id: string) => string }): string {
	const name = <T extends { id: string }>(items: T[], id: string | null | undefined, pick: (x: T) => string) =>
		id ? (items.find((x) => x.id === id) ? pick(items.find((x) => x.id === id)!) : '') : '';
	return toCsv([
		EXPORT_HEADER,
		...tasks.map((t) => [
			lk.project(t.projectId),
			t.number,
			t.title,
			t.type,
			t.status.replace('_', ' '),
			t.priority,
			t.assigneeIds.map((id) => lk.people.find((p) => p.id === id)?.name ?? '').filter(Boolean).join('; '),
			t.manDays || '',
			t.storyPoints ?? '',
			t.startDate ?? '',
			t.dueDate ?? '',
			t.tags.join('; '),
			name(lk.tasks, t.parentId, (x) => `#${x.number}`),
			name(lk.milestones, t.milestoneId, (x) => x.name),
			name(lk.sprints, t.sprintId, (x) => x.name),
			name(lk.areas, t.areaId, (x) => x.path),
			t.description ?? ''
		])
	]);
}
