import { describe, expect, it } from 'vitest';
import { translate as t } from './i18n';

describe('t', () => {
	it('translates known text, keeps the spaces around it, and leaves unknown text', () => {
		expect(t('Save')).toBe('Kaydet');
		expect(t('  Cancel ')).toBe('  İptal ');
		expect(t('Some task title')).toBe('Some task title');
	});
	it('handles text with numbers', () => {
		expect(t('3 items')).toBe('3 öğe');
		expect(t('Due 2026-12-01')).toBe('Bitiş 2026-12-01');
	});
});
