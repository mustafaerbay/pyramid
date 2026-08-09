// Shared UTC-safe date-math helpers.
//
// Every page that compares a plain "YYYY-MM-DD" date string against "today"
// used to hand-roll its own copy of this logic — projects/[id]/+page.svelte,
// ProjectCard.svelte, and the home dashboard (+page.svelte) each had their
// own toUtc()/todayUtc() pair. Centralized here, following the same
// "shared, tested module" pattern the Badge/MetricCard extraction
// established for presentation logic.
//
// Parsing as `${dateStr}T00:00:00Z` and comparing against a UTC-midnight
// "today" (built from UTC getters, not local wall-clock time) means results
// never shift with the viewer's timezone — a person in UTC-8 and a person in
// UTC+8 get the same "3 days left" for the same project.

export const DAY_MS = 86_400_000;

/** Milliseconds since epoch for a plain "YYYY-MM-DD" date, at UTC midnight. */
export function toUtc(dateStr: string): number {
	return new Date(`${dateStr}T00:00:00Z`).getTime();
}

/** Milliseconds since epoch for today, at UTC midnight. */
export function todayUtc(): number {
	const now = new Date();
	return Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
}

/**
 * Whole days from `fromMs` (defaults to "now") until `dateStr`, UTC-safe.
 * Positive = in the future, negative = in the past, 0 = today. `fromMs` is
 * mainly for tests — real callers should rely on the default.
 */
export function daysUntil(dateStr: string, fromMs: number = todayUtc()): number {
	return Math.round((toUtc(dateStr) - fromMs) / DAY_MS);
}
