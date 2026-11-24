<script lang="ts">
	import { renderMarkdown, type MarkdownOptions } from '$lib/markdown';
	import { baseUrl } from '$lib/api/client';

	let {
		text,
		people = [],
		item,
		images = {},
		class: cls = ''
	}: {
		text: string;
		people?: string[];
		item?: MarkdownOptions['item'];
		/** Local previews (pending:N → blob: URL) of images not uploaded yet. */
		images?: Record<string, string>;
		class?: string;
	} = $props();
	const image = (src: string) => (src.startsWith('pending:') ? (images[src] ?? null) : baseUrl + src);
	// renderMarkdown escapes all input; only its own markup is emitted.
	const html = $derived(renderMarkdown(text ?? '', { people, item, image }));
	let zoomed = $state<{ src: string; alt: string } | null>(null);
	function open(e: MouseEvent) {
		const t = e.target as HTMLElement;
		if (t instanceof HTMLImageElement) zoomed = { src: t.src, alt: t.alt };
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="md {cls}" onclick={open}>{@html html}</div>

<svelte:window onkeydown={(e) => zoomed && e.key === 'Escape' && (zoomed = null)} />

{#if zoomed}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-[60] flex items-center justify-center bg-slate-900/80 p-4" onclick={() => (zoomed = null)}>
		<figure class="max-h-full max-w-full">
			<img src={zoomed.src} alt={zoomed.alt} class="max-h-[85vh] max-w-full rounded bg-white object-contain shadow-2xl" />
			<figcaption class="mt-2 flex justify-between gap-4 text-sm text-white">
				<span class="truncate">{zoomed.alt}</span>
				<a class="shrink-0 underline" href={zoomed.src} target="_blank" rel="noopener noreferrer" onclick={(e) => e.stopPropagation()}>Open in a new tab</a>
			</figcaption>
		</figure>
	</div>
{/if}
