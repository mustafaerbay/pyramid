// Minimal .xlsx writer and reader (no dependencies). Writing stores the
// files uncompressed; reading inflates with the browser's
// DecompressionStream. Only the first sheet, values only.

const enc = new TextEncoder();

/** 0 → A, 25 → Z, 26 → AA. */
export function colName(i: number): string {
	let s = '';
	for (let n = i + 1; n > 0; n = Math.floor((n - 1) / 26)) s = String.fromCharCode(65 + ((n - 1) % 26)) + s;
	return s;
}
const colIndex = (ref: string) => {
	const letters = /^[A-Z]+/.exec(ref)?.[0] ?? 'A';
	let n = 0;
	for (const ch of letters) n = n * 26 + (ch.charCodeAt(0) - 64);
	return n - 1;
};

const CRC_TABLE = (() => {
	const t = new Uint32Array(256);
	for (let n = 0; n < 256; n++) {
		let c = n;
		for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
		t[n] = c >>> 0;
	}
	return t;
})();
function crc32(b: Uint8Array): number {
	let c = 0xffffffff;
	for (let i = 0; i < b.length; i++) c = CRC_TABLE[(c ^ b[i]) & 0xff] ^ (c >>> 8);
	return (c ^ 0xffffffff) >>> 0;
}

type Entry = { name: string; data: Uint8Array; method: 0 | 8; size: number; crc: number };

function zip(entries: Entry[]): Uint8Array {
	const parts: Uint8Array[] = [];
	const central: Uint8Array[] = [];
	let offset = 0;
	for (const e of entries) {
		const name = enc.encode(e.name);
		const local = new Uint8Array(30 + name.length);
		const v = new DataView(local.buffer);
		v.setUint32(0, 0x04034b50, true);
		v.setUint16(4, 20, true);
		v.setUint16(6, 0x0800, true); // UTF-8 names
		v.setUint16(8, e.method, true);
		v.setUint32(14, e.crc, true);
		v.setUint32(18, e.data.length, true);
		v.setUint32(22, e.size, true);
		v.setUint16(26, name.length, true);
		local.set(name, 30);
		const cd = new Uint8Array(46 + name.length);
		const w = new DataView(cd.buffer);
		w.setUint32(0, 0x02014b50, true);
		w.setUint16(4, 20, true);
		w.setUint16(6, 20, true);
		w.setUint16(8, 0x0800, true);
		w.setUint16(10, e.method, true);
		w.setUint32(16, e.crc, true);
		w.setUint32(20, e.data.length, true);
		w.setUint32(24, e.size, true);
		w.setUint16(28, name.length, true);
		w.setUint32(42, offset, true);
		cd.set(name, 46);
		parts.push(local, e.data);
		central.push(cd);
		offset += local.length + e.data.length;
	}
	const cdSize = central.reduce((s, c) => s + c.length, 0);
	const end = new Uint8Array(22);
	const d = new DataView(end.buffer);
	d.setUint32(0, 0x06054b50, true);
	d.setUint16(8, entries.length, true);
	d.setUint16(10, entries.length, true);
	d.setUint32(12, cdSize, true);
	d.setUint32(16, offset, true);
	const all = [...parts, ...central, end];
	const out = new Uint8Array(all.reduce((s, p) => s + p.length, 0));
	let at = 0;
	for (const p of all) {
		out.set(p, at);
		at += p.length;
	}
	return out;
}

const stored = (name: string, text: string): Entry => {
	const data = enc.encode(text);
	return { name, data, method: 0, size: data.length, crc: crc32(data) };
};

async function pipe(data: Uint8Array, stream: CompressionStream | DecompressionStream): Promise<Uint8Array> {
	const res = new Response(new Blob([data as BlobPart]).stream().pipeThrough(stream));
	return new Uint8Array(await res.arrayBuffer());
}

/** A zip with deflated entries (used by tests to mimic files Excel writes). */
export async function zipDeflated(files: Record<string, string>): Promise<Uint8Array> {
	const entries: Entry[] = [];
	for (const [name, text] of Object.entries(files)) {
		const raw = enc.encode(text);
		entries.push({ name, data: await pipe(raw, new CompressionStream('deflate-raw')), method: 8, size: raw.length, crc: crc32(raw) });
	}
	return zip(entries);
}

const xmlEsc = (s: string) =>
	s
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		// Characters XML 1.0 can't hold.
		.replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/g, '');

