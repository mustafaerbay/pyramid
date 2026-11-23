import { describe, expect, it } from 'vitest';
import { areaIdsUnder, areaLabel } from './areas';

const areas = [
	{ id: '1', path: 'API' },
	{ id: '2', path: 'Web' },
	{ id: '3', path: 'Web/Checkout' },
	{ id: '4', path: 'Web/Checkout/Pay' },
	{ id: '5', path: 'Webhooks' }
];

describe('areaIdsUnder', () => {
	it('includes the area and its sub-areas, not look-alike names', () => {
		expect([...areaIdsUnder(areas, '2')].sort()).toEqual(['2', '3', '4']);
		expect([...areaIdsUnder(areas, '1')]).toEqual(['1']);
	});
});

describe('areaLabel', () => {
	it('indents by depth and shows the last level', () => {
		expect(areaLabel('Web/Checkout/Pay')).toBe('    Pay');
		expect(areaLabel('API')).toBe('API');
	});
});
