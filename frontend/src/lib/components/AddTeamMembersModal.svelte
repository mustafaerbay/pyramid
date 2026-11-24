<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { PROJECT_ROLE_OPTIONS, projectRoleLabel } from '$lib/projectRoles';

	type Person = components['schemas']['Person'];
	type ProjectRole = components['schemas']['ProjectRole'];
	type ProjectMember = components['schemas']['ProjectMember'];

	let {
		projectId,
		existingPersonIds = [],
		onClose,
		onSaved
	}: {
		projectId: string;
		// People already actively on this project, excluded from the candidate list.
		existingPersonIds?: string[];
		onClose: () => void;
		// Called once with however many members were successfully created, so the
		// parent can refresh its list and show a summary banner.
		onSaved: (addedCount: number) => void;
	} = $props();

	let people = $state<Person[]>([]);
	let loadingPeople = $state(true);

	// personId -> total allocation % committed on other active projects, so a
	// PM can spot an over-allocation risk before batch-adding someone.
	let elsewhereAllocation = $state<Record<string, number>>({});
	let elsewhereCount = $state<Record<string, number>>({});

	let search = $state('');
	let selected = $state<Set<string>>(new Set());

	let role = $state<ProjectRole>('contributor');
	let allocationPct = $state(100);

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const candidates = $derived(people.filter((p) => !existingPersonIds.includes(p.id)));

	const filtered = $derived(
		(() => {
			const q = search.trim().toLowerCase();
			if (!q) return candidates;
			return candidates.filter((p) => {
				const name = `${p.firstName} ${p.lastName}`.toLowerCase();
				return (
					name.includes(q) ||
					p.email.toLowerCase().includes(q) ||
					(p.title ?? '').toLowerCase().includes(q)
				);
			});
		})()
	);

	const allVisibleSelected = $derived(
		filtered.length > 0 && filtered.every((p) => selected.has(p.id))
	);

	function toggleOne(id: string) {
		const next = new Set(selected);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selected = next;
	}

	function toggleAllVisible() {
		const next = new Set(selected);
		if (allVisibleSelected) {
			for (const p of filtered) next.delete(p.id);
		} else {
			for (const p of filtered) next.add(p.id);
		}
		selected = next;
	}

	function fullName(p: Person): string {
		return `${p.firstName} ${p.lastName}`;
	}

	onMount(async () => {
		loadingPeople = true;
		const [peopleRes, membersRes] = await Promise.all([
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/project-members', { params: { query: { pageSize: 100 } } })
		]);
		if (peopleRes.data) people = peopleRes.data.data;
		if (membersRes.data) {
			const alloc: Record<string, number> = {};
			const count: Record<string, number> = {};
			for (const m of membersRes.data.data) {
				if (m.leftAt) continue;
				if (m.projectId === projectId) continue;
				alloc[m.personId] = (alloc[m.personId] ?? 0) + m.allocationPct;
				count[m.personId] = (count[m.personId] ?? 0) + 1;
			}
			elsewhereAllocation = alloc;
			elsewhereCount = count;
		}
		loadingPeople = false;
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (selected.size === 0) {
			formError = 'Select at least one person to add.';
			return;
		}

		saving = true;
		try {
			const ids = Array.from(selected);
			const results = await Promise.allSettled(
				ids.map((personId) =>
					api.POST('/api/v1/project-members', {
						body: { projectId, personId, role, allocationPct }
					})
				)
			);

			let succeeded = 0;
			const failedNames: string[] = [];
			results.forEach((r, i) => {
				const ok = r.status === 'fulfilled' && !r.value.error && r.value.data;
				if (ok) {
					succeeded += 1;
				} else {
					const p = people.find((person) => person.id === ids[i]);
					failedNames.push(p ? fullName(p) : ids[i]);
				}
			});

			if (succeeded > 0) onSaved(succeeded);

			if (failedNames.length > 0) {
				formError = `Couldn't add: ${failedNames.join(', ')}.`;
				// Drop the successfully-added people from the selection so a retry
				// only targets what actually failed.
				const next = new Set(selected);
				for (const id of ids) {
					const p = people.find((person) => person.id === id);
					if (p && !failedNames.includes(fullName(p))) next.delete(id);
				}
				selected = next;
			} else {
				onClose();
			}
		} finally {
			saving = false;
		}
	}
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onClose()} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 sm:p-4"
	role="presentation"
	onclick={onClose}
