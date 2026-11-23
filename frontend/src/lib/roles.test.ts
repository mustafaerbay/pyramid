import { describe, expect, it } from 'vitest';
import { roleAtLeast, safeNext } from './roles';

describe('roleAtLeast', () => {
	it('orders viewer < member < manager < admin', () => {
		expect(roleAtLeast('admin', 'manager')).toBe(true);
		expect(roleAtLeast('manager', 'manager')).toBe(true);
		expect(roleAtLeast('member', 'manager')).toBe(false);
		expect(roleAtLeast('viewer', 'member')).toBe(false);
		expect(roleAtLeast(undefined, 'viewer')).toBe(false);
	});
});

describe('safeNext', () => {
	it('keeps same-site paths and drops anything else', () => {
		expect(safeNext('/projects/1?x=2')).toBe('/projects/1?x=2');
		expect(safeNext(null)).toBe('/');
		expect(safeNext('https://evil.example')).toBe('/');
		expect(safeNext('//evil.example')).toBe('/');
		expect(safeNext('/login')).toBe('/');
	});
});
