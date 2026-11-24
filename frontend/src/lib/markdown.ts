// A small, safe Markdown renderer for descriptions, comments, notes and the
// wiki. Everything is HTML-escaped first; only the markup made here is
// emitted, and links must be http(s), mailto or a path in the app.

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

export type MarkdownOptions = {
	/** Names of people, highlighted when written as @Name. */
	people?: string[];
	/** Link for an item number (#12), or null. */
	item?: (n: number) => string | null;
	/** Where an image's src loads from: a Pyramid attachment path or a
	 * pending:N placeholder. Return null to not show it. */
	image?: (src: string) => string | null;
};

const ATTACHMENT_SRC = /^\/api\/v1\/attachments\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\/download\?inline=1$/;

/** Markdown for an uploaded image attachment. */
export function attachmentImage(id: string, name: string): string {
	return `![${name.replace(/[[\]\n]/g, '').trim() || 'image'}](/api/v1/attachments/${id}/download?inline=1)`;
}

/** Swaps pending:N placeholders for the attachments they became. */
export function fillPending(text: string, ids: Record<string, string>): string {
	return text.replace(/!\[([^\]]*)\]\((pending:\d+)\)/g, (m, alt, key) => (ids[key] ? attachmentImage(ids[key], alt) : m));
}

function inline(raw: string, o: MarkdownOptions): string {
	// Code spans first, kept out of the other rules.
	const codes: string[] = [];
	let s = raw.replace(/`([^`]+)`/g, (_, c) => {
		codes.push(`<code>${esc(c)}</code>`);
		return `\u0000${codes.length - 1}\u0000`;
	});
	const links: string[] = [];
	// Images only from Pyramid's own attachments (or a placeholder the
	// resolver knows); any other image is shown as a plain link.
	s = s.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, (m, alt, url) => {
		const known = ATTACHMENT_SRC.test(url) || /^pending:\d+$/.test(url);
		const src = known ? (o.image ? o.image(url) : ATTACHMENT_SRC.test(url) ? url : null) : null;
		if (src) {
			links.push(`<img src="${esc(src)}" alt="${esc(alt)}" loading="lazy">`);
		} else if (/^https?:\/\//i.test(url)) {
			links.push(`<a href="${esc(url)}" target="_blank" rel="noopener noreferrer">${esc(alt || url)}</a>`);
		} else return m;
		return `\u0001${links.length - 1}\u0001`;
	});
	s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (m, text, url) => {
		if (!/^(https?:\/\/|mailto:|\/(?!\/))/i.test(url)) return m;
		const ext = /^https?:/i.test(url);
		links.push(`<a href="${esc(url)}"${ext ? ' target="_blank" rel="noopener noreferrer"' : ''}>${esc(text)}</a>`);
		return `\u0001${links.length - 1}\u0001`;
	});
	s = esc(s);
	s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>').replace(/(^|[^*\w])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>');
	s = s.replace(/~~([^~]+)~~/g, '<del>$1</del>');
	for (const name of [...(o.people ?? [])].sort((a, b) => b.length - a.length)) {
		const n = esc(name);
		s = s.split(`@${n}`).join(`<span class="mention">@${n}</span>`);
	}
	if (o.item) {
		s = s.replace(/(^|[\s(])#(\d{1,6})\b/g, (m, pre, num) => {
			const href = o.item!(Number(num));
			return href ? `${pre}<a href="${esc(href)}">#${num}</a>` : m;
		});
	}
	s = s.replace(/(^|[\s(])(https?:\/\/[^\s<]+[^\s<.,;:!?)])/g, '$1<a href="$2" target="_blank" rel="noopener noreferrer">$2</a>');
	s = s.replace(/\u0001(\d+)\u0001/g, (_, i) => links[Number(i)]);
	return s.replace(/\u0000(\d+)\u0000/g, (_, i) => codes[Number(i)]);
}

