// Delivery plan layout: lane positions and dependency lines.

/** Top of each lane and the total height; a lane is a header strip plus one row per item (at least one). */
export function laneTops(rows: readonly number[], size: { head: number; row: number; pad: number }) {
	const tops: number[] = [];
	let y = 0;
	for (const n of rows) {
		tops.push(y);
		y += size.head + Math.max(n, 1) * size.row + size.pad;
	}
	return { tops, height: y };
}

type Placed = { left: number; width: number; y: number; start: string; due: string };

/**
 * A line from the end of each blocker to the start of each item waiting on
 * it (both must be on the plan). Conflict: the item starts before its
 * blocker is due.
 */
export function dependencyLinks(items: readonly { id: string; blockedByIds: string[] }[], placed: Map<string, Placed>) {
	const out: { from: string; to: string; x1: number; y1: number; x2: number; y2: number; conflict: boolean }[] = [];
	for (const t of items) {
		const to = placed.get(t.id);
		if (!to) continue;
		for (const b of t.blockedByIds) {
			const from = placed.get(b);
			if (!from) continue;
			out.push({ from: b, to: t.id, x1: from.left + from.width, y1: from.y, x2: to.left, y2: to.y, conflict: to.start <= from.due });
		}
	}
	return out;
}
