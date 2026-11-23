// Helpers for man-day capacity views (People, Dashboard, project totals).
// Months are passed around as 'YYYY-MM' keys, the same format the
// GET /api/v1/capacity?month= parameter takes.

export function monthKey(d: Date): string {
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
}

export function shiftMonth(key: string, delta: number): string {
	const [y, m] = key.split('-').map(Number);
	return monthKey(new Date(y, m - 1 + delta, 1));
}

export function monthLabel(key: string): string {
	const [y, m] = key.split('-').map(Number);
	return new Date(y, m - 1, 1).toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
}

/** Man-days with at most one decimal, no trailing ".0". */
export function formatManDays(n: number): string {
	return String(Math.round(n * 10) / 10);
}

/** Colour band for planned ÷ capacity: over 100 %, 70–100 %, below 70 %. */
export function loadBand(pct: number): { bar: string; text: string; label: string } {
	if (pct > 100) return { bar: 'bg-red-500', text: 'text-red-700', label: 'Over capacity' };
	if (pct >= 70) return { bar: 'bg-emerald-500', text: 'text-slate-700', label: 'Fully planned' };
	return { bar: 'bg-slate-400', text: 'text-slate-500', label: 'Has capacity' };
}

/**
 * Splits `total` man-days across `count` people in half-day steps: everyone
 * gets the same base share and the leftover half-days go to the first ones,
 * so the parts always add back up to the total.
 */
export function splitEvenly(total: number, count: number): number[] {
	if (count <= 0) return [];
	const halves = Math.round(Math.max(total, 0) * 2);
	const base = Math.floor(halves / count);
	const extra = halves % count;
	return Array.from({ length: count }, (_, i) => (base + (i < extra ? 1 : 0)) / 2);
}

type Fit = { personId: string; manDays: number; availableDays?: number | null };

/**
 * Assignments whose man-days don't fit in the working days the person has
 * between the task's start and due dates (after approved leave). Tasks
 * without both dates have no availableDays and are never listed.
 */
export function shortfalls<T extends Fit>(assignments: T[]): T[] {
	return assignments.filter(
		(a) => a.manDays > 0 && a.availableDays != null && a.manDays > a.availableDays
	);
}

/** "12 md in 5 available days" — the warning shown for a shortfall. */
export function fitText(manDays: number, availableDays: number): string {
	if (availableDays === 0) return `${formatManDays(manDays)} md, no available days (on leave)`;
	return `${formatManDays(manDays)} md in ${availableDays} available day${availableDays === 1 ? '' : 's'}`;
}
