<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import LeaveRequestFormModal from '$lib/components/LeaveRequestFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';

	type LeaveRequest = components['schemas']['LeaveRequest'];
	type LeaveStatus = components['schemas']['LeaveStatus'];

	const pageSize = 50;

	let requests = $state<LeaveRequest[]>([]);
	let total = $state(0);
	let currentPage = $state(1);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Toggleable status chips, not a dropdown — matches the Contributions
	// page's type filter (see typeFilters there). Empty array = no filter
	// (show every status). Server-side filter: every toggle resets to page 1
	// and re-fetches, since this page paginates on the backend.
	let statusFilters = $state<LeaveStatus[]>([]);
	const statusFilterOptions: { value: LeaveStatus; label: string }[] = [
		{ value: 'pending', label: 'Pending' },
		{ value: 'approved', label: 'Approved' },
		{ value: 'rejected', label: 'Rejected' },
		{ value: 'cancelled', label: 'Cancelled' }
	];
	let peopleNames = $state<Record<string, string>>({});

	let modalMode = $state<'create' | 'edit' | null>(null);
	let editingRequest = $state<LeaveRequest | null>(null);

	let deletingRequest = $state<LeaveRequest | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function loadPeopleNames() {
		const { data } = await api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } });
		if (data) {
			const map: Record<string, string> = {};
			for (const p of data.data) map[p.id] = `${p.firstName} ${p.lastName}`;
			peopleNames = map;
		}
	}

	async function load() {
		loading = true;
		error = null;

		const { data, error: err } = await api.GET('/api/v1/leave-requests', {
			params: {
				query: {
					page: currentPage,
					pageSize,
					...(statusFilters.length > 0 ? { status: statusFilters } : {})
				}
			}
		});

		if (err) {
			error = err.message ?? 'Failed to load leave requests';
		} else if (data) {
			requests = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

	function personName(id: string): string {
		return peopleNames[id] ?? 'Unknown person';
	}

	function openCreate() {
		editingRequest = null;
		modalMode = 'create';
	}

	function openEdit(r: LeaveRequest) {
		editingRequest = r;
		modalMode = 'edit';
	}

	function closeModal() {
		modalMode = null;
		editingRequest = null;
	}

	function handleSaved(saved: LeaveRequest) {
		const wasEdit = editingRequest !== null;
		closeModal();
		flash(
			'success',
			wasEdit ? `Leave request for ${personName(saved.personId)} was updated.` : 'Leave request submitted.'
		);
		load();
	}

	async function setStatus(r: LeaveRequest, status: LeaveStatus) {
		const { data, error: err } = await api.PATCH('/api/v1/leave-requests/{id}', {
			params: { path: { id: r.id } },
			body: { status }
		});
		if (err) {
			flash('error', err.message ?? 'Failed to update status.');
			return;
		}
		if (data) {
			flash('success', `Marked as ${status}.`);
			load();
		}
	}

	function askDelete(r: LeaveRequest) {
		deleteError = null;
		deletingRequest = r;
	}

	async function confirmDelete() {
		if (!deletingRequest) return;
		const target = deletingRequest;
		deleting = true;
		deleteError = null;

		const { error: err } = await api.DELETE('/api/v1/leave-requests/{id}', {
			params: { path: { id: target.id } }
		});

		deleting = false;
		if (err) {
			deleteError = err.message ?? 'Failed to delete leave request.';
			return;
		}
		deletingRequest = null;
		flash('success', 'Leave request removed.');
		load();
	}

	function toggleStatusFilter(status: LeaveStatus) {
		statusFilters = statusFilters.includes(status)
			? statusFilters.filter((s) => s !== status)
			: [...statusFilters, status];
		currentPage = 1;
		load();
	}

	function clearStatusFilters() {
		statusFilters = [];
		currentPage = 1;
		load();
	}

	function statusTone(status: LeaveStatus): BadgeTone {
		switch (status) {
			case 'approved':
				return 'success';
			case 'pending':
				return 'warning';
			case 'rejected':
				return 'danger';
			case 'cancelled':
				return 'neutralMuted';
		}
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
		loadPeopleNames();
	});
</script>

<section class="space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Time Off</h1>
			<p class="mt-0.5 text-sm text-slate-500">
				{total}
				{total === 1 ? 'request' : 'requests'}
			</p>
		</div>
		<button data-write type="button" class="btn-primary" onclick={openCreate}>+ New request</button>
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

	<div class="flex flex-wrap items-center gap-1.5" role="group" aria-label="Filter by status">
		{#each statusFilterOptions as opt (opt.value)}
			{@const selected = statusFilters.includes(opt.value)}
			<button
				type="button"
				class="rounded-full px-2.5 py-1 text-xs font-medium transition-colors {selected
					? badgeToneClass[statusTone(opt.value)] + ' ring-1 ring-inset ring-transparent'
					: 'bg-white text-slate-500 ring-1 ring-inset ring-slate-200 hover:bg-slate-50'}"
				aria-pressed={selected}
				onclick={() => toggleStatusFilter(opt.value)}
			>
				{opt.label}
			</button>
		{/each}
		{#if statusFilters.length > 0}
			<button
				type="button"
				class="px-1 text-xs text-slate-500 hover:text-slate-700 hover:underline"
				onclick={clearStatusFilters}
			>
				Clear
			</button>
		{/if}
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
	{:else if requests.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			No leave requests yet. Click "New request" to submit one.
		</div>
	{:else}
		<ul class="m-list">
			{#each requests as request (request.id)}
				<li class="m-card">
					<div class="flex items-start justify-between gap-2">
						<p class="font-medium text-slate-900">{personName(request.personId)}</p>
						<Badge tone={statusTone(request.status)}>{request.status}</Badge>
					</div>
					<div class="m-meta">
						<span class="capitalize">{request.type}</span>
						<span>{request.startDate} → {request.endDate}</span>
					</div>
					{#if request.status === 'pending'}
						<div class="m-actions">
							<button data-write
								type="button"
								class="btn-secondary text-emerald-700 hover:bg-emerald-50"
								onclick={() => setStatus(request, 'approved')}
							>
								Approve
							</button>
							<button data-write
								type="button"
								class="btn-secondary text-red-600 hover:bg-red-50"
								onclick={() => setStatus(request, 'rejected')}
							>
								Reject
							</button>
						</div>
					{/if}
					<div class="m-actions {request.status === 'pending' ? 'mt-2 border-t-0 pt-0' : ''}">
						<button data-write type="button" class="btn-secondary" onclick={() => openEdit(request)}>Edit</button>
						<button data-write
							type="button"
							class="btn-secondary text-red-600 hover:bg-red-50"
							onclick={() => askDelete(request)}
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
						<th class="px-4 py-3 font-medium">Person</th>
						<th class="px-4 py-3 font-medium">Type</th>
						<th class="px-4 py-3 font-medium">Dates</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each requests as request (request.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<tr class="cursor-pointer hover:bg-slate-50" onclick={() => openEdit(request)}>
							<td class="px-4 py-3 font-medium">{personName(request.personId)}</td>
							<td class="px-4 py-3 text-slate-600 capitalize">{request.type}</td>
							<td class="px-4 py-3 text-slate-600">{request.startDate} → {request.endDate}</td>
							<td class="px-4 py-3">
								<Badge tone={statusTone(request.status)}>{request.status}</Badge>
							</td>
							<td class="px-4 py-3 text-right whitespace-nowrap" onclick={(e) => e.stopPropagation()}>
								{#if request.status === 'pending'}
									<button data-write
										type="button"
										class="btn-ghost px-2 py-1 text-emerald-700 hover:bg-emerald-50 hover:text-emerald-700"
										onclick={() => setStatus(request, 'approved')}
									>
										Approve
									</button>
									<button data-write
										type="button"
										class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
										onclick={() => setStatus(request, 'rejected')}
									>
										Reject
									</button>
								{/if}
								<button data-write type="button" class="btn-ghost px-2 py-1" onclick={() => openEdit(request)}>
									Edit
								</button>
								<button data-write
									type="button"
									class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
									onclick={() => askDelete(request)}
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
	<LeaveRequestFormModal leaveRequest={editingRequest} onClose={closeModal} onSaved={handleSaved} />
{/if}

<ConfirmDialog
	open={deletingRequest !== null}
	title="Delete leave request"
	message={deletingRequest
		? `Remove this leave request for ${personName(deletingRequest.personId)}? This can't be undone.`
		: ''}
	confirmLabel={deleting ? 'Deleting…' : 'Delete'}
	danger
	error={deleteError}
	onConfirm={confirmDelete}
	onCancel={() => (deletingRequest = null)}
/>
