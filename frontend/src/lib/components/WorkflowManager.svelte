<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { fieldKey } from '$lib/customFields';

	type State = components['schemas']['WorkflowState'];

	let { states, onSave, onClose }: { states: State[]; onSave: (s: State[]) => Promise<string | null>; onClose: () => void } = $props();

	// svelte-ignore state_referenced_locally
	let list = $state<State[]>(states.map((s) => ({ ...s })));
	let newLabel = $state('');
	let newCategory = $state<State['category']>('in_progress');
	let error = $state<string | null>(null);
	let saving = $state(false);

	const CATEGORIES: { value: State['category']; label: string }[] = [
		{ value: 'backlog', label: 'Backlog' },
		{ value: 'todo', label: 'To do' },
		{ value: 'in_progress', label: 'In progress' },
		{ value: 'in_review', label: 'In review' },
		{ value: 'blocked', label: 'Blocked' },
		{ value: 'done', label: 'Done' },
		{ value: 'cancelled', label: 'Cancelled' }
	];

	function move(i: number, dir: -1 | 1) {
		const j = i + dir;
		if (j < 0 || j >= list.length) return;
		const next = [...list];
		[next[i], next[j]] = [next[j], next[i]];
		list = next;
	}
	function add(e: SubmitEvent) {
		e.preventDefault();
		if (!newLabel.trim()) return;
		list = [...list, { key: fieldKey(newLabel, list.map((s) => s.key)), label: newLabel.trim(), category: newCategory }];
		newLabel = '';
	}
	async function save() {
		saving = true;
		error = await onSave(list);
		saving = false;
		if (!error) onClose();
	}
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Workflow">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Workflow</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">
			This project's states, in board order. Each belongs to a category that capacity, sprints and charts understand. A
			state still holding items can't be removed — move them first.
		</p>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}
		<ul class="mb-4 space-y-2">
			{#each list as s, i (s.key)}
				<li class="flex flex-wrap items-center gap-2 text-sm">
					<span class="flex flex-col">
						<button type="button" class="px-1 leading-none text-slate-400 hover:text-slate-700 disabled:opacity-30" aria-label="Move {s.label} up" disabled={i === 0} onclick={() => move(i, -1)}>▲</button>
						<button type="button" class="px-1 leading-none text-slate-400 hover:text-slate-700 disabled:opacity-30" aria-label="Move {s.label} down" disabled={i === list.length - 1} onclick={() => move(i, 1)}>▼</button>
					</span>
					<input class="input min-w-0 flex-1" maxlength="40" aria-label="Name of state {i + 1}" bind:value={list[i].label} />
					<select class="input w-auto" aria-label="Category of {s.label}" bind:value={list[i].category}>
						{#each CATEGORIES as c (c.value)}<option value={c.value}>{c.label}</option>{/each}
					</select>
					<button type="button" class="text-xs text-red-600 hover:underline disabled:opacity-30" disabled={list.length <= 2} onclick={() => (list = list.filter((x) => x.key !== s.key))}>Remove</button>
				</li>
			{/each}
		</ul>
		{#if list.length < 20}
			<form class="mb-4 flex flex-wrap gap-2 rounded-md bg-slate-50 p-3 text-sm" onsubmit={add}>
				<input class="input min-w-0 flex-1" placeholder="New state, e.g. Code review" aria-label="New state name" maxlength="40" bind:value={newLabel} />
				<select class="input w-auto" aria-label="New state category" bind:value={newCategory}>
					{#each CATEGORIES as c (c.value)}<option value={c.value}>{c.label}</option>{/each}
				</select>
				<button type="submit" class="btn-secondary">Add state</button>
			</form>
		{/if}
		<div class="flex gap-2">
			<button type="button" class="btn-primary" disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save workflow'}</button>
			<button type="button" class="btn-ghost" onclick={onClose}>Cancel</button>
		</div>
	</div>
</div>
