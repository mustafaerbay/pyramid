<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import GoalFormModal from '$lib/components/GoalFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import type { BadgeTone } from '$lib/badgeTones';

	type Goal = components['schemas']['Goal'];
	type GoalStatus = components['schemas']['GoalStatus'];

	const pageSize = 50;

	let goals = $state<Goal[]>([]);
	let total = $state(0);
	let currentPage = $state(1);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let search = $state('');
	let statusFilter = $state<GoalStatus | ''>('');

	let peopleNames = $state<Record<string, string>>({});

	let modalMode = $state<'create' | 'edit' | null>(null);
	let editingGoal = $state<Goal | null>(null);

	let deletingGoal = $state<Goal | null>(null);
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

		const { data, error: err } = await api.GET('/api/v1/goals', {
			params: {
				query: {
					page: currentPage,
					pageSize,
					...(statusFilter ? { status: statusFilter } : {})
				}
			}
		});

		if (err) {
			error = err.message ?? 'Failed to load goals';
		} else if (data) {
			goals = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	const filtered = $derived(
		goals.filter((g) => {
			if (!search.trim()) return true;
			const q = search.trim().toLowerCase();
			return g.title.toLowerCase().includes(q) || (g.description ?? '').toLowerCase().includes(q);
		})
	);

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

	function openCreate() {
		editingGoal = null;
		modalMode = 'create';
	}

	function openEdit(g: Goal) {
		editingGoal = g;
		modalMode = 'edit';
	}

	function closeModal() {
		modalMode = null;
		editingGoal = null;
	}

	function handleSaved(saved: Goal) {
		const wasEdit = editingGoal !== null;
		closeModal();
		flash('success', wasEdit ? `${saved.title} was updated.` : `${saved.title} was added.`);
		load();
	}

	function askDelete(g: Goal) {
		deleteError = null;
		deletingGoal = g;
	}

	async function confirmDelete() {
		if (!deletingGoal) return;
		const target = deletingGoal;
		deleting = true;
		deleteError = null;

		const { error: err } = await api.DELETE('/api/v1/goals/{id}', {
			params: { path: { id: target.id } }
		});

		deleting = false;
		if (err) {
			deleteError = err.message ?? 'Failed to delete goal.';
			return;
		}
		deletingGoal = null;
		flash('success', `${target.title} was removed.`);
		load();
	}

	function applyStatusFilter(value: string) {
		statusFilter = value as GoalStatus | '';
		currentPage = 1;
		load();
	}

	function statusTone(status: GoalStatus): BadgeTone {
		switch (status) {
			case 'active':
				return 'success';
			case 'draft':
				return 'neutralMuted';
			case 'achieved':
				return 'info';
			case 'partially_achieved':
				return 'warning';
			case 'missed':
			case 'cancelled':
				return 'danger';
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
			<h1 class="text-2xl font-semibold tracking-tight">Growth</h1>
			<p class="mt-0.5 text-sm text-slate-500">
				{total}
				{total === 1 ? 'goal' : 'goals'}
			</p>
		</div>
		<button data-write type="button" class="btn-primary" onclick={openCreate}>+ Add goal</button>
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
		<input class="input max-w-xs" type="search" aria-label="Search goals" placeholder="Search goals…" bind:value={search} />
		<select
			class="input w-auto"
			aria-label="Filter goals by status"
			value={statusFilter}
			onchange={(e) => applyStatusFilter(e.currentTarget.value)}
		>
			<option value="">All statuses</option>
			<option value="draft">Draft</option>
			<option value="active">Active</option>
			<option value="achieved">Achieved</option>
			<option value="partially_achieved">Partially achieved</option>
			<option value="missed">Missed</option>
			<option value="cancelled">Cancelled</option>
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
			{#if goals.length === 0}
				No goals yet. Click "Add goal" to create the first one.
			{:else}
				No goals match your search.
			{/if}
		</div>
	{:else}
		<div class="space-y-3">
			{#each filtered as goal (goal.id)}
				<div class="rounded-lg border border-slate-200 bg-white p-4">
					<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
						<div>
							<div class="flex items-center gap-2">
								<h3 class="font-medium text-slate-900">{goal.title}</h3>
								<Badge tone={statusTone(goal.status)}>{goal.status.replace('_', ' ')}</Badge>
								<Badge tone="neutral" capitalize>{goal.scope}</Badge>
							</div>
							{#if goal.description}
								<p class="mt-1 text-sm text-slate-500">{goal.description}</p>
							{/if}
							<p class="mt-1 text-xs text-slate-500">
								{goal.ownerPersonId ? (peopleNames[goal.ownerPersonId] ?? 'Unknown owner') : 'Unassigned'}
								{#if goal.dueDate}
									· Due {goal.dueDate}
								{/if}
							</p>
						</div>
						<div class="flex shrink-0 gap-1 max-sm:order-last max-sm:gap-2 max-sm:border-t max-sm:border-slate-100 max-sm:pt-3 max-sm:[&>*]:flex-1">
							<button data-write type="button" class="btn-ghost px-2 py-1" onclick={() => openEdit(goal)}>
								Edit
							</button>
							<button data-write
								type="button"
								class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
								onclick={() => askDelete(goal)}
							>
								Delete
							</button>
						</div>
					</div>
					<div class="mt-3 flex items-center gap-2">
						<div class="h-1.5 flex-1 overflow-hidden rounded-full bg-slate-100">
							<div
								class="h-full rounded-full bg-slate-900"
								style="width: {goal.progressPct}%"
							></div>
						</div>
						<span class="w-10 text-right text-xs text-slate-500">{goal.progressPct}%</span>
					</div>
				</div>
			{/each}
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
	<GoalFormModal goal={editingGoal} onClose={closeModal} onSaved={handleSaved} />
{/if}

<ConfirmDialog
	open={deletingGoal !== null}
	title="Delete goal"
	message={deletingGoal ? `Remove "${deletingGoal.title}"? This can't be undone.` : ''}
	confirmLabel={deleting ? 'Deleting…' : 'Delete'}
	danger
	error={deleteError}
	onConfirm={confirmDelete}
	onCancel={() => (deletingGoal = null)}
/>
