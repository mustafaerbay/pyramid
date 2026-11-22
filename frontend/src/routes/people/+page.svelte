<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonFormModal from '$lib/components/PersonFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import type { BadgeTone } from '$lib/badgeTones';

	type Person = components['schemas']['Person'];
	type PersonStatus = components['schemas']['PersonStatus'];

	const pageSize = 50;

	let people = $state<Person[]>([]);
	let total = $state(0);
	let currentPage = $state(1);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let search = $state('');
	let statusFilter = $state<PersonStatus | ''>('');

	let modalMode = $state<'create' | 'edit' | null>(null);
	let editingPerson = $state<Person | null>(null);

	let deletingPerson = $state<Person | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function load() {
		loading = true;
		error = null;

		const { data, error: err } = await api.GET('/api/v1/people', {
			params: {
				query: {
					page: currentPage,
					pageSize,
					...(statusFilter ? { status: statusFilter } : {})
				}
			}
		});

		if (err) {
			error = err.message ?? 'Failed to load people';
		} else if (data) {
			people = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	// Search filters the currently loaded page client-side; the API doesn't
	// take a free-text query param yet, only status/departmentId.
	const filtered = $derived(
		people.filter((p) => {
			if (!search.trim()) return true;
			const q = search.trim().toLowerCase();
			return (
				`${p.firstName} ${p.lastName}`.toLowerCase().includes(q) ||
				p.email.toLowerCase().includes(q) ||
				(p.title ?? '').toLowerCase().includes(q)
			);
		})
	);

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

	function fullName(p: Person): string {
		return `${p.firstName} ${p.lastName}`;
	}

	function openCreate() {
		editingPerson = null;
		modalMode = 'create';
	}

	function openEdit(p: Person) {
		editingPerson = p;
		modalMode = 'edit';
	}

	function closeModal() {
		modalMode = null;
		editingPerson = null;
	}

	function handleSaved(saved: Person) {
		const wasEdit = editingPerson !== null;
		closeModal();
		flash('success', wasEdit ? `${fullName(saved)} was updated.` : `${fullName(saved)} was added.`);
		load();
	}

	function askDelete(p: Person) {
		deleteError = null;
		deletingPerson = p;
	}

	async function confirmDelete() {
		if (!deletingPerson) return;
		const target = deletingPerson;
		deleting = true;
		deleteError = null;

		const { error: err } = await api.DELETE('/api/v1/people/{id}', {
			params: { path: { id: target.id } }
		});

		deleting = false;
		if (err) {
			deleteError = err.message ?? 'Failed to delete person.';
			return;
		}
		deletingPerson = null;
		flash('success', `${fullName(target)} was removed.`);
		load();
	}

	function applyStatusFilter(value: string) {
		statusFilter = value as PersonStatus | '';
		currentPage = 1;
		load();
	}

	function statusTone(status: PersonStatus): BadgeTone {
		switch (status) {
			case 'active':
				return 'success';
			case 'on_leave':
				return 'warning';
			case 'offboarding':
				return 'orange';
			case 'inactive':
				return 'neutralMuted';
		}
	}

	// Workload = sum of allocation % on current, open projects (server-side).
	// Bands: idle/light < 70, healthy 70–100, over-allocated > 100.
	function workloadBand(pct: number): { bar: string; text: string; label: string } {
		if (pct > 100) return { bar: 'bg-red-500', text: 'text-red-700', label: 'Over-allocated' };
		if (pct >= 70) return { bar: 'bg-emerald-500', text: 'text-slate-700', label: 'Healthy load' };
		return { bar: 'bg-slate-400', text: 'text-slate-500', label: 'Has capacity' };
	}

	function prevPage() {
		if (currentPage > 1) {
			currentPage -= 1;
			load();
		}
	}

	function nextPage() {
		if (currentPage < totalPages) {
			currentPage += 1;
			load();
		}
	}

	onMount(load);
</script>

<section class="space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">People</h1>
			<p class="mt-0.5 text-sm text-slate-500">
				{total}
				{total === 1 ? 'person' : 'people'} in the department
			</p>
		</div>
		<button type="button" class="btn-primary" onclick={openCreate}>+ Add person</button>
	</div>

	{#if banner}
		<div
			class="rounded-md border px-3 py-2 text-sm {banner.kind === 'success'
				? 'border-emerald-200 bg-emerald-50 text-emerald-700'
				: 'border-red-200 bg-red-50 text-red-700'}"
		>
			{banner.text}
		</div>
	{/if}

	<div class="flex flex-wrap items-center gap-3">
		<input
			class="input max-w-xs"
			type="search"
			placeholder="Search by name, email, or title…"
			bind:value={search}
		/>
		<select
			class="input w-auto"
			value={statusFilter}
			onchange={(e) => applyStatusFilter(e.currentTarget.value)}
		>
			<option value="">All statuses</option>
			<option value="active">Active</option>
			<option value="on_leave">On leave</option>
			<option value="offboarding">Offboarding</option>
			<option value="inactive">Inactive</option>
		</select>
	</div>

	{#if loading}
		<p class="text-slate-500">Loading…</p>
	{:else if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
			<p class="font-medium">Couldn't reach the API.</p>
			<p class="text-sm">{error}</p>
			<p class="mt-2 text-sm text-red-600">
				Make sure the Go backend is running on <code>localhost:8080</code>.
			</p>
		</div>
	{:else if filtered.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			{#if people.length === 0}
				No people yet. Click "Add person" to create the first one.
			{:else}
				No people match your search.
			{/if}
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
					<tr>
						<th class="px-4 py-3 font-medium">Name</th>
						<th class="px-4 py-3 font-medium">Email</th>
						<th class="px-4 py-3 font-medium">Title</th>
						<th class="px-4 py-3 font-medium">Seniority</th>
						<th class="px-4 py-3 font-medium">Employment</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium">Workload</th>
						<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each filtered as person (person.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<tr class="cursor-pointer hover:bg-slate-50" onclick={() => openEdit(person)}>
							<td class="px-4 py-3 font-medium">{fullName(person)}</td>
							<td class="px-4 py-3 text-slate-600">{person.email}</td>
							<td class="px-4 py-3 text-slate-600">{person.title ?? '—'}</td>
							<td class="px-4 py-3 text-slate-600">{person.seniority ?? '—'}</td>
							<td class="px-4 py-3 text-slate-600">{person.employmentType.replace('_', ' ')}</td>
							<td class="px-4 py-3">
								<Badge tone={statusTone(person.status)}>{person.status.replace('_', ' ')}</Badge>
							</td>
							<td class="px-4 py-3">
								{#if person.workloadPct != null}
									{@const pct = person.workloadPct}
									{@const band = workloadBand(pct)}
									<div class="flex items-center gap-2" title="{band.label}: {Math.round(pct)}% allocated">
										<div
											class="h-2 w-24 overflow-hidden rounded-full bg-slate-100"
											role="meter"
											aria-label="Workload"
											aria-valuemin={0}
											aria-valuemax={100}
											aria-valuenow={Math.round(pct)}
											aria-valuetext="{Math.round(pct)}% allocated, {band.label.toLowerCase()}"
										>
											<div
												class="h-full rounded-full {band.bar}"
												style="width: {Math.min(pct, 100)}%"
											></div>
										</div>
										<span class="w-10 text-right text-xs tabular-nums font-medium {band.text}">
											{Math.round(pct)}%
										</span>
									</div>
								{:else}
									<span class="text-slate-400">—</span>
								{/if}
							</td>
							<td
								class="px-4 py-3 text-right whitespace-nowrap"
								onclick={(e) => e.stopPropagation()}
							>
								<button type="button" class="btn-ghost px-2 py-1" onclick={() => openEdit(person)}>
									Edit
								</button>
								<button
									type="button"
									class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
									onclick={() => askDelete(person)}
								>
									Delete
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if totalPages > 1}
			<div class="flex items-center justify-between pt-2 text-sm text-slate-600">
				<span>Page {currentPage} of {totalPages}</span>
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" disabled={currentPage <= 1} onclick={prevPage}>
						Previous
					</button>
					<button
						type="button"
						class="btn-secondary"
						disabled={currentPage >= totalPages}
						onclick={nextPage}
					>
						Next
					</button>
				</div>
			</div>
		{/if}
	{/if}
</section>

{#if modalMode}
	<PersonFormModal person={editingPerson} onClose={closeModal} onSaved={handleSaved} />
{/if}

<ConfirmDialog
	open={deletingPerson !== null}
	title="Delete person"
	message={deletingPerson ? `Remove ${fullName(deletingPerson)}? This can't be undone.` : ''}
	confirmLabel={deleting ? 'Deleting…' : 'Delete'}
	danger
	error={deleteError}
	onConfirm={confirmDelete}
	onCancel={() => (deletingPerson = null)}
/>
