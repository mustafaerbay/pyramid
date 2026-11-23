import { describe, expect, it } from 'vitest';
import { blockerOptions, blocksOf, openBlockers } from './dependencies';

type T = { id: string; status: string; blockedByIds: string[] };
const t = (id: string, blockedByIds: string[] = [], status = 'todo'): T => ({ id, status, blockedByIds });

describe('openBlockers', () => {
	it('lists blockers that are not done or cancelled', () => {
		const all = [t('a', [], 'done'), t('b'), t('c', [], 'cancelled'), t('x', ['a', 'b', 'c'])];
		expect(openBlockers(all[3], all).map((x) => x.id)).toEqual(['b']);
	});
});

describe('blocksOf', () => {
	it('lists tasks waiting on this one', () => {
		const all = [t('a'), t('b', ['a']), t('c', ['a', 'b'])];
		expect(blocksOf('a', all).map((x) => x.id)).toEqual(['b', 'c']);
	});
});

describe('blockerOptions', () => {
	it('leaves out the task itself and anything that already waits on it', () => {
		// c waits on b, b waits on a: a can't wait on b or c.
		const all = [t('a'), t('b', ['a']), t('c', ['b']), t('d')];
		expect(blockerOptions(all, 'a').map((x) => x.id)).toEqual(['d']);
		expect(blockerOptions(all, 'c').map((x) => x.id)).toEqual(['a', 'b', 'd']);
		expect(blockerOptions(all, null).map((x) => x.id)).toEqual(['a', 'b', 'c', 'd']);
	});
});
