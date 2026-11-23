import { describe, expect, it } from 'vitest';
import { boardColumns, boardStatuses, DEFAULT_BOARD_STATUSES } from './board';

type T = { id: string; number: number; status: string; priority: 'low' | 'medium' | 'high' | 'urgent' };
const t = (number: number, status: string, priority: T['priority'] = 'medium'): T => ({
	id: `t${number}`,
	number,
	status,
	priority
});

describe('boardStatuses', () => {
	it('shows every status but cancelled by default', () => {
		expect(boardStatuses([])).toEqual(DEFAULT_BOARD_STATUSES);
		expect(DEFAULT_BOARD_STATUSES).not.toContain('cancelled');
	});
	it('shows only the filtered statuses, in board order', () => {
		expect(boardStatuses(['done', 'cancelled', 'todo'])).toEqual(['todo', 'done', 'cancelled']);
	});
});

describe('boardColumns', () => {
	it('puts each task in its status column', () => {
		const cols = boardColumns([t(1, 'todo'), t(2, 'done'), t(3, 'todo')], ['todo', 'in_progress', 'done']);
		expect(cols.map((c) => [c.status, c.tasks.map((x) => x.number)])).toEqual([
			['todo', [1, 3]],
			['in_progress', []],
			['done', [2]]
		]);
	});
	it('leaves out tasks whose status has no column', () => {
		const cols = boardColumns([t(1, 'cancelled')], ['todo']);
		expect(cols[0].tasks).toEqual([]);
	});
	it('sorts by priority, then task number', () => {
		const cols = boardColumns(
			[t(1, 'todo', 'low'), t(2, 'todo', 'urgent'), t(3, 'todo', 'medium'), t(4, 'todo', 'urgent'), t(5, 'todo', 'high')],
			['todo']
		);
		expect(cols[0].tasks.map((x) => x.number)).toEqual([2, 4, 5, 3, 1]);
	});
});
