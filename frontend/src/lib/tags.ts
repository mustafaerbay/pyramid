// Task tags: shared across projects, matched ignoring case. The backend
// keeps the first spelling; these helpers mirror that in the form.

export const MAX_TAG_LENGTH = 40;
export const MAX_TAGS = 20;

/** Badge classes; a tag always gets the same one, picked from its name. */
export const TAG_COLORS = [
	'bg-indigo-50 text-indigo-700 ring-indigo-200',
	'bg-emerald-50 text-emerald-700 ring-emerald-200',
	'bg-amber-50 text-amber-800 ring-amber-200',
	'bg-sky-50 text-sky-700 ring-sky-200',
	'bg-violet-50 text-violet-700 ring-violet-200',
	'bg-rose-50 text-rose-700 ring-rose-200',
	'bg-cyan-50 text-cyan-800 ring-cyan-200',
	'bg-slate-100 text-slate-700 ring-slate-300'
] as const;

export function tagColor(tag: string): string {
	let h = 0;
	for (const ch of tag.toLowerCase()) h = (h * 31 + ch.codePointAt(0)!) >>> 0;
	return TAG_COLORS[h % TAG_COLORS.length];
}

const same = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();

/**
 * Adds `raw` to `tags` unless it's blank or already there (ignoring case).
 * When a tag with that name already exists anywhere (`known`), its
 * spelling is used.
 */
export function addTag(tags: string[], raw: string, known: string[] = []): string[] {
	const tag = raw.trim().slice(0, MAX_TAG_LENGTH);
	if (!tag || tags.some((t) => same(t, tag))) return tags;
	return [...tags, known.find((k) => same(k, tag)) ?? tag];
}

/** True when `tags` carries every tag in `selected` (ignoring case). */
export function hasAllTags(tags: string[], selected: string[]): boolean {
	return selected.every((s) => tags.some((t) => same(t, s)));
}
