import { describe, expect, it } from 'vitest';
import { afterIdToPlaceBefore, boardColumns, boardStates, cardRuleColor, cardShows, splitColumn } from './board';

type T = { id: string; number: number; rank: number; state: string; priority: 'low' | 'medium' | 'high' | 'urgent' };
const t = (number: number, status: string, priority: T['priority'] = 'medium', rank = number): T => ({
	id: `t${number}`,
	number,
	rank,
	state: status,
	priority
});

describe('boardStates', () => {
	const wf = [
		{ key: 'new', label: 'New', category: 'backlog' },
		{ key: 'doing', label: 'Doing', category: 'in_progress' },
		{ key: 'qa', label: 'QA', category: 'in_review' },
		{ key: 'shipped', label: 'Shipped', category: 'done' },
		{ key: 'dropped', label: 'Dropped', category: 'cancelled' }
	];
	it('shows every state but cancelled ones by default, in workflow order', () => {
		expect(boardStates(wf, []).map((s) => s.key)).toEqual(['new', 'doing', 'qa', 'shipped']);
	});
	it('shows the states of the filtered categories', () => {
		expect(boardStates(wf, ['in_review', 'cancelled']).map((s) => s.key)).toEqual(['qa', 'dropped']);
	});
});

describe('boardColumns', () => {
	it('puts each task in its status column', () => {
		const cols = boardColumns([t(1, 'todo'), t(2, 'done'), t(3, 'todo')], ['todo', 'in_progress', 'done']);
		expect(cols.map((c) => [c.state, c.tasks.map((x) => x.number)])).toEqual([
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

import { boardLanes, wipState } from './board';

describe('wipState', () => {
	it('says when a column is at or over its limit', () => {
		expect(wipState(2, undefined)).toBe('none');
		expect(wipState(2, 3)).toBe('ok');
		expect(wipState(3, 3)).toBe('full');
		expect(wipState(4, 3)).toBe('over');
	});
});

describe('boardLanes', () => {
	type L = { id: string; assigneeIds: string[]; priority: string; type: string; parentId?: string | null };
	const items: L[] = [
		{ id: '1', assigneeIds: ['b'], priority: 'low', type: 'bug' },
		{ id: '2', assigneeIds: [], priority: 'urgent', type: 'task', parentId: 'p' },
		{ id: '3', assigneeIds: ['a', 'b'], priority: 'low', type: 'task' }
	];
	const names = { person: (id: string) => ({ a: 'Ada', b: 'Bo' })[id] ?? '?', task: () => 'Parent P' };
	it('is one lane when off', () => {
		expect(boardLanes(items, 'none', names).map((l) => l.tasks.length)).toEqual([3]);
	});
	it('groups by first assignee by name, unassigned last', () => {
		expect(boardLanes(items, 'assignee', names).map((l) => [l.label, l.tasks.map((t) => t.id)])).toEqual([
			['Ada', ['3']],
			['Bo', ['1']],
			['Unassigned', ['2']]
		]);
	});
	it('orders priority and type lanes from the top', () => {
		expect(boardLanes(items, 'priority', names).map((l) => l.label)).toEqual(['Urgent', 'Low']);
		expect(boardLanes(items, 'type', names).map((l) => l.label)).toEqual(['Bug', 'Task']);
		expect(boardLanes(items, 'parent', names).map((l) => l.label)).toEqual(['Parent P', 'No parent']);
	});
});

describe('cardRuleColor', () => {
	const base = { priority: 'medium', type: 'task', customType: null, tags: ['ui'], dueDate: null, status: 'todo', blockedByIds: [], assigneeIds: ['p1'] };
	it('picks the first matching rule', () => {
		const rules = [
			{ field: 'tag', value: 'UI', color: 'sky' },
			{ field: 'priority', value: 'medium', color: 'red' }
		] as const;
		expect(cardRuleColor(base, [...rules], '2026-11-23')).toBe('sky');
	});
	it('matches overdue and blocked only when true', () => {
		expect(cardRuleColor({ ...base, dueDate: '2026-11-22' }, [{ field: 'overdue', value: 'true', color: 'red' }], '2026-11-23')).toBe('red');
		expect(cardRuleColor({ ...base, dueDate: '2026-11-22', status: 'done' }, [{ field: 'overdue', value: 'true', color: 'red' }], '2026-11-23')).toBeNull();
		expect(cardRuleColor(base, [{ field: 'blocked', value: 'true', color: 'amber' }], '2026-11-23')).toBeNull();
		expect(cardRuleColor({ ...base, blockedByIds: ['x'] }, [{ field: 'blocked', value: 'true', color: 'amber' }], '2026-11-23')).toBe('amber');
	});
	it('matches a project type by its key and an assignee by id', () => {
		expect(cardRuleColor({ ...base, customType: 'spike' }, [{ field: 'type', value: 'spike', color: 'pink' }], '2026-11-23')).toBe('pink');
		expect(cardRuleColor(base, [{ field: 'assignee', value: 'p1', color: 'violet' }], '2026-11-23')).toBe('violet');
	});
});

describe('cardShows', () => {
	it('uses the defaults when nothing is set', () => {
		expect(cardShows(undefined, 'assignees')).toBe(true);
		expect(cardShows(undefined, 'remaining')).toBe(false);
	});
	it('follows the saved list', () => {
		expect(cardShows(['due'], 'assignees')).toBe(false);
		expect(cardShows(['due'], 'due')).toBe(true);
	});
});

describe('splitColumn', () => {
	it('splits a column by boardDone', () => {
		const { doing, done } = splitColumn([
			{ id: 'a', boardDone: false },
			{ id: 'b', boardDone: true }
		]);
		expect(doing.map((t) => t.id)).toEqual(['a']);
		expect(done.map((t) => t.id)).toEqual(['b']);
	});
});