/** A one-sheet workbook; every cell is text, the first row is bold-free header text. */
export function writeXlsx(rows: readonly (readonly (string | number | null | undefined)[])[], sheet = 'Sheet1'): Uint8Array {
	const body = rows
		.map(
			(r, i) =>
				`<row r="${i + 1}">${r
					.map((v, j) => {
						const s = v == null ? '' : String(v);
						return s === '' ? '' : `<c r="${colName(j)}${i + 1}" t="inlineStr"><is><t xml:space="preserve">${xmlEsc(s)}</t></is></c>`;
					})
					.join('')}</row>`
		)
		.join('');
	const ns = 'http://schemas.openxmlformats.org/';
	return zip([
		stored(
			'[Content_Types].xml',
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="${ns}package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`
		),
		stored(
			'_rels/.rels',
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="${ns}package/2006/relationships"><Relationship Id="rId1" Type="${ns}officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`
		),
		stored(
			'xl/workbook.xml',
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="${ns}spreadsheetml/2006/main" xmlns:r="${ns}officeDocument/2006/relationships"><sheets><sheet name="${xmlEsc(sheet.slice(0, 31))}" sheetId="1" r:id="rId1"/></sheets></workbook>`
		),
		stored(
			'xl/_rels/workbook.xml.rels',
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="${ns}package/2006/relationships"><Relationship Id="rId1" Type="${ns}officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`
		),
		stored(
			'xl/worksheets/sheet1.xml',
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="${ns}spreadsheetml/2006/main"><sheetData>${body}</sheetData></worksheet>`
		)
	]);
}

function unzip(b: Uint8Array): Map<string, { method: number; data: Uint8Array }> {
	const v = new DataView(b.buffer, b.byteOffset, b.byteLength);
	let end = -1;
	for (let i = b.length - 22; i >= Math.max(0, b.length - 65557); i--) {
		if (v.getUint32(i, true) === 0x06054b50) {
			end = i;
			break;
		}
	}
	if (end < 0) throw new Error('Not an .xlsx file.');
	const count = v.getUint16(end + 10, true);
	let p = v.getUint32(end + 16, true);
	const out = new Map<string, { method: number; data: Uint8Array }>();
	const dec = new TextDecoder();
	for (let i = 0; i < count; i++) {
		if (v.getUint32(p, true) !== 0x02014b50) throw new Error('Damaged .xlsx file.');
		const method = v.getUint16(p + 10, true);
		const csize = v.getUint32(p + 20, true);
		const nlen = v.getUint16(p + 28, true);
		const xlen = v.getUint16(p + 30, true);
		const clen = v.getUint16(p + 32, true);
		const local = v.getUint32(p + 42, true);
		const name = dec.decode(b.subarray(p + 46, p + 46 + nlen));
		const start = local + 30 + v.getUint16(local + 26, true) + v.getUint16(local + 28, true);
		out.set(name, { method, data: b.subarray(start, start + csize) });
		p += 46 + nlen + xlen + clen;
	}
	return out;
}

const unesc = (s: string) =>
	s
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&apos;/g, "'")
		.replace(/&#(\d+);/g, (_, n) => String.fromCodePoint(Number(n)))
		.replace(/&#x([0-9a-f]+);/gi, (_, n) => String.fromCodePoint(parseInt(n, 16)))
		.replace(/&amp;/g, '&');
// Text of every <t> inside a fragment (rich text runs joined).
const texts = (xml: string) => [...xml.matchAll(/<t(?:\s[^>]*)?>([\s\S]*?)<\/t>|<t\s*\/>/g)].map((m) => unesc(m[1] ?? '')).join('');

/** The first sheet's cells as text, rows padded to the same width. */
export async function readXlsx(bytes: Uint8Array): Promise<string[][]> {
	const files = unzip(bytes);
	const read = async (name: string) => {
		const f = files.get(name);
		if (!f) return null;
		const data = f.method === 8 ? await pipe(f.data, new DecompressionStream('deflate-raw')) : f.data;
		return new TextDecoder().decode(data);
	};
	// First sheet's file, through the workbook relationships.
	let sheetPath = 'xl/worksheets/sheet1.xml';
	const wb = await read('xl/workbook.xml');
	const rels = await read('xl/_rels/workbook.xml.rels');
	const rid = wb && /<sheet\b[^>]*\br:id="([^"]+)"/.exec(wb)?.[1];
	if (rid && rels) {
		const target = new RegExp(`<Relationship\\b[^>]*Id="${rid}"[^>]*Target="([^"]+)"`).exec(rels)?.[1] ??
			new RegExp(`<Relationship\\b[^>]*Target="([^"]+)"[^>]*Id="${rid}"`).exec(rels)?.[1];
		if (target) sheetPath = target.startsWith('/') ? target.slice(1) : `xl/${target}`;
	}
	const sheet = await read(sheetPath);
	if (!sheet) throw new Error('The workbook has no sheet.');
	const sstXml = await read('xl/sharedStrings.xml');
	const shared = sstXml ? [...sstXml.matchAll(/<si>([\s\S]*?)<\/si>/g)].map((m) => texts(m[1])) : [];

	const rows: string[][] = [];
	for (const rm of sheet.matchAll(/<row\b([^>]*?)(?:\/>|>([\s\S]*?)<\/row>)/g)) {
		const r = Number(/\br="(\d+)"/.exec(rm[1])?.[1] ?? rows.length + 1) - 1;
		const row: string[] = [];
		let next = 0;
		for (const cm of (rm[2] ?? '').matchAll(/<c\b([^>]*?)(?:\/>|>([\s\S]*?)<\/c>)/g)) {
			const attrs = cm[1];
			const ref = /\br="([A-Z]+)\d*"/.exec(attrs)?.[1];
			const col = ref ? colIndex(ref) : next;
			next = col + 1;
			const t = /\bt="([^"]+)"/.exec(attrs)?.[1];
			const inner = cm[2] ?? '';
			const v = /<v>([\s\S]*?)<\/v>/.exec(inner)?.[1];
			let val = '';
			if (t === 's') val = shared[Number(v)] ?? '';
			else if (t === 'inlineStr') val = texts(inner);
			else if (t === 'b') val = v === '1' ? 'TRUE' : 'FALSE';
			else val = v != null ? unesc(v) : '';
			row[col] = val;
		}
		rows[r] = Array.from(row, (x) => x ?? '');
	}
	const width = Array.from(rows, (r) => r?.length ?? 0).reduce((a, b) => Math.max(a, b), 0);
	return Array.from(rows, (r) => {
		const x = r ?? [];
		return [...x, ...Array(width - x.length).fill('')];
	});
}
