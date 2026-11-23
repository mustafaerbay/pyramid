import { describe, expect, it } from 'vitest';
import { fieldKey, valuesForSave } from './customFields';

describe('fieldKey', () => {
	it('slugs the label and avoids taken keys', () => {
		expect(fieldKey('Go-live date', [])).toBe('go_live_date');
		expect(fieldKey('Müşteri adı', [])).toBe('musteri_adi');
		expect(fieldKey('Risk', ['risk'])).toBe('risk_2');
		expect(fieldKey('123', [])).toBe('f_123');
	});
});

describe('valuesForSave', () => {
	it('keeps filled values with the right types', () => {
		const fields = [
			{ key: 'a', type: 'text' },
			{ key: 'n', type: 'number' },
			{ key: 'd', type: 'date' },
			{ key: 's', type: 'select' }
		];
		expect(valuesForSave(fields, { a: ' x ', n: '12.5', d: '', s: 'High', extra: 'y' })).toEqual({ a: ' x ', n: 12.5, s: 'High' });
	});
});
