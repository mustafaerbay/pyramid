<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { fieldKey } from '$lib/customFields';
	import { TYPE_OPTIONS } from '$lib/workItems';
	import TypeBadge from './TypeBadge.svelte';

	type Def = components['schemas']['WorkItemTypeDef'];

	let { types, onSave, onClose }: { types: Def[]; onSave: (t: Def[]) => Promise<string | null>; onClose: () => void } = $props();

	// svelte-ignore state_referenced_locally
	let list = $state<Def[]>(types.map((t) => ({ ...t })));
	let label = $state('');
	let base = $state<Def['base']>('task');
	let color = $state<Def['color']>('pink');
	let error = $state<string | null>(null);
	let saving = $state(false);
	const COLORS: Def['color'][] = ['orange', 'violet', 'sky', 'red', 'amber', 'emerald', 'pink', 'slate'];
	const builtIn = TYPE_OPTIONS.map((t) => t.value as string);

	function add(e: SubmitEvent) {
		e.preventDefault();
		if (!label.trim()) return;
		list = [...list, { key: fieldKey(label, [...list.map((t) => t.key), ...builtIn]), label: label.trim(), base, color }];
		label = '';
	}
	async function save() {
		saving = true;
		error = await onSave(list);
		saving = false;
		if (!error) onClose();
	}
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Work item types">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Work item types</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">
			This project's own types, next to Epic, Feature, Story, Bug and Task. Each behaves like the type it's based on (where
			it can sit, how its work counts). A type still used by items can't be removed.
		</p>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}
		<ul class="mb-4 space-y-2">
			{#each list as t, i (t.key)}
				<li class="flex flex-wrap items-center gap-2 text-sm">
					<TypeBadge type={t.base} label={t.label} color={t.color} />
					<input class="input min-w-0 flex-1" maxlength="30" aria-label="Name of type {i + 1}" bind:value={list[i].label} />
					<select class="input w-auto" aria-label="Based on" bind:value={list[i].base}>
						{#each TYPE_OPTIONS as o (o.value)}<option value={o.value}>like {o.label}</option>{/each}
					</select>
					<select class="input w-auto" aria-label="Colour" bind:value={list[i].color}>
						{#each COLORS as c (c)}<option value={c}>{c}</option>{/each}
					</select>
					<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => (list = list.filter((x) => x.key !== t.key))}>Remove</button>
				</li>
			{:else}
				<li class="text-sm text-slate-500">No types of its own yet.</li>
			{/each}
		</ul>
		{#if list.length < 20}
			<form class="mb-4 flex flex-wrap gap-2 rounded-md bg-slate-50 p-3 text-sm" onsubmit={add}>
				<input class="input min-w-0 flex-1" placeholder="New type, e.g. Spike" aria-label="New type name" maxlength="30" bind:value={label} />
				<select class="input w-auto" aria-label="New type based on" bind:value={base}>
					{#each TYPE_OPTIONS as o (o.value)}<option value={o.value}>like {o.label}</option>{/each}
				</select>
				<select class="input w-auto" aria-label="New type colour" bind:value={color}>
					{#each COLORS as c (c)}<option value={c}>{c}</option>{/each}
				</select>
				<button type="submit" class="btn-secondary">Add type</button>
			</form>
		{/if}
		<div class="flex gap-2">
			<button type="button" class="btn-primary" disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save types'}</button>
			<button type="button" class="btn-ghost" onclick={onClose}>Cancel</button>
		</div>
	</div>
</div>
