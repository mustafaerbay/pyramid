// Area paths ("Web/Checkout"): sub-area lookups and select labels.

type Area = { id: string; path: string };

/** The area and every area below it. */
export function areaIdsUnder(areas: readonly Area[], id: string): Set<string> {
	const root = areas.find((a) => a.id === id);
	if (!root) return new Set();
	const p = root.path.toLowerCase();
	return new Set(areas.filter((a) => a.path.toLowerCase() === p || a.path.toLowerCase().startsWith(p + '/')).map((a) => a.id));
}

/** Last level, indented two spaces per level, for <option> lists. */
export function areaLabel(path: string): string {
	const parts = path.split('/');
	return '  '.repeat(parts.length - 1) + parts[parts.length - 1];
}
