<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonFormModal from '$lib/components/PersonFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import type { BadgeTone } from '$lib/badgeTones';
	import { formatManDays, loadBand, monthKey, monthLabel, shiftMonth } from '$lib/capacity';

	type Person = components['schemas']['Person'];
	type PersonStatus = components['schemas']['PersonStatus'];
	type CapacityReport = components['schemas']['CapacityReport'];

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

	// Workload = man-days planned for the selected month ÷ the person's
	// monthly capacity (GET /api/v1/capacity spreads each task over its
	// working days). Inactive people aren't in the report: their capacity
	// comes from the person and only the current month's figure is known.
	const thisMonth = monthKey(new Date());
	let month = $state(thisMonth);
	let report = $state<CapacityReport | null>(null);

	async function loadCapacity() {
		const { data } = await api.GET('/api/v1/capacity', { params: { query: { month } } });
		report = data ?? null;
	}

	function changeMonth(delta: number) {
		month = shiftMonth(month, delta);
		loadCapacity();
	}

	const byPerson = $derived(new Map((report?.people ?? []).map((p) => [p.personId, p])));

	function workloadOf(person: Person): { planned: number; capacity: number; pct: number } {
		const r = byPerson.get(person.id);
		const capacity = r?.capacityManDays ?? person.monthlyCapacityManDays ?? 20;
		const planned = r?.plannedManDays ?? (month === thisMonth ? (person.monthManDays ?? 0) : 0);
		return { planned, capacity, pct: capacity > 0 ? (planned / capacity) * 100 : 0 };
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

	onMount(() => {
		load();
		loadCapacity();
	});
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
		<div class="flex items-center gap-1 sm:ml-auto" role="group" aria-label="Workload month">
			<button type="button" class="btn-ghost px-2" aria-label="Previous month" onclick={() => changeMonth(-1)}>‹</button>
			<span class="min-w-32 text-center text-sm font-medium text-slate-700">{monthLabel(month)}</span>
			<button type="button" class="btn-ghost px-2" aria-label="Next month" onclick={() => changeMonth(1)}>›</button>
		</div>
	</div>

	{#if report}
		<p class="text-sm text-slate-600">
			Team load in {monthLabel(month)}:
			<span class="font-semibold text-slate-900 tabular-nums">{formatManDays(report.plannedManDays)}</span>
			of
			<span class="font-semibold text-slate-900 tabular-nums">{formatManDays(report.capacityManDays)}</span>
			man-days planned ({report.workingDays} working days).
		</p>
	{/if}

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
		<ul class="m-list">
			{#each filtered as person (person.id)}
				{@const w = workloadOf(person)}
				{@const band = loadBand(w.pct)}
				<li class="m-card">
					<button type="button" class="block w-full text-left" onclick={() => openEdit(person)}>
						<div class="flex items-start justify-between gap-2">
							<div class="min-w-0">
								<p class="truncate font-medium text-slate-900">{fullName(person)}</p>
								<p class="truncate text-slate-500">{person.title ?? '—'}</p>
							</div>
							<Badge tone={statusTone(person.status)}>{person.status.replace('_', ' ')}</Badge>
						</div>
						<p class="mt-1 truncate text-slate-600">{person.email}</p>
						<div class="m-meta">
							{#if person.seniority}<span>{person.seniority}</span>{/if}
							<span>{person.employmentType.replace('_', ' ')}</span>
						</div>
						<div class="mt-2 flex items-center gap-2">
							<div
								class="h-2 flex-1 overflow-hidden rounded-full bg-slate-100"
								role="meter"
								aria-label="Workload"
								aria-valuemin={0}
								aria-valuemax={100}
								aria-valuenow={Math.round(w.pct)}
								aria-valuetext="{formatManDays(w.planned)} of {formatManDays(w.capacity)} man-days, {band.label.toLowerCase()}"
							>
								<div class="h-full rounded-full {band.bar}" style="width: {Math.min(w.pct, 100)}%"></div>
							</div>
							<span class="text-xs font-medium tabular-nums {band.text}">
								{formatManDays(w.planned)} / {formatManDays(w.capacity)} md
							</span>
						</div>
					</button>
					<div class="m-actions">
						<button type="button" class="btn-secondary" onclick={() => openEdit(person)}>Edit</button>
						<button
							type="button"
							class="btn-secondary text-red-600 hover:bg-red-50"
							onclick={() => askDelete(person)}
						>
							Delete
						</button>
					</div>
				</li>
			{/each}
		</ul>

		<div class="table-wrap">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
					<tr>
						<th class="px-4 py-3 font-medium">Name</th>
						<th class="px-4 py-3 font-medium">Email</th>
						<th class="px-4 py-3 font-medium">Title</th>
						<th class="px-4 py-3 font-medium">Seniority</th>
						<th class="px-4 py-3 font-medium">Employment</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium">Workload ({monthLabel(month)})</th>
						<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each filtered as person (person.id)}
				{@const w = workloadOf(person)}
				{@const band = loadBand(w.pct)}
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
								<div
									class="flex items-center gap-2"
									title="{band.label}: {formatManDays(w.planned)} of {formatManDays(w.capacity)} man-days ({Math.round(w.pct)}%)"
								>
									<div
										class="h-2 w-24 overflow-hidden rounded-full bg-slate-100"
										role="meter"
										aria-label="Workload"
										aria-valuemin={0}
										aria-valuemax={100}
										aria-valuenow={Math.round(w.pct)}
										aria-valuetext="{formatManDays(w.planned)} of {formatManDays(w.capacity)} man-days, {band.label.toLowerCase()}"
									>
										<div class="h-full rounded-full {band.bar}" style="width: {Math.min(w.pct, 100)}%"></div>
									</div>
									<span class="text-xs whitespace-nowrap tabular-nums font-medium {band.text}">
										{formatManDays(w.planned)} / {formatManDays(w.capacity)} md
									</span>
								</div>
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
