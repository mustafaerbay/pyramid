import { describe, expect, it } from 'vitest';
import { formatManDays, loadBand, monthKey, monthLabel, shiftMonth } from './capacity';

describe('monthKey', () => {
	it('formats a date as YYYY-MM', () => {
		expect(monthKey(new Date(2026, 10, 22))).toBe('2026-11');
		expect(monthKey(new Date(2026, 0, 1))).toBe('2026-01');
	});
});

describe('shiftMonth', () => {
	it('moves forward and back across years', () => {
		expect(shiftMonth('2026-11', 1)).toBe('2026-12');
		expect(shiftMonth('2026-12', 1)).toBe('2027-01');
		expect(shiftMonth('2026-01', -1)).toBe('2025-12');
		expect(shiftMonth('2026-05', -14)).toBe('2025-03');
	});
});

describe('monthLabel', () => {
	it('reads as month and year', () => {
		expect(monthLabel('2026-11')).toBe('November 2026');
	});
});

describe('formatManDays', () => {
	it('drops trailing zeros and keeps one decimal at most', () => {
		expect(formatManDays(20)).toBe('20');
		expect(formatManDays(2.5)).toBe('2.5');
		expect(formatManDays(14.2857)).toBe('14.3');
		expect(formatManDays(0)).toBe('0');
	});
});

describe('loadBand', () => {
	it('flags over-planned, healthy and free capacity', () => {
		expect(loadBand(120).label).toBe('Over capacity');
		expect(loadBand(100).label).toBe('Fully planned');
		expect(loadBand(70).label).toBe('Fully planned');
		expect(loadBand(69).label).toBe('Has capacity');
	});
});
