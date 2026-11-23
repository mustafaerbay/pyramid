<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { runLimited } from '$lib/bulk';
	import { TYPE_OPTIONS } from '$lib/workItems';

	type Task = components['schemas']['Task'];
	type TaskUpdate = components['schemas']['TaskUpdate'];

	let {
		tasks,
		people,
		onDone,
		onClear
	}: { tasks: Task[]; people: { id: string; name: string }[]; onDone: (message: string) => void; onClear: () => void } = $props();

	let busy = $state(false);
	let tag = $state('');

	async function apply(label: string, change: (t: Task) => TaskUpdate | null) {
		busy = true;
		const results = await runLimited(tasks, 5, async (t) => {
			const body = change(t);
			if (!body) return;
			const { error } = await api.PATCH('/api/v1/tasks/{id}', { params: { path: { id: t.id } }, body });
			if (error) throw new Error(`${t.title}: ${error.message}`);
		});
		busy = false;
		report(label, results);
	}
	async function remove() {
		if (!confirm(`Delete ${tasks.length} item${tasks.length === 1 ? '' : 's'}? This can't be undone.`)) return;
		busy = true;
		const results = await runLimited(tasks, 5, async (t) => {
			const { error } = await api.DELETE('/api/v1/tasks/{id}', { params: { path: { id: t.id } } });
			if (error) throw new Error(`${t.title}: ${error.message}`);
		});
		busy = false;
		report('Deleted', results);
	}
	function report(label: string, results: { ok: boolean; error?: unknown }[]) {
		const failed = results.filter((r) => !r.ok);
		const okCount = results.length - failed.length;
		onDone(
			failed.length
				? `${label} ${okCount}; ${failed.length} failed — ${(failed[0].error as Error)?.message ?? 'error'}`
				: `${label} ${okCount} item${okCount === 1 ? '' : 's'}.`
		);
	}
	const pick = (e: Event) => {
		const el = e.currentTarget as HTMLSelectElement;
		const v = el.value;
		el.value = '';
		return v;
	};
</script>

<div class="sticky bottom-2 z-30 flex flex-wrap items-center gap-2 rounded-lg border border-slate-300 bg-white p-2 text-sm shadow-lg" role="toolbar" aria-label="Change selected items">
	<span class="px-1 font-medium">{tasks.length} selected</span>
	<select class="input w-auto" aria-label="Set status" disabled={busy} onchange={(e) => { const v = pick(e); if (v) apply('Updated', () => ({ status: v as Task['status'] })); }}>
		<option value="">Status…</option>
		{#each ['backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'] as s (s)}<option value={s}>{s.replace('_', ' ')}</option>{/each}
	</select>
	<select class="input w-auto" aria-label="Set priority" disabled={busy} onchange={(e) => { const v = pick(e); if (v) apply('Updated', () => ({ priority: v as Task['priority'] })); }}>
		<option value="">Priority…</option>
		{#each ['urgent', 'high', 'medium', 'low'] as p (p)}<option value={p}>{p}</option>{/each}
	</select>
	<select class="input w-auto" aria-label="Set type" disabled={busy} onchange={(e) => { const v = pick(e); if (v) apply('Updated', () => ({ type: v as Task['type'] })); }}>
		<option value="">Type…</option>
		{#each TYPE_OPTIONS as t (t.value)}<option value={t.value}>{t.label}</option>{/each}
	</select>
	<select
		class="input w-auto"
		aria-label="Assign a person"
		disabled={busy}
		onchange={(e) => {
			const v = pick(e);
			if (v)
				apply('Assigned', (t) =>
					t.assigneeIds.includes(v) ? null : { assignments: [...t.assignments.map((a) => ({ personId: a.personId, manDays: a.manDays })), { personId: v, manDays: 0 }] }
				);
		}}
	>
		<option value="">Assign…</option>
		{#each people as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
	</select>
	<span class="flex items-center gap-1">
		<input class="input w-28" placeholder="Tag" aria-label="Tag to add or remove" bind:value={tag} />
		<button type="button" class="btn-secondary px-2" disabled={busy || !tag.trim()} onclick={() => apply('Tagged', (t) => (t.tags.some((x) => x.toLowerCase() === tag.trim().toLowerCase()) ? null : { tags: [...t.tags, tag.trim()] }))}>+ Tag</button>
		<button type="button" class="btn-secondary px-2" disabled={busy || !tag.trim()} onclick={() => apply('Untagged', (t) => (t.tags.some((x) => x.toLowerCase() === tag.trim().toLowerCase()) ? { tags: t.tags.filter((x) => x.toLowerCase() !== tag.trim().toLowerCase()) } : null))}>− Tag</button>
	</span>
	<button type="button" class="btn-secondary text-red-600 hover:bg-red-50" disabled={busy} onclick={remove}>Delete</button>
	<button type="button" class="btn-ghost ml-auto" onclick={onClear}>{busy ? 'Working…' : 'Clear'}</button>
</div>
