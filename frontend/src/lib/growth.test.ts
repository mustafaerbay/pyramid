import { describe, expect, it } from 'vitest';
import { contributionTypeTone, goalStatusTone } from './growth';

describe('growth tones', () => {
	it('gives goals the same tone on every page', () => {
		expect(goalStatusTone('active')).toBe('success');
		expect(goalStatusTone('draft')).toBe('neutralMuted');
		expect(goalStatusTone('achieved')).toBe('info');
		expect(goalStatusTone('partially_achieved')).toBe('warning');
		expect(goalStatusTone('missed')).toBe('danger');
		expect(goalStatusTone('cancelled')).toBe('danger');
	});

	it('gives contribution types the same tone on every page', () => {
		expect(contributionTypeTone('commit')).toBe('neutralMuted');
		expect(contributionTypeTone('pull_request')).toBe('violet');
		expect(contributionTypeTone('code_review')).toBe('info');
		expect(contributionTypeTone('issue')).toBe('warning');
		expect(contributionTypeTone('deployment')).toBe('success');
		expect(contributionTypeTone('design')).toBe('neutral');
	});
});
