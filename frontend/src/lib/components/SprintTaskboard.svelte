<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import TypeBadge from './TypeBadge.svelte';

	type Task = components['schemas']['Task'];
	type State = components['schemas']['WorkflowState'];

	let {
		items,
		workflow,
		people,
		onMove
	}: { items: Task[]; workflow: State[]; people: Record<string, string>; onMove: (t: Task, state: string) => void } = $props();

	const columns = $derived(workflow.filter((s) => s.category !== 'cancelled'));
	// Rows: sprint items with children in the sprint; the rest share one row.
	const ids = $derived(new Set(items.map((t) => t.id)));
	const parents = $derived(items.filter((t) => items.some((c) => c.parentId === t.id)));
	const loose = $derived(items.filter((t) => !parents.includes(t) && !(t.parentId && ids.has(t.parentId))));
	const rows = $derived([
		...parents.map((p) => ({ key: p.id, parent: p as Task | null, cards: items.filter((c) => c.parentId === p.id) })),
		...(loose.length ? [{ key: '-', parent: null, cards: loose }] : [])
	]);
	let dragging = $state<string | null>(null);
	let over = $state<string | null>(null);
</script>

<div class="overflow-x-auto">
	<table class="w-full min-w-[48rem] table-fixed border-separate border-spacing-1 text-sm">
		<thead>
			<tr>
				<th class="w-48 text-left text-xs font-medium text-slate-500">Item</th>
				{#each columns as c (c.key)}<th class="text-left text-xs font-medium text-slate-500">{c.label}</th>{/each}
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.key)}
				<tr>
					<td class="rounded-md bg-slate-50 p-2 align-top">
						{#if row.parent}
							<TypeBadge type={row.parent.type} label={row.parent.customTypeLabel} color={row.parent.customTypeColor} />
							<div class="mt-1 font-medium text-slate-800">{row.parent.title}</div>
							<div class="text-xs text-slate-500">{workflow.find((w) => w.key === row.parent?.state)?.label}</div>
						{:else}
							<div class="text-slate-500">Other items</div>
						{/if}
					</td>
					{#each columns as c (c.key)}
						{@const cellKey = `${row.key}|${c.key}`}
						<td
							class="rounded-md p-1 align-top {over === cellKey ? 'bg-indigo-50 ring-2 ring-indigo-200' : 'bg-slate-50/60'}"
							ondragover={(e) => {
								if (!dragging) return;
								e.preventDefault();
								over = cellKey;
							}}
							ondragleave={() => (over = null)}
							ondrop={(e) => {
								e.preventDefault();
								const t = items.find((x) => x.id === dragging);
								dragging = null;
								over = null;
								if (t && t.state !== c.key) onMove(t, c.key);
							}}
						>
							{#each row.cards.filter((t) => t.state === c.key) as t (t.id)}
								<div
									class="mb-1 cursor-grab rounded border border-slate-200 bg-white p-1.5 text-xs shadow-sm {dragging === t.id ? 'opacity-50' : ''}"
									draggable={can('member')}
									role="listitem"
									ondragstart={() => (dragging = t.id)}
									ondragend={() => (dragging = null)}
								>
									<div class="font-medium text-slate-800">{t.title}</div>
									<div class="text-slate-500">
										{t.assigneeIds.map((p) => people[p]?.split(' ')[0] ?? '?').join(', ') || '—'}{t.remainingManDays != null ? ` · ${t.remainingManDays} md left` : ''}
									</div>
								</div>
							{/each}
						</td>
					{/each}
				</tr>
			{:else}
				<tr><td colspan={columns.length + 1} class="p-4 text-center text-slate-500">No items in this sprint.</td></tr>
			{/each}
		</tbody>
	</table>
</div>
