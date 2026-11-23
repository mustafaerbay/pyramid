<script lang="ts">
	import { mentionQuery } from '$lib/markdown';
	import Markdown from './Markdown.svelte';

	// A textarea for Markdown with Write / Preview and @person suggestions.
	let {
		value = $bindable(''),
		people,
		rows = 3,
		placeholder = '',
		label,
		maxlength,
		id
	}: {
		value: string;
		/** Display names ("First Last") to suggest after @. */
		people: string[];
		rows?: number;
		placeholder?: string;
		label: string;
		maxlength?: number;
		id?: string;
	} = $props();

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
	<div class="mb-1 flex gap-1 text-xs">
		<button type="button" class="rounded px-2 py-0.5 {preview ? 'text-slate-500 hover:text-slate-700' : 'bg-slate-100 font-medium text-slate-800'}" onclick={() => (preview = false)}>Write</button>
		<button type="button" class="rounded px-2 py-0.5 {preview ? 'bg-slate-100 font-medium text-slate-800' : 'text-slate-500 hover:text-slate-700'}" onclick={() => (preview = true)}>Preview</button>
		<span class="ml-auto self-center text-slate-400">Markdown · @ to mention</span>
	</div>
	{#if preview}
		<div class="min-h-16 rounded-md border border-slate-200 p-2 text-sm">
			{#if value.trim()}<Markdown text={value} {people} />{:else}<span class="text-slate-400">Nothing to preview.</span>{/if}
		</div>
	{:else}
		<textarea
			bind:this={el}
			{id}
			class="input"
			{rows}
			{placeholder}
			{maxlength}
			aria-label={label}
			aria-autocomplete="list"
			bind:value
			oninput={update}
			onclick={update}
			onkeydown={keydown}
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
