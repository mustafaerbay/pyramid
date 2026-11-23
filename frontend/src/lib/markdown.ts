// A small, safe Markdown renderer for descriptions, comments, notes and the
// wiki. Everything is HTML-escaped first; only the markup made here is
// emitted, and links must be http(s), mailto or a path in the app.

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

export type MarkdownOptions = {
	/** Names of people, highlighted when written as @Name. */
	people?: string[];
	/** Link for an item number (#12), or null. */
	item?: (n: number) => string | null;
};

function inline(raw: string, o: MarkdownOptions): string {
	// Code spans first, kept out of the other rules.
	const codes: string[] = [];
	let s = raw.replace(/`([^`]+)`/g, (_, c) => {
		codes.push(`<code>${esc(c)}</code>`);
		return `\u0000${codes.length - 1}\u0000`;
	});
	const links: string[] = [];
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
