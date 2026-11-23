// Minimal RFC 4180 CSV: quoted fields with commas, quotes and newlines.

/** Parses CSV text into rows; drops a BOM and blank lines. */
export function parseCsv(text: string): string[][] {
	const s = text.replace(/^﻿/, '');
	const rows: string[][] = [];
	let row: string[] = [];
	let field = '';
	let quoted = false;
	for (let i = 0; i < s.length; i++) {
		const ch = s[i];
		if (quoted) {
			if (ch === '"' && s[i + 1] === '"') {
				field += '"';
				i++;
			} else if (ch === '"') quoted = false;
			else field += ch;
		} else if (ch === '"') quoted = true;
		else if (ch === ',') {
			row.push(field);
			field = '';
		} else if (ch === '\n' || ch === '\r') {
			if (ch === '\r' && s[i + 1] === '\n') i++;
			row.push(field);
			field = '';
			if (!(row.length === 1 && row[0] === '')) rows.push(row);
			row = [];
		} else field += ch;
	}
	row.push(field);
	if (!(row.length === 1 && row[0] === '')) rows.push(row);
	return rows;
}

// Cells a spreadsheet would run as a formula get a leading apostrophe.
const FORMULA = /^[=+\-@\t\r]/;

export function toCsv(rows: readonly (readonly (string | number | null | undefined)[])[]): string {
	return rows
		.map((r) =>
			r
				.map((v) => {
					let s = v == null ? '' : String(v);
					if (FORMULA.test(s)) s = `'${s}`;
					return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
				})
				.join(',')
		)
		.join('\r\n')
		.concat('\r\n');
}

/** Offers `text` as a file download in the browser. */
export function download(filename: string, text: string) {
	const url = URL.createObjectURL(new Blob(['﻿' + text], { type: 'text/csv;charset=utf-8' }));
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	a.click();
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}
