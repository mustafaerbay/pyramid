<script lang="ts">
	import Self from './QueryGroupEditor.svelte';
	import { FIELDS, OPS, needsValue, type FieldKind, type Group, type Op } from '$lib/querylang';
	import { TYPE_OPTIONS } from '$lib/workItems';

	type Custom = { key: string; label: string; type: string; options?: string[] | null };
	let {
		group = $bindable(),
		depth = 1,
		people,
		tags,
		customFields,
		onRemove
	}: {
		group: Group;
		depth?: number;
		people: { id: string; name: string }[];
		tags: string[];
		customFields: Custom[];
		onRemove?: () => void;
	} = $props();

	const CHOICES: Record<string, { value: string; label: string }[]> = {
		type: TYPE_OPTIONS.map((t) => ({ value: t.value, label: t.label })),
		status: ['backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'].map((s) => ({ value: s, label: s.replace('_', ' ') })),
		priority: ['urgent', 'high', 'medium', 'low'].map((s) => ({ value: s, label: s }))
	};
	const allFields = $derived([
		...FIELDS,
		...customFields.map((f) => ({
			key: `custom:${f.key}`,
			label: f.label,
			kind: (f.type === 'number' ? 'number' : f.type === 'date' ? 'date' : 'text') as FieldKind,
			options: f.type === 'select' ? (f.options ?? []) : null
		}))
	]);
	const fieldOf = (key: string) => allFields.find((f) => f.key === key) ?? FIELDS[0];
	const optionsOf = (key: string) => (fieldOf(key) as { options?: string[] | null }).options ?? null;

	function setField(i: number, key: string) {
		const ops = OPS[fieldOf(key).kind];
		group.conditions[i] = { field: key, op: ops[0].op, value: '' };
	}
	const add = () => (group.conditions = [...group.conditions, { field: 'title', op: 'contains', value: '' }]);
	const addGroup = () =>
		(group.groups = [...(group.groups ?? []), { match: group.match === 'all' ? 'any' : 'all', conditions: [{ field: 'title', op: 'contains', value: '' }] }]);
</script>

<div class="space-y-2 rounded-md border border-slate-200 p-2 {depth > 1 ? 'bg-slate-50' : 'bg-white'}">
	<div class="flex flex-wrap items-center gap-2 text-xs text-slate-600">
		<span>Match</span>
		<select class="input w-auto py-1 text-xs" aria-label="Match" bind:value={group.match}>
			<option value="all">all (AND)</option>
			<option value="any">any (OR)</option>
		</select>
		<span>of these</span>
		{#if onRemove}<button type="button" class="ml-auto text-slate-500 hover:text-red-600" onclick={onRemove}>Remove group</button>{/if}
	</div>
	{#each group.conditions as c, i (i)}
		{@const f = fieldOf(c.field)}
		{@const opts = optionsOf(c.field)}
		<div class="flex flex-wrap items-center gap-1.5">
			<select class="input w-auto py-1 text-xs" aria-label="Field" value={c.field} onchange={(e) => setField(i, e.currentTarget.value)}>
				{#each allFields as x (x.key)}<option value={x.key}>{x.label}</option>{/each}
			</select>
			<select class="input w-auto py-1 text-xs" aria-label="Operator" bind:value={c.op}>
				{#each OPS[f.kind] as o (o.op)}<option value={o.op}>{o.label}</option>{/each}
			</select>
			{#if needsValue(c.op as Op)}
				{#if CHOICES[c.field]}
					<select class="input w-auto py-1 text-xs" aria-label="Value" bind:value={c.value}>
						<option value="">…</option>
						{#each CHOICES[c.field] as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
					</select>
				{:else if opts}
					<select class="input w-auto py-1 text-xs" aria-label="Value" bind:value={c.value}>
						<option value="">…</option>
						{#each opts as o (o)}<option value={o}>{o}</option>{/each}
					</select>
				{:else if f.kind === 'person'}
					<select class="input w-auto py-1 text-xs" aria-label="Person" bind:value={c.value}>
						<option value="">…</option>
						{#each people as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
					</select>
				{:else if f.kind === 'tag'}
					<select class="input w-auto py-1 text-xs" aria-label="Tag" bind:value={c.value}>
						<option value="">…</option>
						{#each tags as t (t)}<option value={t}>{t}</option>{/each}
					</select>
				{:else if f.kind === 'date'}
					<input class="input w-40 py-1 text-xs" aria-label="Date" placeholder="2026-12-01 or @today-7" bind:value={c.value} />
				{:else if f.kind === 'number'}
					<input class="input w-24 py-1 text-xs" type="number" step="any" aria-label="Number" bind:value={c.value} />
				{:else}
					<input class="input w-40 py-1 text-xs" aria-label="Value" bind:value={c.value} />
				{/if}
			{/if}
			<button
				type="button"
				class="px-1 text-slate-400 hover:text-red-600"
				aria-label="Remove condition"
				onclick={() => (group.conditions = group.conditions.filter((_, j) => j !== i))}>✕</button
			>
		</div>
	{/each}
	{#each group.groups ?? [] as _, gi (gi)}
		<Self
			bind:group={group.groups![gi]}
			depth={depth + 1}
			{people}
			{tags}
			{customFields}
			onRemove={() => (group.groups = (group.groups ?? []).filter((_, j) => j !== gi))}
		/>
	{/each}
	<div class="flex gap-2 text-xs">
		<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={add}>+ Condition</button>
		{#if depth < 3}<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={addGroup}>+ Group</button>{/if}
	</div>
</div>
