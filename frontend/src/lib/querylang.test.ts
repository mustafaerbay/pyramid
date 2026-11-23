import { describe, expect, it } from 'vitest';
import { cleanGroup, needsValue, treeDepth } from './querylang';

describe('cleanGroup', () => {
	it('drops unfinished conditions and empty groups', () => {
		const g = cleanGroup({
			match: 'all',
			conditions: [
				{ field: 'title', op: 'contains', value: ' bug ' },
				{ field: 'priority', op: 'eq', value: '' },
				{ field: 'assignee', op: 'empty', value: '' }
			],
			groups: [{ match: 'any', conditions: [{ field: 'tag', op: 'eq', value: '' }], groups: [] }]
		});
		expect(g).toEqual({
			match: 'all',
			conditions: [
				{ field: 'title', op: 'contains', value: 'bug' },
				{ field: 'assignee', op: 'empty' }
			]
		});
	});
	it('is null when nothing is left', () => {
		expect(cleanGroup({ match: 'all', conditions: [{ field: 'title', op: 'eq', value: '' }], groups: [] })).toBeNull();
	});
});

describe('needsValue', () => {
	it('is false for empty checks', () => {
		expect(needsValue('empty')).toBe(false);
		expect(needsValue('eq')).toBe(true);
	});
});

describe('treeDepth', () => {
	it('indents children under their parent in the result order', () => {
		const rows = [
			{ id: 'a', parentId: null },
			{ id: 'b', parentId: 'a' },
			{ id: 'c', parentId: 'b' },
			{ id: 'd', parentId: null }
		];
		expect(treeDepth(rows, new Set(['b', 'c']))).toEqual({ a: 0, b: 1, c: 2, d: 0 });
	});
});
