import { describe, expect, it } from 'vitest';
import { idealRemaining, metricOf, sprintLength, velocity } from './sprints';

const day = { date: '2026-12-01', scopeCount: 4, doneCount: 1, scopePoints: 13, donePoints: 5, scopeManDays: 10, doneManDays: 2.5 };

describe('metricOf', () => {
	it('picks scope and done for the chosen unit', () => {
		expect(metricOf(day, 'points')).toEqual({ scope: 13, done: 5, remaining: 8 });
		expect(metricOf(day, 'manDays')).toEqual({ scope: 10, done: 2.5, remaining: 7.5 });
		expect(metricOf(day, 'count')).toEqual({ scope: 4, done: 1, remaining: 3 });
	});
});

describe('sprintLength / idealRemaining', () => {
	it('counts days end-inclusive and draws a straight line to zero on the last day', () => {
		expect(sprintLength('2026-12-01', '2026-12-10')).toBe(10);
		expect(idealRemaining(18, 0, 10)).toBe(18);
		expect(idealRemaining(18, 9, 10)).toBe(0);
		expect(idealRemaining(18, 3, 10)).toBe(12);
		expect(idealRemaining(5, 0, 1)).toBe(0);
	});
});

describe('velocity', () => {
	it('lists the last six completed sprints, oldest first', () => {
		const mk = (n: number, status: string) => ({
			name: `S${n}`,
			status,
			startDate: `2026-0${n}-01`,
			stats: { points: n * 10, donePoints: n * 8 }
		});
		const list = [mk(1, 'completed'), mk(2, 'active'), ...[3, 4, 5, 6, 7, 8, 9].map((n) => mk(n, 'completed'))];
		expect(velocity(list).map((v) => v.name)).toEqual(['S4', 'S5', 'S6', 'S7', 'S8', 'S9']);
		expect(velocity(list)[0]).toEqual({ name: 'S4', committed: 40, done: 32 });
	});
});