export function renderMarkdown(text: string, o: MarkdownOptions = {}): string {
	const lines = text.replace(/\r\n?/g, '\n').split('\n');
	const out: string[] = [];
	let para: string[] = [];
	const flush = () => {
		if (para.length) out.push(`<p>${para.map((l) => inline(l, o)).join('<br>')}</p>`);
		para = [];
	};
	for (let i = 0; i < lines.length; i++) {
		const line = lines[i];
		if (/^```/.test(line)) {
			flush();
			const code: string[] = [];
			for (i++; i < lines.length && !/^```/.test(lines[i]); i++) code.push(lines[i]);
			out.push(`<pre><code>${esc(code.join('\n'))}</code></pre>`);
			continue;
		}
		const h = /^(#{1,3})\s+(.*)$/.exec(line);
		if (h) {
			flush();
			out.push(`<h${h[1].length + 2}>${inline(h[2], o)}</h${h[1].length + 2}>`);
			continue;
		}
		if (/^\s*[-*]\s+/.test(line) || /^\s*\d+[.)]\s+/.test(line)) {
			flush();
			const ordered = /^\s*\d/.test(line);
			const re = ordered ? /^\s*\d+[.)]\s+(.*)$/ : /^\s*[-*]\s+(.*)$/;
			const items: string[] = [];
			while (i < lines.length && re.test(lines[i])) {
				const m = re.exec(lines[i])![1];
				const task = /^\[( |x)\]\s+(.*)$/i.exec(m);
				items.push(task ? `<li>${task[1] === ' ' ? '☐' : '☑'} ${inline(task[2], o)}</li>` : `<li>${inline(m, o)}</li>`);
				i++;
			}
			i--;
			out.push(ordered ? `<ol>${items.join('')}</ol>` : `<ul>${items.join('')}</ul>`);
			continue;
		}
		if (/^>\s?/.test(line)) {
			flush();
			const q: string[] = [];
			while (i < lines.length && /^>\s?/.test(lines[i])) q.push(lines[i++].replace(/^>\s?/, ''));
			i--;
			out.push(`<blockquote>${q.map((l) => inline(l, o)).join('<br>')}</blockquote>`);
			continue;
		}
		if (line.trim() === '') {
			flush();
			continue;
		}
		para.push(line);
	}
	flush();
	return out.join('');
}

/** The @mention being typed just before the caret: where its @ is and what follows it. */
export function mentionQuery(text: string, caret: number): { start: number; query: string } | null {
	const before = text.slice(0, caret);
	const at = before.lastIndexOf('@');
	if (at < 0 || (at > 0 && !/\s|\(/.test(before[at - 1]))) return null;
	const q = before.slice(at + 1);
	if (q.length > 30 || /[^\p{L}\p{M}' -]/u.test(q) || / {2}/.test(q)) return null;
	return { start: at, query: q };
}

export type Format = 'bold' | 'italic' | 'code' | 'list' | 'link';

/** Applies a toolbar format to text[start, end); returns the new text and selection. */
export function applyFormat(text: string, start: number, end: number, f: Format): { text: string; start: number; end: number } {
	const sel = text.slice(start, end);
	if (f === 'list') {
		const from = text.lastIndexOf('\n', start - 1) + 1;
		const toNl = text.indexOf('\n', Math.max(end - 1, from));
		const to = toNl < 0 ? text.length : toNl;
		const block = text
			.slice(from, to)
			.split('\n')
			.map((l) => (/^\s*[-*]\s/.test(l) ? l : `- ${l}`))
			.join('\n');
		return { text: text.slice(0, from) + block + text.slice(to), start: from, end: from + block.length };
	}
	if (f === 'link') {
		const label = sel || 'link';
		const out = `[${label}](https://)`;
		const urlAt = start + label.length + 3;
		return { text: text.slice(0, start) + out + text.slice(end), start: urlAt, end: urlAt + 8 };
	}
	const mark = f === 'bold' ? '**' : f === 'italic' ? '*' : '`';
	const inner = sel || 'text';
	return { text: text.slice(0, start) + mark + inner + mark + text.slice(end), start: start + mark.length, end: start + mark.length + inner.length };
}
