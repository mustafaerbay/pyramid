import { describe, expect, it } from 'vitest';
import { DAY_MS, daysUntil, todayUtc, toUtc } from './dateMath';

describe('toUtc', () => {
	it('parses a plain date string as UTC midnight', () => {
		expect(toUtc('2026-01-01')).toBe(Date.UTC(2026, 0, 1));
	});

	it('handles a leap-year February 29th', () => {
		expect(toUtc('2024-02-29')).toBe(Date.UTC(2024, 1, 29));
	});

	it('is unaffected by the runtime timezone (always UTC midnight)', () => {
		const ms = toUtc('2026-08-09');
		const d = new Date(ms);
		expect(d.getUTCHours()).toBe(0);
		expect(d.getUTCMinutes()).toBe(0);
		expect(d.getUTCSeconds()).toBe(0);
		expect(d.getUTCMilliseconds()).toBe(0);
	});
});

describe('todayUtc', () => {
	it('returns a timestamp with zero time-of-day components', () => {
		const d = new Date(todayUtc());
		expect(d.getUTCHours()).toBe(0);
		expect(d.getUTCMinutes()).toBe(0);
		expect(d.getUTCSeconds()).toBe(0);
		expect(d.getUTCMilliseconds()).toBe(0);
	});

	it('matches toUtc() of today\'s own date string', () => {
		const now = new Date();
		const iso = now.toISOString().slice(0, 10);
		expect(todayUtc()).toBe(toUtc(iso));
	});
});

describe('daysUntil', () => {
	// 2026-08-09, matching this session's "today" — daysUntil takes an
	// explicit fromMs so these tests never depend on the actual clock.
	const reference = Date.UTC(2026, 7, 9);

	it('returns 0 for today', () => {
		expect(daysUntil('2026-08-09', reference)).toBe(0);
	});

	it('returns a positive count for a future date', () => {
		expect(daysUntil('2026-08-19', reference)).toBe(10);
	});

	it('returns a negative count for a past date', () => {
		expect(daysUntil('2026-07-30', reference)).toBe(-10);
	});

	it('crosses a year boundary correctly', () => {
		expect(daysUntil('2027-01-01', Date.UTC(2026, 11, 31))).toBe(1);
	});

	it('crosses a leap-day correctly (2024 was a leap year)', () => {
		expect(daysUntil('2024-03-01', Date.UTC(2024, 1, 28))).toBe(2);
	});

	it('stays consistent with DAY_MS and toUtc', () => {
		expect(daysUntil('2026-08-10', reference) * DAY_MS).toBe(toUtc('2026-08-10') - reference);
	});

	it('defaults fromMs to todayUtc() when omitted', () => {
		expect(daysUntil(new Date(todayUtc()).toISOString().slice(0, 10))).toBe(0);
	});
});
