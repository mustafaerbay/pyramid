import { describe, expect, it } from 'vitest';
import { teamFilter } from './teams';

const areas = [
	{ id: 'w', path: 'Web' },
	{ id: 'wc', path: 'Web/Checkout' },
	{ id: 'a', path: 'API' }
];

describe('teamFilter', () => {
	it('keeps items in the team areas and below', () => {
		const f = teamFilter({ memberIds: ['p1'], areaIds: ['w'] }, areas);
		expect(f({ areaId: 'wc', assigneeIds: [] } as never)).toBe(true);
		expect(f({ areaId: 'a', assigneeIds: ['p1'] } as never)).toBe(false);
		expect(f({ areaId: null, assigneeIds: ['p1'] } as never)).toBe(false);
	});
	it('falls back to members when the team has no areas', () => {
		const f = teamFilter({ memberIds: ['p1'], areaIds: [] }, areas);
		expect(f({ areaId: 'a', assigneeIds: ['p1', 'p2'] } as never)).toBe(true);
		expect(f({ areaId: 'a', assigneeIds: ['p3'] } as never)).toBe(false);
	});
});
