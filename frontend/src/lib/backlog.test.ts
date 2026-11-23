import { describe, expect, it } from 'vitest';
import { backlogLevel, forecast, rollup, velocityOf } from './backlog';

type T = { id: string; parentId?: string | null; type: string; status: string; storyPoints?: number | null; manDays: number };
const t = (id: string, type: string, parentId: string | null = null, status = 'todo', storyPoints: number | null = null, manDays = 0): T => ({
	id, type, parentId, status, storyPoints, manDays
});

describe('backlogLevel', () => {
	it('puts bugs with stories', () => {
		expect(['epic', 'feature', 'story', 'bug', 'task'].map(backlogLevel)).toEqual(['epics', 'features', 'stories', 'stories', 'tasks']);
	});
});

describe('rollup', () => {
	it('adds up points and man-days below and counts what is done', () => {
		const all = [
			t('f', 'feature'),
			t('s1', 'story', 'f', 'done', 3),
			t('s2', 'story', 'f', 'todo', 5),
			t('k1', 'task', 's2', 'done', null, 2),
			t('k2', 'task', 's2', 'todo', null, 1),
			t('x', 'story', 'f', 'cancelled', 8)
		];
		expect(rollup(all[0], all)).toEqual({ points: 8, donePoints: 3, manDays: 3, doneManDays: 2, pctDone: 38 });
		expect(rollup(all[2], all).manDays).toBe(3);
	});
});

describe('velocityOf', () => {
	it('averages the done points of the last three finished sprints', () => {
		const sprints = [
			{ status: 'completed', startDate: '2026-01-01', stats: { donePoints: 10 } },
			{ status: 'completed', startDate: '2026-01-15', stats: { donePoints: 20 } },
			{ status: 'completed', startDate: '2026-02-01', stats: { donePoints: 30 } },
			{ status: 'completed', startDate: '2026-02-15', stats: { donePoints: 40 } },
			{ status: 'active', startDate: '2026-03-01', stats: { donePoints: 99 } }
		];
		expect(velocityOf(sprints)).toBe(30);
		expect(velocityOf([])).toBe(0);
	});
});

describe('forecast', () => {
	it('fills sprints in backlog order with the velocity, skipping finished items', () => {
		const items = [
			{ id: 'a', status: 'todo', storyPoints: 5 },
			{ id: 'b', status: 'done', storyPoints: 8 },
			{ id: 'c', status: 'todo', storyPoints: 5 },
			{ id: 'd', status: 'in_progress', storyPoints: 3 },
			{ id: 'e', status: 'todo', storyPoints: null },
			{ id: 'f', status: 'todo', storyPoints: 4 }
		];
		const f = forecast(items, 10, ['Sprint 7', 'Sprint 8']);
		expect(f.get('a')).toBe('Sprint 7');
		expect(f.get('c')).toBe('Sprint 7');
		expect(f.get('d')).toBe('Sprint 8');
		expect(f.get('e')).toBe('Sprint 8');
		expect(f.get('f')).toBe('Sprint 8');
		expect(f.has('b')).toBe(false);
		expect(forecast(items, 0, ['S']).size).toBe(0);
	});
	it('names sprints past the planned ones', () => {
		const f = forecast([{ id: 'a', status: 'todo', storyPoints: 25 }], 10, ['Sprint 7']);
		expect(f.get('a')).toBe('Sprint 7 + 2');
	});
});
