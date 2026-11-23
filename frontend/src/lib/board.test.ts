import { describe, expect, it } from 'vitest';
import { afterIdToPlaceBefore, boardColumns, boardStatuses, DEFAULT_BOARD_STATUSES } from './board';

type T = { id: string; number: number; rank: number; status: string; priority: 'low' | 'medium' | 'high' | 'urgent' };
const t = (number: number, status: string, priority: T['priority'] = 'medium', rank = number): T => ({
	id: `t${number}`,
	number,
	rank,
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
	it('follows the manual order (rank), then task number', () => {
		const cols = boardColumns([t(1, 'todo', 'urgent', 30), t(2, 'todo', 'low', 10), t(3, 'todo', 'high', 20), t(4, 'todo', 'low', 10)], ['todo']);
		expect(cols[0].tasks.map((x) => x.number)).toEqual([2, 4, 3, 1]);
	});
});

describe('afterIdToPlaceBefore', () => {
	const order = [{ id: 'a' }, { id: 'b' }, { id: 'c' }, { id: 'd' }];
	it('returns the item before the target, skipping the one being moved', () => {
		expect(afterIdToPlaceBefore(order, 'd', 'c')).toBe('b');
		expect(afterIdToPlaceBefore(order, 'b', 'c')).toBe('a');
	});
	it('returns null to go first', () => {
		expect(afterIdToPlaceBefore(order, 'c', 'a')).toBeNull();
		expect(afterIdToPlaceBefore(order, 'a', 'b')).toBeNull();
	});
});
