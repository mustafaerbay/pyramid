import { describe, expect, it } from 'vitest';
import { SERIES, seriesColor, stackGroups, topWithOther, trendSeries } from './dashcharts';

const names = { person: (id: string) => ({ a: 'Ann', b: 'Bo' })[id] ?? id, project: (id: string) => id };
const row = (status: string, priority: string, assigneeIds: string[] = []) => ({ status, type: 'task', priority, projectId: 'p', assigneeIds });

describe('stackGroups', () => {
	it('counts each group split by the stack field', () => {
		const s = stackGroups([row('todo', 'high'), row('todo', 'low'), row('done', 'high')], 'status', 'priority', names);
		expect(s.groups.map((g) => g.label)).toEqual(['todo', 'done']);
		expect(s.series.map((x) => x.key)).toEqual(['high', 'low']);
		expect(s.counts.todo).toEqual({ high: 1, low: 1 });
		expect(s.counts.done).toEqual({ high: 1 });
	});
	it('splits people into one series each', () => {
		const s = stackGroups([row('todo', 'high', ['a', 'b']), row('todo', 'high')], 'status', 'assignee', names);
		expect(s.series.map((x) => x.label)).toEqual(['Ann', 'Bo', 'Unassigned']);
	});
});

describe('topWithOther', () => {
	it('folds everything after the first n into Other', () => {
		const d = topWithOther(
			[
				{ key: 'a', label: 'A', count: 5 },
				{ key: 'b', label: 'B', count: 3 },
				{ key: 'c', label: 'C', count: 1 },
				{ key: 'd', label: 'D', count: 1 }
			],
			2
		);
		expect(d).toEqual([
			{ key: 'a', label: 'A', count: 5 },
			{ key: 'b', label: 'B', count: 3 },
			{ key: '__other', label: 'Other', count: 2 }
		]);
	});
});

describe('trendSeries', () => {
	it('counts open and finished items at the end of each day', () => {
		const rows = [
			{ createdAt: '2026-11-01T10:00:00Z', completedAt: '2026-11-03T09:00:00Z' },
			{ createdAt: '2026-11-02T10:00:00Z', completedAt: null },
			{ createdAt: '2026-11-05T10:00:00Z', completedAt: null }
		];
		expect(trendSeries(rows, '2026-11-01', '2026-11-04')).toEqual([
			{ day: '2026-11-01', open: 1, done: 0 },
			{ day: '2026-11-02', open: 2, done: 0 },
			{ day: '2026-11-03', open: 1, done: 1 },
			{ day: '2026-11-04', open: 1, done: 1 }
		]);
	});
});

describe('seriesColor', () => {
	it('uses the categorical slots in order, never cycling', () => {
		expect(seriesColor(0)).toBe(SERIES[0]);
		expect(seriesColor(7)).toBe(SERIES[7]);
		expect(seriesColor(8)).toBe('#94a3b8');
	});
});
