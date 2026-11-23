import { describe, expect, it } from 'vitest';
import { mentionQuery, renderMarkdown } from './markdown';

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
