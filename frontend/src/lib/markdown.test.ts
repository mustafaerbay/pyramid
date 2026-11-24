import { describe, expect, it } from 'vitest';
import { applyFormat, attachmentImage, fillPending, mentionQuery, renderMarkdown } from './markdown';

describe('images', () => {
	const id = '0b7c3a52-4d1e-4f7a-9a3c-2f1d5e6a7b8c';
	const src = `/api/v1/attachments/${id}/download?inline=1`;
	it('shows a Pyramid attachment as an image', () => {
		expect(renderMarkdown(`![login error](${src})`)).toBe(`<p><img src="${src}" alt="login error" loading="lazy"></p>`);
		expect(renderMarkdown(`![a](${src})`, { image: (s) => 'http://api' + s })).toBe(`<p><img src="http://api${src}" alt="a" loading="lazy"></p>`);
	});
	it('never loads an image from elsewhere; it becomes a link', () => {
		const out = renderMarkdown('![x](https://evil.example/p.png)');
		expect(out).not.toContain('<img');
		expect(out).toBe('<p><a href="https://evil.example/p.png" target="_blank" rel="noopener noreferrer">x</a></p>');
		expect(renderMarkdown('![x](javascript:alert(1))')).not.toContain('<img');
		expect(renderMarkdown('![x" onerror="y](/api/v1/attachments/../x)')).not.toContain('<img');
	});
	it('shows a not-yet-uploaded image through the resolver', () => {
		expect(renderMarkdown('![s](pending:2)', { image: (s) => (s === 'pending:2' ? 'blob:http://x/1' : null) })).toBe(
			'<p><img src="blob:http://x/1" alt="s" loading="lazy"></p>'
		);
		expect(renderMarkdown('![s](pending:2)')).not.toContain('<img');
	});
	it('writes and fills image markup', () => {
		expect(attachmentImage(id, 'shot [1].png')).toBe(`![shot 1.png](${src})`);
		expect(fillPending('a ![s](pending:1) b ![t](pending:2)', { 'pending:1': id })).toBe(`a ![s](${src}) b ![t](pending:2)`);
	});
});

describe('renderMarkdown', () => {
	it('escapes HTML before anything else', () => {
		expect(renderMarkdown('<script>alert(1)</script>')).toBe('<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>');
		expect(renderMarkdown('[x](javascript:alert(1))')).toBe('<p>[x](javascript:alert(1))</p>');
	});
	it('formats inline text', () => {
		expect(renderMarkdown('**bold** and *it* and `a<b`')).toBe('<p><strong>bold</strong> and <em>it</em> and <code>a&lt;b</code></p>');
		expect(renderMarkdown('[docs](https://x.dev/a?b=1&c=2)')).toBe(
			'<p><a href="https://x.dev/a?b=1&amp;c=2" target="_blank" rel="noopener noreferrer">docs</a></p>'
		);
	});
	it('builds headings, lists, quotes and code blocks', () => {
		const html = renderMarkdown('# Title\n\n- one\n- two\n\n1. first\n2. second\n\n> said\n\n```\nx < 1\n```');
		expect(html).toBe(
			'<h3>Title</h3><ul><li>one</li><li>two</li></ul><ol><li>first</li><li>second</li></ol><blockquote>said</blockquote><pre><code>x &lt; 1</code></pre>'
		);
	});
	it('keeps line breaks inside a paragraph', () => {
		expect(renderMarkdown('a\nb\n\nc')).toBe('<p>a<br>b</p><p>c</p>');
	});
	it('highlights mentions of known people', () => {
		expect(renderMarkdown('hi @Ada Kim!', { people: ['Ada Kim'] })).toBe('<p>hi <span class="mention">@Ada Kim</span>!</p>');
	});
	it('turns item numbers into links when a resolver is given', () => {
		expect(renderMarkdown('see #12', { item: (n) => (n === 12 ? '/t/12' : null) })).toBe('<p>see <a href="/t/12">#12</a></p>');
	});
});

describe('mentionQuery', () => {
	it('finds the @word being typed before the caret', () => {
		expect(mentionQuery('hello @ad', 9)).toEqual({ start: 6, query: 'ad' });
		expect(mentionQuery('hello @Ada K', 12)).toEqual({ start: 6, query: 'Ada K' });
		expect(mentionQuery('mail a@b.c', 10)).toBeNull();
		expect(mentionQuery('done @ada. next', 15)).toBeNull();
	});
});

describe('applyFormat', () => {
	it('wraps the selection and keeps it selected', () => {
		expect(applyFormat('make it bold', 8, 12, 'bold')).toEqual({ text: 'make it **bold**', start: 10, end: 14 });
		expect(applyFormat('x', 1, 1, 'italic')).toEqual({ text: 'x*text*', start: 2, end: 6 });
		expect(applyFormat('a b', 2, 3, 'code')).toEqual({ text: 'a `b`', start: 3, end: 4 });
	});
	it('turns the selected lines into a list', () => {
		expect(applyFormat('one\ntwo', 0, 7, 'list').text).toBe('- one\n- two');
		expect(applyFormat('intro\nstep', 6, 6, 'list').text).toBe('intro\n- step');
	});
	it('makes a link around the selection', () => {
		expect(applyFormat('see docs', 4, 8, 'link')).toEqual({ text: 'see [docs](https://)', start: 11, end: 19 });
	});
});
