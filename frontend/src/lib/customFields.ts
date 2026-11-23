// Project custom fields: keys from labels, and values ready to send.

/** A lowercase key from a label (ASCII, _ for gaps), not in `taken`. */
export function fieldKey(label: string, taken: readonly string[]): string {
	let base = label
		.toLowerCase()
		.replace(/ı/g, 'i')
		.normalize('NFKD')
		.replace(/[̀-ͯ]/g, '')
		.replace(/[^a-z0-9]+/g, '_')
		.replace(/^_+|_+$/g, '')
		.slice(0, 36);
	if (!/^[a-z]/.test(base)) base = `f_${base}`.replace(/_+$/, '');
	let key = base;
	for (let i = 2; taken.includes(key); i++) key = `${base}_${i}`;
	return key;
}

/** Form values → customValues: blanks left out, numbers as numbers. */
export function valuesForSave(fields: readonly { key: string; type: string }[], form: Record<string, unknown>): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const f of fields) {
		const v = form[f.key];
		if (v == null || (typeof v === 'string' && v.trim() === '')) continue;
		out[f.key] = f.type === 'number' ? Number(v) : v;
	}
	return out;
}
