import { describe, expect, it } from 'vitest';
import { FLOW_ORDER, percentile, planBar, stackDay } from './flow';

describe('stackDay', () => {
	it('stacks statuses in flow order, done at the bottom', () => {
		expect(FLOW_ORDER[0]).toBe('done');
		const s = stackDay({ todo: 2, done: 3, in_progress: 1 });
		expect(s.find((x) => x.status === 'done')).toEqual({ status: 'done', y0: 0, y1: 3 });
		expect(s.find((x) => x.status === 'in_progress')).toEqual({ status: 'in_progress', y0: 3, y1: 4 });
		expect(s.at(-1)!.y1).toBe(6);
	});
});

describe('percentile', () => {
	it('uses the nearest rank', () => {
		expect(percentile([], 85)).toBeNull();
		expect(percentile([5, 1, 3, 2, 4], 50)).toBe(3);
		expect(percentile([1, 2, 3, 4, 5, 6, 7, 8, 9, 10], 85)).toBe(9);
	});
});

describe('planBar', () => {
	// December: from the 1st up to (not including) 1 January.
	const range = { from: Date.parse('2026-12-01'), to: Date.parse('2027-01-01') };
	it('places dated items as a share of the range', () => {
		const b = planBar('2026-12-01', '2026-12-15', range)!; // 15 of 31 days
		expect(b.left).toBeCloseTo(0);
		expect(b.width).toBeCloseTo((15 / 31) * 100);
	});
	it('clips to the range, shows a due-only item as one day, and skips the undated or out of range', () => {
		expect(planBar('2026-11-01', '2026-12-04', range)!.left).toBe(0);
		expect(planBar(null, '2026-12-31', range)!.width).toBeCloseTo(100 / 31);
		expect(planBar(null, null, range)).toBeNull();
		expect(planBar('2027-02-01', '2027-02-10', range)).toBeNull();
	});
});
