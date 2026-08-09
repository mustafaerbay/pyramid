<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ContributionFormModal from '$lib/components/ContributionFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import type { BadgeTone } from '$lib/badgeTones';

	type Contribution = components['schemas']['Contribution'];
	type ContributionType = components['schemas']['ContributionType'];
	type Project = components['schemas']['Project'];

	const pageSize = 50;

	let contributions = $state<Contribution[]>([]);
	let total = $state(0);
	let currentPage = $state(1);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let typeFilter = $state<ContributionType | ''>('');
	let peopleNames = $state<Record<string, string>>({});
	let projectKeys = $state<Record<string, string>>({});

	let modalMode = $state<'create' | 'edit' | null>(null);
	let editingContribution = $state<Contribution | null>(null);

	let deletingContribution = $state<Contribution | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function loadLookups() {
		const [peopleRes, projectsRes] = await Promise.all([
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } })
		]);
		if (peopleRes.data) {
			const map: Record<string, string> = {};
			for (const p of peopleRes.data.data) map[p.id] = `${p.firstName} ${p.lastName}`;
			peopleNames = map;
		}
		if (projectsRes.data) {
			const map: Record<string, string> = {};
			for (const p of projectsRes.data.data as Project[]) map[p.id] = p.key;
			projectKeys = map;
		}
	}

	async function load() {
		loading = true;
		error = null;

		const { data, error: err } = await api.GET('/api/v1/contributions', {
			params: {
				query: {
					page: currentPage,
					pageSize,
					...(typeFilter ? { type: typeFilter } : {})
				}
			}
		});

		if (err) {
			error = err.message ?? 'Failed to load contributions';
		} else if (data) {
			contributions = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

	function personName(id: string): string {
		return peopleNames[id] ?? 'Unknown person';
	}

	function openCreate() {
		editingContribution = null;
		modalMode = 'create';
	}

	function openEdit(c: Contribution) {
		editingContribution = c;
		modalMode = 'edit';
	}

	function closeModal() {
		modalMode = null;
		editingContribution = null;
	}

	function handleSaved(saved: Contribution) {
		const wasEdit = editingContribution !== null;
		closeModal();
		flash('success', wasEdit ? 'Contribution updated.' : `Logged a contribution for ${personName(saved.personId)}.`);
		load();
	}

	function askDelete(c: Contribution) {
		deleteError = null;
		deletingContribution = c;
	}

	async function confirmDelete() {
		if (!deletingContribution) return;
		const target = deletingContribution;
		deleting = true;
		deleteError = null;

		const { error: err } = await api.DELETE('/api/v1/contributions/{id}', {
			params: { path: { id: target.id } }
		});

		deleting = false;
		if (err) {
			deleteError = err.message ?? 'Failed to delete contribution.';
			return;
		}
		deletingContribution = null;
		flash('success', 'Contribution removed.');
		load();
	}

	function applyTypeFilter(value: string) {
		typeFilter = value as ContributionType | '';
		currentPage = 1;
		load();
	}

	function typeTone(type: ContributionType): BadgeTone {
		switch (type) {
			case 'commit':
				return 'neutralMuted';
			case 'pull_request':
				return 'violet';
			case 'code_review':
				return 'info';
			case 'issue':
				return 'warning';
			case 'deployment':
				return 'success';
			default:
				return 'neutral';
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
		loadLookups();
	});
</script>

<section class="space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Contributions</h1>
			<p class="mt-0.5 text-sm text-slate-500">
				{total}
				{total === 1 ? 'entry' : 'entries'} in the activity feed
			</p>
		</div>
		<button type="button" class="btn-primary" onclick={openCreate}>+ Log contribution</button>
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
		<select
			class="input w-auto"
			value={typeFilter}
			onchange={(e) => applyTypeFilter(e.currentTarget.value)}
		>
			<option value="">All types</option>
			<option value="commit">Commit</option>
			<option value="pull_request">Pull request</option>
			<option value="code_review">Code review</option>
			<option value="issue">Issue</option>
			<option value="design">Design</option>
			<option value="documentation">Documentation</option>
			<option value="deployment">Deployment</option>
			<option value="support">Support</option>
			<option value="other">Other</option>
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
	{:else if contributions.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			No contributions logged yet. Click "Log contribution" to add one.
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
					<tr>
						<th class="px-4 py-3 font-medium">Person</th>
						<th class="px-4 py-3 font-medium">Type</th>
						<th class="px-4 py-3 font-medium">Title</th>
						<th class="px-4 py-3 font-medium">Project</th>
						<th class="px-4 py-3 font-medium">Date</th>
						<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each contributions as contribution (contribution.id)}
						<tr class="hover:bg-slate-50">
							<td class="px-4 py-3 font-medium">{personName(contribution.personId)}</td>
							<td class="px-4 py-3">
								<Badge tone={typeTone(contribution.type)}>{contribution.type.replace('_', ' ')}</Badge>
							</td>
							<td class="px-4 py-3 text-slate-600">
								{#if contribution.url}
									<a
										href={contribution.url}
										target="_blank"
										rel="noopener noreferrer"
										class="text-slate-700 underline decoration-slate-300 hover:decoration-slate-500"
									>
										{contribution.title ?? contribution.url}
									</a>
								{:else}
									{contribution.title ?? '—'}
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-xs text-slate-500">
								{contribution.projectId ? (projectKeys[contribution.projectId] ?? '—') : '—'}
							</td>
							<td class="px-4 py-3 text-slate-600">{contribution.occurredAt.slice(0, 10)}</td>
							<td class="px-4 py-3 text-right whitespace-nowrap">
								<button
									type="button"
									class="btn-ghost px-2 py-1"
									onclick={() => openEdit(contribution)}
								>
									Edit
								</button>
								<button
									type="button"
									class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
									onclick={() => askDelete(contribution)}
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
	<ContributionFormModal
		contribution={editingContribution}
		onClose={closeModal}
		onSaved={handleSaved}
	/>
{/if}

<ConfirmDialog
	open={deletingContribution !== null}
	title="Delete contribution"
	message="Remove this contribution entry? This can't be undone."
	confirmLabel={deleting ? 'Deleting…' : 'Delete'}
	danger
	error={deleteError}
	onConfirm={confirmDelete}
	onCancel={() => (deletingContribution = null)}
/>
