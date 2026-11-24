// Turkish translation of English UI text (see i18n.svelte.ts).
import { TR } from './i18n.tr';

/** Rules for text with numbers or names in it. */
const PATTERNS: [RegExp, string][] = [
	[/^(\d+) items?$/, '$1 öğe'],
	[/^(\d+) tasks?$/, '$1 görev'],
	[/^(\d+) cases$/, '$1 test case'],
	[/^(\d+) widgets?$/, '$1 widget'],
	[/^(\d+) members?$/, '$1 üye'],
	[/^(\d+) active members?$/, '$1 aktif üye'],
	[/^(\d+) milestones?$/, '$1 kilometre taşı'],
	[/^(\d+) without dates$/, '$1 tarihsiz'],
	[/^(\d+) steps$/, '$1 adım'],
	[/^(\d+) events$/, '$1 olay'],
	[/^(\d+) children done$/, '$1 alt öğe bitti'],
	[/^(\d+)\/(\d+) children done$/, '$1/$2 alt öğe bitti'],
	[/^Due (.+)$/, 'Bitiş $1'],
	[/^Copied as #(\d+)\.$/, '#$1 olarak kopyalandı.'],
	[/^Results for “(.+)”$/, '“$1” için sonuçlar'],
	[/^All results for “(.+)” →$/, '“$1” için tüm sonuçlar →'],
	[/^Page (\d+) of (\d+)$/, 'Sayfa $1 / $2'],
	[/^(\d+) in total$/, 'Toplam $1'],
	[/^(.+) progress$/, '$1 ilerlemesi']
];

/** The Turkish for an English UI text, or the text itself. */
export function translate(s: string): string {
	const key = s.replace(/\s+/g, ' ').trim();
	if (!key) return s;
	// Keep the spaces around the text (they separate it from what's next to it).
	const lead = /^\s*/.exec(s)![0];
	const trail = /\s*$/.exec(s)![0];
	const hit = TR[key];
	if (hit) return lead + hit + trail;
	for (const [re, out] of PATTERNS) if (re.test(key)) return lead + key.replace(re, out) + trail;
	return s;
}

