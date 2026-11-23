import { describe, expect, it } from 'vitest';
import { formatBytes } from './files';

describe('formatBytes', () => {
	it('picks a unit', () => {
		expect(formatBytes(500)).toBe('500 B');
		expect(formatBytes(1536)).toBe('1.5 KB');
		expect(formatBytes(10 * 1024 * 1024)).toBe('10 MB');
	});
});
