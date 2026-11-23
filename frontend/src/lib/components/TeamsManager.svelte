<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ChipSelect from './ChipSelect.svelte';

	type Team = components['schemas']['ProjectTeam'];

	let {
		projectId,
		teams,
		people,
		areas,
		onChange,
		onClose
	}: {
		projectId: string;
		teams: Team[];
		people: { id: string; name: string }[];
		areas: { id: string; path: string }[];
		onChange: () => void;
		onClose: () => void;
	} = $props();

	let editing = $state<Team | null>(null);
	let name = $state('');
	let memberIds = $state<string[]>([]);
	let areaIds = $state<string[]>([]);
	let error = $state<string | null>(null);

	function edit(t: Team | null) {
		editing = t;
		name = t?.name ?? '';
		memberIds = [...(t?.memberIds ?? [])];
		areaIds = [...(t?.areaIds ?? [])];
		error = null;
	}
	async function save(e: SubmitEvent) {
		e.preventDefault();
		const body = { projectId, name, memberIds, areaIds };
		const res = editing
			? await api.PATCH('/api/v1/project-teams/{id}', { params: { path: { id: editing.id } }, body })
			: await api.POST('/api/v1/project-teams', { body });
		if (res.error) return (error = res.error.message ?? 'Could not save the team.');
		edit(null);
		onChange();
	}
	async function remove(t: Team) {
		if (!confirm(`Delete team "${t.name}"? Its sprints stay, without a team.`)) return;
		await api.DELETE('/api/v1/project-teams/{id}', { params: { path: { id: t.id } } });
		if (editing?.id === t.id) edit(null);
		onChange();
	}
	const personName = (id: string) => people.find((p) => p.id === id)?.name ?? 'Someone';
	const areaPath = (id: string) => areas.find((a) => a.id === id)?.path ?? '?';
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Teams">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Teams</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">
			A team owns area paths: its backlog and board show the items in those areas (and below). A team without areas sees
			what its members are assigned. Sprints can belong to a team.
		</p>
		<ul class="mb-4 divide-y divide-slate-100 rounded-md border border-slate-200 text-sm">
			{#each teams as t (t.id)}
				<li class="flex items-start gap-2 px-3 py-2">
					<span class="min-w-0 flex-1">
						<span class="font-medium">{t.name}</span>
						<span class="block text-xs text-slate-500">
							{t.memberIds.map(personName).join(', ') || 'No members'} · {t.areaIds.map(areaPath).join(', ') || 'no areas'}
						</span>
					</span>
					<button type="button" class="text-xs text-slate-600 hover:underline" onclick={() => edit(t)}>Edit</button>
					<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => remove(t)}>Delete</button>
				</li>
			{:else}
				<li class="px-3 py-2 text-slate-500">No teams yet.</li>
			{/each}
		</ul>
		<form class="space-y-3 rounded-md bg-slate-50 p-3 text-sm" onsubmit={save}>
			<div class="font-medium">{editing ? `Edit ${editing.name}` : 'New team'}</div>
			{#if error}<p class="text-red-700" role="alert">{error}</p>{/if}
			<input class="input" placeholder="Team name" aria-label="Team name" required maxlength="60" bind:value={name} />
			<ChipSelect label="Members" options={people.map((p) => ({ value: p.id, label: p.name }))} bind:values={memberIds} />
			<ChipSelect label="Areas" options={areas.map((a) => ({ value: a.id, label: a.path }))} bind:values={areaIds} />
			<div class="flex gap-2">
				<button type="submit" class="btn-primary">{editing ? 'Save team' : 'Add team'}</button>
				{#if editing}<button type="button" class="btn-ghost" onclick={() => edit(null)}>Cancel</button>{/if}
			</div>
		</form>
	</div>
</div>
