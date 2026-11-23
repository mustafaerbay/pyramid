import { describe, expect, it } from 'vitest';
import { dependencyLinks, laneTops } from './plans';

describe('laneTops', () => {
	it('stacks lanes by their row count (at least one row each)', () => {
		expect(laneTops([2, 0, 1], { head: 26, row: 30, pad: 2 })).toEqual({ tops: [0, 88, 146], height: 204 });
	});
});

describe('dependencyLinks', () => {
	const placed = new Map([
		['a', { left: 10, width: 10, y: 40, start: '2026-12-01', due: '2026-12-10' }],
		['b', { left: 30, width: 10, y: 70, start: '2026-12-15', due: '2026-12-20' }],
		['c', { left: 15, width: 10, y: 100, start: '2026-12-05', due: '2026-12-12' }]
	]);
	it('links each blocker to what waits on it, flagging ones that start too early', () => {
		const items = [
			{ id: 'b', blockedByIds: ['a'] },
			{ id: 'c', blockedByIds: ['a', 'zz'] }
		];
		expect(dependencyLinks(items, placed)).toEqual([
			{ from: 'a', to: 'b', x1: 20, y1: 40, x2: 30, y2: 70, conflict: false },
			{ from: 'a', to: 'c', x1: 20, y1: 40, x2: 15, y2: 100, conflict: true }
		]);
	});
});
