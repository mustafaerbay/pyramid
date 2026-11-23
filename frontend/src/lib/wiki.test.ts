import { describe, expect, it } from 'vitest';
import { breadcrumb, wikiTree } from './wiki';

const pages = [
	{ id: 'b', parentId: 'a', title: 'Setup', position: 2 },
	{ id: 'a', parentId: null, title: 'Home', position: 1 },
	{ id: 'c', parentId: 'a', title: 'API', position: 2 },
	{ id: 'd', parentId: 'gone', title: 'Orphan', position: 3 }
];

describe('wikiTree', () => {
	it('orders children under parents; orphans go to the top level', () => {
		expect(wikiTree(pages).map((x) => `${x.depth}:${x.page.title}`)).toEqual(['0:Home', '1:API', '1:Setup', '0:Orphan']);
	});
});

describe('breadcrumb', () => {
	it('walks up to the top', () => {
		expect(breadcrumb(pages, 'b').map((p) => p.title)).toEqual(['Home', 'Setup']);
	});
});
