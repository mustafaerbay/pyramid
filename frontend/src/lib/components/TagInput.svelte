<script lang="ts">
	import { api } from '$lib/api/client';
	import { addTag, MAX_TAG_LENGTH, MAX_TAGS, tagColor } from '$lib/tags';

	let { tags = $bindable([]) }: { tags: string[] } = $props();

	const listId = `tag-suggestions-${Math.random().toString(36).slice(2, 8)}`;
	let draft = $state('');
	let known = $state<string[]>([]);

	$effect(() => {
		api.GET('/api/v1/tags').then(({ data }) => (known = data?.data ?? []));
	});

	const suggestions = $derived(
		known.filter((k) => !tags.some((t) => t.toLowerCase() === k.toLowerCase()))
	);

	function commit() {
		if (tags.length >= MAX_TAGS) return;
		tags = addTag(tags, draft, known);
		draft = '';
	}

	function remove(tag: string) {
		tags = tags.filter((t) => t !== tag);
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			commit();
		} else if (e.key === 'Backspace' && draft === '' && tags.length > 0) {
			tags = tags.slice(0, -1);
		}
	}

	// Picking a suggestion from the datalist fills the input in one go.
	function oninput() {
		if (draft.endsWith(',')) {
			draft = draft.slice(0, -1);
			commit();
		} else if (known.some((k) => k === draft)) {
			commit();
		}
	}
</script>

<div class="input flex min-h-10 flex-wrap items-center gap-1.5 py-1.5">
	{#each tags as tag (tag)}
		<span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset {tagColor(tag)}">
			{tag}
			<button
				type="button"
				class="-mr-1 rounded-full px-1 leading-none opacity-60 hover:opacity-100"
				aria-label="Remove tag {tag}"
				onclick={() => remove(tag)}>×</button
			>
		</span>
	{/each}
	{#if tags.length < MAX_TAGS}
		<input
			class="min-w-24 flex-1 border-0 bg-transparent p-0 text-sm outline-none focus:ring-0"
			list={listId}
			maxlength={MAX_TAG_LENGTH}
			placeholder={tags.length ? '' : 'Type a tag, press Enter'}
			aria-label="Add tag"
			bind:value={draft}
			{onkeydown}
			{oninput}
			onblur={commit}
		/>
	{/if}
	<datalist id={listId}>
		{#each suggestions as s (s)}
			<option value={s}></option>
		{/each}
	</datalist>
</div>
