// Türkçe / English. English is the source text; Turkish comes from a
// dictionary keyed by that text. Pages can call t() directly; everything
// else is translated in the page itself (text and placeholder / title /
// aria-label) as it renders, skipping what people wrote (Markdown,
// inputs, code). Switching back to English reloads the page.

import { translate } from './i18n';

export type Locale = 'en' | 'tr';
const KEY = 'pyramid.locale';

function initial(): Locale {
	try {
		const saved = localStorage.getItem(KEY);
		if (saved === 'tr' || saved === 'en') return saved;
		return navigator.language?.toLowerCase().startsWith('tr') ? 'tr' : 'en';
	} catch {
		return 'en';
	}
}

export const i18n = $state<{ locale: Locale }>({ locale: typeof window === 'undefined' ? 'en' : initial() });

export function t(s: string): string {
	return i18n.locale === 'tr' ? translate(s) : s;
}

export function setLocale(l: Locale) {
	try {
		localStorage.setItem(KEY, l);
	} catch {
		/* private mode: this page only */
	}
	if (l === i18n.locale) return;
	i18n.locale = l;
	if (l === 'en') location.reload();
	else translateTree(document.body);
	document.documentElement.lang = l;
}

const SKIP = '.md, [data-no-i18n], textarea, script, style, code, pre, [contenteditable="true"]';
const ATTRS = ['placeholder', 'title', 'aria-label'];

function translateText(n: Text) {
	const v = n.data;
	if (!v || !/[A-Za-z]/.test(v)) return;
	const p = n.parentElement;
	if (!p || p.closest(SKIP)) return;
	const out = t(v);
	if (out !== v) n.data = out;
}

function translateEl(el: Element) {
	if (el.closest(SKIP)) return;
	for (const a of ATTRS) {
		const v = el.getAttribute(a);
		if (v && /[A-Za-z]/.test(v)) {
			const out = t(v);
			if (out !== v) el.setAttribute(a, out);
		}
	}
}

export function translateTree(root: Node) {
	if (i18n.locale !== 'tr') return;
	if (root.nodeType === Node.TEXT_NODE) return translateText(root as Text);
	if (root.nodeType !== Node.ELEMENT_NODE) return;
	translateEl(root as Element);
	const w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT);
	for (let n = w.nextNode(); n; n = w.nextNode()) {
		if (n.nodeType === Node.TEXT_NODE) translateText(n as Text);
		else translateEl(n as Element);
	}
}

/** Keeps the page translated as Svelte renders and updates it. */
export function watchTranslations(): () => void {
	document.documentElement.lang = i18n.locale;
	translateTree(document.body);
	const mo = new MutationObserver((records) => {
		if (i18n.locale !== 'tr') return;
		for (const r of records) {
			if (r.type === 'characterData') translateText(r.target as Text);
			else if (r.type === 'attributes') translateEl(r.target as Element);
			else r.addedNodes.forEach((n) => translateTree(n));
		}
	});
	mo.observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ATTRS });
	return () => mo.disconnect();
}
