// Wiki page tree helpers.

type P = { id: string; parentId?: string | null; title: string; position: number };

/** Pages in tree order with their depth (children after their parent, by position then title). */
export function wikiTree<T extends P>(pages: readonly T[]): { page: T; depth: number }[] {
	const ids = new Set(pages.map((p) => p.id));
	const kids = new Map<string, T[]>();
	for (const p of pages) {
		const k = p.parentId && ids.has(p.parentId) ? p.parentId : '';
		kids.set(k, [...(kids.get(k) ?? []), p]);
	}
	const out: { page: T; depth: number }[] = [];
	const walk = (k: string, depth: number) => {
		for (const p of [...(kids.get(k) ?? [])].sort((a, b) => a.position - b.position || a.title.localeCompare(b.title))) {
			out.push({ page: p, depth });
			if (depth < 20) walk(p.id, depth + 1);
		}
	};
	walk('', 0);
	return out;
}

/** The titles from the top page down to this one. */
export function breadcrumb<T extends P>(pages: readonly T[], id: string): T[] {
	const byId = new Map(pages.map((p) => [p.id, p]));
	const out: T[] = [];
	for (let p = byId.get(id); p && out.length < 20; p = p.parentId ? byId.get(p.parentId) : undefined) out.unshift(p);
	return out;
}
