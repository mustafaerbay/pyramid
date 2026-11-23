<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { areaLabel } from '$lib/areas';

	type Area = components['schemas']['Area'];

	let { projectId, areas, onChange, onClose }: { projectId: string; areas: Area[]; onChange: () => void; onClose: () => void } =
		$props();

	let newPath = $state('');
	let editingId = $state<string | null>(null);
	let editPath = $state('');
	let error = $state<string | null>(null);

	async function add(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		const { error: err } = await api.POST('/api/v1/areas', { body: { projectId, path: newPath } });
		if (err) return (error = err.message ?? 'Could not add the area.');
		newPath = '';
		onChange();
	}
	async function rename(a: Area) {
		error = null;
		const { error: err } = await api.PATCH('/api/v1/areas/{id}', { params: { path: { id: a.id } }, body: { path: editPath } });
		if (err) return (error = err.message ?? 'Could not rename the area.');
		editingId = null;
		onChange();
	}
	async function remove(a: Area) {
		if (!confirm(`Delete area "${a.path}"? Its items keep going but lose their area.`)) return;
		error = null;
		const { error: err } = await api.DELETE('/api/v1/areas/{id}', { params: { path: { id: a.id } } });
		if (err) return (error = err.message ?? 'Could not delete the area.');
		onChange();
	}
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 p-0 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Areas">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-lg sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Areas</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">Group work by product area. Use "/" for levels, e.g. Web/Checkout (up to 4 levels).</p>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}
		<form class="mb-4 flex gap-2" onsubmit={add}>
			<input class="input flex-1" placeholder="New area, e.g. Web/Checkout" aria-label="New area path" required bind:value={newPath} />
			<button type="submit" class="btn-primary">Add</button>
		</form>
		{#if areas.length === 0}
			<p class="text-sm text-slate-500">No areas yet.</p>
		{/if}
		<ul class="divide-y divide-slate-100 text-sm">
			{#each areas as a (a.id)}
				<li class="flex items-center gap-2 py-2">
					{#if editingId === a.id}
						<input class="input flex-1" aria-label="Area path" bind:value={editPath} />
						<button type="button" class="btn-primary px-3 py-1 text-xs" onclick={() => rename(a)}>Save</button>
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (editingId = null)}>Cancel</button>
					{:else}
						<span class="min-w-0 flex-1 truncate whitespace-pre" title={a.path}>{areaLabel(a.path)}</span>
						<button type="button" class="text-xs text-slate-500 hover:underline" onclick={() => ((editingId = a.id), (editPath = a.path))}>Rename</button>
						<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => remove(a)}>Delete</button>
					{/if}
				</li>
			{/each}
		</ul>
	</div>
</div>
