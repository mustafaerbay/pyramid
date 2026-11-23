// Work items to and from CSV (export from lists, import into a project).

import { toCsv } from './csv';

export type Lookups = {
	people: { id: string; name: string; email: string }[];
	tasks: { id: string; number: number; title: string }[];
	milestones: { id: string; name: string }[];
	sprints: { id: string; name: string }[];
	areas: { id: string; path: string }[];
	/** The project's custom fields; columns named like their labels fill them. */
	customFields?: { key: string; label: string; type: 'text' | 'number' | 'date' | 'select'; required: boolean; options?: string[] }[];
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

export type ImportRow = {
	line: number;
	body: Record<string, unknown> & { title: string; projectId: string };
	/** Parent is another row of the same file (created first). */
	parentLine?: number;
};

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
	const customCols = (lk.customFields ?? [])
		.map((f) => ({ f, i: header.findIndex((h) => h === f.label.trim().toLowerCase() || h === f.key) }))
		.filter((c) => c.i >= 0);

	// Titles in the file, so a Parent can name another row.
	const fileTitles = new Map<string, number>();
	rows.slice(1).forEach((r, idx) => {
		const t = col.title === undefined ? '' : (r[col.title] ?? '').trim().toLowerCase();
		if (t && !fileTitles.has(t)) fileTitles.set(t, idx + 2);
	});
	const parentLines = new Map<number, number>();

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
				const inFile = /^#\d+$/.test(parent) ? undefined : fileTitles.get(parent.toLowerCase());
				if (inFile !== undefined && inFile !== line) parentLines.set(line, inFile);
				else {
					const m = /^#\d+$/.test(parent)
						? lk.tasks.find((t) => t.number === Number(parent.slice(1)))
						: lk.tasks.find((t) => same(t.title, parent));
					if (!m) throw new Error(`No item "${parent}" in the file or the project to put this under`);
					body.parentId = m.id;
				}
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
			const values: Record<string, unknown> = {};
			for (const { f, i } of customCols) {
				const v = (r[i] ?? '').trim();
				if (!v) continue;
				if (f.type === 'number') {
					const n = Number(v.replace(',', '.'));
					if (!Number.isFinite(n)) throw new Error(`${f.label} must be a number`);
					values[f.key] = n;
				} else if (f.type === 'select') {
					const o = (f.options ?? []).find((x) => same(x, v));
					if (!o) throw new Error(`${f.label} must be one of: ${(f.options ?? []).join(', ')}`);
					values[f.key] = o;
				} else if (f.type === 'date') {
					if (!/^\d{4}-\d{2}-\d{2}$/.test(v)) throw new Error(`${f.label} must look like 2026-12-31`);
					values[f.key] = v;
				} else values[f.key] = v;
			}
			if (Object.keys(values).length) body.customValues = values;
			ok.push({ line, body, ...(parentLines.has(line) ? { parentLine: parentLines.get(line) } : {}) });
		} catch (e) {
			errors.push({ line, message: (e as Error).message });
		}
	});

	// Drop rows whose parent row has a problem or that form a loop, until
	// nothing changes.
	let changed = true;
	while (changed) {
		changed = false;
		const good = new Set(ok.map((r) => r.line));
		for (const r of [...ok]) {
			if (r.parentLine === undefined) continue;
			let bad = !good.has(r.parentLine) ? `Its parent row (line ${r.parentLine}) has a problem` : '';
			const seen = new Set([r.line]);
			for (let p: number | undefined = r.parentLine; !bad && p !== undefined; p = parentLines.get(p)) {
				if (seen.has(p)) bad = 'Parent rows point at each other in a loop';
				seen.add(p);
			}
			if (bad) {
				ok.splice(ok.indexOf(r), 1);
				errors.push({ line: r.line, message: bad });
				changed = true;
			}
		}
	}
	errors.sort((a, b) => a.line - b.line);
	return { ok, errors };
}

/** Rows in creation order: parents before the rows under them. */
export function importWaves(rows: readonly ImportRow[]): ImportRow[][] {
	const waves: ImportRow[][] = [];
	const placed = new Set<number>();
	let left = [...rows];
	while (left.length) {
		const wave = left.filter((r) => r.parentLine === undefined || placed.has(r.parentLine));
		if (wave.length === 0) break;
		wave.forEach((r) => placed.add(r.line));
		waves.push(wave);
		left = left.filter((r) => !placed.has(r.line));
	}
	return waves;
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
	customValues?: Record<string, unknown>;
};

export const EXPORT_HEADER = ['Project', '#', 'Title', 'Type', 'Status', 'Priority', 'Assignees', 'Man-days', 'Story points', 'Start', 'Due', 'Tags', 'Parent', 'Milestone', 'Sprint', 'Area', 'Description'];

export function tasksToCsv(tasks: readonly ExportTask[], lk: Lookups & { project: (id: string) => string }): string {
	const name = <T extends { id: string }>(items: T[], id: string | null | undefined, pick: (x: T) => string) =>
		id ? (items.find((x) => x.id === id) ? pick(items.find((x) => x.id === id)!) : '') : '';
	const extra = lk.customFields ?? [];
	return toCsv([
		[...EXPORT_HEADER, ...extra.map((f) => f.label)],
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
			t.description ?? '',
			...extra.map((f) => {
				const v = t.customValues?.[f.key];
				return v == null ? '' : String(v);
			})
		])
	]);
}
