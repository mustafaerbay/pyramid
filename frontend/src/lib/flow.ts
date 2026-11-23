// Cumulative flow and delivery plan helpers.

/** Bottom to top: finished work first, waiting work on top. */
export const FLOW_ORDER = ['done', 'in_review', 'in_progress', 'blocked', 'todo', 'backlog'] as const;

/** Reference categorical slots 1–6 in fixed order (validated adjacent pairs). */
export const FLOW_COLORS: Record<string, string> = {
	done: '#2a78d6',
	in_review: '#eb6834',
	in_progress: '#1baf7a',
	blocked: '#eda100',
	todo: '#e87ba4',
	backlog: '#008300'
};

export function stackDay(counts: Record<string, number>): { status: string; y0: number; y1: number }[] {
	let y = 0;
	return FLOW_ORDER.map((status) => {
		const y0 = y;
		y += counts[status] ?? 0;
		return { status, y0, y1: y };
	});
}

/** Nearest-rank percentile; null for no values. */
export function percentile(values: readonly number[], p: number): number | null {
	if (values.length === 0) return null;
	const sorted = [...values].sort((a, b) => a - b);
	const rank = Math.ceil((p / 100) * sorted.length);
	return sorted[Math.min(Math.max(rank, 1), sorted.length) - 1];
}

const DAY = 86_400_000;

/**
 * Where an item with these dates sits in [range.from, range.to) (ms), as
 * left/width percentages. A due-only (or start-only) item is one day long.
 * Null when it has no dates or falls outside the range.
 */
export function planBar(
	start: string | null | undefined,
	due: string | null | undefined,
	range: { from: number; to: number }
): { left: number; width: number } | null {
	if (!start && !due) return null;
	const s = Date.parse((start ?? due)!);
	const e = Date.parse((due ?? start)!) + DAY;
	const span = range.to - range.from;
	if (e <= range.from || s >= range.to) return null;
	const a = Math.max(s, range.from);
	const b = Math.min(e, range.to);
	return { left: ((a - range.from) / span) * 100, width: Math.max(((b - a) / span) * 100, 0.5) };
}
