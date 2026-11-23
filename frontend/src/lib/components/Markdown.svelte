<script lang="ts">
	import { renderMarkdown, type MarkdownOptions } from '$lib/markdown';

	let { text, people = [], item, class: cls = '' }: { text: string; people?: string[]; item?: MarkdownOptions['item']; class?: string } = $props();
	// renderMarkdown escapes all input; only its own markup is emitted.
	const html = $derived(renderMarkdown(text ?? '', { people, item }));
</script>

<div class="md {cls}">{@html html}</div>
