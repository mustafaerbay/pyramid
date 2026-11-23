import { describe, expect, it } from 'vitest';
import { parseCsv, toCsv } from './csv';
import { rowsToTasks, tasksToCsv, type Lookups } from './taskCsv';
import { runLimited } from './bulk';

describe('csv', () => {
	it('round-trips quotes, commas and newlines', () => {
		const rows = [['Title', 'Description'], ['A, "quoted"', 'line 1\nline 2'], ['plain', '']];
		expect(parseCsv(toCsv(rows))).toEqual(rows);
	});
	it('reads CRLF, a BOM, semicolon-free text and skips blank lines', () => {
		expect(parseCsv('﻿a,b\r\n1,2\r\n\r\n3,"4"\r\n')).toEqual([['a', 'b'], ['1', '2'], ['3', '4']]);
	});
	it('guards against spreadsheet formulas on export', () => {
		expect(toCsv([['=HYPERLINK("x")', '+1', '-2', '@a', 'ok']])).toBe(`"'=HYPERLINK(""x"")",'+1,'-2,'@a,ok\r\n`);
	});
});

const lookups: Lookups = {
	people: [
		{ id: 'p1', name: 'Ada Kim', email: 'ada@x.dev' },
		{ id: 'p2', name: 'Bo Li', email: 'bo@x.dev' }
	],
	tasks: [{ id: 't12', number: 12, title: 'Checkout' }],
	milestones: [{ id: 'm1', name: 'M1' }],
	sprints: [{ id: 's1', name: 'Sprint 1' }],
	areas: [{ id: 'a1', path: 'Web/Checkout' }]
};

describe('rowsToTasks', () => {
	it('maps columns by header name, ignoring case', () => {
		const { ok, errors } = rowsToTasks(
			[
				['title', 'TYPE', 'Status', 'Priority', 'Assignees', 'Man-days', 'Story points', 'Start', 'Due', 'Tags', 'Parent', 'Milestone', 'Sprint', 'Area', 'Description'],
				['Pay button', 'story', 'in progress', 'High', 'ada@x.dev; Bo Li', '3', '5', '2026-12-01', '2026-12-05', 'ui; web', '#12', 'M1', 'Sprint 1', 'web/checkout', 'Hi']
			],
			'proj',
			lookups
		);
		expect(errors).toEqual([]);
		expect(ok[0].line).toBe(2);
		expect(ok[0].body).toEqual({
			projectId: 'proj',
			title: 'Pay button',
			type: 'story',
			status: 'in_progress',
			priority: 'high',
			assigneeIds: ['p1', 'p2'],
			estimateManDays: 3,
			storyPoints: 5,
			startDate: '2026-12-01',
			dueDate: '2026-12-05',
			tags: ['ui', 'web'],
			parentId: 't12',
			milestoneId: 'm1',
			sprintId: 's1',
			areaId: 'a1',
			description: 'Hi'
		});
	});
	it('reports problems per line and needs a Title column', () => {
		expect(rowsToTasks([['Name'], ['x']], 'p', lookups).errors).toEqual([{ line: 1, message: 'No "Title" column' }]);
		const { ok, errors } = rowsToTasks(
			[
				['Title', 'Type', 'Assignees', 'Due', 'Man-days', 'Parent'],
				['', 'task', '', '', '', ''],
				['A', 'saga', '', '', '', ''],
				['B', '', 'nobody@x.dev', '', '', ''],
				['C', '', '', '31/12/2026', '', ''],
				['D', '', '', '', 'lots', ''],
				['E', '', '', '', '', 'Nope'],
				['F', '', '', '', '', 'checkout']
			],
			'p',
			lookups
		);
		expect(errors.map((e) => e.line)).toEqual([2, 3, 4, 5, 6, 7]);
		expect(ok.map((o) => o.body.title)).toEqual(['F']);
		expect(ok[0].body.parentId).toBe('t12');
	});
});

describe('rowsToTasks with custom fields', () => {
	it('reads columns named like the fields, by label', () => {
		const lk = {
			...lookups,
			customFields: [
				{ key: 'customer', label: 'Customer', type: 'text' as const, required: true },
				{ key: 'risk', label: 'Risk', type: 'select' as const, required: false, options: ['Low', 'High'] },
				{ key: 'budget', label: 'Budget', type: 'number' as const, required: false }
			]
		};
		const { ok, errors } = rowsToTasks([['Title', 'customer', 'Risk', 'Budget'], ['A', 'ACME', 'high', '1200,5'], ['B', 'X', 'Huge', '']], 'p', lk);
		expect(ok[0].body.customValues).toEqual({ customer: 'ACME', risk: 'High', budget: 1200.5 });
		expect(errors).toEqual([{ line: 3, message: 'Risk must be one of: Low, High' }]);
	});
});

describe('tasksToCsv', () => {
	it('writes readable columns', () => {
		const csv = tasksToCsv(
			[
				{
					number: 3, title: 'T', type: 'bug', status: 'in_progress', priority: 'high', assigneeIds: ['p1'],
					manDays: 2, storyPoints: null, startDate: null, dueDate: '2026-12-01', tags: ['a', 'b'],
					parentId: 't12', milestoneId: null, sprintId: 's1', areaId: 'a1', description: null, projectId: 'x'
				}
			],
			{ ...lookups, project: () => 'Proj' }
		);
		const [head, row] = parseCsv(csv);
		expect(head.slice(0, 4)).toEqual(['Project', '#', 'Title', 'Type']);
		expect(row).toEqual(['Proj', '3', 'T', 'bug', 'in progress', 'high', 'Ada Kim', '2', '', '', '2026-12-01', 'a; b', '#12', '', 'Sprint 1', 'Web/Checkout', '']);
	});
});

describe('runLimited', () => {
	it('runs everything with at most n at once and keeps results in order', async () => {
		let running = 0;
		let peak = 0;
		const out = await runLimited([1, 2, 3, 4, 5, 6, 7], 3, async (x) => {
			running++;
			peak = Math.max(peak, running);
			await new Promise((r) => setTimeout(r, 5));
			running--;
			if (x === 4) throw new Error('nope');
			return x * 2;
		});
		expect(peak).toBeLessThanOrEqual(3);
		expect(out.map((r) => (r.ok ? r.value : 'err'))).toEqual([2, 4, 6, 'err', 10, 12, 14]);
	});
});

describe('tasksToCsv with custom fields', () => {
	it('adds a column per field', () => {
		const [head, row] = parseCsv(
			tasksToCsv(
				[{ number: 1, title: 'T', type: 'task', status: 'todo', priority: 'low', assigneeIds: [], manDays: 0, tags: [], projectId: 'x', customValues: { risk: 'High' } }],
				{ ...lookups, project: () => 'P', customFields: [{ key: 'risk', label: 'Risk', type: 'select', required: false, options: ['High'] }] }
			)
		);
		expect(head.at(-1)).toBe('Risk');
		expect(row.at(-1)).toBe('High');
	});
});
