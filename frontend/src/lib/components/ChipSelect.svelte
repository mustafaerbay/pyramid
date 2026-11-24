<script lang="ts">
	// Pick several values: chips for the picked ones, a select to add more.
	let {
		label,
		options,
		values = $bindable([])
	}: { label: string; options: { value: string; label: string }[]; values: string[] } = $props();

	const name = (v: string) => options.find((o) => o.value === v)?.label ?? v;
</script>

<div class="min-w-0 text-sm">
	<select
		class="input w-full"
		aria-label="Add {label}"
		value=""
		onchange={(e) => {
			const v = e.currentTarget.value;
			if (v && !values.includes(v)) values = [...values, v];
			e.currentTarget.value = '';
		}}
	>
		<option value="">{label}{values.length ? ` (${values.length})` : ''}…</option>
		{#each options.filter((o) => !values.includes(o.value)) as o (o.value)}
			<option value={o.value}>{o.label}</option>
		{/each}
	</select>
	{#if values.length}
		<div class="mt-1 flex flex-wrap gap-1">
			{#each values as v (v)}
				<button
					type="button"
					class="inline-flex min-h-7 max-w-full items-center gap-1 rounded-full bg-slate-100 px-2.5 text-xs text-slate-700 hover:bg-slate-200"
					aria-label="Remove {name(v)} from {label}"
					onclick={() => (values = values.filter((x) => x !== v))}
				>
					<span class="truncate">{name(v)}</span><span aria-hidden="true">×</span>
				</button>
			{/each}
		</div>
	{/if}
</div>
