<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { applyFormat, mentionQuery, type Format } from '$lib/markdown';
	import Markdown from './Markdown.svelte';

	// A textarea for Markdown with Write / Preview and @person suggestions.
	let {
		value = $bindable(''),
		people,
		rows = 3,
		placeholder = '',
		label,
		maxlength,
		id,
		onImage,
		images = {},
		autofocus = false
	}: {
		value: string;
		/** Display names ("First Last") to suggest after @. */
		people: string[];
		rows?: number;
		placeholder?: string;
		label: string;
		maxlength?: number;
		id?: string;
		/** Uploads a pasted / dropped / picked image; returns the Markdown to insert, or null. */
		onImage?: (file: File) => Promise<string | null>;
		/** Local previews of images not uploaded yet (pending:N → blob: URL). */
		images?: Record<string, string>;
		autofocus?: boolean;
	} = $props();
	let uploading = $state(0);
	let dragging = $state(false);
	let imageError = $state<string | null>(null);
	let picker = $state<HTMLInputElement | null>(null);
	function setSelection(a: number, b: number) {
		queueMicrotask(() => {
			el?.focus();
			el?.setSelectionRange(a, b);
		});
	}
	function format(f: Format) {
		preview = false;
		const a = el?.selectionStart ?? value.length;
		const b = el?.selectionEnd ?? value.length;
		const r = applyFormat(value, a, b, f);
		value = r.text;
		setSelection(r.start, r.end);
	}
	async function addImages(files: File[]) {
		const list = files.filter((f) => f.type.startsWith('image/'));
		if (!list.length || !onImage) return;
		imageError = null;
		preview = false;
		for (const file of list) {
			if (file.size > 10 << 20) {
				imageError = `${file.name || 'The image'} is over 10 MB.`;
				continue;
			}
			uploading++;
			const md = await onImage(file).catch(() => null);
			uploading--;
			if (!md) {
				imageError = `Could not add ${file.name || 'the image'}.`;
				continue;
			}
			const at = el?.selectionEnd ?? value.length;
			const before = value.slice(0, at);
			const pre = before && !before.endsWith('\n') ? '\n' : '';
			const insert = `${pre}${md}\n`;
			value = before + insert + value.slice(at);
			setSelection(at + insert.length, at + insert.length);
		}
	}
	function paste(e: ClipboardEvent) {
		const files = [...(e.clipboardData?.files ?? [])].filter((f) => f.type.startsWith('image/'));
		if (!files.length || !onImage) return;
		e.preventDefault();
		addImages(files);
	}
	function drop(e: DragEvent) {
		dragging = false;
		const files = [...(e.dataTransfer?.files ?? [])];
		if (!files.some((f) => f.type.startsWith('image/')) || !onImage) return;
		e.preventDefault();
		addImages(files);
	}

	let el = $state<HTMLTextAreaElement | null>(null);
	let preview = $state(false);
	let q = $state<{ start: number; query: string } | null>(null);
	let active = $state(0);
	const matches = $derived(
		q ? people.filter((p) => p.toLowerCase().split(/\s+/).some((w) => w.startsWith(q!.query.toLowerCase().split(' ')[0])) && p.toLowerCase().includes(q!.query.toLowerCase())).slice(0, 6) : []
	);
	function update() {
		if (!el) return;
		q = mentionQuery(value, el.selectionStart ?? value.length);
		active = 0;
	}
	function pick(name: string) {
		if (!q || !el) return;
		const end = el.selectionStart ?? value.length;
		value = `${value.slice(0, q.start)}@${name} ${value.slice(end)}`;
		const caret = q.start + name.length + 2;
		q = null;
		queueMicrotask(() => {
			el?.focus();
			el?.setSelectionRange(caret, caret);
		});
	}
	function keydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
			(e.currentTarget as HTMLTextAreaElement).form?.requestSubmit();
			return;
		}
		if ((e.metaKey || e.ctrlKey) && (e.key === 'b' || e.key === 'i')) {
			e.preventDefault();
			format(e.key === 'b' ? 'bold' : 'italic');
			return;
		}
		if (!matches.length) return;
		if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
			e.preventDefault();
			active = (active + (e.key === 'ArrowDown' ? 1 : matches.length - 1)) % matches.length;
		} else if (e.key === 'Enter' || e.key === 'Tab') {
			e.preventDefault();
			pick(matches[active]);
		} else if (e.key === 'Escape') {
			q = null;
		}
	}
