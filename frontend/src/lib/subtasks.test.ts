import { describe, expect, it } from 'vitest';
import { countsTowardTotals, nestTasks, parentOptions, subtaskSummary } from './subtasks';

type T = { id: string; parentId?: string | null; status: string; manDays: number; title: string };
const t = (id: string, parentId: string | null = null, status = 'todo', manDays = 0): T => ({
	id,
	parentId,
	status,
	manDays,
	title: id
});

describe('nestTasks', () => {
	it('puts subtasks right after their parent', () => {
		const all = [t('a'), t('b'), t('a1', 'a'), t('b1', 'b'), t('a2', 'a')];
		expect(nestTasks(all, all).map((r) => [r.task.id, r.depth])).toEqual([
			['a', 0],
			['a1', 1],
			['a2', 1],
			['b', 0],
			['b1', 1]
		]);
	});
	it('shows a subtask whose parent is not listed at the top level, with its parent', () => {
		const all = [t('a'), t('a1', 'a')];
		const rows = nestTasks([all[1]], all);
		expect(rows.map((r) => [r.task.id, r.depth, r.parent?.id])).toEqual([['a1', 0, 'a']]);
	});
});

describe('subtaskSummary', () => {
	it('counts done and man-days of non-cancelled subtasks', () => {
		const all = [t('p'), t('c1', 'p', 'done', 2), t('c2', 'p', 'todo', 3), t('c3', 'p', 'cancelled', 9)];
		expect(subtaskSummary('p', all)).toEqual({ count: 2, done: 1, manDays: 5 });
		expect(subtaskSummary('c1', all)).toEqual({ count: 0, done: 0, manDays: 0 });
	});
});

describe('countsTowardTotals', () => {
	it('skips a task whose live subtasks carry the work', () => {
		const all = [t('p', null, 'todo', 10), t('c', 'p', 'todo', 4), t('q', null, 'todo', 1), t('x', 'q', 'cancelled', 1)];
		expect(countsTowardTotals(all[0], all)).toBe(false);
		expect(countsTowardTotals(all[1], all)).toBe(true);
		expect(countsTowardTotals(all[2], all)).toBe(true);
	});
});

describe('parentOptions', () => {
	it('offers top-level tasks other than the task itself', () => {
		const all = [t('a'), t('b'), t('a1', 'a')];
		expect(parentOptions(all, 'b').map((x) => x.id)).toEqual(['a']);
		expect(parentOptions(all, null).map((x) => x.id)).toEqual(['a', 'b']);
	});
	it('offers nothing for a task that has subtasks', () => {
		const all = [t('a'), t('b'), t('a1', 'a')];
		expect(parentOptions(all, 'a')).toEqual([]);
	});
});
