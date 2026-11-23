<script lang="ts">
	// Chips for picked tasks plus a select to add one more.
	type Option = { id: string; number: number; title: string };

	let {
		label,
		placeholder,
		ids = $bindable([]),
		options,
		all,
		tone = 'slate'
	}: {
		label: string;
		placeholder: string;
		ids: string[];
		/** Tasks that may be added. */
		options: Option[];
		/** Every task, to name the picked ones. */
		all: Option[];
		tone?: 'slate' | 'amber';
	} = $props();

	const ref = (id: string) => {
		const t = all.find((x) => x.id === id);
		return t ? `#${t.number} ${t.title}` : 'Unknown task';
	};
	const chip = $derived(
		tone === 'amber' ? 'bg-amber-50 text-amber-800 ring-amber-200' : 'bg-slate-100 text-slate-700 ring-slate-200'
	);
</script>

<div class="block text-sm">
	<span class="mb-1 block font-medium text-slate-700">{label}</span>
	{#if ids.length > 0}
		<ul class="mb-2 flex flex-wrap gap-1.5">
			{#each ids as id (id)}
				<li class="inline-flex max-w-full items-center gap-1 rounded-full px-2 py-0.5 text-xs ring-1 ring-inset {chip}">
					<span class="truncate">{ref(id)}</span>
					<button
						type="button"
						class="-mr-1 rounded-full px-1 leading-none opacity-60 hover:opacity-100"
						aria-label="Remove {ref(id)} from {label}"
						onclick={() => (ids = ids.filter((x) => x !== id))}>×</button
					>
				</li>
			{/each}
		</ul>
	{/if}
	<select
		class="input"
		aria-label="Add to {label}"
		value=""
		onchange={(e) => {
			const v = e.currentTarget.value;
			if (v && !ids.includes(v)) ids = [...ids, v];
			e.currentTarget.value = '';
		}}
	>
		<option value="">{placeholder}</option>
		{#each options.filter((t) => !ids.includes(t.id)) as t (t.id)}
			<option value={t.id}>#{t.number} {t.title}</option>
		{/each}
	</select>
</div>