>
	<div
		class="flex h-dvh max-h-dvh w-full max-w-2xl flex-col overflow-hidden bg-white shadow-xl sm:h-auto sm:max-h-[90vh] sm:rounded-lg"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="add-members-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between border-b border-slate-200 px-6 py-4">
			<h3 id="add-members-modal-title" class="text-lg font-semibold text-slate-900">
				Add people to project
			</h3>
			<button
				type="button"
				class="text-slate-500 hover:text-slate-600"
				onclick={onClose}
				aria-label="Close"
			>
				✕
			</button>
		</div>

		<form class="flex min-h-0 flex-1 flex-col" onsubmit={handleSubmit}>
			<div class="space-y-3 px-6 pt-4">
				{#if formError}
					<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
						{formError}
					</div>
				{/if}

				<input
					class="input"
					type="search"
					placeholder="Search by name, email, or title…"
					bind:value={search}
				/>

				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Role for selected</span>
						<select class="input" bind:value={role}>
							{#each PROJECT_ROLE_OPTIONS as opt (opt)}
								<option value={opt}>{projectRoleLabel[opt]}</option>
							{/each}
						</select>
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Allocation for selected</span>
						<div class="flex items-center gap-2">
							<input
								type="number"
								min="0"
								max="100"
								step="5"
								class="input"
								bind:value={allocationPct}
							/>
							<span class="text-sm text-slate-500">%</span>
						</div>
					</label>
				</div>
			</div>

			<div class="mt-3 min-h-0 flex-1 overflow-y-auto border-t border-slate-200">
				{#if loadingPeople}
					<p class="p-6 text-sm text-slate-500">Loading people…</p>
				{:else if filtered.length === 0}
					<p class="p-6 text-sm text-slate-500">
						{candidates.length === 0
							? 'Everyone is already assigned to this project.'
							: 'No one matches your search.'}
					</p>
				{:else}
					<table class="w-full text-left text-sm">
						<thead class="sticky top-0 border-b border-slate-200 bg-slate-50 text-slate-600">
							<tr>
								<th class="w-10 px-4 py-2">
									<input
										type="checkbox"
										aria-label="Select all visible"
										checked={allVisibleSelected}
										onchange={toggleAllVisible}
									/>
								</th>
								<th class="px-2 py-2 font-medium">Person</th>
								<th class="hidden px-2 py-2 font-medium sm:table-cell">Title</th>
								<th class="px-2 py-2 font-medium">Elsewhere</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-slate-100">
							{#each filtered as person (person.id)}
								{@const otherPct = elsewhereAllocation[person.id] ?? 0}
								{@const otherCount = elsewhereCount[person.id] ?? 0}
								{@const overCapacity = otherPct + (selected.has(person.id) ? allocationPct : 0) > 100}
								<tr
									class="cursor-pointer hover:bg-slate-50 {selected.has(person.id) ? 'bg-slate-50' : ''}"
									onclick={() => toggleOne(person.id)}
								>
									<td class="px-4 py-2">
										<input
											type="checkbox"
											checked={selected.has(person.id)}
											onclick={(e) => e.stopPropagation()}
											onchange={() => toggleOne(person.id)}
											aria-label={`Select ${fullName(person)}`}
										/>
									</td>
									<td class="px-2 py-2 font-medium text-slate-800">
										{fullName(person)}
										{#if person.title}<span class="block text-xs font-normal text-slate-500 sm:hidden">{person.title}</span>{/if}
										<span class="block text-xs font-normal break-all text-slate-500">{person.email}</span>
									</td>
									<td class="hidden px-2 py-2 text-slate-600 sm:table-cell">{person.title ?? '—'}</td>
									<td class="px-2 py-2">
										{#if otherCount > 0}
											<span class="text-xs {overCapacity ? 'font-medium text-amber-700' : 'text-slate-500'}">
												{otherPct}% / {otherCount}
												{otherCount === 1 ? 'project' : 'projects'}
											</span>
										{:else}
											<span class="text-xs text-slate-500">free</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>

			<div class="flex items-center justify-between border-t border-slate-200 px-6 py-4">
				<span class="text-sm text-slate-500">
					{selected.size} selected
				</span>
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={saving || selected.size === 0}>
						{saving ? 'Adding…' : `Add ${selected.size || ''} to project`}
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
