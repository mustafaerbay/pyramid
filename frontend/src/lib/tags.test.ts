import { describe, expect, it } from 'vitest';
import { addTag, hasAllTags, tagColor, TAG_COLORS } from './tags';

describe('addTag', () => {
	it('trims and appends a new tag', () => {
		expect(addTag(['ui'], '  bug ')).toEqual(['ui', 'bug']);
	});
	it('ignores blanks and case-insensitive duplicates', () => {
		expect(addTag(['Bug'], '   ')).toEqual(['Bug']);
		expect(addTag(['Bug'], 'bug')).toEqual(['Bug']);
	});
	it('reuses the spelling of a known tag', () => {
		expect(addTag([], 'backend', ['Backend', 'UI'])).toEqual(['Backend']);
	});
	it('cuts tags to 40 characters', () => {
		expect(addTag([], 'x'.repeat(50))[0]).toHaveLength(40);
	});
});

describe('tagColor', () => {
	it('is stable and ignores case', () => {
		expect(tagColor('Backend')).toBe(tagColor('backend'));
		expect(TAG_COLORS).toContain(tagColor('anything'));
	});
	it('spreads different tags over more than one colour', () => {
		const names = ['ui', 'api', 'bug', 'infra', 'docs', 'qa', 'design', 'mobile'];
		expect(new Set(names.map(tagColor)).size).toBeGreaterThan(2);
	});
});

describe('hasAllTags', () => {
	it('matches everything when no tag is selected', () => {
		expect(hasAllTags([], [])).toBe(true);
	});
	it('needs every selected tag, ignoring case', () => {
		expect(hasAllTags(['UI', 'Bug'], ['ui', 'bug'])).toBe(true);
		expect(hasAllTags(['UI'], ['ui', 'bug'])).toBe(false);
	});
});
