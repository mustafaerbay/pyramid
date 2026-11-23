import { describe, expect, it } from 'vitest';
import { cleanFilters, groupResults } from './queries';

const tasks = [
	{ status: 'todo', type: 'bug', priority: 'high', projectId: 'p1', assigneeIds: ['a', 'b'] },
	{ status: 'todo', type: 'task', priority: 'low', projectId: 'p1', assigneeIds: [] },
	{ status: 'done', type: 'bug', priority: 'high', projectId: 'p2', assigneeIds: ['a'] }
];
const names = { person: (id: string) => ({ a: 'Ada', b: 'Bo' })[id] ?? '?', project: (id: string) => ({ p1: 'One', p2: 'Two' })[id] ?? '?' };

describe('groupResults', () => {
	it('counts by status, biggest first', () => {
		expect(groupResults(tasks, 'status', names)).toEqual([
			{ key: 'todo', label: 'todo', count: 2 },
			{ key: 'done', label: 'done', count: 1 }
		]);
	});
	it('counts each assignee, and unassigned', () => {
		expect(groupResults(tasks, 'assignee', names).map((g) => [g.label, g.count])).toEqual([
			['Ada', 2],
			['Bo', 1],
			['Unassigned', 1]
		]);
	});
	it('names projects', () => {
		expect(groupResults(tasks, 'project', names).map((g) => g.label)).toEqual(['One', 'Two']);
	});
});

describe('cleanFilters', () => {
	it('drops empty lists and blank text', () => {
		expect(cleanFilters({ types: [], statuses: ['todo'], text: '  ', tags: ['x'], unassigned: false })).toEqual({
			statuses: ['todo'],
			tags: ['x']
		});
	});
});