</script>

<div class="relative">
	<div class="mb-1 flex flex-wrap items-center gap-1 text-xs">
		<button type="button" class="inline-flex min-h-7 items-center rounded px-2 {preview ? 'text-slate-500 hover:text-slate-700' : 'bg-slate-100 font-medium text-slate-800'}" onclick={() => (preview = false)}>Write</button>
		<button type="button" class="inline-flex min-h-7 items-center rounded px-2 {preview ? 'bg-slate-100 font-medium text-slate-800' : 'text-slate-500 hover:text-slate-700'}" onclick={() => (preview = true)}>Preview</button>
		<span class="mx-1 h-4 w-px bg-slate-200"></span>
		<button type="button" class="tool font-bold" title="Bold (Cmd/Ctrl+B)" aria-label="Bold" onclick={() => format('bold')}>B</button>
		<button type="button" class="tool italic" title="Italic (Cmd/Ctrl+I)" aria-label="Italic" onclick={() => format('italic')}>I</button>
		<button type="button" class="tool" title="Code" aria-label="Code" onclick={() => format('code')}><Icon name="code" size={14} /></button>
		<button type="button" class="tool" title="List" aria-label="List" onclick={() => format('list')}><Icon name="list" size={14} /></button>
		<button type="button" class="tool" title="Link" aria-label="Link" onclick={() => format('link')}><Icon name="link" size={14} /></button>
		{#if onImage}
			<button type="button" class="tool gap-1 px-2" title="Add an image" aria-label="Add an image" onclick={() => picker?.click()}><Icon name="image" size={14} /><span>Image</span></button>
			<input bind:this={picker} type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple class="sr-only" tabindex="-1" onchange={(e) => { addImages([...(e.currentTarget.files ?? [])]); e.currentTarget.value = ''; }} />
		{/if}
		<span class="ml-auto self-center text-slate-500">{uploading ? 'Uploading image…' : onImage ? 'Markdown · @ to mention · paste images' : 'Markdown · @ to mention'}</span>
	</div>
	{#if imageError}<p class="mb-1 text-xs text-red-600">{imageError}</p>{/if}
	{#if preview}
		<div class="min-h-16 rounded-md border border-slate-200 p-2 text-sm">
			{#if value.trim()}<Markdown text={value} {people} {images} />{:else}<span class="text-slate-500">Nothing to preview.</span>{/if}
		</div>
	{:else}
		<!-- svelte-ignore a11y_autofocus -->
		<textarea
			bind:this={el}
			{id}
			{autofocus}
			class="input {dragging ? 'ring-2 ring-indigo-300' : ''}"
			{rows}
			{placeholder}
			{maxlength}
			aria-label={label}
			aria-autocomplete="list"
			bind:value
			oninput={update}
			onclick={update}
			onkeydown={keydown}
			onpaste={paste}
			ondragover={(e) => {
				if (onImage && e.dataTransfer?.types.includes('Files')) {
					e.preventDefault();
					dragging = true;
				}
			}}
			ondragleave={() => (dragging = false)}
			ondrop={drop}
			onblur={() => setTimeout(() => (q = null), 150)}
		></textarea>
		{#if matches.length}
			<ul class="absolute left-2 z-30 mt-1 w-56 overflow-hidden rounded-md border border-slate-200 bg-white text-sm shadow-lg" role="listbox" aria-label="People">
				{#each matches as m, i (m)}
					<li role="option" aria-selected={i === active}>
						<button type="button" class="block w-full px-3 py-1.5 text-left {i === active ? 'bg-indigo-50 text-indigo-800' : 'hover:bg-slate-50'}" onmousedown={(e) => e.preventDefault()} onclick={() => pick(m)}
							>@{m}</button
						>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</div>
